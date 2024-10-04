package sc

import (
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

func GetUser(permalink string) (User, error) {
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

	u.Fix()

	usersCacheLock.Lock()
	usersCache[permalink] = cached[User]{Value: u, Expires: time.Now().Add(cfg.UserTTL)}
	usersCacheLock.Unlock()

	return u, err
}

func SearchUsers(args string) (*Paginated[*User], error) {
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
		u.Fix()
	}

	return &p, nil
}

func (u User) GetTracks(args string) (*Paginated[Track], error) {
	p := Paginated[Track]{
		Next: "https://" + api + "/users/" + u.ID + "/tracks" + args,
	}

	err := p.Proceed()
	if err != nil {
		return nil, err
	}

	for _, u := range p.Collection {
		u.Fix()
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

func (u *User) Fix() {
	u.Avatar = strings.Replace(u.Avatar, "-large.", "-t200x200.", 1)
	ls := strings.Split(u.ID, ":")
	u.ID = ls[len(ls)-1]
}
