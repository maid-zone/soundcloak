package cfg

import (
	"log"
	"os"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// You can now use a soundcloak.json file to configure!
// or you can specify a different path with environment variable SOUNDCLOAK_CONFIG
// it's not needed to specify every key in your config, it will use the default values for keys you haven't specified
// also, when specifying time, specify it in seconds (instead of nanoseconds as time.Duration)

// // config // //

// fully loads the track on page load
// this option is here since the stream expires after some time (5 minutes? correct me if im wrong)
// if the stream isn't fully loaded before it expires - you'll need to reload the page
var FullyPreloadTrack = false

// proxy images (user avatars, track/playlist covers)
var ProxyImages = false
var ImageCacheControl = "max-age=600; public" // 10 minutes by default, only used for proxied images

// proxy streams (hls playlist files and track parts)
var ProxyStreams = false

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

// what JSON library should be used
var JSON = jsoniter.ConfigFastest

func init() {
	filename := "soundcloak.json"
	if env := os.Getenv("SOUNDCLOAK_CONFIG"); env != "" {
		filename = env
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("failed to load config from %s: %s\n", filename, err)
		return
	}

	var config struct {
		FullyPreloadTrack       *bool
		ProxyImages             *bool
		ImageCacheControl       *string
		ProxyStreams            *bool
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

	err = JSON.Unmarshal(data, &config)
	if err != nil {
		log.Printf("failed to parse config from %s: %s\n", filename, err)
		return
	}

	// tedious
	// i've decided to fully override to make it easier to change default config later on
	if config.FullyPreloadTrack != nil {
		FullyPreloadTrack = *config.FullyPreloadTrack
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
}
