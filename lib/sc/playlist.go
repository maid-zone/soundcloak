package sc

import (
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
)

var PlaylistsCache = map[string]cached[Playlist]{}
var playlistsCacheLock = &sync.RWMutex{}

// Functions/structures related to playlists

type Playlist struct {
	Artwork       string  `json:"artwork_url"`
	CreatedAt     string  `json:"created_at"`
	Description   string  `json:"description"`
	Kind          string  `json:"kind"` // should always be "playlist"! or "system-playlist"
	LastModified  string  `json:"last_modified"`
	Permalink     string  `json:"permalink"`
	TagList       string  `json:"tag_list"`
	Title         string  `json:"title"`
	Type          string  `json:"set_type"`
	MissingTracks string  `json:"-"`
	Tracks        []Track `json:"tracks"`
	Author        User    `json:"user"`
	Likes         int64   `json:"likes_count"`
	TrackCount    int64   `json:"track_count"`
	Album         bool    `json:"is_album"`
}

func GetPlaylist(cid string, permalink string) (Playlist, error) {
	playlistsCacheLock.RLock()
	if cell, ok := PlaylistsCache[permalink]; ok {
		playlistsCacheLock.RUnlock()
		return cell.Value, nil
	}
	playlistsCacheLock.RUnlock()

	var p Playlist
	var err error
	if cid == "" {
		cid, err = GetClientID()
		if err != nil {
			return p, err
		}
	}

	err = Resolve(cid, permalink, &p)
	if err != nil {
		return p, err
	}

	if p.Kind != "playlist" && p.Kind != "system-playlist" {
		return p, ErrKindNotCorrect
	}

	err = p.Fix(cid, true, true)
	if err != nil {
		return p, err
	}

	playlistsCacheLock.Lock()
	PlaylistsCache[permalink] = cached[Playlist]{Value: p, Expires: time.Now().Add(cfg.PlaylistTTL)}
	playlistsCacheLock.Unlock()

	return p, nil
}

func SearchPlaylists(cid string, prefs cfg.Preferences, args string) (*Paginated[*Playlist], error) {
	p := Paginated[*Playlist]{Next: "https://" + api + "/search/playlists" + args}
	err := p.Proceed(cid, true)
	if err != nil {
		return nil, err
	}

	for _, p := range p.Collection {
		p.Fix("", false, false)
		p.Postfix(prefs, false, false)
	}

	return &p, nil
}

func (p *Playlist) Fix(cid string, cached bool, fixAuthor bool) error {
	if cached {
		err := p.GetMissingTracks(cid)
		if err != nil {
			return err
		}

		for i, t := range p.Tracks {
			t.Fix(false, false)
			p.Tracks[i] = t
		}

		p.Artwork = strings.Replace(p.Artwork, "-large.", "-t500x500.", 1)
	} else {
		p.Artwork = strings.Replace(p.Artwork, "-large.", "-t200x200.", 1)
	}

	if fixAuthor {
		p.Author.Fix(false)
	}

	return nil
}

func (p *Playlist) Postfix(prefs cfg.Preferences, fixTracks bool, fixAuthor bool) []Track {
	if cfg.ProxyImages && *prefs.ProxyImages && p.Artwork != "" {
		p.Artwork = "/_/proxy/images?url=" + url.QueryEscape(p.Artwork)
	}

	if fixAuthor {
		p.Author.Postfix(prefs)
	}

	if fixTracks {
		var fixed = make([]Track, len(p.Tracks))
		for i, t := range p.Tracks {
			t.Postfix(prefs, false)
			fixed[i] = t
		}
		return fixed
	}

	return nil
}

func (p Playlist) FormatDescription() string {
	desc := p.Description
	if p.Description != "" {
		desc += "\n\n"
	}

	desc += strconv.FormatInt(int64(len(p.Tracks)), 10) + " tracks"
	desc += "\n" + strconv.FormatInt(p.Likes, 10) + " ❤️"
	desc += "\nCreated: " + p.CreatedAt
	desc += "\nLast modified: " + p.LastModified
	if len(p.TagList) != 0 {
		desc += "\nTags: " + strings.Join(TagListParser(p.TagList), ", ")
	}

	return desc
}

type MissingTrack struct {
	ID    string
	Index int
}

func JoinMissingTracks(missing []MissingTrack) (st string) {
	for i, track := range missing {
		st += track.ID
		if i != len(missing)-1 {
			st += ","
		}
	}
	return
}

func GetMissingTracks(cid string, missing []MissingTrack) (res []Track, next []MissingTrack, err error) {
	if len(missing) > 50 {
		next = missing[50:]
		missing = missing[:50]
	}

	res, err = GetTracks(cid, JoinMissingTracks(missing))
	return
}

func GetNextMissingTracks(cid string, raw string) (res []Track, next []string, err error) {
	missing := strings.Split(raw, ",")
	if len(missing) > 50 {
		next = missing[50:]
		missing = missing[:50]
	}

	res, err = GetTracks(cid, strings.Join(missing, ","))
	return
}

func (p *Playlist) GetMissingTracks(cid string) error {
	missing := []MissingTrack{}
	for i, track := range p.Tracks {
		if track.Title == "" {
			missing = append(missing, MissingTrack{ID: string(track.ID), Index: i})
		}
	}

	if len(missing) == 0 {
		return nil
	}

	res, next, err := GetMissingTracks(cid, missing)
	if err != nil {
		return err
	}

	for _, oldTrack := range missing {
		for _, newTrack := range res {
			if string(newTrack.ID) == string(oldTrack.ID) {
				p.Tracks[oldTrack.Index] = newTrack
			}
		}
	}

	p.MissingTracks = JoinMissingTracks(next)

	return nil
}

func (p Playlist) Href() string {
	if p.Kind == "system-playlist" {
		return "/discover/sets/" + p.Permalink
	}

	return "/" + p.Author.Permalink + "/sets/" + p.Permalink
}

func (p Playlist) TracksCount() int64 {
	if p.TrackCount != 0 {
		return p.TrackCount
	}

	return int64(len(p.Tracks))
}
