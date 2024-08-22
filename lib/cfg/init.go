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

// default fasthttp one was causing connections to be stuck? todo make it cycle browser useragents or just choose random at startup
const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0"

var JSON = jsoniter.ConfigFastest
