package sc

import (
	"bytes"
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

// don't jus leak the proxy like that lol
func scrub(err error) error {
	if cfg.SoundcloudApiProxy != "" && err != nil {
		s := err.Error()
		if strings.HasPrefix(s, "could not connect to proxyAddr") || strings.HasPrefix(s, "socks connect") {
			return ProxyErr
		}
	}

	return err
}

type clientIdCache struct {
	NextCheck time.Time
	ClientID  string
	Version   string
}

var ClientIDCache clientIdCache

const api = "api-v2.soundcloud.com"

const H = len("https://" + api)

var newline = []byte("\n")

var sc_version = []byte(`<script>window.__sc_version="`)
var script0 = []byte(`<script crossorigin src="https://a-v2.sndcdn.com/assets/0-`)
var script = []byte(`<script crossorigin src="https://a-v2.sndcdn.com/assets/`)
var httpc = &fasthttp.HostClient{
	Addr:                api + ":443",
	IsTLS:               true,
	MaxIdleConnDuration: cfg.MaxIdleConnDuration,
	DialDualStack:       cfg.DialDualStack,
}

var genericClient = &fasthttp.Client{
	DialDualStack: cfg.DialDualStack,
}

// var verRegex = regexp2.MustCompile(`^<script>window\.__sc_version="([0-9]{10})"</script>$`, 2)
// var scriptsRegex = regexp2.MustCompile(`^<script crossorigin src="(https://a-v2\.sndcdn\.com/assets/.+\.js)"></script>$`, 2)
// var scriptRegex = regexp2.MustCompile(`^<script crossorigin src="(https://a-v2\.sndcdn\.com/assets/0-.+\.js)"></script>$`, 2)

//go:generate go tool regexp2cg -package sc -o regexp2_codegen.go
var clientIdRegex = regexp2.MustCompile(`client_id:"([A-Za-z0-9]{32})"`, 0) //regexp2.MustCompile(`\("client_id=([A-Za-z0-9]{32})"\)`, 0)
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

	m2, _ := clientIdRegex.FindStringMatch(string(data))
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

// Experimental method, which asserts that the clientId is inside the file that starts with "0-"
const experimental_GetClientID = false

// inspired by github.com/imputnet/cobalt
func GetClientID() (string, error) {
	if cfg.ClientID != "" {
		return cfg.ClientID, nil
	}

	if ClientIDCache.NextCheck.After(time.Now()) {
		misc.Log("clientidcache hit @ 1")
		return ClientIDCache.ClientID, nil
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("https://soundcloud.com")
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetryAll(genericClient, req, resp)
	if err != nil {
		return "", err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	if experimental_GetClientID {
		var ver string
		var scriptUrl []byte
		for l := range bytes.SplitSeq(data, newline) { // version usually comes earlier, but retest this sometimes !!!
			if ver == "" && bytes.HasPrefix(l, sc_version) {
				ver = cfg.B2s(l[len(sc_version) : len(l)-len(`"</script>`)])
				misc.Log("found ver:", ver)
				if ClientIDCache.Version != "" && ver == ClientIDCache.Version {
					goto verCacheHit
				}
			} else if bytes.HasPrefix(l, script0) {
				scriptUrl = l[len(`<script crossorigin src="`) : len(l)-len(`"></script>`)]
				misc.Log("found scriptUrl:", string(scriptUrl))
				break
			}
		}

		if ver == "" {
			return "", ErrVersionNotFound
		}

		if scriptUrl == nil {
			return "", ErrScriptNotFound
		}

		req.SetRequestURIBytes(scriptUrl)
		err = DoWithRetryAll(genericClient, req, resp)
		if err != nil {
			return "", err
		}

		data, err = resp.BodyUncompressed()
		if err != nil {
			data = resp.Body()
		}

		m, _ := clientIdRegex.FindStringMatch(cfg.B2s(data))
		if m != nil {
			g := m.GroupByNumber(1)
			if g != nil {
				ClientIDCache.ClientID = g.String()
				ClientIDCache.Version = ver
				ClientIDCache.NextCheck = time.Now().Add(cfg.ClientIDTTL)
				misc.Log(ClientIDCache)
				return ClientIDCache.ClientID, nil
			}
		}
	} else {
		ch := make(chan string, 1)
		wg := &sync.WaitGroup{}
		done := false

		var ver string
		var scriptUrls = make([][]byte, 0, 10) // Usually only 8 chunks, so I went with jsut a bit more to be safe
		for l := range bytes.SplitSeq(data, newline) {
			if ver == "" && bytes.HasPrefix(l, sc_version) {
				ver = cfg.B2s(l[len(sc_version) : len(l)-len(`"</script>`)])
				if ver == ClientIDCache.Version {
					goto verCacheHit
				}
			} else if bytes.HasPrefix(l, script) {
				scriptUrls = append(scriptUrls, l[len(`<script crossorigin src="`):len(l)-len(`"></script>`)])
			}
		}

		if ver == "" {
			return "", ErrVersionNotFound
		}

		if len(scriptUrls) == 0 {
			return "", ErrScriptNotFound
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
			ClientIDCache.ClientID = res
			ClientIDCache.Version = ver
			ClientIDCache.NextCheck = time.Now().Add(cfg.ClientIDTTL)
			misc.Log(ClientIDCache)
		}

		return res, err
	}

	return "", ErrIDNotFound

verCacheHit:
	misc.Log("clientidcache hit @ ver")
	ClientIDCache.NextCheck = time.Now().Add(cfg.ClientIDTTL)
	return ClientIDCache.ClientID, nil
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

func Resolve(cid string, path string, out any) error {
	var err error
	if cid == "" {
		cid, err = GetClientID()
		if err != nil {
			return err
		}
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("https://" + api + "/resolve?url=https%3A%2F%2Fsoundcloud.com%2F" + url.QueryEscape(path) + "&client_id=" + cid)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = DoWithRetry(httpc, req, resp)
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
	Next       string `json:"next_href"`
	Collection []T    `json:"collection"`
	Total      int64  `json:"total_results"`
}

func (p *Paginated[T]) Proceed(cid string, shouldUnfold bool) error {
	var err error
	if cid == "" {
		cid, err = GetClientID()
		if err != nil {
			return err
		}
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	oldNext := p.Next
	req.SetRequestURI(p.Next + "&client_id=" + cid)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = DoWithRetry(httpc, req, resp)
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

	if p.Next == oldNext { // prevent loops of nothingness
		p.Next = ""
	}

	// in soundcloud api, pagination may not immediately return you something!
	// loading users who haven't released anything recently may require you to do a bunch of requests for nothing :/
	// maybe there could be a way to cache the last useless layer of pagination so soundcloak can start loading from there? might be a bit complicated, but would be great

	// another note: in featured tracks it seems to just be forever stuck after 2-3~ pages so i added a way to disable this behaviour
	if shouldUnfold && len(p.Collection) == 0 && p.Next != "" {
		// this will make sure that we actually proceed to something useful and not emptiness
		return p.Proceed(cid, true)
	}

	return nil
}

func TagListParser(taglist string) (res []string) {
	inString := false
	cur := []rune{}
	for i, c := range taglist {
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
			cur = []rune{}
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

func GetSearchSuggestions(cid string, query string) ([]string, error) {
	p := Paginated[SearchSuggestion]{Next: "https://" + api + "/search/queries?limit=10&q=" + url.QueryEscape(query)}
	err := p.Proceed(cid, false)
	if err != nil {
		return nil, err
	}

	l := make([]string, len(p.Collection)) // just so the response is a bit smaller, why not
	for i, r := range p.Collection {
		l[i] = r.Query
	}

	return l, nil
}

// could probably make a generic function, whatever
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
