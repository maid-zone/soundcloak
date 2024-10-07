package sc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/valyala/fasthttp"
)

// Functions/structures related to users

var ErrIncompatibleStream = errors.New("incompatible stream")
var ErrNoURL = errors.New("no url")

var tracksCache = map[string]cached[Track]{}
var tracksCacheLock = &sync.RWMutex{}

type Track struct {
	Artwork     string `json:"artwork_url"`
	Comments    int    `json:"comment_count"`
	CreatedAt   string `json:"created_at"`
	Description string `json:"description"`
	//Duration      int    `json:"duration"` // there are duration and full_duration fields wtf does that mean
	Genre         string `json:"genre"`
	Kind          string `json:"kind"` // should always be "track"!
	LastModified  string `json:"last_modified"`
	License       string `json:"license"`
	Likes         int64  `json:"likes_count"`
	Permalink     string `json:"permalink"`
	Played        int64  `json:"playback_count"`
	TagList       string `json:"tag_list"`
	Title         string `json:"title"`
	ID            string `json:"urn"`
	Media         Media  `json:"media"`
	Authorization string `json:"track_authorization"`
	Author        User   `json:"user"`

	IDint int64 `json:"id"`
}

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
	if cell, ok := tracksCache[permalink]; ok && cell.Expires.After(time.Now()) {
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

	t.Fix(true)

	tracksCacheLock.Lock()
	tracksCache[permalink] = cached[Track]{Value: t, Expires: time.Now().Add(cfg.TrackTTL)}
	tracksCacheLock.Unlock()

	return t, nil
}

func SearchTracks(args string) (*Paginated[*Track], error) {
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	p := Paginated[*Track]{Next: "https://" + api + "/search/tracks" + args + "&client_id=" + cid}
	err = p.Proceed()
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(false)
	}

	return &p, nil
}

func GetTracks(ids string) ([]*Track, error) {
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

	err = DoWithRetry(req, resp)
	if err != nil {
		return nil, err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var res []*Track
	err = cfg.JSON.Unmarshal(data, &res)
	for _, t := range res {
		t.Fix(false)
	}
	return res, err
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
		return "", fmt.Errorf("getstream: got status code %d", resp.StatusCode())
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

func (t *Track) Fix(large bool) {
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

	t.Author.Fix(false)
}

func (t Track) FormatDescription() string {
	desc := t.Description
	if t.Description != "" {
		desc += "\n\n"
	}

	desc += strconv.FormatInt(t.Likes, 10) + " ❤️ | " + strconv.FormatInt(t.Played, 10) + " ▶️"
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
	for _, cell := range tracksCache {
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

	err = DoWithRetry(req, resp)
	if err != nil {
		return t, err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	err = cfg.JSON.Unmarshal(data, &t)
	if err != nil {
		return t, err
	}

	if t.Kind != "track" {
		return t, ErrKindNotCorrect
	}

	t.Fix(true)

	tracksCacheLock.Lock()
	tracksCache[t.Permalink] = cached[Track]{Value: t, Expires: time.Now().Add(cfg.TrackTTL)}
	tracksCacheLock.Unlock()

	return t, nil
}
