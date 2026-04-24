package sc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"github.com/a-h/templ"
	templruntime "github.com/a-h/templ/runtime"
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

// Functions/structures related to tracks

var ErrIncompatibleStream = errors.New("incompatible stream")
var ErrNoURL = errors.New("no url")

var TracksCache = map[string]cached[Track]{}
var tracksCacheLock = &sync.RWMutex{}

type Track struct {
	Artwork       string      `json:"artwork_url"`
	CreatedAt     string      `json:"created_at"`
	Description   string      `json:"description"`
	Genre         string      `json:"genre"`
	Kind          string      `json:"kind"` // should always be "track"!
	LastModified  string      `json:"last_modified"`
	License       string      `json:"license"`
	Permalink     string      `json:"permalink"`
	TagList       string      `json:"tag_list"`
	Title         string      `json:"title"`
	ID            json.Number `json:"id"`
	Authorization string      `json:"track_authorization"`
	Policy        TrackPolicy `json:"policy"`
	Station       string      `json:"station_permalink"`
	Media         Media       `json:"media"`
	Author        User        `json:"user"`
	Comments      int         `json:"comment_count"`
	Likes         int64       `json:"likes_count"`
	Played        int64       `json:"playback_count"`
	Reposted      int64       `json:"reposts_count"`
	Duration      uint32      `json:"full_duration"`
	Waveform      string      `json:"waveform_url"`
}

type TrackPolicy string

const (
	PolicyMonetize TrackPolicy = "MONETIZE" // seems like only certain countries get this policy? sometimes protected by widevine and fairplay
	PolicyBlock    TrackPolicy = "BLOCK"    // not available (in your country)
	PolicySnip     TrackPolicy = "SNIP"     // 30-second snippet available
	PolicyAllow    TrackPolicy = "ALLOW"    // all good
)

type Protocol string

const (
	ProtocolHLS             Protocol = "hls"
	ProtocolProgressive     Protocol = "progressive"
	ProtocolEncryptedHLS    Protocol = "encrypted-hls"     // idk, haven't seen in the wild
	ProtocolCTREncryptedHLS Protocol = "ctr-encrypted-hls" // google's widevine
	ProtocolCBCEncryptedHLS Protocol = "cbc-encrypted-hls" // apple's fairplay
)

type Format struct {
	Protocol Protocol `json:"protocol"`
	MimeType string   `json:"mime_type"`
}

type Transcoding struct {
	URL     string `json:"url"`
	Preset  string `json:"preset"`
	Format  Format `json:"format"`
	Quality string `json:"quality"`
}

func (t Transcoding) Slug(tr Track) string {
	return tr.Author.Permalink + "/" +
		tr.Permalink + "/" +
		t.Quality + "/" +
		t.Preset + "/" +
		string(t.Format.Protocol)
}

type Media struct {
	Transcodings []Transcoding `json:"transcodings"`
}

type Stream struct {
	URL string `json:"url"`
}

type Comment struct {
	Kind      string `json:"kind"` // "comment"
	Body      string `json:"body"`
	Author    User   `json:"user"`
	Timestamp int    `json:"timestamp"`
}

// func (m Media) SelectCompatible(mode string, restream bool) (*Transcoding, string) {
// 	switch mode {
// 	case cfg.AudioBest:
// 	case cfg.AudioAAC:
// 		for _, t := range m.Transcodings {
// 			if t.Format.Protocol == ProtocolHLS && t.Preset == "aac_160k" {
// 				return &t, cfg.AudioAAC
// 			}
// 		}
// 	}

// 	if restream {
// 		for _, t := range m.Transcodings {
// 			if t.Format.Protocol == ProtocolProgressive && t.Format.MimeType == "audio/mpeg" {
// 				return &t, cfg.AudioMP3
// 			}
// 		}
// 	}
// 	for _, t := range m.Transcodings {
// 		if t.Format.Protocol == ProtocolHLS && t.Format.MimeType == "audio/mpeg" {
// 			return &t, cfg.AudioMP3
// 		}
// 	}
// 	return nil, ""
// }

func (m Media) SelectCompatibleRestream(mode string) (*Transcoding, string) {
	var b1 *Transcoding
	// note that best is deprecated, left for compatibility
	if mode == cfg.AudioAAC || mode == cfg.AudioBest {
		// reduce iterations count :)
		var b2 *Transcoding
		for _, t := range m.Transcodings {
			switch t.Format.Protocol {
			case ProtocolHLS:
				if t.Preset == "aac_160k" {
					return &t, cfg.AudioAAC
				} else if b1 == nil && t.Format.MimeType == "audio/mpeg" {
					b1 = &t
				}
			case ProtocolProgressive:
				if b2 == nil && t.Format.MimeType == "audio/mpeg" {
					b2 = &t
				}
			}
		}
		// progressive prefered instead of hls because less processing
		if b2 != nil {
			return b2, cfg.AudioMP3
		}
		if b1 != nil {
			return b1, cfg.AudioMP3
		}
		return nil, ""
	}

	for _, t := range m.Transcodings {
		if t.Format.MimeType == "audio/mpeg" {
			switch t.Format.Protocol {
			case ProtocolProgressive:
				return &t, cfg.AudioMP3
			case ProtocolHLS:
				b1 = &t
			}
		}
	}
	if b1 != nil {
		return b1, cfg.AudioMP3
	}
	return nil, ""
}

func (m Media) SelectCompatibleHLS(mode string) (*Transcoding, string) {
	if mode == cfg.AudioAAC || mode == cfg.AudioBest {
		var b1 *Transcoding
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolHLS {
				if t.Preset == "aac_160k" {
					return &t, cfg.AudioAAC
				} else if b1 == nil && t.Format.MimeType == "audio/mpeg" {
					b1 = &t
				}
			}
		}
		if b1 != nil {
			return b1, cfg.AudioMP3
		}
		return nil, ""
	}

	for _, t := range m.Transcodings {
		if t.Format.Protocol == ProtocolHLS && t.Format.MimeType == "audio/mpeg" {
			return &t, cfg.AudioMP3
		}
	}
	return nil, ""
}

func (m Media) SelectCompatibleProgressive() *Transcoding {
	for _, t := range m.Transcodings {
		if t.Format.Protocol == ProtocolProgressive && t.Format.MimeType == "audio/mpeg" {
			return &t
		}
	}
	return nil
}

func GetTrack(permalink string) (Track, error) {
	tracksCacheLock.RLock()
	if cell, ok := TracksCache[permalink]; ok {
		tracksCacheLock.RUnlock()
		return cell.Value, nil
	}
	tracksCacheLock.RUnlock()

	var t Track
	err := Resolve(permalink, &t)
	if err != nil {
		return t, err
	}

	if t.Kind != "track" {
		return t, ErrKindNotCorrect
	}

	t.Fix(true, true)

	tracksCacheLock.Lock()
	TracksCache[permalink] = cached[Track]{Value: t, Expires: time.Now().Add(cfg.TrackTTL)}
	tracksCacheLock.Unlock()

	return t, nil
}

// Currently supports:
// http/https links:
// - api.soundcloud.com/tracks/<id> (api-v2 subdomain also supported)
// - soundcloud.com/<user>/<track>
//
// plain permalink/id:
// - <user>/<track>
// - <id>
func GetArbitraryTrack(data string) (Track, error) {
	if len(data) > 8 && (data[:8] == "https://" || data[:7] == "http://") {
		u, err := url.Parse(data)
		if err == nil {
			if (u.Host == "api.soundcloud.com" || u.Host == "api-v2.soundcloud.com") && len(u.Path) > 8 && u.Path[:8] == "/tracks/" {
				return GetTrackByID(u.Path[8:])
			}

			if u.Host == "soundcloud.com" {
				if len(u.Path) < 4 {
					return Track{}, ErrNoURL
				}

				u.Path = u.Path[1:]
				if u.Path[len(u.Path)-1] == '/' {
					u.Path = u.Path[:len(u.Path)-1]
				}

				var n uint = 0
				for _, c := range u.Path {
					if c == '/' {
						n++
					}

					if n == 2 {
						return Track{}, ErrKindNotCorrect
					}
				}

				if n != 1 {
					return Track{}, ErrKindNotCorrect
				}

				return GetTrack(u.Path)
			}
		} else {
			return Track{}, err
		}
	}

	valid := true
	for _, n := range data {
		if n < '0' || n > '9' {
			valid = false
			break
		}
	}

	if valid {
		return GetTrackByID(data)
	}

	// this part should be at the end since it manipulates data
	if len(data) < 4 {
		return Track{}, ErrNoURL
	}

	if data[0] == '/' {
		data = data[1:]
	}

	if data[len(data)-1] == '/' {
		data = data[:len(data)-1]
	}
	var n uint = 0
	for _, c := range data {
		if c == '/' {
			n++
		}
	}

	if n == 1 {
		return GetTrack(data)
	}

	// failed to find a data point
	return Track{}, ErrKindNotCorrect
}

func SearchTracks(prefs cfg.Preferences, args []byte) (*Paginated[*Track], error) {
	uri := baseUri()
	uri.SetPath("/search/tracks")
	uri.SetQueryStringBytes(args)
	p := Paginated[*Track]{Next: uri}
	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(false, false)
		t.Postfix(prefs, false)
	}

	return &p, nil
}

func GetTracks(ids string) ([]Track, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	uri := baseUri()
	uri.SetPath("/tracks")
	uri.QueryArgs().Set("ids", ids)
	uri.QueryArgs().Set("client_id", ClientID)
	req.SetURI(uri)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return nil, err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var res []Track
	err = json.Unmarshal(data, &res)
	for i, t := range res {
		t.Fix(false, false)
		res[i] = t
	}
	return res, err
}

type CachedStream struct {
	Playlist *fasthttp.URI
	Base     *fasthttp.URI
}

var StreamCache = map[string]cached[CachedStream]{}
var StreamCacheMut = sync.RWMutex{}

func (tr Transcoding) GetStream(slug string, t Track) (cached[CachedStream], error) {
	if slug == "" {
		slug = tr.Slug(t)
	}

	StreamCacheMut.RLock()
	s, ok := StreamCache[slug]
	StreamCacheMut.RUnlock()
	if ok && s.Expires.After(time.Now()) {
		misc.Log("cache hit", s)
		return s, nil
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(tr.URL)
	req.URI().QueryArgs().Set("client_id", ClientID)
	req.URI().QueryArgs().Set("track_authorization", t.Authorization)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return s, err
	}

	if resp.StatusCode() != 200 {
		return s, fmt.Errorf("getstream: got status code %d", resp.StatusCode())
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var st Stream
	err = json.Unmarshal(data, &st)
	if err != nil {
		return s, err
	}

	misc.Log(st)

	if st.URL == "" {
		return s, ErrNoURL
	}

	if s.Value.Playlist == nil {
		s.Value.Playlist = fasthttp.AcquireURI()
	}
	err = s.Value.Playlist.Parse(nil, cfg.S2b(st.URL))
	if err == nil {
		s.Expires = time.Now().Add(time.Duration(t.Duration)*time.Millisecond + 105*time.Second)
		StreamCacheMut.Lock()
		StreamCache[slug] = s
		StreamCacheMut.Unlock()
	}
	return s, err
}

func (t *Track) Fix(large bool, fixAuthor bool) {
	if large {
		t.Artwork = strings.Replace(t.Artwork, "-large.", "-t500x500.", 1)
	} else {
		t.Artwork = strings.Replace(t.Artwork, "-large.", "-t200x200.", 1)
	}

	if fixAuthor {
		t.Author.Fix(false)
	}
}

func (t *Track) Postfix(prefs cfg.Preferences, fixAuthor bool) {
	if cfg.ProxyImages && *prefs.ProxyImages && t.Artwork != "" {
		t.Artwork = "/_/proxy/images?url=" + url.QueryEscape(t.Artwork)
	}

	if fixAuthor {
		t.Author.Postfix(prefs)
	}
}

func (t Track) FormatDescription() string {
	desc := t.Description
	if t.Description != "" {
		desc += "\n\n"
	}

	desc += strconv.FormatInt(t.Likes, 10) + " ❤️ | " + strconv.FormatInt(t.Played, 10) + " ▶️ | " + strconv.FormatInt(t.Reposted, 10) + " 🔁"
	if t.Genre != "" {
		desc += "\nGenre: " + t.Genre
	}
	desc += "\nCreated: " + t.CreatedAt
	desc += "\nLast modified: " + t.LastModified
	if len(t.TagList) != 0 {
		desc += "\nTags: " + TagListParser(t.TagList)
	}

	return desc
}

func GetTrackByID(id string) (Track, error) {
	tracksCacheLock.RLock()
	for _, cell := range TracksCache {
		if string(cell.Value.ID) == string(id) {
			tracksCacheLock.RUnlock()
			return cell.Value, nil
		}
	}
	tracksCacheLock.RUnlock()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	baseUriReq(req)
	req.URI().SetPath("/tracks/" + id)
	req.URI().QueryArgs().Set("client_id", ClientID)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	var t Track
	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return t, err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	err = json.Unmarshal(data, &t)
	if err != nil {
		return t, err
	}

	if t.Kind != "track" {
		return t, ErrKindNotCorrect
	}

	t.Fix(true, true)

	tracksCacheLock.Lock()
	TracksCache[t.Author.Permalink+"/"+t.Permalink] = cached[Track]{Value: t, Expires: time.Now().Add(cfg.TrackTTL)}
	tracksCacheLock.Unlock()

	return t, nil
}

func (t Track) Href() string {
	return "/" + t.Author.Permalink + "/" + t.Permalink
}

func RecentTracks(prefs cfg.Preferences, tag, args string) (*Paginated[*Track], error) {
	uri := baseUri()
	uri.SetPath("/recent-tracks/" + tag)
	uri.SetQueryString(args)
	p := Paginated[*Track]{Next: uri}
	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(false, false)
		t.Postfix(prefs, false)
	}

	return &p, nil
}

func (t Track) baseUri(subpath, args string) *fasthttp.URI {
	uri := baseUri()
	uri.SetPath("/tracks/" + string(t.ID) + "/" + subpath)
	uri.SetQueryString(args)
	return uri
}

func (t Track) GetRelated(prefs cfg.Preferences, args string) (*Paginated[*Track], error) {
	p := Paginated[*Track]{Next: t.baseUri("related", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(false, false)
		t.Postfix(prefs, false)
	}

	return &p, nil
}

func (t Track) GetPlaylists(prefs cfg.Preferences, args string) (*Paginated[*Playlist], error) {
	p := Paginated[*Playlist]{Next: t.baseUri("playlists_without_albums", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, p := range p.Collection {
		p.Fix(false, false)
		p.Postfix(prefs, false, false)
	}

	return &p, nil
}

func (t Track) GetAlbums(prefs cfg.Preferences, args string) (*Paginated[*Playlist], error) {
	p := Paginated[*Playlist]{Next: t.baseUri("albums", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, p := range p.Collection {
		p.Fix(false, false)
		p.Postfix(prefs, false, false)
	}

	return &p, nil
}

func (t Track) GetComments(prefs cfg.Preferences, args string) (*Paginated[*Comment], error) {
	p := Paginated[*Comment]{Next: t.baseUri("comments", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, p := range p.Collection {
		p.Author.Fix(false)
		p.Author.Postfix(prefs)
	}

	return &p, nil
}

func ToExt(audio string) string {
	switch audio {
	case cfg.AudioAAC:
		return "m4a"
	case cfg.AudioMP3:
		return "mp3"
	}

	return ""
}

type Waveform struct {
	//Width   int   `json:"width"`
	Height  uint64   `json:"height"`
	Samples []uint64 `json:"samples"`
}

func (t *Track) RenderWaveform() templ.Component {
	if t.Waveform == "" {
		return templ.NopComponent
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(t.Waveform)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return templ.NopComponent
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var wf Waveform
	err = json.Unmarshal(data, &wf)
	if err != nil || len(wf.Samples) == 0 || wf.Height == 0 {
		return templ.NopComponent
	}

	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		ww := w.(*templruntime.Buffer)
		_, err := ww.WriteString(`<svg class="waveform" viewBox="0 0 200 100" preserveAspectRatio="none"><defs><clipPath id="wf-p"><rect x="0" y="0" width="0" height="100"/></clipPath></defs><path d="`)
		if err != nil {
			return err
		}
		const (
			targetBars = 200
			svgHeight  = 100
			center     = svgHeight / 2
		)

		step := len(wf.Samples) / targetBars
		if step < 1 {
			step = 1
		}

		var count uint64
		b := make([]byte, 1, 10)
		b[0] = 'M'
		for i := 0; i < len(wf.Samples) && count < targetBars; i += step {
			h := wf.Samples[i] * center / wf.Height
			if h < 1 {
				h = 1
			}

			b = b[:1]
			b = strconv.AppendUint(b, count, 10)
			b = append(b, ',')
			b = strconv.AppendUint(b, center-h, 10)
			b = append(b, 'V')
			b = strconv.AppendUint(b, center+h, 10)
			count++
			_, err = ww.Write(b)
			if err != nil {
				return err
			}
		}
		_, err = ww.WriteString(`" stroke="var(--0)" fill="none" stroke-width="0.6"/></svg><script async src="/_/static/waveform.js"></script>`)
		if err != nil {
			return err
		}
		return nil
	})
}
