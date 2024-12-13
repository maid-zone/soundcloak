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

	if dst.HLSAudio == nil {
		dst.HLSAudio = cfg.DefaultPreferences.HLSAudio
	}

	if dst.RestreamAudio == nil {
		dst.RestreamAudio = cfg.DefaultPreferences.RestreamAudio
	}

	if dst.DownloadAudio == nil {
		dst.DownloadAudio = cfg.DefaultPreferences.DownloadAudio
	}

	if dst.ShowAudio == nil {
		dst.ShowAudio = cfg.DefaultPreferences.ShowAudio
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
	HLSAudio            string
	RestreamAudio       string
	DownloadAudio       string
	ShowAudio           string
}

type Export struct {
	Preferences *cfg.Preferences `json:",omitempty"`
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

		if *old.AutoplayNextTrack {
			old.DefaultAutoplayMode = &p.DefaultAutoplayMode
		}

		if p.AutoplayNextTrack == "on" {
			old.AutoplayNextTrack = &cfg.True
		} else {
			old.AutoplayNextTrack = &cfg.False
		}

		if p.ShowAudio == "on" {
			old.ShowAudio = &cfg.True
		} else {
			old.ShowAudio = &cfg.False
		}

		if *old.Player == cfg.HLSPlayer {
			if cfg.ProxyStreams {
				if p.ProxyStreams == "on" {
					old.ProxyStreams = &cfg.ProxyStreams // true!
				} else if p.ProxyStreams == "" {
					old.ProxyStreams = &cfg.False
				}
			}

			if p.FullyPreloadTrack == "on" {
				old.FullyPreloadTrack = &cfg.True
			} else if p.FullyPreloadTrack == "" {
				old.FullyPreloadTrack = &cfg.False
			}

			old.HLSAudio = &p.HLSAudio
		}

		if cfg.Restream {
			if *old.Player == cfg.RestreamPlayer {
				old.RestreamAudio = &p.RestreamAudio
			}

			old.DownloadAudio = &p.DownloadAudio
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

	r.Get("/_/preferences/reset", func(c *fiber.Ctx) error {
		// c.ClearCookie("prefs")
		c.Cookie(&fiber.Cookie{ // I've had some issues with c.ClearCookie() method, so using this workaround for now
			Name:     "prefs",
			Value:    "{}",
			Expires:  time.Now().Add(400 * 24 * time.Hour),
			HTTPOnly: true,
			SameSite: "strict",
		})

		return c.Redirect("/_/preferences")
	})

	r.Get("/_/preferences/export", func(c *fiber.Ctx) error {
		p, err := Get(c)
		if err != nil {
			return err
		}

		return c.JSON(Export{Preferences: &p})
	})

	r.Post("/_/preferences/import", func(c *fiber.Ctx) error {
		f, err := c.FormFile("prefs")
		if err != nil {
			return err
		}

		fd, err := f.Open()
		if err != nil {
			return err
		}
		defer fd.Close()

		dec := json.NewDecoder(fd)

		var p Export
		err = dec.Decode(&p)
		if err != nil {
			return err
		}

		if p.Preferences == nil {
			p.Preferences = &cfg.Preferences{}
		}

		Defaults(p.Preferences)

		data, err := json.Marshal(p.Preferences)
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
