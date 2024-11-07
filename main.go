package main

import (
	"context"
	"log"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/earlydata"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/valyala/fasthttp"

	"github.com/maid-zone/soundcloak/lib/cfg"
	proxyimages "github.com/maid-zone/soundcloak/lib/proxy_images"
	proxystreams "github.com/maid-zone/soundcloak/lib/proxy_streams"
	"github.com/maid-zone/soundcloak/lib/restream"
	"github.com/maid-zone/soundcloak/lib/sc"
	"github.com/maid-zone/soundcloak/templates"
)

func main() {
	app := fiber.New(fiber.Config{
		Prefork:     cfg.Prefork,
		JSONEncoder: cfg.JSON.Marshal,
		JSONDecoder: cfg.JSON.Unmarshal,

		EnableTrustedProxyCheck: cfg.TrustedProxyCheck,
		TrustedProxies:          cfg.TrustedProxies,
	})

	app.Use(recover.New())
	if cfg.EarlyData {
		app.Use(earlydata.New())
	}
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))

	app.Static("/", "assets", fiber.Static{Compress: true, MaxAge: 3600})
	app.Static("/js/hls.js/", "node_modules/hls.js/dist", fiber.Static{Compress: true, MaxAge: 14400})

	app.Get("/search", func(c *fiber.Ctx) error {
		q := c.Query("q")
		t := c.Query("type")
		switch t {
		case "tracks":
			p, err := sc.SearchTracks(c.Query("pagination", "?q="+url.QueryEscape(q)))
			if err != nil {
				log.Printf("error getting tracks for %s: %s\n", q, err)
				return err
			}

			c.Set("Content-Type", "text/html")
			return templates.Base("tracks: "+q, templates.SearchTracks(p), nil).Render(context.Background(), c)

		case "users":
			p, err := sc.SearchUsers(c.Query("pagination", "?q="+url.QueryEscape(q)))
			if err != nil {
				log.Printf("error getting users for %s: %s\n", q, err)
				return err
			}

			c.Set("Content-Type", "text/html")
			return templates.Base("users: "+q, templates.SearchUsers(p), nil).Render(context.Background(), c)

		case "playlists":
			p, err := sc.SearchPlaylists(c.Query("pagination", "?q="+url.QueryEscape(q)))
			if err != nil {
				log.Printf("error getting users for %s: %s\n", q, err)
				return err
			}

			c.Set("Content-Type", "text/html")
			return templates.Base("playlists: "+q, templates.SearchPlaylists(p), nil).Render(context.Background(), c)
		}

		return c.SendStatus(404)
	})

	app.Get("/on/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.ErrNotFound
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.Header.SetMethod("HEAD")
		req.SetRequestURI("https://on.soundcloud.com/" + id)
		req.Header.Set("User-Agent", cfg.UserAgent)

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)

		err := fasthttp.Do(req, resp)
		if err != nil {
			return err
		}

		loc := resp.Header.Peek("location")
		if len(loc) == 0 {
			return fiber.ErrNotFound
		}

		//fmt.Println(c.Hostname(), c.Protocol(), c.IPs())

		u, err := url.Parse(string(loc))
		if err != nil {
			return err
		}

		return c.Redirect(u.Path)
	})

	app.Get("/w/player", func(c *fiber.Ctx) error {
		u := c.Query("url")
		if u == "" {
			return fiber.ErrNotFound
		}

		track, err := sc.GetArbitraryTrack(u)

		if err != nil {
			log.Printf("error getting %s: %s\n", u, err)
			return err
		}
		displayErr := ""

		stream, err := track.GetStream()
		if err != nil {
			log.Printf("error getting %s stream from %s: %s\n", c.Params("track"), c.Params("user"), err)
			displayErr = "Failed to get track stream: " + err.Error()
			if track.Policy == sc.PolicyBlock {
				displayErr += "\nThis track may be blocked in the country where this instance is hosted."
			}
		}

		c.Set("Content-Type", "text/html")
		return templates.TrackEmbed(track, stream, displayErr).Render(context.Background(), c)
	})

	if cfg.ProxyImages {
		proxyimages.Load(app)
	}

	if cfg.ProxyStreams {
		proxystreams.Load(app)
	}

	if cfg.InstanceInfo {
		type info struct {
			ProxyImages       bool
			ProxyStreams      bool
			FullyPreloadTrack bool
			Restream          bool
		}

		app.Get("/_/info", func(c *fiber.Ctx) error {
			return c.JSON(info{
				ProxyImages:       cfg.ProxyImages,
				ProxyStreams:      cfg.ProxyStreams,
				FullyPreloadTrack: cfg.FullyPreloadTrack,
				Restream:          cfg.Restream,
			})
		})
	}

	if cfg.Restream {
		restream.Load(app)
	}

	app.Get("/:user/sets", func(c *fiber.Ctx) error {
		user, err := sc.GetUser(c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (playlists): %s\n", c.Params("user"), err)
			return err
		}

		pl, err := user.GetPlaylists(c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s playlists: %s\n", c.Params("user"), err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(user.Username, templates.UserPlaylists(user, pl), templates.UserHeader(user)).Render(context.Background(), c)
	})

	app.Get("/:user/albums", func(c *fiber.Ctx) error {
		user, err := sc.GetUser(c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (albums): %s\n", c.Params("user"), err)
			return err
		}

		pl, err := user.GetAlbums(c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s albums: %s\n", c.Params("user"), err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(user.Username, templates.UserAlbums(user, pl), templates.UserHeader(user)).Render(context.Background(), c)
	})

	app.Get("/:user/reposts", func(c *fiber.Ctx) error {
		user, err := sc.GetUser(c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (reposts): %s\n", c.Params("user"), err)
			return err
		}

		p, err := user.GetReposts(c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s reposts: %s\n", c.Params("user"), err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(user.Username, templates.UserReposts(user, p), templates.UserHeader(user)).Render(context.Background(), c)
	})

	app.Get("/:user/:track", func(c *fiber.Ctx) error {
		track, err := sc.GetTrack(c.Params("user") + "/" + c.Params("track"))
		if err != nil {
			log.Printf("error getting %s from %s: %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}
		displayErr := ""

		stream, err := track.GetStream()
		if err != nil {
			log.Printf("error getting %s stream from %s: %s\n", c.Params("track"), c.Params("user"), err)
			displayErr = "Failed to get track stream: " + err.Error()
			if track.Policy == sc.PolicyBlock {
				displayErr += "\nThis track may be blocked in the country where this instance is hosted."
			}
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(track.Title+" by "+track.Author.Username, templates.Track(track, stream, displayErr), templates.TrackHeader(track)).Render(context.Background(), c)
	})

	app.Get("/:user", func(c *fiber.Ctx) error {
		//h := time.Now()
		usr, err := sc.GetUser(c.Params("user"))
		if err != nil {
			log.Printf("error getting %s: %s\n", c.Params("user"), err)
			return err
		}
		//fmt.Println("getuser", time.Since(h))

		//h = time.Now()
		p, err := usr.GetTracks(c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s tracks: %s\n", c.Params("user"), err)
			return err
		}
		//fmt.Println("gettracks", time.Since(h))

		c.Set("Content-Type", "text/html")
		return templates.Base(usr.Username, templates.User(usr, p), templates.UserHeader(usr)).Render(context.Background(), c)
	})

	app.Get("/:user/sets/:playlist", func(c *fiber.Ctx) error {
		playlist, err := sc.GetPlaylist(c.Params("user") + "/sets/" + c.Params("playlist"))
		if err != nil {
			log.Printf("error getting %s playlist from %s: %s\n", c.Params("playlist"), c.Params("user"), err)
			return err
		}

		p := c.Query("pagination")
		if p != "" {
			tracks, next, err := sc.GetNextMissingTracks(p)
			if err != nil {
				log.Printf("error getting %s playlist tracks from %s: %s\n", c.Params("playlist"), c.Params("user"), err)
				return err
			}

			playlist.Tracks = tracks
			playlist.MissingTracks = strings.Join(next, ",")
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(playlist.Title+" by "+playlist.Author.Username, templates.Playlist(playlist), templates.PlaylistHeader(playlist)).Render(context.Background(), c)
	})

	log.Fatal(app.Listen(cfg.Addr))
}
