package preferences

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/maid-zone/soundcloak/templates"
)

func Defaults(dst *cfg.Preferences) {
	if dst.Player == nil {
		dst.Player = cfg.DefaultPreferences.Player
	}

	if dst.ProxyStreams == nil {
		dst.ProxyStreams = cfg.DefaultPreferences.ProxyStreams
	}

	if dst.FullyPreloadTrack == nil {
		dst.FullyPreloadTrack = cfg.DefaultPreferences.FullyPreloadTrack
	}

	if dst.ProxyImages == nil {
		dst.ProxyImages = cfg.DefaultPreferences.ProxyImages
	}

	if dst.ParseDescriptions == nil {
		dst.ParseDescriptions = cfg.DefaultPreferences.ParseDescriptions
	}

	if dst.AutoplayNextTrack == nil {
		dst.AutoplayNextTrack = cfg.DefaultPreferences.AutoplayNextTrack
	}

	if dst.DefaultAutoplayMode == nil {
		dst.DefaultAutoplayMode = cfg.DefaultPreferences.DefaultAutoplayMode
	}
}

func Get(c *fiber.Ctx) (cfg.Preferences, error) {
	rawprefs := c.Cookies("prefs", "{}")
	var p cfg.Preferences

	err := json.Unmarshal([]byte(rawprefs), &p)
	if err != nil {
		return p, err
	}
	Defaults(&p)

	return p, err
}

type PrefsForm struct {
	ProxyImages         string
	ParseDescriptions   string
	Player              string
	ProxyStreams        string
	FullyPreloadTrack   string
	AutoplayNextTrack   string
	DefaultAutoplayMode string
}

func Load(r fiber.Router) {
	r.Get("/_/preferences", func(c *fiber.Ctx) error {
		p, err := Get(c)
		if err != nil {
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base("preferences", templates.Preferences(p), nil).Render(context.Background(), c)
	})

	r.Post("/_/preferences", func(c *fiber.Ctx) error {
		var p PrefsForm
		err := c.BodyParser(&p)
		if err != nil {
			return err
		}

		old, err := Get(c)
		if err != nil {
			return err
		}

		if *old.Player == "hls" {
			if cfg.ProxyStreams {
				if p.ProxyStreams == "on" {
					old.ProxyStreams = &cfg.ProxyStreams // true!
				} else if p.ProxyStreams == "" {
					old.ProxyStreams = &cfg.False
				}

				if p.AutoplayNextTrack == "on" {
					old.AutoplayNextTrack = &cfg.True
				} else {
					old.AutoplayNextTrack = &cfg.False
				}
			}

			if p.FullyPreloadTrack == "on" {
				old.FullyPreloadTrack = &cfg.True
			} else if p.FullyPreloadTrack == "" {
				old.FullyPreloadTrack = &cfg.False
			}
		}

		if cfg.ProxyImages {
			if p.ProxyImages == "on" {
				old.ProxyImages = &cfg.ProxyImages // true!
			} else if p.ProxyImages == "" {
				old.ProxyImages = &cfg.False
			}
		}

		if p.ParseDescriptions == "on" {
			old.ParseDescriptions = &cfg.True
		} else {
			old.ParseDescriptions = &cfg.False
		}

		if *old.AutoplayNextTrack {
			old.DefaultAutoplayMode = &p.DefaultAutoplayMode
		}
		old.Player = &p.Player

		data, err := json.Marshal(old)
		if err != nil {
			return err
		}

		c.Cookie(&fiber.Cookie{
			Name:     "prefs",
			Value:    string(data),
			Expires:  time.Now().Add(400 * 24 * time.Hour),
			HTTPOnly: true,
			SameSite: "strict",
		})

		return c.Redirect("/_/preferences")
	})
}
