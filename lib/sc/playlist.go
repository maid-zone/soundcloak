package sc

import (
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maid-zone/soundcloak/lib/cfg"
)

var playlistsCache = map[string]cached[Playlist]{}
var playlistsCacheLock = &sync.RWMutex{}

// Functions/structures related to playlists

type Playlist struct {
	Artwork      string `json:"artwork_url"`
	CreatedAt    string `json:"created_at"`
	Description  string `json:"description"`
	Kind         string `json:"kind"` // should always be "playlist"!
	LastModified string `json:"last_modified"`
	Likes        int64  `json:"likes_count"`
	Permalink    string `json:"permalink"`
	//ReleaseDate  string  `json:"release_date"`
	TagList    string   `json:"tag_list"`
	Title      string   `json:"title"`
	Type       string   `json:"set_type"`
	Album      bool     `json:"is_album"`
	Author     User     `json:"user"`
	Tracks     []*Track `json:"tracks"`
	TrackCount int64    `json:"track_count"`

	MissingTracks string `json:"-"`
}

func GetPlaylist(permalink string) (Playlist, error) {
	playlistsCacheLock.RLock()
	if cell, ok := playlistsCache[permalink]; ok && cell.Expires.After(time.Now()) {
		playlistsCacheLock.RUnlock()
		return cell.Value, nil
	}
	playlistsCacheLock.RUnlock()

	var p Playlist
	err := Resolve(permalink, &p)
	if err != nil {
		return p, err
	}

	if p.Kind != "playlist" {
		return p, ErrKindNotCorrect
	}

	err = p.Fix(true)
	if err != nil {
		return p, err
	}

	playlistsCacheLock.Lock()
	playlistsCache[permalink] = cached[Playlist]{Value: p, Expires: time.Now().Add(cfg.PlaylistTTL)}
	playlistsCacheLock.Unlock()

	return p, nil
}

func SearchPlaylists(args string) (*Paginated[*Playlist], error) {
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	p := Paginated[*Playlist]{Next: "https://" + api + "/search/playlists" + args + "&client_id=" + cid}
	err = p.Proceed()
	if err != nil {
		return nil, err
	}

	for _, p := range p.Collection {
		p.Fix(false)
	}

	return &p, nil
}

func (p *Playlist) Fix(cached bool) error {
	if cached {
		for _, t := range p.Tracks {
			t.Fix(false)
		}

		err := p.GetMissingTracks()
		if err != nil {
			return err
		}

		p.Artwork = strings.Replace(p.Artwork, "-large.", "-t500x500.", 1)
	} else {
		p.Artwork = strings.Replace(p.Artwork, "-large.", "-t200x200.", 1)
	}

	if cfg.ProxyImages && p.Artwork != "" {
		p.Artwork = "/proxy/images?url=" + url.QueryEscape(p.Artwork)
	}

	p.Author.Fix(false)

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

func GetMissingTracks(missing []MissingTrack) (res []*Track, next []MissingTrack, err error) {
	if len(missing) > 50 {
		next = missing[50:]
		missing = missing[:50]
	}

	res, err = GetTracks(JoinMissingTracks(missing))
	return
}

func GetNextMissingTracks(raw string) (res []*Track, next []string, err error) {
	missing := strings.Split(raw, ",")
	if len(missing) > 50 {
		next = missing[50:]
		missing = missing[:50]
	}

	res, err = GetTracks(strings.Join(missing, ","))
	return
}

func (p *Playlist) GetMissingTracks() error {
	missing := []MissingTrack{}
	for i, track := range p.Tracks {
		if track.Title == "" {
			//fmt.Println(track.ID)
			missing = append(missing, MissingTrack{ID: track.ID, Index: i})
		}
	}

	if len(missing) == 0 {
		return nil
	}

	res, next, err := GetMissingTracks(missing)
	if err != nil {
		return err
	}

	for _, oldTrack := range missing {
		for _, newTrack := range res {
			if newTrack.ID == oldTrack.ID {
				p.Tracks[oldTrack.Index] = newTrack
			}
		}
	}

	p.MissingTracks = JoinMissingTracks(next)

	return nil
}
