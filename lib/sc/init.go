package sc

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"syscall"
	"time"

	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/segmentio/encoding/json"
	"github.com/valyala/fasthttp"
)

var clientIdCache struct {
	ClientID  string
	Version   []byte
	NextCheck time.Time
}

const api = "api-v2.soundcloud.com"

var httpc = &fasthttp.HostClient{
	Addr:                api + ":443",
	IsTLS:               true,
	DialDualStack:       true,
	Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
	MaxIdleConnDuration: 1<<63 - 1, //improves performance but seems to cause some issues, need more testing
}

var verRegex = regexp.MustCompile(`(?m)^<script>window\.__sc_version="([0-9]{10})"</script>$`)
var scriptsRegex = regexp.MustCompile(`(?m)^<script crossorigin src="(https://a-v2\.sndcdn\.com/assets/.+\.js)"></script>$`)
var clientIdRegex = regexp.MustCompile(`\("client_id=([A-Za-z0-9]{32})"\)`)
var ErrVersionNotFound = errors.New("version not found")
var ErrScriptNotFound = errors.New("script not found")
var ErrIDNotFound = errors.New("clientid not found")
var ErrKindNotCorrect = errors.New("entity of incorrect kind")

type cached[T any] struct {
	Value   T
	Expires time.Time
}

// inspired by github.com/imputnet/cobalt (mostly stolen lol)
func GetClientID() (string, error) {
	if clientIdCache.NextCheck.After(time.Now()) {
		return clientIdCache.ClientID, nil
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("https://soundcloud.com/h") // 404 page
	req.Header.Set("User-Agent", cfg.UserAgent)   // the connection is stuck with fasthttp useragent lol, maybe randomly select from a list of browser useragents in the future? low priority for now
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := fasthttp.Do(req, resp)
	if err != nil {
		return "", err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	res := verRegex.FindSubmatch(data)
	if len(res) != 2 {
		return "", ErrVersionNotFound
	}

	if bytes.Equal(res[1], clientIdCache.Version) {
		clientIdCache.NextCheck = time.Now().Add(cfg.ClientIDTTL)
		return clientIdCache.ClientID, nil
	}

	ver := res[1]

	scripts := scriptsRegex.FindAllSubmatch(data, -1)
	if len(scripts) == 0 {
		return "", ErrScriptNotFound
	}

	for _, scr := range scripts {
		if len(scr) != 2 {
			continue
		}

		req.SetRequestURIBytes(scr[1])

		err = fasthttp.Do(req, resp)
		if err != nil {
			continue
		}

		data, err = resp.BodyUncompressed()
		if err != nil {
			data = resp.Body()
		}

		res = clientIdRegex.FindSubmatch(data)
		if len(res) != 2 {
			continue
		}

		clientIdCache.ClientID = string(res[1])
		clientIdCache.Version = ver
		clientIdCache.NextCheck = time.Now().Add(cfg.ClientIDTTL)
		return clientIdCache.ClientID, nil
	}

	return "", ErrIDNotFound
}

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
	}

	return
}

func Resolve(path string, out any) error {
	cid, err := GetClientID()
	if err != nil {
		return err
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

func (p *Paginated[T]) Proceed(shouldUnfold bool) error {
	cid, err := GetClientID()
	if err != nil {
		return err
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
		return p.Proceed(true)
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

	return
}

// could probably make a generic function, whatever
func init() {
	go func() {
		ticker := time.NewTicker(cfg.UserCacheCleanDelay)
		for range ticker.C {
			usersCacheLock.Lock()

			for key, val := range usersCache {
				if val.Expires.Before(time.Now()) {
					delete(usersCache, key)
				}
			}

			usersCacheLock.Unlock()
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.TrackCacheCleanDelay)
		for range ticker.C {
			tracksCacheLock.Lock()

			for key, val := range tracksCache {
				if val.Expires.Before(time.Now()) {
					delete(tracksCache, key)
				}
			}

			tracksCacheLock.Unlock()
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.PlaylistCacheCleanDelay)
		for range ticker.C {
			playlistsCacheLock.Lock()

			for key, val := range playlistsCache {
				if val.Expires.Before(time.Now()) {
					delete(playlistsCache, key)
				}
			}

			playlistsCacheLock.Unlock()
		}
	}()
}
