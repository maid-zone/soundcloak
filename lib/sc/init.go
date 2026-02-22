package sc

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"github.com/dlclark/regexp2"
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/net/http/httpproxy"
)

var ProxyErr = errors.New("could not connect to proxy")
var parsedproxy string

// don't jus leak the proxy like that lol
func scrub(err error) error {
	if cfg.SoundcloudApiProxy != "" && err != nil {
		if parsedproxy == "" {
			u, err := url.Parse(cfg.SoundcloudApiProxy)
			if err == nil {
				parsedproxy = u.Host
			} else {
				parsedproxy = cfg.SoundcloudApiProxy
			}
		}
		s := err.Error()
		if strings.HasPrefix(s, "could not connect to proxyAddr") || strings.HasPrefix(s, "socks connect") || strings.Contains(s, parsedproxy) {
			return ProxyErr
		}
	}

	return err
}

var ClientID string
var Version string

const api = "api-v2.soundcloud.com"

const H = len("https://" + api)

var newline = []byte("\n")

const sc_version = `<script>window.__sc_version="`
const sc_hydration = `<script>window.__sc_hydration = `
const script0 = `<script crossorigin src="https://a-v2.sndcdn.com/assets/0-`
const script = `<script crossorigin src="https://a-v2.sndcdn.com/assets/`

var tlsConfig = &tls.Config{
	CipherSuites: []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	},
}

var httpc = &fasthttp.HostClient{
	Addr:                api + ":443",
	IsTLS:               true,
	MaxIdleConnDuration: cfg.MaxIdleConnDuration,
	DialDualStack:       cfg.DialDualStack,
	TLSConfig:           tlsConfig,
}

var genericClient = &fasthttp.Client{
	DialDualStack: cfg.DialDualStack,
	TLSConfig:     tlsConfig,
}

// var verRegex = regexp2.MustCompile(`^<script>window\.__sc_version="([0-9]{10})"</script>$`, 2)
// var scriptsRegex = regexp2.MustCompile(`^<script crossorigin src="(https://a-v2\.sndcdn\.com/assets/.+\.js)"></script>$`, 2)
// var scriptRegex = regexp2.MustCompile(`^<script crossorigin src="(https://a-v2\.sndcdn\.com/assets/0-.+\.js)"></script>$`, 2)

//go:generate go tool regexp2cg -package sc -o regexp2_codegen.go
var clientIdRegex = regexp2.MustCompile(`client_id:"([A-Za-z0-9]{32})"`, 0) //regexp2.MustCompile(`\("client_id=([A-Za-z0-9]{32})"\)`, 0)
var hydrationClientIdRegex = regexp2.MustCompile(`{"hydratable":"apiClient","data":{"id":"([A-Za-z0-9]{32})"`, 0)
var ErrVersionNotFound = errors.New("version not found")
var ErrScriptNotFound = errors.New("script not found")
var ErrIDNotFound = errors.New("clientid not found")
var ErrKindNotCorrect = errors.New("entity of incorrect kind")

type cached[T any] struct {
	Value   T
	Expires time.Time
}

// don't be spooked by misc.Log, it will be removed during compilation if cfg.Debug == false
func processFile(wg *sync.WaitGroup, ch chan string, uri []byte, isDone *bool) {
	misc.Log(string(uri))
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURIBytes(uri)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	defer wg.Done()

	if *isDone {
		misc.Log("early 1")
		return
	}

	err := DoWithRetryAll(genericClient, req, resp)
	if err != nil {
		misc.Log(err)
		return
	}

	if *isDone {
		misc.Log("early 2")
		return
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	if *isDone {
		misc.Log("early 3")
		return
	}

	m2, _ := clientIdRegex.FindStringMatch(cfg.B2s(data))
	if m2 != nil {
		g := m2.GroupByNumber(1)
		if g != nil {
			if *isDone {
				misc.Log("early 4")
				return
			}

			ch <- g.String()
			misc.Log("found in", string(uri))
			return
		}
	}

	misc.Log("not found in", string(uri))
}

// dont use
func GetClientID() error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.URI().SetScheme("https")
	req.URI().SetHost("soundcloud.com")
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetryAll(genericClient, req, resp)
	if err != nil {
		return err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var ver string
	var hydration []byte
	for l := range bytes.SplitSeq(data, newline) { // version usually comes earlier, but retest this sometimes !!!
		if ver == "" && len(l) > len(sc_version)+len(`"</script>`) && string(l[:len(sc_version)]) == sc_version {
			ver = cfg.B2s(l[len(sc_version) : len(l)-len(`"</script>`)])
			misc.Log("found ver:", ver)
			if Version != "" && ver == Version {
				misc.Log("clientidcache hit @ ver")
				return nil
			}
		} else if len(l) > len(sc_hydration)+len(`;</script>`) && string(l[:len(sc_hydration)]) == sc_hydration {
			hydration = l[len(sc_hydration) : len(l)-len(`;</script>`)]
			misc.Log("found hydration:", cfg.B2s(hydration))
			break
		}
	}

	if ver == "" {
		return ErrVersionNotFound
	}

	// inspired a bit by 4get
	if hydration != nil {
		m, _ := hydrationClientIdRegex.FindStringMatch(cfg.B2s(hydration))
		if m != nil {
			g := m.GroupByNumber(1)
			if g != nil {
				misc.Log("found using sc_hydration")
				ClientID = g.String()
				Version = ver
				misc.Log(ClientID, Version)
				return nil
			}
		}
	}

	// fallback to searching inside JS scripts, inspired by cobalt
	ch := make(chan string, 1)
	wg := &sync.WaitGroup{}
	done := false

	var scriptUrls = make([][]byte, 0, 9)
	for l := range bytes.SplitSeq(data, newline) {
		if len(l) > len(script)+len(`"></script>`) && string(l[:len(script)]) == script {
			scriptUrls = append(scriptUrls, l[len(`<script crossorigin src="`):len(l)-len(`"></script>`)])
		}
	}

	if ver == "" {
		return ErrVersionNotFound
	}

	if len(scriptUrls) == 0 {
		return ErrScriptNotFound
	}

	for _, s := range scriptUrls {
		wg.Add(1)
		go processFile(wg, ch, s, &done)
	}

	go func() {
		defer func() {
			err := recover()
			misc.Log("-- GetClientID recovered:", err)
		}()

		wg.Wait()

		//time.Sleep(time.Millisecond) // maybe race?
		if !done {
			ch <- ""
		}
	}()

	res := <-ch
	done = true
	close(ch)
	if res == "" {
		err = ErrIDNotFound
	} else {
		ClientID = res
		Version = ver
		misc.Log(ClientID, Version)
	}

	return err
}

// Just retry any kind of errors, why not
func DoWithRetryAll(httpc *fasthttp.Client, req *fasthttp.Request, resp *fasthttp.Response) (err error) {
	for range 10 {
		err = httpc.Do(req, resp)
		if err == nil {
			return nil
		}
	}

	err = scrub(err)

	return
}

// Since the http client is setup to always keep connections idle (great for speed, no need to open a new one everytime), those connections may be closed by soundcloud after some time of inactivity, this ensures that we retry those requests that fail due to the connection closing/timing out
func DoWithRetry(httpc *fasthttp.HostClient, req *fasthttp.Request, resp *fasthttp.Response) (err error) {
	for range 10 {
		err = httpc.Do(req, resp)
		if err == nil {
			return nil
		}

		if err != fasthttp.ErrTimeout &&
			err != fasthttp.ErrDialTimeout &&
			err != fasthttp.ErrTLSHandshakeTimeout &&
			err != fasthttp.ErrConnectionClosed &&
			!os.IsTimeout(err) &&
			!errors.Is(err, syscall.EPIPE) && // EPIPE is "broken pipe" error
			err.Error() != "timeout" {
			err = scrub(err)
			return
		}

		misc.Log("we failed haha", err)
	}

	err = scrub(err)

	return
}

func Resolve(path string, out any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.URI().SetScheme("https")
	req.URI().SetHost(api)
	req.URI().SetPath("/resolve")
	req.URI().QueryArgs().Set("url", "https://soundcloud.com/"+path)
	req.URI().QueryArgs().Set("client_id", ClientID)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("resolve: got status code %d", resp.StatusCode())
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	return json.Unmarshal(data, out)
}

type Paginated[T any] struct {
	NextHref   string        `json:"next_href"`
	Next       *fasthttp.URI `json:"-"`
	Collection []T           `json:"collection"`
	Total      int64         `json:"total_results"`
}

func (p *Paginated[T]) Proceed(shouldUnfold bool) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	oldNext := p.NextHref
	if p.NextHref == "" {
		oldNext = p.Next.String()
		req.SetURI(p.Next)
	} else {
		req.SetRequestURI(p.NextHref)
	}
	req.URI().QueryArgs().Set("client_id", ClientID)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5") // you get captcha without it :)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("paginated.proceed: got status code %d", resp.StatusCode())
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	err = json.Unmarshal(data, p)
	if err != nil {
		return err
	}

	if p.NextHref == oldNext { // prevent loops of nothingness
		p.NextHref = ""
	}

	// in soundcloud api, pagination may not immediately return you something!
	// loading users who haven't released anything recently may require you to do a bunch of requests for nothing :/
	// maybe there could be a way to cache the last useless layer of pagination so soundcloak can start loading from there? might be a bit complicated, but would be great

	// another note: in featured tracks it seems to just be forever stuck after 2-3~ pages so i added a way to disable this behaviour
	if shouldUnfold && len(p.Collection) == 0 && p.NextHref != "" {
		// this will make sure that we actually proceed to something useful and not emptiness
		return p.Proceed(true)
	}

	return nil
}

func TagListParser(taglist string) (res []string) {
	inString := false
	cur := []byte{}
	for i, c := range cfg.S2b(taglist) {
		if c == '"' {
			if i == len(taglist)-1 {
				res = append(res, string(cur))
				return
			}

			inString = !inString
			continue
		}

		if !inString && c == ' ' {
			res = append(res, string(cur))
			cur = cur[:0]
			continue
		}

		cur = append(cur, c)
	}

	if len(cur) != 0 {
		res = append(res, string(cur))
	}

	return
}

type SearchSuggestion struct {
	Query string `json:"query"`
}

func GetSearchSuggestions(query string) ([]string, error) {
	uri := baseUri()
	uri.SetPath("/search/queries")
	uri.QueryArgs().Set("limit", "10")
	uri.QueryArgs().Set("q", query)

	p := Paginated[SearchSuggestion]{Next: uri}
	err := p.Proceed(false)
	if err != nil {
		return nil, err
	}

	l := make([]string, len(p.Collection)) // just so the response is a bit smaller, why not
	for i, r := range p.Collection {
		l[i] = r.Query
	}

	return l, nil
}

// polyglot type struct lol
type UserPlaylistTrack struct {
	Kind      string `json:"kind"` // "playlist" or "system-playlist" or "user" or "track"
	Permalink string `json:"permalink"`

	// User
	Avatar   string `json:"avatar_url"`
	Username string `json:"username"`
	FullName string `json:"full_name"`

	// Playlist/track
	Title  string `json:"title"`
	Author struct {
		Permalink string `string:"permalink"`
		Username  string `json:"username"`
	} `json:"user"`
	Artwork string `json:"artwork_url"`

	// Playlist
	Tracks     []struct{} `json:"tracks"` // stub
	TrackCount int64      `json:"track_count"`
}

func (p UserPlaylistTrack) Href() string {
	switch p.Kind {
	case "system-playlist":
		return "/discover/sets/" + p.Permalink
	case "playlist":
		return "/" + p.Author.Permalink + "/sets/" + p.Permalink
	case "track":
		return "/" + p.Author.Permalink + "/" + p.Permalink
	default:
		return "/" + p.Permalink
	}
}

func (p *UserPlaylistTrack) Fix(prefs cfg.Preferences) {
	switch p.Kind {
	case "user":
		if p.Avatar == "https://a1.sndcdn.com/images/default_avatar_large.png" {
			p.Avatar = ""
		} else {
			p.Avatar = strings.Replace(p.Avatar, "-large.", "-t200x200.", 1)
		}

		if p.Avatar != "" && cfg.ProxyImages && *prefs.ProxyImages {
			p.Avatar = "/_/proxy/images?url=" + url.QueryEscape(p.Avatar)
		}
	default:
		if p.Artwork != "" {
			p.Artwork = strings.Replace(p.Artwork, "-large.", "-t200x200.", 1)
			if cfg.ProxyImages && *prefs.ProxyImages {
				p.Artwork = "/_/proxy/images?url=" + url.QueryEscape(p.Artwork)
			}
		}
	}
}

func (p UserPlaylistTrack) TracksCount() int64 {
	if p.TrackCount != 0 {
		return p.TrackCount
	}

	return int64(len(p.Tracks))
}

func Search(prefs cfg.Preferences, args []byte) (*Paginated[*UserPlaylistTrack], error) {
	uri := baseUri()
	uri.SetPath("/search")
	uri.SetQueryStringBytes(args)
	p := Paginated[*UserPlaylistTrack]{Next: uri}
	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(prefs)
	}

	return &p, nil
}

func init() {
	if cfg.SoundcloudApiProxy != "" {
		d := fasthttpproxy.Dialer{Config: httpproxy.Config{HTTPProxy: cfg.SoundcloudApiProxy, HTTPSProxy: cfg.SoundcloudApiProxy}, DialDualStack: cfg.DialDualStack}
		dialer, err := d.GetDialFunc(false)
		if err != nil {
			log.Println("[warning] failed to get dialer for proxy", err)
		}

		genericClient.Dial = dialer
		httpc.Dial = dialer
	}

	if cfg.ClientID != "" {
		ClientID = cfg.ClientID
	} else {
		err := GetClientID()
		if err != nil {
			log.Println("Failed to get ClientID:", err)
			log.Println("please report this as  issue")
			log.Println("For temporary workaround, you can manually extract this token and set in your config: https://git.maid.zone/stuff/soundcloak/src/branch/main/docs/INSTANCE_GUIDE.md#script-version-clientid-not-found")
			os.Exit(1)
			return
		}

		go func() {
			ticker := time.NewTicker(cfg.ClientIDTTL)
			for range ticker.C {
				err := GetClientID()
				if err != nil {
					fmt.Println("Got error extracting ClientID, using previously extracted, please report as issue:", err)
				}
			}
		}()
	}

	// could probably make a generic function, whatever
	go func() {
		ticker := time.NewTicker(cfg.UserCacheCleanDelay)
		for range ticker.C {
			usersCacheLock.Lock()

			now := time.Now()
			for key, val := range UsersCache {
				if val.Expires.Before(now) {
					delete(UsersCache, key)
				}
			}

			usersCacheLock.Unlock()
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.TrackCacheCleanDelay)
		for range ticker.C {
			tracksCacheLock.Lock()

			now := time.Now()
			for key, val := range TracksCache {
				if val.Expires.Before(now) {
					delete(TracksCache, key)
				}
			}

			tracksCacheLock.Unlock()
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.PlaylistCacheCleanDelay)
		for range ticker.C {
			playlistsCacheLock.Lock()

			now := time.Now()
			for key, val := range PlaylistsCache {
				if val.Expires.Before(now) {
					delete(PlaylistsCache, key)
				}
			}

			playlistsCacheLock.Unlock()
		}
	}()
}

func baseUri() *fasthttp.URI {
	uri := fasthttp.AcquireURI()
	uri.SetScheme("https")
	uri.SetHost(api)

	return uri
}

func baseUriReq(req *fasthttp.Request) {
	req.URI().SetScheme("https")
	req.URI().SetHost(api)
}
