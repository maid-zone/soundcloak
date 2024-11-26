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
	"github.com/segmentio/encoding/json"
	"github.com/valyala/fasthttp"

	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/maid-zone/soundcloak/lib/preferences"
	proxyimages "github.com/maid-zone/soundcloak/lib/proxy_images"
	proxystreams "github.com/maid-zone/soundcloak/lib/proxy_streams"
	"github.com/maid-zone/soundcloak/lib/restream"
	"github.com/maid-zone/soundcloak/lib/sc"
	"github.com/maid-zone/soundcloak/templates"
)

func main() {
	app := fiber.New(fiber.Config{
		Prefork:     cfg.Prefork,
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,

		EnableTrustedProxyCheck: cfg.TrustedProxyCheck,
		TrustedProxies:          cfg.TrustedProxies,
	})

	app.Use(recover.New())
	if cfg.EarlyData {
		app.Use(earlydata.New())
	}
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))

	app.Static("/", "assets", fiber.Static{Compress: true, MaxAge: 3600})                              // 1hour
	app.Static("/js/hls.js/", "node_modules/hls.js/dist", fiber.Static{Compress: true, MaxAge: 28800}) // 8 hours

	app.Get("/search", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		q := c.Query("q")
		t := c.Query("type")
		switch t {
		case "tracks":
			p, err := sc.SearchTracks(prefs, c.Query("pagination", "?q="+url.QueryEscape(q)))
			if err != nil {
				log.Printf("error getting tracks for %s: %s\n", q, err)
				return err
			}

			c.Set("Content-Type", "text/html")
			return templates.Base("tracks: "+q, templates.SearchTracks(p), nil).Render(context.Background(), c)

		case "users":
			p, err := sc.SearchUsers(prefs, c.Query("pagination", "?q="+url.QueryEscape(q)))
			if err != nil {
				log.Printf("error getting users for %s: %s\n", q, err)
				return err
			}

			c.Set("Content-Type", "text/html")
			return templates.Base("users: "+q, templates.SearchUsers(p), nil).Render(context.Background(), c)

		case "playlists":
			p, err := sc.SearchPlaylists(prefs, c.Query("pagination", "?q="+url.QueryEscape(q)))
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

		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		track, err := sc.GetArbitraryTrack(prefs, u)

		if err != nil {
			log.Printf("error getting %s: %s\n", u, err)
			return err
		}
		displayErr := ""
		stream := ""

		if *prefs.Player != cfg.NonePlayer {
			tr := track.Media.SelectCompatible()
			if tr == nil {
				err = sc.ErrIncompatibleStream
			} else if *prefs.Player == cfg.HLSPlayer {
				stream, err = tr.GetStream(prefs, track.Authorization)
			}

			if err != nil {
				displayErr = "Failed to get track stream: " + err.Error()
				if track.Policy == sc.PolicyBlock {
					displayErr += "\nThis track may be blocked in the country where this instance is hosted."
				}
			}
		}

		c.Set("Content-Type", "text/html")
		return templates.TrackEmbed(prefs, track, stream, displayErr).Render(context.Background(), c)
	})

	app.Get("/_/featured", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		tracks, err := sc.GetFeaturedTracks(prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting featured tracks: %s\n", err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base("Featured Tracks", templates.FeaturedTracks(tracks), nil).Render(context.Background(), c)
	})

	app.Get("/discover", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		selections, err := sc.GetSelections(prefs) // There is no pagination
		if err != nil {
			log.Printf("error getting selections: %s\n", err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base("Discover", templates.Discover(selections), nil).Render(context.Background(), c)
	})

	if cfg.ProxyImages {
		proxyimages.Load(app)
	}

	if cfg.ProxyStreams {
		proxystreams.Load(app)
	}

	if cfg.InstanceInfo {
		type info struct {
			ProxyImages        bool
			ProxyStreams       bool
			Restream           bool
			DefaultPreferences cfg.Preferences
		}

		app.Get("/_/info", func(c *fiber.Ctx) error {
			return c.JSON(info{
				ProxyImages:        cfg.ProxyImages,
				ProxyStreams:       cfg.ProxyStreams,
				Restream:           cfg.Restream,
				DefaultPreferences: cfg.DefaultPreferences,
			})
		})
	}

	if cfg.Restream {
		restream.Load(app)
	}

	preferences.Load(app)

	app.Get("/:user/sets", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		user, err := sc.GetUser(prefs, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (playlists): %s\n", c.Params("user"), err)
			return err
		}

		pl, err := user.GetPlaylists(prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s playlists: %s\n", c.Params("user"), err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(user.Username, templates.UserPlaylists(prefs, user, pl), templates.UserHeader(user)).Render(context.Background(), c)
	})

	app.Get("/:user/albums", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		user, err := sc.GetUser(prefs, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (albums): %s\n", c.Params("user"), err)
			return err
		}

		pl, err := user.GetAlbums(prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s albums: %s\n", c.Params("user"), err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(user.Username, templates.UserAlbums(prefs, user, pl), templates.UserHeader(user)).Render(context.Background(), c)
	})

	app.Get("/:user/reposts", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		user, err := sc.GetUser(prefs, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (reposts): %s\n", c.Params("user"), err)
			return err
		}

		p, err := user.GetReposts(prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s reposts: %s\n", c.Params("user"), err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(user.Username, templates.UserReposts(prefs, user, p), templates.UserHeader(user)).Render(context.Background(), c)
	})

	app.Get("/:user/:track", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		track, err := sc.GetTrack(prefs, c.Params("user")+"/"+c.Params("track"))
		if err != nil {
			log.Printf("error getting %s from %s: %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}
		displayErr := ""
		stream := ""

		if *prefs.Player != cfg.NonePlayer {
			tr := track.Media.SelectCompatible()
			if tr == nil {
				err = sc.ErrIncompatibleStream
			} else if *prefs.Player == cfg.HLSPlayer {
				stream, err = tr.GetStream(prefs, track.Authorization)
			}

			if err != nil {
				displayErr = "Failed to get track stream: " + err.Error()
				if track.Policy == sc.PolicyBlock {
					displayErr += "\nThis track may be blocked in the country where this instance is hosted."
				}
			}
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(track.Title+" by "+track.Author.Username, templates.Track(prefs, track, stream, displayErr), templates.TrackHeader(prefs, track)).Render(context.Background(), c)
	})

	app.Get("/:user", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		//h := time.Now()
		usr, err := sc.GetUser(prefs, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s: %s\n", c.Params("user"), err)
			return err
		}
		//fmt.Println("getuser", time.Since(h))

		//h = time.Now()
		p, err := usr.GetTracks(prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s tracks: %s\n", c.Params("user"), err)
			return err
		}
		//fmt.Println("gettracks", time.Since(h))

		c.Set("Content-Type", "text/html")
		return templates.Base(usr.Username, templates.User(prefs, usr, p), templates.UserHeader(usr)).Render(context.Background(), c)
	})

	app.Get("/:user/sets/:playlist", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		playlist, err := sc.GetPlaylist(prefs, c.Params("user")+"/sets/"+c.Params("playlist"))
		if err != nil {
			log.Printf("error getting %s playlist from %s: %s\n", c.Params("playlist"), c.Params("user"), err)
			return err
		}

		p := c.Query("pagination")
		if p != "" {
			tracks, next, err := sc.GetNextMissingTracks(prefs, p)
			if err != nil {
				log.Printf("error getting %s playlist tracks from %s: %s\n", c.Params("playlist"), c.Params("user"), err)
				return err
			}

			playlist.Tracks = tracks
			playlist.MissingTracks = strings.Join(next, ",")
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(playlist.Title+" by "+playlist.Author.Username, templates.Playlist(prefs, playlist), templates.PlaylistHeader(playlist)).Render(context.Background(), c)
	})

	log.Fatal(app.Listen(cfg.Addr))
}
