package sc

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/maid-zone/soundcloak/lib/textparsing"
	"github.com/segmentio/encoding/json"
	"github.com/valyala/fasthttp"
)

// Functions/structures related to users

var UsersCache = map[string]cached[User]{}
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

	WebProfiles []Link
}

type Link struct {
	URL   string `json:"url"`
	Title string `json:"title"`
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

func (r Repost) Fix(prefs cfg.Preferences) {
	switch r.Type {
	case TrackRepost:
		if r.Track != nil {
			r.Track.Fix(false, false)
			r.Track.Postfix(prefs, false)
		}
		return
	case PlaylistRepost:
		if r.Playlist != nil {
			r.Playlist.Fix(false, false) // err always nil if cached == false
			r.Playlist.Postfix(prefs, false, false)
		}
		return
	}
}

// same thing
type Like struct {
	Track    *Track
	Playlist *Playlist
}

func (l Like) Fix(prefs cfg.Preferences) {
	if l.Track != nil {
		l.Track.Fix(false, false)
		l.Track.Postfix(prefs, false)
	} else if l.Playlist != nil {
		l.Playlist.Fix(false, false)
		l.Playlist.Postfix(prefs, false, false)
	}
}
func GetUser(permalink string) (User, error) {
	usersCacheLock.RLock()
	if cell, ok := UsersCache[permalink]; ok && cell.Expires.After(time.Now()) {
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

	if cfg.GetWebProfiles {
		err = u.GetWebProfiles()
		if err != nil {
			return u, err
		}
	}
	u.Fix(true)

	usersCacheLock.Lock()
	UsersCache[permalink] = cached[User]{Value: u, Expires: time.Now().Add(cfg.UserTTL)}
	usersCacheLock.Unlock()

	return u, err
}

func SearchUsers(prefs cfg.Preferences, args string) (*Paginated[*User], error) {
	cid, err := GetClientID()
	if err != nil {
		return nil, err
	}

	p := Paginated[*User]{Next: "https://" + api + "/search/users" + args + "&client_id=" + cid}
	err = p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, u := range p.Collection {
		u.Fix(false)
		u.Postfix(prefs)
	}

	return &p, nil
}

func (u User) GetTracks(prefs cfg.Preferences, args string) (*Paginated[*Track], error) {
	p := Paginated[*Track]{
		Next: "https://" + api + "/users/" + u.ID + "/tracks" + args,
	}

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

func (u *User) Fix(large bool) {
	if large {
		u.Avatar = strings.Replace(u.Avatar, "-large.", "-t500x500.", 1)
	} else {
		u.Avatar = strings.Replace(u.Avatar, "-large.", "-t200x200.", 1)
	}

	// maybe hardcoding it isn't the best decision, but it should be ok
	if u.Avatar == "https://a1.sndcdn.com/images/default_avatar_large.png" {
		u.Avatar = ""
	}

	ls := strings.Split(u.ID, ":")
	u.ID = ls[len(ls)-1]

	for i, l := range u.WebProfiles {
		if textparsing.IsEmail(l.URL) {
			l.URL = "mailto:" + l.URL
			u.WebProfiles[i] = l
		} else {
			parsed, err := url.Parse(l.URL)
			if err == nil {
				if parsed.Host == "soundcloud.com" || strings.HasSuffix(parsed.Host, ".soundcloud.com") {
					l.URL = "/" + strings.Join(strings.Split(l.URL, "/")[3:], "/")
					if parsed.Host == "on.soundcloud.com" {
						l.URL = "/on" + l.URL
					}

					u.WebProfiles[i] = l
				}
			}
		}
	}
}

func (u *User) Postfix(prefs cfg.Preferences) {
	if cfg.ProxyImages && *prefs.ProxyImages && u.Avatar != "" {
		u.Avatar = "/_/proxy/images?url=" + url.QueryEscape(u.Avatar)
	}
}

func (u User) GetPlaylists(prefs cfg.Preferences, args string) (*Paginated[*Playlist], error) {
	p := Paginated[*Playlist]{
		Next: "https://" + api + "/users/" + u.ID + "/playlists_without_albums" + args,
	}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, pl := range p.Collection {
		pl.Fix(false, false)
		pl.Postfix(prefs, false, false)
	}

	return &p, nil
}

func (u User) GetAlbums(prefs cfg.Preferences, args string) (*Paginated[*Playlist], error) {
	p := Paginated[*Playlist]{
		Next: "https://" + api + "/users/" + u.ID + "/albums" + args,
	}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, pl := range p.Collection {
		pl.Fix(false, false)
		pl.Postfix(prefs, false, false)
	}

	return &p, nil
}

func (u User) GetReposts(prefs cfg.Preferences, args string) (*Paginated[*Repost], error) {
	p := Paginated[*Repost]{
		Next: "https://" + api + "/stream/users/" + u.ID + "/reposts" + args,
	}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, r := range p.Collection {
		r.Fix(prefs)
	}

	return &p, nil
}

func (u User) GetLikes(prefs cfg.Preferences, args string) (*Paginated[*Like], error) {
	p := Paginated[*Like]{
		Next: "https://" + api + "/users/" + u.ID + "/likes" + args,
	}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, l := range p.Collection {
		l.Fix(prefs)
	}

	return &p, nil
}

func (u *User) GetWebProfiles() error {
	cid, err := GetClientID()
	if err != nil {
		return err
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("https://" + api + "/users/" + u.ID + "/web-profiles?client_id=" + cid)
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = DoWithRetry(httpc, req, resp)
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("getwebprofiles: got status code %d", resp.StatusCode())
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	return json.Unmarshal(data, &u.WebProfiles)
}
