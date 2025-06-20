package api

import (
	"log"
	"net/url"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/preferences"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"github.com/gofiber/fiber/v3"
)

func Load(r *fiber.App) {
	r.Get("/_/api/search", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		q := cfg.B2s(c.RequestCtx().QueryArgs().Peek("q"))
		t := cfg.B2s(c.RequestCtx().QueryArgs().Peek("type"))
		args := cfg.B2s(c.RequestCtx().QueryArgs().Peek("pagination"))
		if args == "" {
			args = "?q=" + url.QueryEscape(q)
		}

		switch t {
		case "tracks":
			p, err := sc.SearchTracks("", prefs, args)
			if err != nil {
				log.Printf("[API] error getting tracks for %s: %s\n", q, err)
				return err
			}

			return c.JSON(p)

		case "users":
			p, err := sc.SearchUsers("", prefs, args)
			if err != nil {
				log.Printf("[API] error getting users for %s: %s\n", q, err)
				return err
			}

			return c.JSON(p)

		case "playlists":
			p, err := sc.SearchPlaylists("", prefs, args)
			if err != nil {
				log.Printf("[API] error getting playlists for %s: %s\n", q, err)
				return err
			}

			return c.JSON(p)
		}

		return c.SendStatus(404)
	})
}
