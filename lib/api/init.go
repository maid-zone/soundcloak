package api

import (
	"log"
	"net/url"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
)

func Load(a *fiber.App) {
	r := a.Group("/_/api")

	prefs := cfg.Preferences{ProxyImages: &cfg.False}
	r.Get("/search", func(c fiber.Ctx) error {
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

	r.Get("/track/:id/related", func(c fiber.Ctx) error {
		args := cfg.B2s(c.RequestCtx().QueryArgs().Peek("pagination"))
		if args == "" {
			args = "?limit=20"
		}

		p, err := (sc.Track{ID: json.Number(c.Params("id"))}).GetRelated("", prefs, args)
		if err != nil {
			log.Printf("[API] error getting related tracks for %s: %s\n", c.Params("id"), err)
			return err
		}

		return c.JSON(p)
	})

	r.Get("/playlistByPermalink/:author/sets/:playlist", func(c fiber.Ctx) error {
		p, err := sc.GetPlaylist("", c.Params("author")+"/sets/"+c.Params("playlist"))
		if err != nil {
			log.Printf("[API] error getting %s playlist from %s: %s\n", c.Params("playlist"), c.Params("author"), err)
			return err
		}

		return c.JSON(p)
	})

	r.Get("/playlistByPermalink/:author/sets/:playlist/tracks", func(c fiber.Ctx) error {
		p, err := sc.GetPlaylist("", c.Params("author")+"/sets/"+c.Params("playlist"))
		if err != nil {
			log.Printf("[API] error getting %s playlist tracks from %s: %s\n", c.Params("playlist"), c.Params("author"), err)
			return err
		}

		tracks := make([]json.Number, len(p.Tracks))
		for i, t := range p.Tracks {
			tracks[i] = t.ID
		}

		return c.JSON(tracks)
	})

	r.Get("/track/:id", func(c fiber.Ctx) error {
		t, err := sc.GetTrackByID("", c.Params("id"))
		if err != nil {
			log.Printf("[API] error getting track %s: %s\n", c.Params("id"), err)
			return err
		}

		return c.JSON(t)
	})

	r.Get("/tracks", func(c fiber.Ctx) error {
		ids := cfg.B2s(c.RequestCtx().QueryArgs().Peek("ids"))
		t, err := sc.GetTracks("", ids)
		if err != nil {
			log.Printf("[API] error getting %s tracks: %s\n", ids, err)
			return err
		}

		return c.JSON(t)
	})
}
