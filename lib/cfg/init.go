package cfg

import (
	"time"

	jsoniter "github.com/json-iterator/go"
)

// time-to-live for clientid cache
// larger number will improve performance (no need to recheck everytime) but might make soundcloak unusable after soundcloud updates the website
const ClientIDTTL = 30 * time.Minute

// time-to-live for user profile cache
const UserTTL = 5 * time.Minute

// time-to-live for track cache
const TrackTTL = 5 * time.Minute

// time-to-live for playlist cache
const PlaylistTTL = 5 * time.Minute

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

// use X-Forwarded-* headers ONLY when ip is in TrustedProxies list
// when disabled, the X-Forwarded-* headers will be blindly used
const TrustedProxyCheck = false

// ip or ip range of trusted proxies (check above)
var TrustedProxies = []string{}

// what JSON library should be used
var JSON = jsoniter.ConfigFastest
