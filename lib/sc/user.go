package sc

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/textparsing"
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

// Functions/structures related to users

var UsersCache = map[string]cached[User]{}
var usersCacheLock = &sync.RWMutex{}

type User struct {
	Avatar       string      `json:"avatar_url"`
	CreatedAt    string      `json:"created_at"`
	Description  string      `json:"description"`
	FullName     string      `json:"full_name"`
	Kind         string      `json:"kind"` // should always be "user"!
	LastModified string      `json:"last_modified"`
	Permalink    string      `json:"permalink"`
	ID           json.Number `json:"id"`
	Username     string      `json:"username"`
	Station      string      `json:"station_permalink"`
	WebProfiles  []Link      `json:",omitempty"`
	Followers    int64       `json:"followers_count"`
	Following    int64       `json:"followings_count"`
	Liked        int64       `json:"likes_count"`
	Playlists    int64       `json:"playlist_count"`
	Tracks       int64       `json:"track_count"`
	Verified     bool        `json:"verified"`
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
	Track    *Track    // type == track-report
	Playlist *Playlist // type == playlist-repost
	Type     RepostType
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
	if cell, ok := UsersCache[permalink]; ok {
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

func SearchUsers(prefs cfg.Preferences, args []byte) (*Paginated[*User], error) {
	uri := baseUri()
	uri.SetPath("/search/users")
	uri.SetQueryStringBytes(args)
	p := Paginated[*User]{Next: uri}
	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, u := range p.Collection {
		u.Fix(false)
		u.Postfix(prefs)
	}

	return &p, nil
}

func (u User) baseUri(subpath, args string) *fasthttp.URI {
	uri := baseUri()
	uri.SetPath("/users/" + string(u.ID) + "/" + subpath)
	uri.SetQueryString(args)
	return uri
}

func (u User) GetTracks(prefs cfg.Preferences, args string) (*Paginated[*Track], error) {
	p := Paginated[*Track]{Next: u.baseUri("tracks", args)}

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
	p := Paginated[*Playlist]{Next: u.baseUri("playlists_without_albums", args)}

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
	p := Paginated[*Playlist]{Next: u.baseUri("albums", args)}

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
	uri := baseUri()
	uri.SetPath("/stream/users/" + string(u.ID) + "/reposts")
	uri.SetQueryString(args)
	p := Paginated[*Repost]{Next: uri}

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
	p := Paginated[*Like]{Next: u.baseUri("likes", args)}

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
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	baseUriReq(req)
	req.URI().SetPath("/users/soundcloud:users:" + string(u.ID) + "/web-profiles")
	req.URI().QueryArgs().Set("client_id", ClientID)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
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

func (u User) GetRelated(prefs cfg.Preferences) ([]*User, error) {
	uri := baseUri()
	uri.SetPath("/users/" + string(u.ID) + "/relatedartists")
	uri.QueryArgs().Set("page_size", "20")
	p := Paginated[*User]{
		Next: uri,
	}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, u := range p.Collection {
		u.Fix(false)
		u.Postfix(prefs)
	}

	return p.Collection, nil
}

func (u User) GetTopTracks(prefs cfg.Preferences) ([]*Track, error) {
	uri := baseUri()
	uri.SetPath("/users/" + string(u.ID) + "/toptracks")
	uri.QueryArgs().Set("limit", "10")
	p := Paginated[*Track]{Next: uri}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(false, false)
		t.Postfix(prefs, false)
	}

	return p.Collection, nil
}

func (u User) GetFollowers(prefs cfg.Preferences, args string) (*Paginated[*User], error) {
	p := Paginated[*User]{Next: u.baseUri("followers", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, u := range p.Collection {
		u.Fix(false)
		u.Postfix(prefs)
	}

	return &p, nil
}

func (u User) GetFollowing(prefs cfg.Preferences, args string) (*Paginated[*User], error) {
	p := Paginated[*User]{Next: u.baseUri("followings", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, u := range p.Collection {
		u.Fix(false)
		u.Postfix(prefs)
	}

	return &p, nil
}

func t(s string) string {
	parsed, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return parsed.Format(time.RFC1123Z)
	}

	return ""
}

// TODO: maybe add option for caching generated feeds? could benefit when many people follow same artists
func (u *User) GenerateFeed(ctx context.Context, prefs cfg.Preferences, base string) ([]byte, error) {
	tracks, err := u.GetTracks(prefs, "limit=20")
	if err != nil {
		return nil, err
	}

	f := RssFeed{
		Title:          "Tracks from " + u.Username,
		Link:           base + "/" + u.Permalink,
		ManagingEditor: u.Username + " (@" + u.Permalink + ")",

		Category:  "Music",
		Generator: "soundcloak",
		Ttl:       int(cfg.UserTTL / time.Second),
	}
	f.Description = "Recently released tracks by " + f.ManagingEditor

	if len(tracks.Collection) != 0 {
		f.LastBuildDate = t(tracks.Collection[0].LastModified)
		for _, track := range tracks.Collection {
			item := RssItem{
				Title: track.Title,
				Link:  base + "/" + u.Permalink + "/" + track.Permalink,

				Category: track.Genre,
				Guid:     &RssGuid{Id: string(track.ID), IsPermaLink: "false"},
				PubDate:  t(track.LastModified),
			}

			if cfg.ProxyImages && *prefs.ProxyImages {
				track.Artwork = base + track.Artwork
			}

			track.Artwork = strings.Replace(track.Artwork, "-t200x200.", "-original.", 1)

			buf := strings.Builder{}
			err = TrackDescription(prefs, track, item.Link).Render(ctx, &buf)
			if err != nil {
				log.Printf("error generating %s (%s) feed: %s\n", u.Permalink, track.Permalink, err)
				continue
			}

			item.Description = buf.String()
			f.Items = append(f.Items, &item)
		}
	} else {
		f.LastBuildDate = t(u.LastModified)
	}
	f.PubDate = f.LastBuildDate

	return xml.Marshal(RssFeedXml{
		Version:          "2.0",
		Channel:          &f,
		ContentNamespace: "http://purl.org/rss/1.0/modules/content/",
	})
}
