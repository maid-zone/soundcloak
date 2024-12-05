package sc

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/segmentio/encoding/json"
	"github.com/valyala/fasthttp"
)

// Functions/structures related to tracks

var ErrIncompatibleStream = errors.New("incompatible stream")
var ErrNoURL = errors.New("no url")

var TracksCache = map[string]cached[Track]{}
var tracksCacheLock = &sync.RWMutex{}

type Track struct {
	Artwork     string `json:"artwork_url"`
	Comments    int    `json:"comment_count"`
	CreatedAt   string `json:"created_at"`
	Description string `json:"description"`
	//Duration      int    `json:"duration"` // there are duration and full_duration fields wtf does that mean
	Genre         string      `json:"genre"`
	Kind          string      `json:"kind"` // should always be "track"!
	LastModified  string      `json:"last_modified"`
	License       string      `json:"license"`
	Likes         int64       `json:"likes_count"`
	Permalink     string      `json:"permalink"`
	Played        int64       `json:"playback_count"`
	Reposted      int64       `json:"reposts_count"`
	TagList       string      `json:"tag_list"`
	Title         string      `json:"title"`
	ID            string      `json:"urn"`
	Media         Media       `json:"media"`
	Authorization string      `json:"track_authorization"`
	Author        User        `json:"user"`
	Policy        TrackPolicy `json:"policy"`

	IDint int64 `json:"id"`
}

type TrackPolicy string

const (
	PolicyBlock TrackPolicy = "BLOCK"
)

type Protocol string

const (
	ProtocolHLS         Protocol = "hls"
	ProtocolProgressive Protocol = "progressive"
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

type Media struct {
	Transcodings []Transcoding `json:"transcodings"`
}

type Stream struct {
	URL string `json:"url"`
}

func (m Media) SelectCompatible() *Transcoding {
	for _, t := range m.Transcodings {
		if t.Format.Protocol == ProtocolHLS && t.Format.MimeType == "audio/mpeg" {
			return &t
		}
	}

	return nil
}

func GetTrack(permalink string) (Track, error) {
	tracksCacheLock.RLock()
	if cell, ok := TracksCache[permalink]; ok && cell.Expires.After(time.Now()) {
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

func SearchTracks(prefs cfg.Preferences, args string) (*Paginated[*Track], error) {
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	p := Paginated[*Track]{Next: "https://" + api + "/search/tracks" + args + "&client_id=" + cid}
	err = p.Proceed(true)
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
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("https://" + api + "/tracks?ids=" + ids + "&client_id=" + cid)
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = DoWithRetry(httpc, req, resp)
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

func (tr Transcoding) GetStream(prefs cfg.Preferences, authorization string) (string, error) {
	cid, err := GetClientID()
	if err != nil {
		return "", err
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(tr.URL + "?client_id=" + cid + "&track_authorization=" + authorization)
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = DoWithRetry(httpc, req, resp)
	if err != nil {
		return "", err
	}

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("getstream: got status code %d", resp.StatusCode())
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var s Stream
	err = json.Unmarshal(data, &s)
	if err != nil {
		return "", err
	}

	if s.URL == "" {
		return "", ErrNoURL
	}

	if cfg.ProxyStreams && *prefs.ProxyStreams && *prefs.Player == cfg.HLSPlayer {
		return "/_/proxy/streams/playlist?url=" + url.QueryEscape(s.URL), nil
	}

	return s.URL, nil
}

func (t *Track) Fix(large bool, fixAuthor bool) {
	if large {
		t.Artwork = strings.Replace(t.Artwork, "-large.", "-t500x500.", 1)
	} else {
		t.Artwork = strings.Replace(t.Artwork, "-large.", "-t200x200.", 1)
	}

	if t.ID == "" {
		t.ID = strconv.FormatInt(t.IDint, 10)
	} else {
		ls := strings.Split(t.ID, ":")
		t.ID = ls[len(ls)-1]
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
		desc += "\nTags: " + strings.Join(TagListParser(t.TagList), ", ")
	}

	return desc
}

func GetTrackByID(id string) (Track, error) {
	cid, err := GetClientID()
	if err != nil {
		return Track{}, err
	}

	tracksCacheLock.RLock()
	for _, cell := range TracksCache {
		if cell.Value.ID == id && cell.Expires.After(time.Now()) {
			tracksCacheLock.RUnlock()
			return cell.Value, nil
		}
	}
	tracksCacheLock.RUnlock()

	var t Track
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("https://" + api + "/tracks/" + id + "?client_id=" + cid)
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = DoWithRetry(httpc, req, resp)
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

func (t Track) DownloadImage() ([]byte, string, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(t.Artwork)
	req.Header.Set("User-Agent", cfg.UserAgent)
	//req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd") images not big enough to be compressed

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(ImageClient, req, resp)
	if err != nil {
		return nil, "", err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	return data, string(resp.Header.Peek("Content-Type")), nil
}

func (t Track) Href() string {
	return "/" + t.Author.Permalink + "/" + t.Permalink
}
