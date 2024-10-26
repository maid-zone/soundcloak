package cfg

import (
	"time"

	jsoniter "github.com/json-iterator/go"
)

// fully loads the track on page load
// this option is here since the stream expires after some time (5 minutes? correct me if im wrong)
// if the stream isn't fully loaded before it expires - you'll need to reload the page
const FullyPreloadTrack = false

// proxy images (user avatars, track/playlist covers)
const ProxyImages = false
const ImageCacheControl = "max-age=600; public" // 10 minutes by default, only used for proxied images

// proxy streams (hls playlist files and track parts)
const ProxyStreams = false

// enable /_/info endpoint (shows if some settings are enabled/disabled)
const InstanceInfo = true

// time-to-live for clientid cache
// larger number will improve performance (no need to recheck everytime) but might make soundcloak briefly unusable for a larger amount of time if the client id is invalidated
const ClientIDTTL = 30 * time.Minute

// time-to-live for user profile cache
const UserTTL = 10 * time.Minute

// delay between cleanup of user cache
const UserCacheCleanDelay = UserTTL / 4

// time-to-live for track cache
const TrackTTL = 10 * time.Minute

// delay between cleanup of track cache
const TrackCacheCleanDelay = TrackTTL / 4

// time-to-live for playlist cache
const PlaylistTTL = 10 * time.Minute

// delay between cleanup of playlist cache
const PlaylistCacheCleanDelay = PlaylistTTL / 4

// default fasthttp one was causing connections to be stuck? todo make it cycle browser useragents or just choose random at startup
const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.3"

// time-to-live for dns cache
const DNSCacheTTL = 10 * time.Minute

// // // some webserver configuration, put here to make it easier to configure what you need // // //
// more info can be found here: https://docs.gofiber.io/api/fiber#config

// run soundcloak on this address (localhost:4664 by default)
const Addr = ":4664"

// run multiple instances of soundcloud locally to be able to handle more requests
// each one will be a separate process, so they will have separate cache
const Prefork = false

// Enables TLS Early Data (0-RTT / zero round trip time)
// This can reduce latency, but also makes requests replayable (not that much of a concern for soundcloak, since there are no authenticated operations)
// There might be breakage when used together with TrustedProxyCheck and the proxy is untrusted
const EarlyData = false

// use X-Forwarded-* headers ONLY when ip is in TrustedProxies list
// when disabled, the X-Forwarded-* headers will be blindly used
const TrustedProxyCheck = true

// list of ips or ip ranges of trusted proxies (check above)
var TrustedProxies = []string{}

// what JSON library should be used
var JSON = jsoniter.ConfigFastest
