package sc

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
	URN       string `json:"urn"`
	Username  string `json:"username"`
	Verified  bool   `json:"verified"`

	ID string `json:"-"`
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

func (m Media) SelectCompatible() *Transcoding {
	for _, t := range m.Transcodings {
		if t.Format.Protocol == "hls" && t.Format.MimeType == "audio/mpeg" {
			return &t
		}
	}

	return nil
}

type Track struct {
	Artwork     string `json:"artwork_url"`
	Comments    int    `json:"comment_count"`
	CreatedAt   string `json:"created_at"`
	Description string `json:"description"`
	//Duration      int    `json:"duration"` // there are duration and full_duration fields wtf does that mean
	Genre         string `json:"genre"`
	ID            int64  `json:"id"`
	Kind          string `json:"kind"` // should always be "track"!
	LastModified  string `json:"last_modified"`
	Likes         int64  `json:"likes_count"`
	Permalink     string `json:"permalink"`
	Played        int64  `json:"playback_count"`
	TagList       string `json:"tag_list"`
	Title         string `json:"title"`
	Media         Media  `json:"media"`
	Authorization string `json:"track_authorization"`
	Author        User   `json:"user"`
}

type Paginated[T any] struct {
	Collection []T    `json:"collection"`
	Total      int    `json:"total_results"`
	Next       string `json:"next_href"`
}

type Stream struct {
	URL string `json:"url"`
}

type Playlist struct {
	Artwork      string `json:"artwork_url"`
	CreatedAt    string `json:"created_at"`
	Description  string `json:"description"`
	Kind         string `json:"kind"` // should always be "playlist"!
	LastModified string `json:"last_modified"`
	Likes        int64  `json:"likes_count"`
	//ReleaseDate  string  `json:"release_date"`
	TagList string  `json:"tag_list"`
	Title   string  `json:"title"`
	Type    string  `json:"set_type"`
	Album   bool    `json:"is_album"`
	Author  User    `json:"user"`
	Tracks  []Track `json:"tracks"`
}
