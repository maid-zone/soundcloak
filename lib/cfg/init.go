package cfg

import (
	"log"
	"os"
	"time"

	"github.com/segmentio/encoding/json"
)

// You can now use a soundcloak.json file to configure!
// or you can specify a different path with environment variable SOUNDCLOAK_CONFIG
// it's not needed to specify every key in your config, it will use the default values for keys you haven't specified
// also, when specifying time, specify it in seconds (instead of nanoseconds as time.Duration)

const (
	// Downloads the HLS stream on the backend, and restreams it to frontend as a file. Requires no JS, but less stable client-side
	RestreamPlayer string = "restream"
	// Downloads the HLS stream on the frontend (proxying can be enabled). Requires JS, more stable client-side
	HLSPlayer string = "hls"
	// Disables the song player
	NonePlayer string = "none"
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
}

// // config // //

// Default preferences. You can override those preferences in the config file, otherwise they default to values depending on your config
// (so, if you have ProxyStreams enabled - it will be enabled for the user by default and etc, or if you enabled Restream, the default player will be RestreamPlayer instead of HLSPlayer)
var DefaultPreferences Preferences

// proxy images (user avatars, track/playlist covers)
var ProxyImages = false
var ImageCacheControl = "max-age=600, public, immutable" // browser-side 10 minutes cache by default, only used for proxied images

// proxy streams (hls playlist files and track parts)
var ProxyStreams = false

// restream the HLS playlist as just a file, makes soundcloak usable with zero JS
// If this setting is set to true, ProxyStreams and FullyPreloadTrack will be ignored (you could count this as a replacement for having both as true, also should be a bit more effective)
// You can also easily download the songs this way (right click => save audio as..., the only downside is that there is no metadata)
var Restream = false
var RestreamCacheControl = "max-age=604800, public, immutable"

// enable /_/info endpoint (shows if some settings are enabled/disabled)
var InstanceInfo = true

// time-to-live for clientid cache
// larger number will improve performance (no need to recheck everytime) but might make soundcloak briefly unusable for a larger amount of time if the client id is invalidated
var ClientIDTTL = 30 * time.Minute

// time-to-live for user profile cache
var UserTTL = 10 * time.Minute

// delay between cleanup of user cache
var UserCacheCleanDelay = UserTTL / 4

// time-to-live for track cache
var TrackTTL = 10 * time.Minute

// delay between cleanup of track cache
var TrackCacheCleanDelay = TrackTTL / 4

// time-to-live for playlist cache
var PlaylistTTL = 10 * time.Minute

// delay between cleanup of playlist cache
var PlaylistCacheCleanDelay = PlaylistTTL / 4

// default fasthttp one was causing connections to be stuck? todo make it cycle browser useragents or just choose random at startup
var UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.3"

// time-to-live for dns cache
var DNSCacheTTL = 10 * time.Minute

// // // some webserver configuration, put here to make it easier to configure what you need // // //
// more info can be found here: https://docs.gofiber.io/api/fiber#config

// run soundcloak on this address (localhost:4664 by default)
var Addr = ":4664"

// run multiple instances of soundcloud locally to be able to handle more requests
// each one will be a separate process, so they will have separate cache
var Prefork = false

// Enables TLS Early Data (0-RTT / zero round trip time)
// This can reduce latency, but also makes requests replayable (not that much of a concern for soundcloak, since there are no authenticated operations)
// There might be breakage when used together with TrustedProxyCheck and the proxy is untrusted
var EarlyData = false

// use X-Forwarded-* headers ONLY when ip is in TrustedProxies list
// when disabled, the X-Forwarded-* headers will be blindly used
var TrustedProxyCheck = true

// list of ips or ip ranges of trusted proxies (check above)
var TrustedProxies = []string{}

// // end of config // //

func defaultPreferences() {
	var p string
	if Restream {
		p = RestreamPlayer
	} else {
		p = HLSPlayer
	}
	DefaultPreferences.Player = &p

	DefaultPreferences.ProxyStreams = &ProxyStreams

	var f bool
	var t = true
	DefaultPreferences.FullyPreloadTrack = &f

	DefaultPreferences.ProxyImages = &ProxyImages

	DefaultPreferences.ParseDescriptions = &t
}

func init() {
	filename := "soundcloak.json"
	if env := os.Getenv("SOUNDCLOAK_CONFIG"); env != "" {
		filename = env
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("failed to load config from %s: %s\n", filename, err)
		defaultPreferences()
		return
	}

	var config struct {
		DefaultPreferences      *Preferences
		ProxyImages             *bool
		ImageCacheControl       *string
		ProxyStreams            *bool
		Restream                *bool
		RestreamCacheControl    *string
		InstanceInfo            *bool
		ClientIDTTL             *time.Duration
		UserTTL                 *time.Duration
		UserCacheCleanDelay     *time.Duration
		TrackTTL                *time.Duration
		TrackCacheCleanDelay    *time.Duration
		PlaylistTTL             *time.Duration
		PlaylistCacheCleanDelay *time.Duration
		UserAgent               *string
		DNSCacheTTL             *time.Duration
		Addr                    *string
		Prefork                 *bool
		EarlyData               *bool
		TrustedProxyCheck       *bool
		TrustedProxies          *[]string
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Printf("failed to parse config from %s: %s\n", filename, err)
		defaultPreferences()
		return
	}

	// tedious
	// i've decided to fully override to make it easier to change default config later on
	if config.ProxyImages != nil {
		ProxyImages = *config.ProxyImages
	}
	if config.ImageCacheControl != nil {
		ImageCacheControl = *config.ImageCacheControl
	}
	if config.ProxyStreams != nil {
		ProxyStreams = *config.ProxyStreams
	}
	if config.Restream != nil {
		Restream = *config.Restream
	}
	if config.RestreamCacheControl != nil {
		RestreamCacheControl = *config.RestreamCacheControl
	}
	if config.InstanceInfo != nil {
		InstanceInfo = *config.InstanceInfo
	}
	if config.ClientIDTTL != nil {
		ClientIDTTL = *config.ClientIDTTL * time.Second
	}
	if config.UserTTL != nil {
		UserTTL = *config.UserTTL * time.Second
	}
	if config.UserCacheCleanDelay != nil {
		UserCacheCleanDelay = *config.UserCacheCleanDelay * time.Second
	}
	if config.TrackTTL != nil {
		TrackTTL = *config.TrackTTL * time.Second
	}
	if config.TrackCacheCleanDelay != nil {
		TrackCacheCleanDelay = *config.TrackCacheCleanDelay * time.Second
	}
	if config.PlaylistTTL != nil {
		PlaylistTTL = *config.PlaylistTTL * time.Second
	}
	if config.PlaylistCacheCleanDelay != nil {
		PlaylistCacheCleanDelay = *config.PlaylistCacheCleanDelay * time.Second
	}
	if config.UserAgent != nil {
		UserAgent = *config.UserAgent
	}
	if config.DNSCacheTTL != nil {
		DNSCacheTTL = *config.DNSCacheTTL * time.Second
	}
	if config.Addr != nil {
		Addr = *config.Addr
	}
	if config.Prefork != nil {
		Prefork = *config.Prefork
	}
	if config.EarlyData != nil {
		EarlyData = *config.EarlyData
	}
	if config.TrustedProxyCheck != nil {
		TrustedProxyCheck = *config.TrustedProxyCheck
	}
	if config.TrustedProxies != nil {
		TrustedProxies = *config.TrustedProxies
	}

	// defaults are:
	// Player: RestreamPlayer if Restream is enabled, otherwise - HLSPlayer
	// ProxyStreams: same as ProxyStreams in your config (false by default)
	// FullyPreloadTrack: false
	// ProxyImages: same as ProxyImages in your config (false by default)
	// ParseDescriptions: true
	if config.DefaultPreferences != nil {
		var f bool
		var t = true
		if config.DefaultPreferences.Player != nil {
			DefaultPreferences.Player = config.DefaultPreferences.Player
		} else {
			var p string
			if Restream {
				p = RestreamPlayer
			} else {
				p = HLSPlayer
			}
			DefaultPreferences.Player = &p
		}

		if config.DefaultPreferences.ProxyStreams != nil {
			DefaultPreferences.ProxyStreams = config.DefaultPreferences.ProxyStreams
		} else {
			DefaultPreferences.ProxyStreams = &ProxyStreams
		}

		if config.DefaultPreferences.FullyPreloadTrack != nil {
			DefaultPreferences.FullyPreloadTrack = config.DefaultPreferences.FullyPreloadTrack
		} else {
			DefaultPreferences.FullyPreloadTrack = &f
		}

		if config.DefaultPreferences.ProxyImages != nil {
			DefaultPreferences.ProxyImages = config.DefaultPreferences.ProxyImages
		} else {
			DefaultPreferences.ProxyImages = &ProxyImages
		}

		if config.DefaultPreferences.ParseDescriptions != nil {
			DefaultPreferences.ParseDescriptions = config.DefaultPreferences.ParseDescriptions
		} else {
			DefaultPreferences.ParseDescriptions = &t
		}
	} else {
		defaultPreferences()
	}
}
