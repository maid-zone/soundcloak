package preferences

import (
	"context"
	"time"

	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/templates"
	"github.com/gofiber/fiber/v3"
)

const on = "on"

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

	if dst.AutoplayNextRelatedTrack == nil {
		dst.AutoplayNextRelatedTrack = cfg.DefaultPreferences.AutoplayNextRelatedTrack
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

	if dst.SearchSuggestions == nil {
		dst.SearchSuggestions = cfg.DefaultPreferences.SearchSuggestions
	}

	if dst.DynamicLoadComments == nil {
		dst.DynamicLoadComments = cfg.DefaultPreferences.DynamicLoadComments
	}

	if dst.KeepPlayerFocus == nil {
		dst.KeepPlayerFocus = cfg.DefaultPreferences.KeepPlayerFocus
	}
}

func Get(c fiber.Ctx) (cfg.Preferences, error) {
	rawprefs := c.Cookies("prefs", "{}")
	var p cfg.Preferences

	err := json.Unmarshal(cfg.S2b(rawprefs), &p)
	if err != nil {
		return p, err
	}
	Defaults(&p)

	return p, err
}

type PrefsForm struct {
	ProxyImages              string
	ParseDescriptions        string
	Player                   string
	ProxyStreams             string
	FullyPreloadTrack        string
	AutoplayNextTrack        string
	AutoplayNextRelatedTrack string
	DefaultAutoplayMode      string
	HLSAudio                 string
	RestreamAudio            string
	DownloadAudio            string
	ShowAudio                string
	SearchSuggestions        string
	DynamicLoadComments      string
	KeepPlayerFocus          string
}

type Export struct {
	Preferences *cfg.Preferences `json:",omitempty"`
}

func setPrefs(c fiber.Ctx, p *cfg.Preferences) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	cookie := fasthttp.AcquireCookie()
	cookie.SetKey("prefs")
	cookie.SetValueBytes(data)
	cookie.SetExpire(time.Now().Add(400 * 24 * time.Hour))
	cookie.SetHTTPOnly(true)
	cookie.SetSameSite(fasthttp.CookieSameSiteStrictMode)
	c.Response().Header.SetCookie(cookie)
	fasthttp.ReleaseCookie(cookie)

	return nil
}

func Load(r *fiber.App) {
	r.Get("/_/preferences", func(c fiber.Ctx) error {
		p, err := Get(c)
		if err != nil {
			return err
		}

		c.Response().Header.SetContentType("text/html")
		return templates.Base("preferences", templates.Preferences(p), nil).Render(context.Background(), c)
	})

	r.Post("/_/preferences", func(c fiber.Ctx) error {
		var p PrefsForm
		err := c.Bind().Body(&p)
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

		if p.AutoplayNextTrack == on {
			old.AutoplayNextTrack = &cfg.True
		} else {
			old.AutoplayNextTrack = &cfg.False
		}

		if p.AutoplayNextRelatedTrack == on {
			old.AutoplayNextRelatedTrack = &cfg.True
		} else {
			old.AutoplayNextRelatedTrack = &cfg.False
		}

		if p.ShowAudio == on {
			old.ShowAudio = &cfg.True
		} else {
			old.ShowAudio = &cfg.False
		}

		if *old.Player == cfg.HLSPlayer {
			if cfg.ProxyStreams {
				switch p.ProxyStreams {
				case on:
					old.ProxyStreams = &cfg.True
				case "":
					old.ProxyStreams = &cfg.False
				}
			}

			switch p.FullyPreloadTrack {
			case on:
				old.FullyPreloadTrack = &cfg.True
			case "":
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
			switch p.ProxyImages {
			case on:
				old.ProxyImages = &cfg.True
			case "":
				old.ProxyImages = &cfg.False
			}
		}

		if p.ParseDescriptions == on {
			old.ParseDescriptions = &cfg.True
		} else {
			old.ParseDescriptions = &cfg.False
		}

		if p.SearchSuggestions == on {
			old.SearchSuggestions = &cfg.True
		} else {
			old.SearchSuggestions = &cfg.False
		}

		if p.DynamicLoadComments == on {
			old.DynamicLoadComments = &cfg.True
		} else {
			old.DynamicLoadComments = &cfg.False
		}

		if p.KeepPlayerFocus == on {
			old.KeepPlayerFocus = &cfg.True
		} else {
			old.KeepPlayerFocus = &cfg.False
		}

		old.Player = &p.Player

		setPrefs(c, &old)

		return c.Redirect().To("/_/preferences")
	})

	r.Get("/_/preferences/reset", func(c fiber.Ctx) error {
		// do not use (fiber.Ctx).Cookie() ts pmo icl

		cookie := fasthttp.AcquireCookie()
		cookie.SetKey("prefs")
		cookie.SetValue("{}")
		cookie.SetExpire(time.Now().Add(400 * 24 * time.Hour))
		cookie.SetHTTPOnly(true)
		cookie.SetSameSite(fasthttp.CookieSameSiteStrictMode)
		c.Response().Header.SetCookie(cookie)
		fasthttp.ReleaseCookie(cookie)

		return c.Redirect().To("/_/preferences")
	})

	r.Get("/_/preferences/export", func(c fiber.Ctx) error {
		p, err := Get(c)
		if err != nil {
			return err
		}

		return c.JSON(Export{Preferences: &p})
	})

	r.Post("/_/preferences/import", func(c fiber.Ctx) error {
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

		setPrefs(c, p.Preferences)

		return c.Redirect().To("/_/preferences")
	})
}
