package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/earlydata"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/maid-zone/soundcloak/lib/sc"
	"github.com/maid-zone/soundcloak/templates"
)

func main() {
	app := fiber.New(fiber.Config{
		Prefork:     cfg.Prefork,
		JSONEncoder: cfg.JSON.Marshal,
		JSONDecoder: cfg.JSON.Unmarshal,
	})
	app.Use(compress.New())
	app.Use(recover.New())
	app.Use(earlydata.New())

	app.Static("/", "assets", fiber.Static{Compress: true, MaxAge: 3600})
	app.Static("/js/hls.js/", "node_modules/hls.js/dist", fiber.Static{Compress: true, MaxAge: 3600})

	app.Get("/search", func(c *fiber.Ctx) error {
		q := c.Query("q")
		t := c.Query("type")
		switch t {
		case "tracks":
			p, err := sc.SearchTracks(c.Query("pagination", "?q="+url.QueryEscape(q)))
			if err != nil {
				fmt.Printf("error getting tracks for %s: %s\n", q, err)
				return err
			}

			c.Set("Content-Type", "text/html")
			return templates.Base("tracks: "+q, templates.SearchTracks(p), nil).Render(context.Background(), c)

		case "users":
			p, err := sc.SearchUsers(c.Query("pagination", "?q="+url.QueryEscape(q)))
			if err != nil {
				fmt.Printf("error getting users for %s: %s\n", q, err)
				return err
			}

			c.Set("Content-Type", "text/html")
			return templates.Base("users: "+q, templates.SearchUsers(p), nil).Render(context.Background(), c)

		case "playlists":
			p, err := sc.SearchPlaylists(c.Query("pagination", "?q="+url.QueryEscape(q)))
			if err != nil {
				fmt.Printf("error getting users for %s: %s\n", q, err)
				return err
			}

			c.Set("Content-Type", "text/html")
			return templates.Base("playlists: "+q, templates.SearchPlaylists(p), nil).Render(context.Background(), c)
		}

		return c.SendStatus(404)
	})

	app.Get("/:user/:track", func(c *fiber.Ctx) error {
		track, err := sc.GetTrack(c.Params("user") + "/" + c.Params("track"))
		if err != nil {
			fmt.Printf("error getting %s from %s: %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}

		stream, err := track.GetStream()
		if err != nil {
			fmt.Printf("error getting %s stream from %s: %s\n", c.Params("track"), c.Params("user"), err)
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(track.Title+" by "+track.Author.Username, templates.Track(track, stream), templates.TrackEmbed(track)).Render(context.Background(), c)
	})

	app.Get("/:user", func(c *fiber.Ctx) error {
		//h := time.Now()
		usr, err := sc.GetUser(c.Params("user"))
		if err != nil {
			fmt.Printf("error getting %s: %s\n", c.Params("user"), err)
			return err
		}
		//fmt.Println("getuser", time.Since(h))

		//h = time.Now()
		p, err := usr.GetTracks(c.Query("pagination", "?limit=20"))
		if err != nil {
			fmt.Printf("error getting %s tracks: %s\n", c.Params("user"), err)
			return err
		}
		//fmt.Println("gettracks", time.Since(h))

		c.Set("Content-Type", "text/html")
		return templates.Base(usr.Username, templates.User(usr, p), templates.UserEmbed(usr)).Render(context.Background(), c)
	})

	app.Get("/:user/sets/:playlist", func(c *fiber.Ctx) error {
		playlist, err := sc.GetPlaylist(c.Params("user") + "/sets/" + c.Params("playlist"))
		if err != nil {
			fmt.Printf("error getting %s playlist from %s: %s\n", c.Params("playlist"), c.Params("user"), err)
			return err
		}

		p := c.Query("pagination")
		if p != "" {
			tracks, next, err := sc.GetNextMissingTracks(p)
			if err != nil {
				fmt.Printf("error getting %s playlist tracks from %s: %s\n", c.Params("playlist"), c.Params("user"), err)
				return err
			}

			playlist.Tracks = tracks
			playlist.MissingTracks = strings.Join(next, ",")
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(playlist.Title+" by "+playlist.Author.Username, templates.Playlist(playlist), templates.PlaylistEmbed(playlist)).Render(context.Background(), c)
	})

	log.Fatal(app.Listen(cfg.Addr))
}
