package sc

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/valyala/fasthttp"
)

var clientIdCache struct {
	ClientID       []byte
	ClientIDString string
	Version        []byte
	NextCheck      time.Time
}

type cached[T any] struct {
	Value   T
	Expires time.Time
}

var httpc = fasthttp.HostClient{
	Addr:          "api-v2.soundcloud.com:443",
	IsTLS:         true,
	DialDualStack: true,
	Dial:          (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
	//MaxIdleConnDuration: 1<<63 - 1,
}

var usersCache = map[string]cached[User]{}
var tracksCache = map[string]cached[Track]{}
var userSearchCache = map[string]cached[Paginated[User]]{}
var trackSearchCache = map[string]cached[Paginated[Track]]{}

var verRegex = regexp.MustCompile(`(?m)^<script>window\.__sc_version="([0-9]{10})"</script>$`)
var scriptsRegex = regexp.MustCompile(`(?m)^<script crossorigin src="(https://a-v2\.sndcdn\.com/assets/.+\.js)"></script>$`)
var clientIdRegex = regexp.MustCompile(`\("client_id=([A-Za-z0-9]{32})"\)`)
var ErrVersionNotFound = errors.New("version not found")
var ErrScriptNotFound = errors.New("script not found")
var ErrIDNotFound = errors.New("clientid not found")
var ErrKindNotCorrect = errors.New("entity of incorrect kind")
var ErrIncompatibleStream = errors.New("incompatible stream")
var ErrNoURL = errors.New("no url")

// inspired by github.com/imputnet/cobalt (mostly stolen lol)
func GetClientID() (string, error) {
	if clientIdCache.NextCheck.After(time.Now()) {
		return clientIdCache.ClientIDString, nil
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

	//fmt.Println(string(data), err)

	res := verRegex.FindSubmatch(data)
	if len(res) != 2 {
		return "", ErrVersionNotFound
	}

	if bytes.Equal(res[1], clientIdCache.Version) {
		return clientIdCache.ClientIDString, nil
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

		clientIdCache.ClientID = res[1]
		clientIdCache.ClientIDString = string(res[1])
		clientIdCache.Version = ver
		clientIdCache.NextCheck = time.Now().Add(cfg.ClientIDTTL)
		return clientIdCache.ClientIDString, nil
	}

	return "", ErrIDNotFound
}

func DoWithRetry(req *fasthttp.Request, resp *fasthttp.Response) (err error) {
	for i := 0; i < 5; i++ {
		err = httpc.Do(req, resp)
		if err == nil {
			return nil
		}

		if !os.IsTimeout(err) && err != fasthttp.ErrTimeout {
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

	req.SetRequestURI("https://api-v2.soundcloud.com/resolve?url=https%3A%2F%2Fsoundcloud.com%2F" + url.QueryEscape(path) + "&client_id=" + cid)
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = DoWithRetry(req, resp)
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

	return cfg.JSON.Unmarshal(data, out)
}

func GetUser(permalink string) (User, error) {
	if cell, ok := usersCache[permalink]; ok && cell.Expires.After(time.Now()) {
		return cell.Value, nil
	}

	var u User
	err := Resolve(permalink, &u)
	if err != nil {
		return u, err
	}

	if u.Kind != "user" {
		err = ErrKindNotCorrect
		return u, err
	}

	u.Avatar = strings.Replace(u.Avatar, "-large.", "-t200x200.", 1)
	ls := strings.Split(u.URN, ":")
	u.ID = ls[len(ls)-1]
	usersCache[permalink] = cached[User]{Value: u, Expires: time.Now().Add(cfg.UserTTL)}

	return u, err
}

func GetTrack(permalink string) (Track, error) {
	if cell, ok := tracksCache[permalink]; ok && cell.Expires.After(time.Now()) {
		return cell.Value, nil
	}

	var u Track
	err := Resolve(permalink, &u)
	if err != nil {
		return u, err
	}

	if u.Kind != "track" {
		return u, ErrKindNotCorrect
	}

	tracksCache[permalink] = cached[Track]{Value: u, Expires: time.Now().Add(cfg.TrackTTL)}

	return u, nil
}

func (p *Paginated[T]) Proceed() error {
	cid, err := GetClientID()
	if err != nil {
		return err
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(p.Next + "&client_id=" + cid)
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = DoWithRetry(req, resp)
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

	return cfg.JSON.Unmarshal(data, p)
}

func (u User) GetTracks(args string) (*Paginated[Track], error) {
	p := Paginated[Track]{
		Next: "https://api-v2.soundcloud.com/users/" + u.ID + "/tracks" + args,
	}

	err := p.Proceed()
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (t Track) GetStream() (string, error) {
	cid, err := GetClientID()
	if err != nil {
		return "", err
	}

	tr := t.Media.SelectCompatible()
	if tr == nil {
		return "", ErrIncompatibleStream
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(tr.URL + "?client_id=" + cid + "&track_authorization=" + t.Authorization)
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = DoWithRetry(req, resp)
	if err != nil {
		return "", err
	}

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("resolve: got status code %d", resp.StatusCode())
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var s Stream
	err = cfg.JSON.Unmarshal(data, &s)
	if err != nil {
		return "", err
	}

	if s.URL == "" {
		return "", ErrNoURL
	}

	return s.URL, nil
}

func SearchTracks(args string) (*Paginated[Track], error) {
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	p := Paginated[Track]{Next: "https://api-v2.soundcloud.com/search/tracks" + args + "&client_id=" + cid}
	err = p.Proceed()
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func SearchUsers(args string) (*Paginated[User], error) {
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	p := Paginated[User]{Next: "https://api-v2.soundcloud.com/search/users" + args + "&client_id=" + cid}
	err = p.Proceed()
	if err != nil {
		return nil, err
	}

	return &p, nil
}
