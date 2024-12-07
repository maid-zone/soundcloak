package main

import (
	"context"
	"log"
	"math/rand"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
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

// see build script/dockerfile
var commit = "unknown"
var repo = "unknown"

func main() {
	app := fiber.New(fiber.Config{
		Prefork:     cfg.Prefork,
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,

		EnableTrustedProxyCheck: cfg.TrustedProxyCheck,
		TrustedProxies:          cfg.TrustedProxies,
	})

	app.Use(recover.New())
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))

	app.Static("/", "assets", fiber.Static{Compress: true, MaxAge: 7200})                              // 2 hours
	app.Static("/js/hls.js/", "node_modules/hls.js/dist", fiber.Static{Compress: true, MaxAge: 28800}) // 8 hours

	// Just for easy inspection of cache in development. Since debug is constant, the compiler will just remove the code below if it's set to false, so this has no runtime overhead.
	const debug = false
	if debug {
		app.Get("/_/cachedump/tracks", func(c *fiber.Ctx) error {
			return c.JSON(sc.TracksCache)
		})

		app.Get("/_/cachedump/playlists", func(c *fiber.Ctx) error {
			return c.JSON(sc.PlaylistsCache)
		})

		app.Get("/_/cachedump/users", func(c *fiber.Ctx) error {
			return c.JSON(sc.UsersCache)
		})

		app.Get("/_/cachedump/clientId", func(c *fiber.Ctx) error {
			return c.JSON(sc.ClientIDCache)
		})
	}

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

		track, err := sc.GetArbitraryTrack(u)
		if err != nil {
			log.Printf("error getting %s: %s\n", u, err)
			return err
		}
		track.Postfix(prefs, true)

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
			Commit             string
			Repo               string
			ProxyImages        bool
			ProxyStreams       bool
			Restream           bool
			GetWebProfiles     bool
			DefaultPreferences cfg.Preferences
		}

		app.Get("/_/info", func(c *fiber.Ctx) error {
			return c.JSON(info{
				Commit:             commit,
				Repo:               repo,
				ProxyImages:        cfg.ProxyImages,
				ProxyStreams:       cfg.ProxyStreams,
				Restream:           cfg.Restream,
				GetWebProfiles:     cfg.GetWebProfiles,
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

		user, err := sc.GetUser(c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (playlists): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

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

		user, err := sc.GetUser(c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (albums): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

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

		user, err := sc.GetUser(c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (reposts): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		p, err := user.GetReposts(prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s reposts: %s\n", c.Params("user"), err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(user.Username, templates.UserReposts(prefs, user, p), templates.UserHeader(user)).Render(context.Background(), c)
	})

	app.Get("/:user/likes", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		user, err := sc.GetUser(c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (likes): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		p, err := user.GetLikes(prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s likes: %s\n", c.Params("user"), err)
			return err
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(user.Username, templates.UserLikes(prefs, user, p), templates.UserHeader(user)).Render(context.Background(), c)
	})

	app.Get("/:user/:track", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		track, err := sc.GetTrack(c.Params("user") + "/" + c.Params("track"))
		if err != nil {
			log.Printf("error getting %s from %s: %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}
		track.Postfix(prefs, true)

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

		var playlist *sc.Playlist
		var nextTrack *sc.Track
		mode := c.Query("mode", *prefs.DefaultAutoplayMode)
		if pl := c.Query("playlist"); pl != "" {
			p, err := sc.GetPlaylist(pl)
			if err != nil {
				log.Printf("error getting %s playlist (track): %s\n", pl, err)
				return err
			}

			p.Tracks = p.Postfix(prefs, true, false)

			nextIndex := -1
			if mode == cfg.AutoplayRandom {
				nextIndex = rand.Intn(len(p.Tracks))
			} else {
				for i, t := range p.Tracks {
					if t.ID == track.ID {
						nextIndex = i + 1
					}
				}

				if nextIndex == len(p.Tracks) {
					nextIndex = 0
				}
			}

			if nextIndex != -1 {
				nextTrack = &p.Tracks[nextIndex]
				playlist = &p

				if nextTrack.Title == "" {
					nt, err := sc.GetTrackByID(nextTrack.ID)
					if err != nil {
						return err
					}

					nt.Postfix(prefs, false)

					nextTrack = &nt
				}
			}
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(track.Title+" by "+track.Author.Username, templates.Track(prefs, track, stream, displayErr, c.Query("autoplay") == "true", playlist, nextTrack, c.Query("volume"), mode), templates.TrackHeader(prefs, track)).Render(context.Background(), c)
	})

	app.Get("/:user", func(c *fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		//h := time.Now()
		usr, err := sc.GetUser(c.Params("user"))
		if err != nil {
			log.Printf("error getting %s: %s\n", c.Params("user"), err)
			return err
		}
		usr.Postfix(prefs)
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

		playlist, err := sc.GetPlaylist(c.Params("user") + "/sets/" + c.Params("playlist"))
		if err != nil {
			log.Printf("error getting %s playlist from %s: %s\n", c.Params("playlist"), c.Params("user"), err)
			return err
		}
		// Don't ask why
		playlist.Tracks = playlist.Postfix(prefs, true, true)

		p := c.Query("pagination")
		if p != "" {
			tracks, next, err := sc.GetNextMissingTracks(p)
			if err != nil {
				log.Printf("error getting %s playlist tracks from %s: %s\n", c.Params("playlist"), c.Params("user"), err)
				return err
			}

			for i, track := range tracks {
				track.Postfix(prefs, false)
				tracks[i] = track
			}

			playlist.Tracks = tracks
			playlist.MissingTracks = strings.Join(next, ",")
		}

		c.Set("Content-Type", "text/html")
		return templates.Base(playlist.Title+" by "+playlist.Author.Username, templates.Playlist(prefs, playlist), templates.PlaylistHeader(playlist)).Render(context.Background(), c)
	})

	log.Fatal(app.Listen(cfg.Addr))
}
