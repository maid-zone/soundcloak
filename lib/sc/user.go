package sc

import (
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maid-zone/soundcloak/lib/cfg"
)

// Functions/structures related to users

var usersCache = map[string]cached[User]{}
var usersCacheLock = &sync.RWMutex{}

type User struct {
	Avatar       string `json:"avatar_url"`
	CreatedAt    string `json:"created_at"`
	Description  string `json:"description"`
	Followers    int64  `json:"followers_count"`
	Following    int64  `json:"followings_count"`
	FullName     string `json:"full_name"`
	Kind         string `json:"kind"` // should always be "user"!
	LastModified string `json:"last_modified"`
	//Liked        int    `json:"likes_count"`
	Permalink string `json:"permalink"`
	Playlists int64  `json:"playlist_count"`
	Tracks    int64  `json:"track_count"`
	ID        string `json:"urn"`
	Username  string `json:"username"`
	Verified  bool   `json:"verified"`
}

type RepostType string

const (
	TrackRepost    RepostType = "track-repost"
	PlaylistRepost RepostType = "playlist-repost"
)

// not worthy of its own file
type Repost struct {
	Type RepostType

	Track    *Track    // type == track-report
	Playlist *Playlist // type == playlist-repost
}

func (r *Repost) Fix(prefs cfg.Preferences) {
	switch r.Type {
	case TrackRepost:
		if r.Track != nil {
			r.Track.Fix(prefs, false)
		}
		return
	case PlaylistRepost:
		if r.Playlist != nil {
			r.Playlist.Fix(prefs, false) // err always nil if cached == false
		}
		return
	}
}

func GetUser(prefs cfg.Preferences, permalink string) (User, error) {
	usersCacheLock.RLock()
	if cell, ok := usersCache[permalink]; ok && cell.Expires.After(time.Now()) {
		usersCacheLock.RUnlock()
		return cell.Value, nil
	}

	usersCacheLock.RUnlock()

	var u User
	err := Resolve(permalink, &u)
	if err != nil {
		return u, err
	}

	if u.Kind != "user" {
		err = ErrKindNotCorrect
		return u, err
	}

	u.Fix(prefs, true)

	usersCacheLock.Lock()
	usersCache[permalink] = cached[User]{Value: u, Expires: time.Now().Add(cfg.UserTTL)}
	usersCacheLock.Unlock()

	return u, err
}

func SearchUsers(prefs cfg.Preferences, args string) (*Paginated[*User], error) {
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	p := Paginated[*User]{Next: "https://" + api + "/search/users" + args + "&client_id=" + cid}
	err = p.Proceed()
	if err != nil {
		return nil, err
	}

	for _, u := range p.Collection {
		u.Fix(prefs, false)
	}

	return &p, nil
}

func (u User) GetTracks(prefs cfg.Preferences, args string) (*Paginated[*Track], error) {
	p := Paginated[*Track]{
		Next: "https://" + api + "/users/" + u.ID + "/tracks" + args,
	}

	err := p.Proceed()
	if err != nil {
		return nil, err
	}

	for _, u := range p.Collection {
		u.Fix(prefs, false)
	}

	return &p, nil
}

func (u User) FormatDescription() string {
	desc := u.Description
	if u.Description != "" {
		desc += "\n\n"
	}

	desc += strconv.FormatInt(u.Followers, 10) + " followers | " + strconv.FormatInt(u.Following, 10) + " following"
	desc += "\n" + strconv.FormatInt(u.Tracks, 10) + " tracks | " + strconv.FormatInt(u.Playlists, 10) + " playlists"
	desc += "\nCreated: " + u.CreatedAt
	desc += "\nLast modified: " + u.LastModified

	return desc
}

func (u User) FormatUsername() string {
	res := u.Username
	if u.Verified {
		res += " ☑️"
	}

	return res
}

func (u *User) Fix(prefs cfg.Preferences, large bool) {
	if large {
		u.Avatar = strings.Replace(u.Avatar, "-large.", "-t500x500.", 1)
	} else {
		u.Avatar = strings.Replace(u.Avatar, "-large.", "-t200x200.", 1)
	}

	// maybe hardcoding it isn't the best decision, but it should be ok
	if u.Avatar == "https://a1.sndcdn.com/images/default_avatar_large.png" {
		u.Avatar = ""
	}

	if cfg.ProxyImages && *prefs.ProxyImages && u.Avatar != "" {
		u.Avatar = "/_/proxy/images?url=" + url.QueryEscape(u.Avatar)
	}

	ls := strings.Split(u.ID, ":")
	u.ID = ls[len(ls)-1]
}

func (u *User) GetPlaylists(prefs cfg.Preferences, args string) (*Paginated[*Playlist], error) {
	p := Paginated[*Playlist]{
		Next: "https://" + api + "/users/" + u.ID + "/playlists_without_albums" + args,
	}

	err := p.Proceed()
	if err != nil {
		return nil, err
	}

	for _, pl := range p.Collection {
		pl.Fix(prefs, false)
	}

	return &p, nil
}

func (u *User) GetAlbums(prefs cfg.Preferences, args string) (*Paginated[*Playlist], error) {
	p := Paginated[*Playlist]{
		Next: "https://" + api + "/users/" + u.ID + "/albums" + args,
	}

	err := p.Proceed()
	if err != nil {
		return nil, err
	}

	for _, pl := range p.Collection {
		pl.Fix(prefs, false)
	}

	return &p, nil
}

func (u *User) GetReposts(prefs cfg.Preferences, args string) (*Paginated[*Repost], error) {
	p := Paginated[*Repost]{
		Next: "https://" + api + "/stream/users/" + u.ID + "/reposts" + args,
	}

	err := p.Proceed()
	if err != nil {
		return nil, err
	}

	for _, r := range p.Collection {
		r.Fix(prefs)
	}

	return &p, nil
}
