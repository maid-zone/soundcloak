package cfg

import (
	"time"
	"unsafe"
)

// seems soundcloud has 4 of these (i1, i2, i3, i4)
// they point to the same ip from my observations, and they all serve the same files
const ImageCDN = "i1.sndcdn.com"
const HLSCDN = "cf-hls-media.sndcdn.com"
const HLSAACCDN = "playback.media-streaming.soundcloud.cloud"

// Note: we don't need DialDualStack for clients, soundcloud has no ipv6 support and operates only over http1.1 :D
const MaxIdleConnDuration = 4 * time.Hour

var True = true
var False = false

const (
	// Downloads the HLS stream on the backend, and restreams it to frontend as a file. Requires no JS, but less stable client-side
	RestreamPlayer string = "restream"
	// Downloads the HLS stream on the frontend (proxying can be enabled). Requires JS, more stable client-side
	HLSPlayer string = "hls"
	// Disables the song player
	NonePlayer string = "none"
)

const (
	// Just plays every song in order, one after another
	AutoplayNormal string = "normal"
	// Randomly selects a song to play from the playlist
	AutoplayRandom string = "random"
)

const (
	// choose best for quality/size (AudioAAC over AudioOpus over AudioMP3)
	AudioBest string = "best"

	// 160kbps m4a AAC audio, rarely available (fallback to AudioMP3 if unavailable)
	AudioAAC string = "aac"

	// 72kbps ogg opus audio, usually available 99% of the time (fallback to AudioMP3 if unavailable)
	AudioOpus string = "opus"

	// 128kbps mp3 audio, always available, good for compatibility
	AudioMP3 string = "mpeg"
)

type Preferences struct {
	Player       *string
	ProxyStreams *bool

	// fully loads the track on page load
	// this option is here since the stream expires after some time (5 minutes? correct me if im wrong)
	// if the stream isn't fully loaded before it expires - you'll need to reload the page
	FullyPreloadTrack *bool

	ProxyImages *bool

	// Highlight @username, https://example.com and email@example.com in text as clickable links
	ParseDescriptions *bool

	// Automatically play next track in playlists
	AutoplayNextTrack *bool

	// Automatically play next related track
	AutoplayNextRelatedTrack *bool

	DefaultAutoplayMode *string // "normal" or "random"

	// Check above for more info
	// Probably best to keep all at "mpeg" by default for compatibility
	HLSAudio      *string // Please don't use "opus" or "best". hls.js doesn't work with ogg/opus
	RestreamAudio *string // You can actually use anything here
	DownloadAudio *string // "aac" may not play well with some players

	ShowAudio *bool // display audio (aac, opus, mpeg etc) under track player

	SearchSuggestions *bool // load search suggestions on main page

	DynamicLoadComments *bool // dynamic comments loader without leaving track page
}

func B2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func S2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
