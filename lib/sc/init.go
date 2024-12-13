package sc

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/segmentio/encoding/json"
	"github.com/valyala/fasthttp"
)

type clientIdCache struct {
	ClientID  string
	Version   string
	NextCheck time.Time
}

var ClientIDCache clientIdCache

const api = "api-v2.soundcloud.com"

var httpc = &fasthttp.HostClient{
	Addr:                api + ":443",
	IsTLS:               true,
	DialDualStack:       true,
	Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
	MaxIdleConnDuration: 1<<63 - 1,
}

var httpc_image = &fasthttp.HostClient{
	Addr:                cfg.ImageCDN + ":443",
	IsTLS:               true,
	DialDualStack:       true,
	Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
	MaxIdleConnDuration: 1<<63 - 1,
	StreamResponseBody:  true,
}

var genericClient = &fasthttp.Client{
	Dial: (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
}

//go:generate regexp2cg -package sc -o regexp2_codegen.go
var verRegex = regexp2.MustCompile(`^<script>window\.__sc_version="([0-9]{10})"</script>$`, 2)
var scriptsRegex = regexp2.MustCompile(`^<script crossorigin src="(https://a-v2\.sndcdn\.com/assets/.+\.js)"></script>$`, 2)
var scriptRegex = regexp2.MustCompile(`^<script crossorigin src="(https://a-v2\.sndcdn\.com/assets/0-.+\.js)"></script>$`, 2)
var clientIdRegex = regexp2.MustCompile(`\("client_id=([A-Za-z0-9]{32})"\)`, 0)
var ErrVersionNotFound = errors.New("version not found")
var ErrScriptNotFound = errors.New("script not found")
var ErrIDNotFound = errors.New("clientid not found")
var ErrKindNotCorrect = errors.New("entity of incorrect kind")

type cached[T any] struct {
	Value   T
	Expires time.Time
}

// don't be spooked by cfg.Log, it will be removed during compilation if cfg.Debug == false
func processFile(wg *sync.WaitGroup, ch chan string, uri string, isDone *bool) {
	cfg.Log(uri)
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(uri)
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	defer wg.Done()

	if *isDone {
		cfg.Log("early 1")
		return
	}

	err := DoWithRetryAll(genericClient, req, resp)
	if err != nil {
		cfg.Log(err)
		return
	}

	if *isDone {
		cfg.Log("early 2")
		return
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	if *isDone {
		cfg.Log("early 3")
		return
	}

	m2, _ := clientIdRegex.FindStringMatch(string(data))
	if m2 != nil {
		g := m2.GroupByNumber(1)
		if g != nil {
			if *isDone {
				cfg.Log("early 4")
				return
			}

			ch <- g.String()
			cfg.Log("found in", uri)
			return
		}
	}

	cfg.Log("not found in", uri)
}

// Experimental method, which asserts that the clientId is inside the file that starts with "0-"
const experimental_GetClientID = true

// inspired by github.com/imputnet/cobalt (mostly stolen lol)
func GetClientID() (string, error) {
	if ClientIDCache.NextCheck.After(time.Now()) {
		cfg.Log("clientidcache hit @ 1")
		return ClientIDCache.ClientID, nil
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("https://soundcloud.com/h") // 404 page
	req.Header.Set("User-Agent", cfg.UserAgent)
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

	m, _ := verRegex.FindStringMatch(string(data))
	if m == nil {
		return "", ErrVersionNotFound
	}

	g := m.GroupByNumber(1)
	if g == nil {
		return "", ErrVersionNotFound
	}

	ver := g.String()
	if ver == ClientIDCache.Version {
		cfg.Log("clientidcache hit @ ver")
		ClientIDCache.NextCheck = time.Now().Add(cfg.ClientIDTTL)
		return ClientIDCache.ClientID, nil
	}

	if experimental_GetClientID {
		m, _ = scriptRegex.FindStringMatch(string(data))
		if m != nil {
			g = m.GroupByNumber(1)
			if g != nil {
				req.SetRequestURI(g.String())
				err = DoWithRetryAll(genericClient, req, resp)
				if err != nil {
					return "", err
				}

				data, err = resp.BodyUncompressed()
				if err != nil {
					data = resp.Body()
				}

				m, _ = clientIdRegex.FindStringMatch(string(data))
				if m != nil {
					g = m.GroupByNumber(1)
					if g != nil {
						ClientIDCache.ClientID = g.String()
						ClientIDCache.Version = ver
						ClientIDCache.NextCheck = time.Now().Add(cfg.ClientIDTTL)
						cfg.Log(ClientIDCache)
						return ClientIDCache.ClientID, nil
					}
				}
			}
		}
	} else {
		ch := make(chan string, 1)
		wg := &sync.WaitGroup{}
		done := false
		m, _ = scriptsRegex.FindStringMatch(string(data))
		for m != nil {
			g = m.GroupByNumber(1)
			if g != nil {
				wg.Add(1)
				go processFile(wg, ch, g.String(), &done)
			}

			m, _ = scriptsRegex.FindNextMatch(m)
		}

		go func() {
			defer func() {
				err := recover()
				cfg.Log("-- GetClientID recovered:", err)
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
			cfg.Log(ClientIDCache)
		}

		return res, err
	}

	return "", ErrIDNotFound
}

// Just retry any kind of errors, why not
func DoWithRetryAll(httpc *fasthttp.Client, req *fasthttp.Request, resp *fasthttp.Response) (err error) {
	for i := 0; i < 10; i++ {
		err = httpc.Do(req, resp)
		if err == nil {
			return nil
		}
	}

	return
}

// Since the http client is setup to always keep connections idle (great for speed, no need to open a new one everytime), those connections may be closed by soundcloud after some time of inactivity, this ensures that we retry those requests that fail due to the connection closing/timing out
func DoWithRetry(httpc *fasthttp.HostClient, req *fasthttp.Request, resp *fasthttp.Response) (err error) {
	for i := 0; i < 10; i++ {
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
			return
		}

		cfg.Log("we failed haha", err)
	}

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
	req.Header.Set("User-Agent", cfg.UserAgent)
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
	Collection []T    `json:"collection"`
	Total      int64  `json:"total_results"`
	Next       string `json:"next_href"`
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
	req.Header.Set("User-Agent", cfg.UserAgent)
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

// could probably make a generic function, whatever
func init() {
	go func() {
		ticker := time.NewTicker(cfg.UserCacheCleanDelay)
		for range ticker.C {
			usersCacheLock.Lock()

			for key, val := range UsersCache {
				if val.Expires.Before(time.Now()) {
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

			for key, val := range TracksCache {
				if val.Expires.Before(time.Now()) {
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

			for key, val := range PlaylistsCache {
				if val.Expires.Before(time.Now()) {
					delete(PlaylistsCache, key)
				}
			}

			playlistsCacheLock.Unlock()
		}
	}()
}
