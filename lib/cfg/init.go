package cfg

import (
	"time"

	jsoniter "github.com/json-iterator/go"
)

// time-to-live for clientid cache
// larger number will improve performance (no need to recheck everytime) but might make soundcloak unusable after soundcloud updates the website
const ClientIDTTL = 5 * time.Minute

// time-to-live for user profile cache
const UserTTL = 5 * time.Minute

// time-to-live for track cache
const TrackTTL = 5 * time.Minute

// default fasthttp one was causing connections to be stuck? todo make it cycle browser useragents or just choose random at startup
const UserAgent = "insomnia/2023.2.0"

var JSON = jsoniter.ConfigFastest
