package sc

import (
	"net/url"
	"strings"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
)

// Functions/structures related to featured/suggested content

type PlaylistOrUser struct {
	Kind      string `json:"kind"` // "playlist" or "system-playlist" or "user"
	Permalink string `json:"permalink"`

	// User-specific
	Avatar   string `json:"avatar_url"`
	Username string `json:"username"`
	FullName string `json:"full_name"`

	// Playlist-specific
	Title  string `json:"title"`
	Author struct {
		Permalink string `string:"permalink"`
	} `json:"author"`
	Artwork    string `json:"artwork_url"`
	TrackCount int64  `json:"track_count"`
}

func (p PlaylistOrUser) Href() string {
	switch p.Kind {
	case "system-playlist":
		return "/discover/sets/" + p.Permalink
	case "playlist":
		return "/" + p.Author.Permalink + "/sets/" + p.Permalink
	default:
		return "/" + p.Permalink
	}
}

func (p *PlaylistOrUser) Fix(prefs cfg.Preferences) {
	switch p.Kind {
	case "user":
		if p.Avatar == "https://a1.sndcdn.com/images/default_avatar_large.png" {
			p.Avatar = ""
		} else {
			p.Avatar = strings.Replace(p.Avatar, "-large.", "-t200x200.", 1)
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

type Selection struct {
	Title string                     `json:"title"`
	Kind  string                     `json:"kind"`  // should always be "selection"!
	Items Paginated[*PlaylistOrUser] `json:"items"` // ?? why
}

func GetSelections(cid string, prefs cfg.Preferences) (*Paginated[*Selection], error) {
	// There is no pagination
	p := Paginated[*Selection]{Next: "https://" + api + "/mixed-selections?limit=20"}
	err := p.Proceed(cid, false)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(prefs)
	}

	return &p, nil
}

func (s *Selection) Fix(prefs cfg.Preferences) {
	for _, p := range s.Items.Collection {
		p.Fix(prefs)
	}
}
