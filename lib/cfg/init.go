package cfg

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

// You can now use a soundcloak.json file to configure!
// or you can specify a different path with environment variable SOUNDCLOAK_CONFIG
// it's not needed to specify every key in your config, it will use the default values for keys you haven't specified
// also, when specifying time, specify it in seconds (instead of nanoseconds as time.Duration)

// // config // //

// Retrieve links users set in their profile (social media, website, etc)
var GetWebProfiles = true

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
var RestreamCacheControl = "max-age=3600, public, immutable"

// enable /_/info endpoint (shows if some settings are enabled/disabled)
var InstanceInfo = true

// time-to-live for clientid cache
// larger number will improve performance (no need to recheck everytime) but might make soundcloak briefly unusable for a larger amount of time if the client id is invalidated
// I went with 4 hours, since those clientids still remain active for quite some time, even after soundcloud updates it
var ClientIDTTL = 4 * time.Hour

// time-to-live for user profile cache
var UserTTL = 20 * time.Minute

// delay between cleanup of user cache
var UserCacheCleanDelay = UserTTL / 4

// time-to-live for track cache
var TrackTTL = 20 * time.Minute

// delay between cleanup of track cache
var TrackCacheCleanDelay = TrackTTL / 4

// time-to-live for playlist cache
var PlaylistTTL = 20 * time.Minute

// delay between cleanup of playlist cache
var PlaylistCacheCleanDelay = PlaylistTTL / 4

// default fasthttp one was causing connections to be stuck? todo make it cycle browser useragents or just choose random at startup
var UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.3"

// time-to-live for dns cache
var DNSCacheTTL = 60 * time.Minute

// enab;e api
var EnableAPI = false

// // // some webserver configuration, put here to make it easier to configure what you need // // //
// more info can be found here: https://docs.gofiber.io/api/fiber#config

// where to listen
// can be tcp4 (bind to ipv4 address), tcp6 (bind to ipv6 address) or unix (bind to unix socket)
var Network = "tcp4"

// run soundcloak on this address (localhost:4664 by default)
var Addr = ":4664"

// unix socket perms wow
var UnixSocketPerms os.FileMode = 0775

// run multiple instances of soundcloak locally to be able to handle more requests
// each one will be a separate process, so they will have separate cache
var Prefork = false

// use X-Forwarded-* headers ONLY when ip is in TrustedProxies list
// when disabled, the X-Forwarded-* headers will be blindly used
var TrustedProxyCheck = true

// list of ips or ip ranges of trusted proxies (check above)
var TrustedProxies = []string{}

// generate code for config (run soundcloakctl config codegen or use docker image - it runs it for you)
var CodegenConfig = false

// use static files embedded in binary
var EmbedFiles = false

// // end of config // //

// defaults are:
// Player: RestreamPlayer if Restream is enabled, otherwise - HLSPlayer
// ProxyStreams: same as ProxyStreams in your config (false by default)
// FullyPreloadTrack: false
// ProxyImages: same as ProxyImages in your config (false by default)
// ParseDescriptions: true
// AutoplayNextTrack: false
// DefaultAutoplayMode: AutoplayNormal
// HLSAudio: AudioMP3
// RestreamAudio: AudioMP3
// DownloadAudio: AudioMP3
// ShowAudio: false
// SearchSuggestions: false
// DynamicLoadComments: false
func defaultPreferences() {
	var p string
	if Restream {
		p = RestreamPlayer
	} else {
		p = HLSPlayer
	}
	DefaultPreferences.Player = &p

	DefaultPreferences.ProxyStreams = &ProxyStreams

	DefaultPreferences.FullyPreloadTrack = &False

	DefaultPreferences.ProxyImages = &ProxyImages

	DefaultPreferences.ParseDescriptions = &True
	DefaultPreferences.AutoplayNextTrack = &False
	DefaultPreferences.AutoplayNextRelatedTrack = &False

	p2 := AutoplayNormal
	DefaultPreferences.DefaultAutoplayMode = &p2

	p3 := AudioMP3
	DefaultPreferences.HLSAudio = &p3
	DefaultPreferences.RestreamAudio = &p3
	DefaultPreferences.DownloadAudio = &p3

	DefaultPreferences.ShowAudio = &False

	DefaultPreferences.SearchSuggestions = &False
	DefaultPreferences.DynamicLoadComments = &False
	DefaultPreferences.KeepPlayerFocus = &False
}

func loadDefaultPreferences(loaded Preferences) {
	if loaded.Player != nil {
		DefaultPreferences.Player = loaded.Player
	} else {
		var p string
		if Restream {
			p = RestreamPlayer
		} else {
			p = HLSPlayer
		}
		DefaultPreferences.Player = &p
	}

	if loaded.ProxyStreams != nil {
		DefaultPreferences.ProxyStreams = loaded.ProxyStreams
	} else {
		DefaultPreferences.ProxyStreams = &ProxyStreams
	}

	if loaded.FullyPreloadTrack != nil {
		DefaultPreferences.FullyPreloadTrack = loaded.FullyPreloadTrack
	} else {
		DefaultPreferences.FullyPreloadTrack = &False
	}

	if loaded.ProxyImages != nil {
		DefaultPreferences.ProxyImages = loaded.ProxyImages
	} else {
		DefaultPreferences.ProxyImages = &ProxyImages
	}

	if loaded.ParseDescriptions != nil {
		DefaultPreferences.ParseDescriptions = loaded.ParseDescriptions
	} else {
		DefaultPreferences.ParseDescriptions = &True
	}

	if loaded.AutoplayNextTrack != nil {
		DefaultPreferences.AutoplayNextTrack = loaded.AutoplayNextTrack
	} else {
		DefaultPreferences.AutoplayNextTrack = &False
	}

	if loaded.AutoplayNextRelatedTrack != nil {
		DefaultPreferences.AutoplayNextRelatedTrack = loaded.AutoplayNextRelatedTrack
	} else {
		DefaultPreferences.AutoplayNextRelatedTrack = &False
	}

	if loaded.DefaultAutoplayMode != nil {
		DefaultPreferences.DefaultAutoplayMode = loaded.DefaultAutoplayMode
	} else {
		p := AutoplayNormal
		DefaultPreferences.DefaultAutoplayMode = &p
	}

	p := AudioMP3
	if loaded.HLSAudio != nil {
		DefaultPreferences.HLSAudio = loaded.HLSAudio
	} else {
		DefaultPreferences.HLSAudio = &p
	}

	if loaded.RestreamAudio != nil {
		DefaultPreferences.RestreamAudio = loaded.RestreamAudio
	} else {
		DefaultPreferences.RestreamAudio = &p
	}

	if loaded.DownloadAudio != nil {
		DefaultPreferences.DownloadAudio = loaded.DownloadAudio
	} else {
		DefaultPreferences.DownloadAudio = &p
	}

	if loaded.ShowAudio != nil {
		DefaultPreferences.ShowAudio = loaded.ShowAudio
	} else {
		DefaultPreferences.ShowAudio = &False
	}

	if loaded.SearchSuggestions != nil {
		DefaultPreferences.SearchSuggestions = loaded.SearchSuggestions
	} else {
		DefaultPreferences.SearchSuggestions = &False
	}

	if loaded.DynamicLoadComments != nil {
		DefaultPreferences.DynamicLoadComments = loaded.DynamicLoadComments
	} else {
		DefaultPreferences.DynamicLoadComments = &False
	}

	if loaded.KeepPlayerFocus != nil {
		DefaultPreferences.KeepPlayerFocus = loaded.KeepPlayerFocus
	} else {
		DefaultPreferences.KeepPlayerFocus = &False
	}
}

func boolean(in string) bool {
	return strings.Trim(strings.ToLower(in), " ") == "true"
}

func fromEnv() error {
	env := os.Getenv("GET_WEB_PROFILES")
	if env != "" {
		GetWebProfiles = boolean(env)
	}

	env = os.Getenv("DEFAULT_PREFERENCES")
	if env != "" {
		var p Preferences
		err := json.Unmarshal(S2b(env), &p)
		if err != nil {
			return err
		}

		loadDefaultPreferences(p)
	} else {
		defaultPreferences()
	}

	env = os.Getenv("PROXY_IMAGES")
	if env != "" {
		ProxyImages = boolean(env)
	}

	env = os.Getenv("IMAGE_CACHE_CONTROL")
	if env != "" {
		ImageCacheControl = env
	}

	env = os.Getenv("PROXY_STREAMS")
	if env != "" {
		ProxyStreams = boolean(env)
	}

	env = os.Getenv("RESTREAM")
	if env != "" {
		Restream = boolean(env)
	}

	env = os.Getenv("RESTREAM_CACHE_CONTROL")
	if env != "" {
		RestreamCacheControl = env
	}

	env = os.Getenv("INSTANCE_INFO")
	if env != "" {
		InstanceInfo = boolean(env)
	}

	env = os.Getenv("CLIENT_ID_TTL")
	if env != "" {
		num, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			return err
		}

		ClientIDTTL = time.Duration(num) * time.Second
	}

	env = os.Getenv("USER_TTL")
	if env != "" {
		num, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			return err
		}

		UserTTL = time.Duration(num) * time.Second
	}

	env = os.Getenv("USER_CACHE_CLEAN_DELAY")
	if env != "" {
		num, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			return err
		}

		UserCacheCleanDelay = time.Duration(num) * time.Second
	}

	env = os.Getenv("TRACK_TTL")
	if env != "" {
		num, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			return err
		}

		TrackTTL = time.Duration(num) * time.Second
	}

	env = os.Getenv("TRACK_CACHE_CLEAN_DELAY")
	if env != "" {
		num, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			return err
		}

		TrackCacheCleanDelay = time.Duration(num) * time.Second
	}

	env = os.Getenv("PLAYLIST_TTL")
	if env != "" {
		num, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			return err
		}

		PlaylistTTL = time.Duration(num) * time.Second
	}

	env = os.Getenv("PLAYLIST_CACHE_CLEAN_DELAY")
	if env != "" {
		num, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			return err
		}

		PlaylistCacheCleanDelay = time.Duration(num) * time.Second
	}

	env = os.Getenv("USER_AGENT")
	if env != "" {
		UserAgent = env
	}

	env = os.Getenv("DNS_CACHE_TTL")
	if env != "" {
		num, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			return err
		}

		DNSCacheTTL = time.Duration(num) * time.Second
	}

	env = os.Getenv("ENABLE_API")
	if env != "" {
		EnableAPI = boolean(env)
	}

	env = os.Getenv("NETWORK")
	if env != "" {
		Network = env
	}

	env = os.Getenv("ADDR")
	if env != "" {
		Addr = env
	}

	env = os.Getenv("UNIX_SOCKET_PERMS")
	if env != "" {
		p, err := strconv.ParseUint(env, 0, 32)
		if err != nil {
			return err
		}

		UnixSocketPerms = os.FileMode(p)
	}

	env = os.Getenv("PREFORK")
	if env != "" {
		Prefork = boolean(env)
	}

	env = os.Getenv("TRUSTED_PROXY_CHECK")
	if env != "" {
		TrustedProxyCheck = boolean(env)
	}

	env = os.Getenv("TRUSTED_PROXIES")
	if env != "" {
		var p []string
		err := json.Unmarshal(S2b(env), &p)
		if err != nil {
			return err
		}

		TrustedProxies = p
	}

	env = os.Getenv("CODEGEN_CONFIG")
	if env != "" {
		CodegenConfig = boolean(env)
	}

	env = os.Getenv("EMBED_FILES")
	if env != "" {
		EmbedFiles = boolean(env)
	}

	return nil
}

func init() {
	filename := "soundcloak.json"
	if env := os.Getenv("SOUNDCLOAK_CONFIG"); env == "FROM_ENV" {
		err := fromEnv()
		if err != nil {
			log.Println("Warning: failed to load config from environment:", err)
			defaultPreferences()
		}

		return
	} else if env != "" {
		filename = env
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Warning: failed to load config from %s: %s\n", filename, err)
		defaultPreferences()
		return
	}

	var config struct {
		GetWebProfiles          *bool
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
		EnableAPI               *bool
		Network                 *string
		Addr                    *string
		UnixSocketPerms         *string
		Prefork                 *bool
		TrustedProxyCheck       *bool
		TrustedProxies          *[]string
		CodegenConfig           *bool
		EmbedFiles              *bool
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Printf("failed to parse config from %s: %s\n", filename, err)
		defaultPreferences()
		return
	}

	// tedious
	// i've decided to fully override to make it easier to change default config later on
	if config.GetWebProfiles != nil {
		GetWebProfiles = *config.GetWebProfiles
	}
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
	if config.EnableAPI != nil {
		EnableAPI = *config.EnableAPI
	}
	if config.Network != nil {
		Network = *config.Network
	}
	if config.Addr != nil {
		Addr = *config.Addr
	}
	if config.UnixSocketPerms != nil {
		p, err := strconv.ParseUint(*config.UnixSocketPerms, 0, 32)
		if err != nil {
			log.Println("failed to parse UnixSocketPerms:", err)
		} else {
			UnixSocketPerms = os.FileMode(p)
		}
	}
	if config.Prefork != nil {
		Prefork = *config.Prefork
	}
	if config.TrustedProxyCheck != nil {
		TrustedProxyCheck = *config.TrustedProxyCheck
	}
	if config.TrustedProxies != nil {
		TrustedProxies = *config.TrustedProxies
	}
	if config.CodegenConfig != nil {
		CodegenConfig = *config.CodegenConfig
	}
	if config.EmbedFiles != nil {
		EmbedFiles = *config.EmbedFiles
	}

	if config.DefaultPreferences != nil {
		loadDefaultPreferences(*config.DefaultPreferences)
	} else {
		defaultPreferences()
	}
}

const Debug = false
