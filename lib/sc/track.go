package sc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"github.com/a-h/templ"
	templruntime "github.com/a-h/templ/runtime"
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

// Functions/structures related to tracks

var ErrIncompatibleStream = errors.New("incompatible stream")
var ErrNoURL = errors.New("no url")

var TracksCache = map[string]Cached[Track]{}
var tracksCacheLock = &sync.RWMutex{}

type Track struct {
	PublisherMetadata PublisherMetadata `json:"publisher_metadata"`
	Artwork           string            `json:"artwork_url"`
	CreatedAt         string            `json:"created_at"`
	Description       string            `json:"description"`
	Genre             string            `json:"genre"`
	Kind              string            `json:"kind"` // should always be "track"!
	LastModified      string            `json:"last_modified"`
	License           string            `json:"license"`
	Permalink         string            `json:"permalink"`
	TagList           string            `json:"tag_list"`
	Title             string            `json:"title"`
	ID                json.Number       `json:"id"`
	Authorization     string            `json:"track_authorization"`
	Policy            TrackPolicy       `json:"policy"`
	Station           string            `json:"station_permalink"`
	Waveform          string            `json:"waveform_url"`
	Media             Media             `json:"media"`
	Author            User              `json:"user"`
	Comments          int               `json:"comment_count"`
	Likes             int64             `json:"likes_count"`
	Played            int64             `json:"playback_count"`
	Reposted          int64             `json:"reposts_count"`
	Duration          uint32            `json:"full_duration"`
}

type PublisherMetadata struct {
	ISRC     string `json:"isrc"`
	UPCorEAN string `json:"upc_or_ean"`
	ISWC     string `json:"iswc"`
}

type TrackPolicy string

const (
	PolicyMonetize TrackPolicy = "MONETIZE" // seems like only certain countries get this policy? sometimes protected by widevine and fairplay
	PolicyBlock    TrackPolicy = "BLOCK"    // not available (in your country)
	PolicySnip     TrackPolicy = "SNIP"     // 30-second snippet available
	PolicyAllow    TrackPolicy = "ALLOW"    // all good
)

type Protocol string

const (
	ProtocolHLS             Protocol = "hls"
	ProtocolProgressive     Protocol = "progressive"
	ProtocolEncryptedHLS    Protocol = "encrypted-hls"     // idk, haven't seen in the wild
	ProtocolCTREncryptedHLS Protocol = "ctr-encrypted-hls" // google's widevine && microsoft playready
	ProtocolCBCEncryptedHLS Protocol = "cbc-encrypted-hls" // apple's fairplay
)

type Format struct {
	Protocol Protocol `json:"protocol"`
	MimeType string   `json:"mime_type"`
}

type Transcoding struct {
	URL              string `json:"url"`
	Preset           string `json:"preset"`
	Format           Format `json:"format"`
	Quality          string `json:"quality"`
	SoundcloakPreset string `json:"-"`
	Legacy           bool   `json:"is_legacy_transcoding"`
}

func (t *Transcoding) HasDRM() bool {
	switch t.Format.Protocol {
	case ProtocolCTREncryptedHLS, ProtocolCBCEncryptedHLS:
		return true
	}
	return false
}

func (t Transcoding) Slug(tr Track) string {
	return tr.Author.Permalink + "/" +
		tr.Permalink + "/" +
		t.Quality + "/" +
		t.Preset + "/" +
		string(t.Format.Protocol)
}

func (t Transcoding) ToExt() string {
	switch t.SoundcloakPreset {
	case cfg.AudioAACHQ, cfg.AudioAAC, cfg.AudioAACLQ:
		return "m4a"
	case cfg.AudioMP3:
		return "mp3"
	}

	return ""
}

type Media struct {
	Transcodings []Transcoding `json:"transcodings"`
}

type Stream struct {
	URL     string `json:"url"`
	License string `json:"licenseAuthToken"`
}

type Comment struct {
	Kind      string `json:"kind"` // "comment"
	Body      string `json:"body"`
	Author    User   `json:"user"`
	Timestamp int    `json:"timestamp"`
}

func (m Media) SelectCompatibleRestream(mode string) *Transcoding {
	// aac_hq - aac_256k, aac_160k, mp3,      aac_96k
	// aac    - aac_160k, aac_256k, mp3,      aac_96k
	// mpeg   - mp3,      aac_256k, aac_160k, aac_96k
	// aac_lq - aac_96k,  mp3,      aac_160k, aac_256k
	// aac_256k has a legacy twin, new preferred (i guess)
	// progressive preferred over hls

	var b [6]*Transcoding
	switch mode {
	case cfg.AudioAACHQ:
		for _, t := range m.Transcodings {
			switch t.Format.Protocol {
			case ProtocolHLS:
				switch t.Preset {
				case "aac_256k":
					t.SoundcloakPreset = cfg.AudioAACHQ
					return &t
				case "aac_160k":
					if b[2] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[2] = &t
					}
				case "aac_96k":
					if b[5] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[5] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[1] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[1] = &t
						}
					case "audio/mpeg":
						if b[4] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[4] = &t
						}
					}
				}
			case ProtocolProgressive:
				switch t.Format.MimeType {
				case `audio/mp4; codecs="mp4a.40.2"`:
					if b[0] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[0] = &t
					}
				case "audio/mpeg":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioMP3
						b[3] = &t
					}
				}
			}
		}
	case cfg.AudioAAC:
		for _, t := range m.Transcodings {
			switch t.Format.Protocol {
			case ProtocolHLS:
				switch t.Preset {
				case "aac_256k":
					if b[0] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[0] = &t
					}
				case "aac_160k":
					t.SoundcloakPreset = cfg.AudioAAC
					return &t
				case "aac_96k":
					if b[5] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[5] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[2] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[2] = &t
						}
					case "audio/mpeg":
						if b[4] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[4] = &t
						}
					}
				}
			case ProtocolProgressive:
				switch t.Format.MimeType {
				case `audio/mp4; codecs="mp4a.40.2"`:
					if b[1] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[1] = &t
					}
				case "audio/mpeg":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioMP3
						b[3] = &t
					}
				}
			}
		}
	case cfg.AudioMP3:
		for _, t := range m.Transcodings {
			switch t.Format.Protocol {
			case ProtocolHLS:
				switch t.Preset {
				case "aac_256k":
					if b[1] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[1] = &t
					}
				case "aac_160k":
					if b[4] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[4] = &t
					}
				case "aac_96k":
					if b[5] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[5] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[3] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[3] = &t
						}
					case "audio/mpeg":
						if b[0] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[0] = &t
						}
					}
				}
			case ProtocolProgressive:
				switch t.Format.MimeType {
				case `audio/mp4; codecs="mp4a.40.2"`:
					if b[2] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[2] = &t
					}
				case "audio/mpeg":
					t.SoundcloakPreset = cfg.AudioMP3
					return &t
				}
			}
		}
	case cfg.AudioAACLQ:
		for _, t := range m.Transcodings {
			switch t.Format.Protocol {
			case ProtocolHLS:
				switch t.Preset {
				case "aac_256k":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[3] = &t
					}
				case "aac_160k":
					if b[2] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[2] = &t
					}
				case "aac_96k":
					t.SoundcloakPreset = cfg.AudioAACLQ
					return &t
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[4] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[4] = &t
						}
					case "audio/mpeg":
						if b[1] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[1] = &t
						}
					}
				}
			case ProtocolProgressive:
				switch t.Format.MimeType {
				case `audio/mp4; codecs="mp4a.40.2"`:
					if b[5] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[5] = &t
					}
				case "audio/mpeg":
					if b[0] == nil {
						t.SoundcloakPreset = cfg.AudioMP3
						b[0] = &t
					}
				}
			}
		}
	}
	for _, t := range b {
		if t != nil {
			return t
		}
	}
	return nil
}

func (m Media) SelectCompatibleDownload(mode string) *Transcoding {
	// aac_hq - aac_256k, aac_160k, mp3,      aac_96k
	// aac    - aac_160k, aac_256k, mp3,      aac_96k
	// mpeg   - mp3,      aac_256k, aac_160k, aac_96k
	// aac_lq - aac_96k,  mp3,      aac_160k, aac_256k
	// aac_256k has a legacy twin, new preferred (i guess)
	// progressive preferred over hls except for aac_256k because mp4 metadata isnt as simple to inject like mp3

	var b [5]*Transcoding
	switch mode {
	case cfg.AudioAACHQ:
		for _, t := range m.Transcodings {
			switch t.Format.Protocol {
			case ProtocolHLS:
				switch t.Preset {
				case "aac_256k":
					t.SoundcloakPreset = cfg.AudioAACHQ
					return &t
				case "aac_160k":
					if b[1] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[1] = &t
					}
				case "aac_96k":
					if b[4] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[4] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[0] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[0] = &t
						}
					case "audio/mpeg":
						if b[3] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[3] = &t
						}
					}
				}
			case ProtocolProgressive:
				if b[2] == nil && t.Format.MimeType == "audio/mpeg" {
					t.SoundcloakPreset = cfg.AudioMP3
					b[2] = &t
				}
			}
		}
	case cfg.AudioAAC:
		for _, t := range m.Transcodings {
			switch t.Format.Protocol {
			case ProtocolHLS:
				switch t.Preset {
				case "aac_256k":
					if b[0] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[0] = &t
					}
				case "aac_160k":
					t.SoundcloakPreset = cfg.AudioAAC
					return &t
				case "aac_96k":
					if b[4] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[4] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[1] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[1] = &t
						}
					case "audio/mpeg":
						if b[3] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[3] = &t
						}
					}
				}
			case ProtocolProgressive:
				if b[2] == nil && t.Format.MimeType == "audio/mpeg" {
					t.SoundcloakPreset = cfg.AudioMP3
					b[2] = &t
				}
			}
		}
	case cfg.AudioMP3:
		for _, t := range m.Transcodings {
			switch t.Format.Protocol {
			case ProtocolHLS:
				switch t.Preset {
				case "aac_256k":
					if b[1] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[1] = &t
					}
				case "aac_160k":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[3] = &t
					}
				case "aac_96k":
					if b[4] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[4] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[2] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[2] = &t
						}
					case "audio/mpeg":
						if b[0] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[0] = &t
						}
					}
				}
			case ProtocolProgressive:
				if t.Format.MimeType == "audio/mpeg" {
					t.SoundcloakPreset = cfg.AudioMP3
					return &t
				}
			}
		}
	case cfg.AudioAACLQ:
		for _, t := range m.Transcodings {
			switch t.Format.Protocol {
			case ProtocolHLS:
				switch t.Preset {
				case "aac_256k":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[3] = &t
					}
				case "aac_160k":
					if b[2] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[2] = &t
					}
				case "aac_96k":
					t.SoundcloakPreset = cfg.AudioAACLQ
					return &t
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[4] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[4] = &t
						}
					case "audio/mpeg":
						if b[1] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[1] = &t
						}
					}
				}
			case ProtocolProgressive:
				if t.Format.MimeType == "audio/mpeg" {
					if b[0] == nil {
						t.SoundcloakPreset = cfg.AudioMP3
						b[0] = &t
					}
				}
			}
		}
	}
	for _, t := range b {
		if t != nil {
			return t
		}
	}
	return nil
}

func (m Media) SelectCompatibleHLS(mode string) *Transcoding {
	// aac_hq - aac_256k, aac_160k, mp3,      aac_96k
	// aac    - aac_160k, aac_256k, mp3,      aac_96k
	// mpeg   - mp3,      aac_256k, aac_160k, aac_96k
	// aac_lq - aac_96k,  mp3,      aac_160k, aac_256k
	// aac_256k has an evil legacy twin

	// to do just one iteration
	var b [4]*Transcoding
	switch mode {
	case cfg.AudioAACHQ:
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolHLS {
				switch t.Preset {
				case "aac_256k":
					t.SoundcloakPreset = cfg.AudioAACHQ
					return &t
				case "aac_160k":
					if b[1] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[1] = &t
					}
				case "aac_96k":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[3] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[0] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[0] = &t
						}
					case "audio/mpeg":
						if b[2] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[2] = &t
						}
					}
				}
			}
		}
	case cfg.AudioAAC:
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolHLS {
				switch t.Preset {
				case "aac_256k":
					if b[0] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[0] = &t
					}
				case "aac_160k":
					t.SoundcloakPreset = cfg.AudioAAC
					return &t
				case "aac_96k":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[3] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[1] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[1] = &t
						}
					case "audio/mpeg":
						if b[2] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[2] = &t
						}
					}
				}
			}
		}
	case cfg.AudioMP3:
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolHLS {
				switch t.Preset {
				case "aac_256k":
					if b[0] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[0] = &t
					}
				case "aac_160k":
					if b[2] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[2] = &t
					}
				case "aac_96k":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[3] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[1] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[1] = &t
						}
					case "audio/mpeg":
						t.SoundcloakPreset = cfg.AudioMP3
						return &t
					}
				}
			}
		}
	case cfg.AudioAACLQ:
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolHLS {
				switch t.Preset {
				case "aac_256k":
					if b[2] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[2] = &t
					}
				case "aac_160k":
					if b[1] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[1] = &t
					}
				case "aac_96k":
					t.SoundcloakPreset = cfg.AudioAACLQ
					return &t
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[3] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[3] = &t
						}
					case "audio/mpeg":
						if b[0] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[0] = &t
						}
					}
				}
			}
		}
	}
	for _, t := range b {
		if t != nil {
			return t
		}
	}
	return nil
}

func (m Media) SelectCompatibleHLSDRM(mode string) *Transcoding {
	// aac_hq - aac_256k, aac_160k, mp3,      aac_96k
	// aac    - aac_160k, aac_256k, mp3,      aac_96k
	// mpeg   - mp3,      aac_256k, aac_160k, aac_96k
	// aac_lq - aac_96k,  mp3,      aac_160k, aac_256k
	// aac_256k has an evil legacy twin
	// this picks only stuff with drm

	// to do just one iteration
	var b [4]*Transcoding
	switch mode {
	case cfg.AudioAACHQ:
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolCTREncryptedHLS {
				switch t.Preset {
				case "aac_256k":
					t.SoundcloakPreset = cfg.AudioAACHQ
					return &t
				case "aac_160k":
					if b[1] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[1] = &t
					}
				case "aac_96k":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[3] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[0] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[0] = &t
						}
					case "audio/mpeg":
						if b[2] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[2] = &t
						}
					}
				}
			}
		}
	case cfg.AudioAAC:
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolCTREncryptedHLS {
				switch t.Preset {
				case "aac_256k":
					if b[0] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[0] = &t
					}
				case "aac_160k":
					t.SoundcloakPreset = cfg.AudioAAC
					return &t
				case "aac_96k":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[3] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[1] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[1] = &t
						}
					case "audio/mpeg":
						if b[2] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[2] = &t
						}
					}
				}
			}
		}
	case cfg.AudioMP3:
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolCTREncryptedHLS {
				switch t.Preset {
				case "aac_256k":
					if b[0] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[0] = &t
					}
				case "aac_160k":
					if b[2] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[2] = &t
					}
				case "aac_96k":
					if b[3] == nil {
						t.SoundcloakPreset = cfg.AudioAACLQ
						b[3] = &t
					}
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[1] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[1] = &t
						}
					case "audio/mpeg":
						t.SoundcloakPreset = cfg.AudioMP3
						return &t
					}
				}
			}
		}
	case cfg.AudioAACLQ:
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolCTREncryptedHLS {
				switch t.Preset {
				case "aac_256k":
					if b[2] == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b[2] = &t
					}
				case "aac_160k":
					if b[1] == nil {
						t.SoundcloakPreset = cfg.AudioAAC
						b[1] = &t
					}
				case "aac_96k":
					t.SoundcloakPreset = cfg.AudioAACLQ
					return &t
				default:
					switch t.Format.MimeType {
					case `audio/mp4; codecs="mp4a.40.2"`:
						if b[3] == nil {
							t.SoundcloakPreset = cfg.AudioAACHQ
							b[3] = &t
						}
					case "audio/mpeg":
						if b[0] == nil {
							t.SoundcloakPreset = cfg.AudioMP3
							b[0] = &t
						}
					}
				}
			}
		}
	}
	for _, t := range b {
		if t != nil {
			return t
		}
	}
	return nil
}

func (m Media) SelectCompatibleProgressive(prefs cfg.Preferences) (b *Transcoding) {
	if *prefs.ProgressiveAudio == cfg.AudioAACHQ {
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolProgressive {
				switch t.Format.MimeType {
				case `audio/mp4; codecs="mp4a.40.2"`:
					t.SoundcloakPreset = cfg.AudioAACHQ
					return &t
				case `audio/mpeg`:
					if b == nil {
						t.SoundcloakPreset = cfg.AudioMP3
						b = &t
					}
				}
			}
		}
	} else {
		for _, t := range m.Transcodings {
			if t.Format.Protocol == ProtocolProgressive {
				switch t.Format.MimeType {
				case `audio/mp4; codecs="mp4a.40.2"`:
					if b == nil {
						t.SoundcloakPreset = cfg.AudioAACHQ
						b = &t
					}
				case `audio/mpeg`:
					t.SoundcloakPreset = cfg.AudioMP3
					return &t
				}
			}
		}
	}
	return b
}

func (m Media) HasDRM() bool {
	for _, t := range m.Transcodings {
		switch t.Format.Protocol {
		case ProtocolCTREncryptedHLS, ProtocolCBCEncryptedHLS:
			return true
		}
	}
	return false
}

func (m Media) SelectCompatibleAnyHLS(prefs cfg.Preferences) *Transcoding {
	if *prefs.DRM {
		t := m.SelectCompatibleHLSDRM(*prefs.HLSAudio)
		if t != nil {
			return t
		}
	}
	return m.SelectCompatibleHLS(*prefs.HLSAudio)
}

func GetTrack(permalink string) (Track, error) {
	tracksCacheLock.RLock()
	if cell, ok := TracksCache[permalink]; ok {
		tracksCacheLock.RUnlock()
		return cell.Value, nil
	}
	tracksCacheLock.RUnlock()

	var t Track
	err := Resolve(permalink, &t)
	if err != nil {
		return t, err
	}

	if t.Kind != "track" {
		return t, ErrKindNotCorrect
	}

	t.Fix(true, true)

	tracksCacheLock.Lock()
	TracksCache[permalink] = Cached[Track]{Value: t, Expires: time.Now().Add(cfg.TrackTTL)}
	tracksCacheLock.Unlock()

	return t, nil
}

// Currently supports:
// http/https links:
// - api.soundcloud.com/tracks/<id> (api-v2 subdomain also supported)
// - soundcloud.com/<user>/<track>
//
// plain permalink/id:
// - <user>/<track>
// - <id>
func GetArbitraryTrack(data string) (Track, error) {
	if len(data) > 8 && (data[:8] == "https://" || data[:7] == "http://") {
		u, err := url.Parse(data)
		if err == nil {
			if (u.Host == "api.soundcloud.com" || u.Host == "api-v2.soundcloud.com") && len(u.Path) > 8 && u.Path[:8] == "/tracks/" {
				return GetTrackByID(u.Path[8:])
			}

			if u.Host == "soundcloud.com" {
				if len(u.Path) < 4 {
					return Track{}, ErrNoURL
				}

				u.Path = u.Path[1:]
				if u.Path[len(u.Path)-1] == '/' {
					u.Path = u.Path[:len(u.Path)-1]
				}

				var n uint = 0
				for _, c := range u.Path {
					if c == '/' {
						n++
					}

					if n == 2 {
						return Track{}, ErrKindNotCorrect
					}
				}

				if n != 1 {
					return Track{}, ErrKindNotCorrect
				}

				return GetTrack(u.Path)
			}
		} else {
			return Track{}, err
		}
	}

	valid := true
	for _, n := range data {
		if n < '0' || n > '9' {
			valid = false
			break
		}
	}

	if valid {
		return GetTrackByID(data)
	}

	// this part should be at the end since it manipulates data
	if len(data) < 4 {
		return Track{}, ErrNoURL
	}

	if data[0] == '/' {
		data = data[1:]
	}

	if data[len(data)-1] == '/' {
		data = data[:len(data)-1]
	}
	var n uint = 0
	for _, c := range data {
		if c == '/' {
			n++
		}
	}

	if n == 1 {
		return GetTrack(data)
	}

	// failed to find a data point
	return Track{}, ErrKindNotCorrect
}

func SearchTracks(prefs cfg.Preferences, args []byte) (*Paginated[*Track], error) {
	uri := baseUri()
	uri.SetPath("/search/tracks")
	uri.SetQueryStringBytes(args)
	p := Paginated[*Track]{Next: uri}
	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(false, false)
		t.Postfix(prefs, false)
	}

	return &p, nil
}

func GetTracks(ids string) ([]Track, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	uri := baseUri()
	uri.SetPath("/tracks")
	uri.QueryArgs().Set("ids", ids)
	uri.QueryArgs().Set("client_id", ClientID)
	req.SetURI(uri)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	Authorize(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return nil, err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var res []Track
	err = json.Unmarshal(data, &res)
	for i, t := range res {
		t.Fix(false, false)
		res[i] = t
	}
	return res, err
}

type CachedStream struct {
	Playlist         *fasthttp.URI
	Base             *fasthttp.URI
	LegacyHlsAacInit *fasthttp.URI
	License          string
	FreshBase        bool
}

var StreamCache = map[string]Cached[CachedStream]{}
var StreamCacheMut = sync.RWMutex{}

func (tr Transcoding) GetStream(slug string, t Track) (Cached[CachedStream], error) {
	if slug == "" {
		slug = tr.Slug(t)
	}

	StreamCacheMut.RLock()
	s, ok := StreamCache[slug]
	StreamCacheMut.RUnlock()
	if ok && s.Expires.After(time.Now()) {
		misc.Log("cache hit", s)
		return s, nil
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(tr.URL)
	req.URI().QueryArgs().Set("client_id", ClientID)
	req.URI().QueryArgs().Set("track_authorization", t.Authorization)
	req.Header.SetUserAgent(cfg.UserAgent)
	Authorize(req)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return s, err
	}

	if resp.StatusCode() != 200 {
		return s, fmt.Errorf("getstream: got status code %d", resp.StatusCode())
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var st Stream
	err = json.Unmarshal(data, &st)
	if err != nil {
		return s, err
	}

	misc.Log(st)

	if st.URL == "" {
		return s, ErrNoURL
	}
	s.Value.License = st.License
	if s.Value.Playlist == nil {
		s.Value.Playlist = fasthttp.AcquireURI()
	}
	err = s.Value.Playlist.Parse(nil, cfg.S2b(st.URL))
	if err == nil {
		s.Expires = time.Now().Add(time.Duration(t.Duration)*time.Millisecond + 105*time.Second)
		s.Value.FreshBase = true
		StreamCacheMut.Lock()
		StreamCache[slug] = s
		StreamCacheMut.Unlock()
	}
	return s, err
}

func (tr Transcoding) GetUncachedStream(slug string, t Track) (Stream, error) {
	if slug == "" {
		slug = tr.Slug(t)
	}

	var s Stream
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(tr.URL)
	req.URI().QueryArgs().Set("client_id", ClientID)
	req.URI().QueryArgs().Set("track_authorization", t.Authorization)
	req.Header.SetUserAgent(cfg.UserAgent)
	Authorize(req)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return s, err
	}

	if resp.StatusCode() != 200 {
		return s, fmt.Errorf("getuncachedstream: got status code %d", resp.StatusCode())
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	err = json.Unmarshal(data, &s)
	if err != nil {
		return s, err
	}

	misc.Log(s)

	if s.URL == "" {
		err = ErrNoURL
	}

	return s, err
}

func (t *Track) Fix(large bool, fixAuthor bool) {
	if large {
		t.Artwork = strings.Replace(t.Artwork, "-large.", "-t500x500.", 1)
	} else {
		t.Artwork = strings.Replace(t.Artwork, "-large.", "-t200x200.", 1)
	}

	if fixAuthor {
		t.Author.Fix(false)
	}
}

func (t *Track) Postfix(prefs cfg.Preferences, fixAuthor bool) {
	if cfg.ProxyImages && *prefs.ProxyImages && t.Artwork != "" {
		t.Artwork = "/_/proxy/images?url=" + url.QueryEscape(t.Artwork)
	}

	if fixAuthor {
		t.Author.Postfix(prefs)
	}
}

func (t Track) FormatDescription() string {
	desc := t.Description
	if t.Description != "" {
		desc += "\n\n"
	}

	desc += strconv.FormatInt(t.Likes, 10) + " ❤️ | " + strconv.FormatInt(t.Played, 10) + " ▶️ | " + strconv.FormatInt(t.Reposted, 10) + " 🔁"
	if t.Genre != "" {
		desc += "\nGenre: " + t.Genre
	}
	desc += "\nCreated: " + t.CreatedAt
	desc += "\nLast modified: " + t.LastModified
	if len(t.TagList) != 0 {
		desc += "\nTags: " + TagListParser(t.TagList)
	}

	return desc
}

func GetTrackByID(id string) (Track, error) {
	tracksCacheLock.RLock()
	for _, cell := range TracksCache {
		if string(cell.Value.ID) == string(id) {
			tracksCacheLock.RUnlock()
			return cell.Value, nil
		}
	}
	tracksCacheLock.RUnlock()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	baseUriReq(req)
	req.URI().SetPath("/tracks/" + id)
	req.URI().QueryArgs().Set("client_id", ClientID)
	Authorize(req)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	var t Track
	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return t, err
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	err = json.Unmarshal(data, &t)
	if err != nil {
		return t, err
	}

	if t.Kind != "track" {
		return t, ErrKindNotCorrect
	}

	t.Fix(true, true)

	tracksCacheLock.Lock()
	TracksCache[t.Author.Permalink+"/"+t.Permalink] = Cached[Track]{Value: t, Expires: time.Now().Add(cfg.TrackTTL)}
	tracksCacheLock.Unlock()

	return t, nil
}

func (t Track) Href() string {
	return "/" + t.Author.Permalink + "/" + t.Permalink
}

func RecentTracks(prefs cfg.Preferences, tag, args string) (*Paginated[*Track], error) {
	uri := baseUri()
	uri.SetPath("/recent-tracks/" + tag)
	uri.SetQueryString(args)
	p := Paginated[*Track]{Next: uri}
	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(false, false)
		t.Postfix(prefs, false)
	}

	return &p, nil
}

func (t Track) baseUri(subpath, args string) *fasthttp.URI {
	uri := baseUri()
	uri.SetPath("/tracks/" + string(t.ID) + "/" + subpath)
	uri.SetQueryString(args)
	return uri
}

func (t Track) GetRelated(prefs cfg.Preferences, args string) (*Paginated[*Track], error) {
	p := Paginated[*Track]{Next: t.baseUri("related", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, t := range p.Collection {
		t.Fix(false, false)
		t.Postfix(prefs, false)
	}

	return &p, nil
}

func (t Track) GetPlaylists(prefs cfg.Preferences, args string) (*Paginated[*Playlist], error) {
	p := Paginated[*Playlist]{Next: t.baseUri("playlists_without_albums", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, p := range p.Collection {
		p.Fix(false, false)
		p.Postfix(prefs, false, false)
	}

	return &p, nil
}

func (t Track) GetAlbums(prefs cfg.Preferences, args string) (*Paginated[*Playlist], error) {
	p := Paginated[*Playlist]{Next: t.baseUri("albums", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, p := range p.Collection {
		p.Fix(false, false)
		p.Postfix(prefs, false, false)
	}

	return &p, nil
}

func (t Track) GetComments(prefs cfg.Preferences, args string) (*Paginated[*Comment], error) {
	p := Paginated[*Comment]{Next: t.baseUri("comments", args)}

	err := p.Proceed(true)
	if err != nil {
		return nil, err
	}

	for _, p := range p.Collection {
		p.Author.Fix(false)
		p.Author.Postfix(prefs)
	}

	return &p, nil
}

type Waveform struct {
	Samples []uint64 `json:"samples"`
	//Width   int   `json:"width"`
	Height uint64 `json:"height"`
}

func (t *Track) RenderWaveform() templ.Component {
	if t.Waveform == "" {
		return templ.NopComponent
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(t.Waveform)
	req.Header.SetUserAgent(cfg.UserAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := DoWithRetry(httpc, req, resp)
	if err != nil {
		return templ.NopComponent
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		data = resp.Body()
	}

	var wf Waveform
	err = json.Unmarshal(data, &wf)
	if err != nil || len(wf.Samples) == 0 || wf.Height == 0 {
		return templ.NopComponent
	}

	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		ww := w.(*templruntime.Buffer)
		ww.WriteString(`<svg class="waveform" viewBox="0 0 200 100" preserveAspectRatio="none"><defs><clipPath id="wf-p"><rect x="0" y="0" width="0" height="100"/></clipPath></defs><path d="`)
		const (
			targetBars = 200
			svgHeight  = 100
			center     = svgHeight / 2
		)

		step := len(wf.Samples) / targetBars
		if step < 1 {
			step = 1
		}

		var count uint64
		b := make([]byte, 1, 10)
		b[0] = 'M'
		for i := 0; i < len(wf.Samples) && count < targetBars; i += step {
			h := wf.Samples[i] * center / wf.Height
			if h < 1 {
				h = 1
			}

			b = b[:1]
			b = strconv.AppendUint(b, count, 10)
			b = append(b, ',')
			b = strconv.AppendUint(b, center-h, 10)
			b = append(b, 'V')
			b = strconv.AppendUint(b, center+h, 10)
			count++
			ww.Write(b)
		}
		ww.WriteString(`" stroke="var(--0)" fill="none" stroke-width="0.6"/></svg><script async src="/_/static/waveform.js"></script>`)
		return nil
	})
}
