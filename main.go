package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"sync"

	"git.maid.zone/stuff/soundcloak/lib/api"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/valyala/fasthttp"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/preferences"
	proxyimages "git.maid.zone/stuff/soundcloak/lib/proxy_images"
	proxystreams "git.maid.zone/stuff/soundcloak/lib/proxy_streams"
	"git.maid.zone/stuff/soundcloak/lib/restream"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"git.maid.zone/stuff/soundcloak/templates"

	static_files "git.maid.zone/stuff/soundcloak/static"
)

func boolean(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}

type compressionMap = map[string][][]byte

func parseCompressionMap(path string, filesystem fs.FS) compressionMap {
	f, err := filesystem.Open(path + "/.compression")
	if err != nil {
		return nil
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil
	}

	sp := bytes.Split(data, []byte("\n"))
	cm := make(compressionMap, len(sp))
	for _, h := range sp {
		sp2 := bytes.Split(h, []byte("|"))
		if len(sp2) != 2 || string(sp2[0]) == ".gitkeep" {
			continue
		}

		h := bytes.Split(sp2[1], []byte(","))
		if len(h) == 0 || len(h[0]) == 0 {
			h = nil
		}

		cm[cfg.B2s(sp2[0])] = h
	}

	return cm
}

func parseCompressionMaps(filesystem fs.FS) compressionMap {
	var (
		external = parseCompressionMap("external", filesystem)
		assets   = parseCompressionMap("assets", filesystem)
		instance = parseCompressionMap("instance", filesystem)
	)

	res := make(compressionMap, len(external)+len(assets)+len(instance))

	for k, v := range external {
		res["external/"+k] = v
	}
	for k, v := range assets {
		res["assets/"+k] = v
	}
	for k, v := range instance {
		res["instance/"+k] = v
	}

	return res
}

type pooledReader struct {
	handle io.ReadSeeker
	p      *sync.Pool
}

func (pr *pooledReader) Read(data []byte) (int, error) {
	return pr.handle.Read(data)
}

func (pr *pooledReader) Close() error {
	pr.handle.Seek(0, io.SeekStart)
	pr.p.Put(pr)
	return nil
}

type pooledFs struct {
	filesystem fs.FS
	pools      map[string]*sync.Pool
	mut        sync.RWMutex
}

func (p *pooledFs) Open(name string) (*pooledReader, error) {
	p.mut.RLock()
	pool := p.pools[name]
	p.mut.RUnlock()

	if pool == nil {
		misc.Log("pool is nil for", name)
		pool = &sync.Pool{}
		p.mut.Lock()
		p.pools[name] = pool
		p.mut.Unlock()

		goto new
	}

	if pr := pool.Get(); pr != nil {
		return pr.(*pooledReader), nil
	}

new:
	h, err := p.filesystem.Open(name)
	if err != nil {
		return nil, err
	}

	return &pooledReader{h.(io.ReadSeeker), pool}, nil
}

func ServeFS(r *fiber.App, filesystem fs.FS) {
	cm := parseCompressionMaps(filesystem)
	misc.Log(cm)

	pfs := &pooledFs{filesystem, make(map[string]*sync.Pool), sync.RWMutex{}}

	const path = "/_/static/"

	if len(cm) == 0 {
		r.Use(path, func(c fiber.Ctx) error {
			// start := time.Now()
			// defer func() { fmt.Println("it took", time.Since(start)) }()
			fp := cfg.B2s(c.RequestCtx().Path()[len(path):])

			if strings.HasSuffix(fp, ".css") {
				c.Response().Header.SetContentType("text/css")
			} else if strings.HasSuffix(fp, ".js") {
				c.Response().Header.SetContentType("text/javascript")
			} else if strings.HasSuffix(fp, ".jpg") {
				c.Response().Header.SetContentType("image/jpeg")
			} else if strings.HasSuffix(fp, ".ttf") {
				c.Response().Header.SetContentType("font/ttf")
			}

			var (
				f   *pooledReader
				err error
			)
			if !strings.HasPrefix(fp, "external/") {
				f, err = pfs.Open("instance/" + fp)
				if err != nil {
					f, err = pfs.Open("assets/" + fp)
				}
			} else {
				f, err = pfs.Open(fp)
			}

			if err != nil {
				return err
			}

			c.Set("Cache-Control", "public, max-age=28800")
			return c.SendStream(f)
		})
	} else {
		r.Use(path, func(c fiber.Ctx) error {
			// start := time.Now()
			// defer func() { fmt.Println("it took", time.Since(start)) }()
			fp := cfg.B2s(c.RequestCtx().Path()[len(path):])

			var (
				encs [][]byte
				ok   bool
			)
			if strings.HasPrefix(fp, "external/") {
				encs, ok = cm[fp]
			} else {
				encs, ok = cm["instance/"+fp]
				if ok {
					fp = "instance/" + fp
				} else {
					fp = "assets/" + fp
					encs, ok = cm[fp]
				}
			}

			if !ok {
				return fiber.ErrNotFound
			}

			if strings.HasSuffix(fp, ".css") {
				c.Response().Header.SetContentType("text/css")
			} else if strings.HasSuffix(fp, ".js") {
				c.Response().Header.SetContentType("text/javascript")
			} else if strings.HasSuffix(fp, ".jpg") {
				c.Response().Header.SetContentType("image/jpeg")
			} else if strings.HasSuffix(fp, ".ttf") {
				c.Response().Header.SetContentType("font/ttf")
			}

			if len(encs) != 0 {
				ae := c.Request().Header.Peek("Accept-Encoding")
				if len(ae) == 1 && ae[0] == '*' {
					c.Response().Header.SetContentEncodingBytes(encs[0])
					fp += "." + cfg.B2s(encs[0])
				} else {
					for _, enc := range encs {
						if bytes.Index(ae, enc) != -1 {
							c.Response().Header.SetContentEncodingBytes(enc)
							fp += "." + cfg.B2s(enc)
							break
						}
					}
				}
			}

			f, err := pfs.Open(fp)
			if err != nil {
				return err
			}

			c.Set("Cache-Control", "public, max-age=28800")
			return c.SendStream(f)
		})
	}
}

func render(c fiber.Ctx, t templ.Component) error {
	c.Response().Header.SetContentType("text/html")
	return t.Render(c.RequestCtx(), c.Response().BodyWriter())
}

func r(c fiber.Ctx, title string, content, head templ.Component) error {
	return render(c, templates.Base(title, content, head))
}

func main() {
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,

		TrustProxy:       cfg.TrustedProxyCheck,
		TrustProxyConfig: fiber.TrustProxyConfig{Proxies: cfg.TrustedProxies},
		ReadBufferSize:   4096 * 2,
	})

	if cfg.Debug {
		app.Server().Logger = fasthttp.Logger(log.New(os.Stdout, "", log.LstdFlags))
	}

	if !cfg.Debug { // you wanna catch any possible panics as soon as possible
		app.Use(recover.New())
	}

	app.Use(compress.New(compress.Config{
		Next: func(c fiber.Ctx) bool {
			return strings.HasPrefix(c.Path(), "/_/static")
		},
		Level: compress.LevelBestSpeed,
	}))

	// Just for easy inspection of cache in development. Since debug is constant, the compiler will just remove the code below if it's set to false, so this has no runtime overhead.
	if cfg.Debug {
		app.Get("/_/cachedump/tracks", func(c fiber.Ctx) error {
			return c.JSON(sc.TracksCache)
		})

		app.Get("/_/cachedump/playlists", func(c fiber.Ctx) error {
			return c.JSON(sc.PlaylistsCache)
		})

		app.Get("/_/cachedump/users", func(c fiber.Ctx) error {
			return c.JSON(sc.UsersCache)
		})

		app.Get("/_/cachedump/clientId", func(c fiber.Ctx) error {
			return c.JSON(sc.ClientIDCache)
		})
	}

	{
		mainPageHandler := func(c fiber.Ctx) error {
			prefs, err := preferences.Get(c)
			if err != nil {
				return err
			}

			return r(c, "", templates.MainPage(prefs), templates.MainPageHead())
		}

		app.Get("/", mainPageHandler)
		app.Get("/index.html", mainPageHandler)
	}

	if cfg.EmbedFiles {
		misc.Log("using embedded files")
		ServeFS(app, static_files.All)
	} else {
		misc.Log("loading files dynamically")
		r, err := os.OpenRoot("static")
		if err != nil {
			panic(err)
		}
		ServeFS(app, r.FS())
	}

	// why? because when you load a page without link rel="icon" the browser will
	// try to load favicon from default location,
	// and this path loads the user "favicon" by default
	app.Get("favicon.ico", func(c fiber.Ctx) error {
		return c.Redirect().Status(fiber.StatusPermanentRedirect).To("/_/static/favicon.ico")
	})

	app.Get("robots.txt", func(c fiber.Ctx) error {
		return c.SendString(`User-agent: *
Disallow: /`)
	})

	app.Get("/search", func(c fiber.Ctx) error {
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
				log.Printf("error getting tracks for %s: %s\n", q, err)
				return err
			}

			return r(c, "tracks: "+q, templates.SearchTracks(p), nil)

		case "users":
			p, err := sc.SearchUsers("", prefs, args)
			if err != nil {
				log.Printf("error getting users for %s: %s\n", q, err)
				return err
			}

			return r(c, "users: "+q, templates.SearchUsers(p), nil)

		case "playlists":
			p, err := sc.SearchPlaylists("", prefs, args)
			if err != nil {
				log.Printf("error getting playlists for %s: %s\n", q, err)
				return err
			}

			return r(c, "playlists: "+q, templates.SearchPlaylists(p), nil)
		}

		return c.SendStatus(404)
	})

	// someone is trying to hit those endpoints on sc.maid.zone at like 4am lol
	// those are authentication-only, planning to make something similar later on
	app.Get("/stream", func(c fiber.Ctx) error {
		return c.Redirect().To("/")
	})

	app.Get("/feed", func(c fiber.Ctx) error {
		return c.Redirect().To("/")
	})

	app.Get("/on/:id", func(c fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.ErrNotFound
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.Header.SetMethod("HEAD")
		req.SetRequestURI("https://on.soundcloud.com/" + id)
		req.Header.SetUserAgent(cfg.UserAgent)

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

		u, err := url.Parse(cfg.B2s(loc))
		if err != nil {
			return err
		}

		return c.Redirect().To(u.Path)
	})

	app.Get("/w/player", func(c fiber.Ctx) error {
		u := c.Query("url")
		if u == "" {
			return fiber.ErrNotFound
		}

		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		track, err := sc.GetArbitraryTrack(cid, u)
		if err != nil {
			log.Printf("error getting %s: %s\n", u, err)
			return err
		}
		track.Postfix(prefs, true)

		displayErr := ""
		stream := ""

		if *prefs.Player != cfg.NonePlayer {
			tr, _ := track.Media.SelectCompatible(*prefs.HLSAudio, false)
			if tr == nil {
				err = sc.ErrIncompatibleStream
			} else if *prefs.Player == cfg.HLSPlayer {
				stream, err = tr.GetStream(cid, prefs, track.Authorization)
			}

			if err != nil {
				displayErr = "Failed to get track stream: " + err.Error()
				if track.Policy == sc.PolicyBlock {
					displayErr += "\nThis track may be blocked in the country where this instance is hosted."
				}
			}
		}

		return render(c, templates.TrackEmbed(prefs, track, stream, displayErr))
	})

	app.Get("/tags/:tag", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		tag := c.Params("tag")
		p, err := sc.RecentTracks("", prefs, c.Query("pagination", tag+"?limit=20"))
		if err != nil {
			log.Printf("error getting %s tagged recent-tracks: %s\n", tag, err)
			return err
		}

		return r(c, "Recent tracks tagged "+tag, templates.RecentTracks(tag, p), nil)
	})

	app.Get("/tags/:tag/popular-tracks", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		tag := c.Params("tag")
		p, err := sc.SearchTracks("", prefs, c.Query("pagination", "?q=*&filter.genre_or_tag="+tag+"&sort=popular"))
		if err != nil {
			log.Printf("error getting %s tagged popular-tracks: %s\n", tag, err)
			return err
		}

		return r(c, "Popular tracks tagged "+tag, templates.PopularTracks(tag, p), nil)
	})

	app.Get("/tags/:tag/playlists", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		tag := c.Params("tag")
		// Using a different method, since /playlists/discovery endpoint seems to be broken :P
		p, err := sc.SearchPlaylists("", prefs, c.Query("pagination", "?q=*&filter.genre_or_tag="+tag))
		if err != nil {
			log.Printf("error getting %s tagged playlists: %s\n", tag, err)
			return err
		}

		return r(c, "Playlists tagged "+tag, templates.TaggedPlaylists(tag, p), nil)
	})

	app.Get("/discover", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		selections, err := sc.GetSelections("", prefs) // There is no pagination
		if err != nil {
			log.Printf("error getting selections: %s\n", err)
			return err
		}

		return r(c, "Discover", templates.Discover(selections), nil)
	})

	if cfg.ProxyImages {
		proxyimages.Load(app)
	}

	if cfg.ProxyStreams {
		proxystreams.Load(app)
	}

	if cfg.EnableAPI {
		api.Load(app)
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
			EnableAPI          bool
		}

		inf, err := json.Marshal(info{
			Commit:             cfg.Commit,
			Repo:               cfg.Repo,
			ProxyImages:        cfg.ProxyImages,
			ProxyStreams:       cfg.ProxyStreams,
			Restream:           cfg.Restream,
			GetWebProfiles:     cfg.GetWebProfiles,
			DefaultPreferences: cfg.DefaultPreferences,
			EnableAPI:          cfg.EnableAPI,
		})
		if err != nil {
			log.Fatalln("failed to marshal info: ", err)
		}

		app.Get("/_/info", func(c fiber.Ctx) error {
			c.Response().Header.SetContentType("application/json")
			return c.Send(inf)
		})
	}

	if cfg.Restream {
		restream.Load(app)
	}

	preferences.Load(app)

	app.Get("/_/searchSuggestions", func(c fiber.Ctx) error {
		q := c.Query("q")
		if q == "" {
			return fiber.ErrBadRequest
		}

		s, err := sc.GetSearchSuggestions("", q)
		if err != nil {
			return err
		}

		return c.JSON(s)
	})

	// Currently, /:user is the tracks page
	app.Get("/:user/tracks", func(c fiber.Ctx) error {
		return c.Redirect().To("/" + c.Params("user"))
	})

	app.Get("/:user/sets", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		user, err := sc.GetUser(cid, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (playlists): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		pl, err := user.GetPlaylists(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s playlists: %s\n", c.Params("user"), err)
			return err
		}

		return r(c, user.Username, templates.UserPlaylists(prefs, user, pl), templates.UserHeader(user))
	})

	app.Get("/:user/albums", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		user, err := sc.GetUser(cid, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (albums): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		pl, err := user.GetAlbums(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s albums: %s\n", c.Params("user"), err)
			return err
		}

		return r(c, user.Username, templates.UserAlbums(prefs, user, pl), templates.UserHeader(user))
	})

	app.Get("/:user/reposts", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		user, err := sc.GetUser(cid, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (reposts): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		p, err := user.GetReposts(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s reposts: %s\n", c.Params("user"), err)
			return err
		}

		return r(c, user.Username, templates.UserReposts(prefs, user, p), templates.UserHeader(user))
	})

	app.Get("/:user/likes", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		user, err := sc.GetUser(cid, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (likes): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		p, err := user.GetLikes(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s likes: %s\n", c.Params("user"), err)
			return err
		}

		return r(c, user.Username, templates.UserLikes(prefs, user, p), templates.UserHeader(user))
	})

	app.Get("/:user/popular-tracks", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		user, err := sc.GetUser(cid, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (popular-tracks): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		p, err := user.GetTopTracks(cid, prefs)
		if err != nil {
			log.Printf("error getting %s popular tracks: %s\n", c.Params("user"), err)
			return err
		}

		return r(c, user.Username, templates.UserTopTracks(prefs, user, p), templates.UserHeader(user))
	})

	app.Get("/:user/followers", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		user, err := sc.GetUser(cid, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (followers): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		p, err := user.GetFollowers(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s followers: %s\n", c.Params("user"), err)
			return err
		}

		return r(c, user.Username, templates.UserFollowers(prefs, user, p), templates.UserHeader(user))
	})

	app.Get("/:user/following", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		user, err := sc.GetUser(cid, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (following): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		p, err := user.GetFollowing(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s following: %s\n", c.Params("user"), err)
			return err
		}

		return r(c, user.Username, templates.UserFollowing(prefs, user, p), templates.UserHeader(user))
	})

	app.Get("/:user/:track", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		track, err := sc.GetTrack(cid, c.Params("user")+"/"+c.Params("track"))
		if err != nil {
			log.Printf("error getting %s from %s: %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}
		track.Postfix(prefs, true)

		displayErr := ""
		stream := ""
		audio := ""

		if *prefs.Player != cfg.NonePlayer {
			if *prefs.Player == cfg.HLSPlayer {
				var tr *sc.Transcoding
				tr, audio = track.Media.SelectCompatible(*prefs.HLSAudio, false)
				if tr == nil {
					err = sc.ErrIncompatibleStream
				} else {
					stream, err = tr.GetStream(cid, prefs, track.Authorization)
				}
			} else {
				_, audio = track.Media.SelectCompatible(*prefs.RestreamAudio, true)
				if audio == "" {
					err = sc.ErrIncompatibleStream
				}
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
			p, err := sc.GetPlaylist(cid, pl)
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
					nt, err := sc.GetTrackByID(cid, string(nextTrack.ID))
					if err != nil {
						return err
					}

					nt.Postfix(prefs, false)

					nextTrack = &nt
				}
			}
		}

		if *prefs.AutoplayNextRelatedTrack && nextTrack == nil && string(c.RequestCtx().QueryArgs().Peek("playRelated")) != "false" {
			rel, err := track.GetRelated(cid, prefs, "?limit=4")
			if err == nil && len(rel.Collection) != 0 {
				prev := c.RequestCtx().QueryArgs().Peek("prev")
				nextTrack = &track
				for i := len(rel.Collection) - 1; i >= 0 && (string(nextTrack.ID) == string(track.ID) || string(nextTrack.ID) == string(prev)); i-- {
					nextTrack = rel.Collection[i]
				}
			}
		}

		var comments *sc.Paginated[*sc.Comment]
		if q := c.Query("pagination"); q != "" {
			comments, err = track.GetComments(cid, prefs, q)
			if err != nil {
				log.Printf("failed to get %s from %s comments: %s\n", c.Params("track"), c.Params("user"), err)
				return err
			}
		}

		var downloadAudio *string
		if cfg.Restream {
			_, audio := track.Media.SelectCompatible(*prefs.DownloadAudio, true)
			downloadAudio = &audio
		}

		return r(c, track.Title+" by "+track.Author.Username, templates.Track(prefs, track, stream, displayErr, string(c.RequestCtx().QueryArgs().Peek("autoplay")) == "true", playlist, nextTrack, c.Query("volume"), mode, audio, downloadAudio, comments), templates.TrackHeader(prefs, track, true))
	})

	app.Get("/_/partials/comments/:id", func(c fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.ErrBadRequest
		}

		pagination := c.RequestCtx().QueryArgs().Peek("pagination")
		if len(pagination) == 0 {
			return fiber.ErrBadRequest
		}

		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		t := sc.Track{ID: json.Number(id)}
		comm, err := t.GetComments("", prefs, cfg.B2s(pagination))
		if err != nil {
			return err
		}

		if comm.Next != "" {
			c.Set("next", "?pagination="+url.QueryEscape(strings.Split(comm.Next, "/comments")[1]))
		} else {
			c.Set("next", "done")
		}

		return render(c, templates.Comments(comm))
	})

	app.Get("/:user", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		usr, err := sc.GetUser(cid, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s: %s\n", c.Params("user"), err)
			return err
		}
		usr.Postfix(prefs)

		p, err := usr.GetTracks(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s tracks: %s\n", c.Params("user"), err)
			return err
		}

		return r(c, usr.Username, templates.User(prefs, usr, p), templates.UserHeader(usr))
	})

	app.Get("/:user/sets/:playlist", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		playlist, err := sc.GetPlaylist(cid, c.Params("user")+"/sets/"+c.Params("playlist"))
		if err != nil {
			log.Printf("error getting %s playlist from %s: %s\n", c.Params("playlist"), c.Params("user"), err)
			return err
		}
		// Don't ask why
		playlist.Tracks = playlist.Postfix(prefs, true, true)

		p := c.Query("pagination")
		if p != "" {
			tracks, next, err := sc.GetNextMissingTracks(cid, p)
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

		return r(c, playlist.Title+" by "+playlist.Author.Username, templates.Playlist(prefs, playlist), templates.PlaylistHeader(playlist))
	})

	app.Get("/:user/_/related", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		user, err := sc.GetUser(cid, c.Params("user"))
		if err != nil {
			log.Printf("error getting %s (related): %s\n", c.Params("user"), err)
			return err
		}
		user.Postfix(prefs)

		rel, err := user.GetRelated(cid, prefs)
		if err != nil {
			log.Printf("error getting %s related users: %s\n", c.Params("user"), err)
			return err
		}

		return r(c, user.Username, templates.UserRelated(prefs, user, rel), templates.UserHeader(user))
	})

	// I'd like to make this "related" but keeping it "recommended" to have the same url as soundcloud
	app.Get("/:user/:track/recommended", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		track, err := sc.GetTrack(cid, c.Params("user")+"/"+c.Params("track"))
		if err != nil {
			log.Printf("error getting %s from %s (related): %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}
		track.Postfix(prefs, true)

		rel, err := track.GetRelated(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s from %s related tracks: %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}

		return r(c, track.Title+" by "+track.Author.Username, templates.RelatedTracks(track, rel), templates.TrackHeader(prefs, track, false))
	})

	app.Get("/:user/:track/sets", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		track, err := sc.GetTrack(cid, c.Params("user")+"/"+c.Params("track"))
		if err != nil {
			log.Printf("error getting %s from %s (sets): %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}
		track.Postfix(prefs, true)

		p, err := track.GetPlaylists(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s from %s sets: %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}

		return r(c, track.Title+" by "+track.Author.Username, templates.TrackInPlaylists(track, p), templates.TrackHeader(prefs, track, false))
	})

	app.Get("/:user/:track/albums", func(c fiber.Ctx) error {
		prefs, err := preferences.Get(c)
		if err != nil {
			return err
		}

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		track, err := sc.GetTrack(cid, c.Params("user")+"/"+c.Params("track"))
		if err != nil {
			log.Printf("error getting %s from %s (albums): %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}
		track.Postfix(prefs, true)

		p, err := track.GetAlbums(cid, prefs, c.Query("pagination", "?limit=20"))
		if err != nil {
			log.Printf("error getting %s from %s albums: %s\n", c.Params("track"), c.Params("user"), err)
			return err
		}

		return r(c, track.Title+" by "+track.Author.Username, templates.TrackInAlbums(track, p), templates.TrackHeader(prefs, track, false))
	})

	// cute
	const art = `
            ⠀⠀⠀⠀⢀⡴⣆⠀⠀⠀⠀⠀⣠⡀⠀⠀⠀⠀⠀⠀⣼⣿⡗⠀⠀⠀⠀
            ⠀⠀⠀⣠⠟⠀⠘⠷⠶⠶⠶⠾⠉⢳⡄⠀⠀⠀⠀⠀⣧⣿⠀⠀⠀⠀⠀
            ⠀⠀⣰⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢻⣤⣤⣤⣤⣤⣿⢿⣄⠀⠀⠀⠀
  ___  ___  ⠀⠀⡇⠀⢀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣧⠀⠀⠀⠀⠀⠀⠙⣷⡴⠶⣦
 / __|/ __| ⠀⠀⢱⡀⠀⠉⠉⠀⠀⠀⠀⠛⠃⠀⢠⡟⠂⠀⠀⢀⣀⣠⣤⠿⠞⠛⠋
 \__ \ (__  ⣠⠾⠋⠙⣶⣤⣤⣤⣤⣤⣀⣠⣤⣾⣿⠴⠶⠚⠋⠉⠁⠀⠀⠀⠀⠀⠀
 |___/\___| ⠛⠒⠛⠉⠉⠀⠀⠀⣴⠟⣣⡴⠛⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
~~~~~~~~~~~~~~~~~~~~⠛⠛~~~~~~~~~~~~~~~~~~~~~~~~`
	fmt.Println(art)
	const sep = " :: "

	// maps in go are unordered..
	table := map[string]string{
		"Built from":     fmt.Sprintf("%s (%s)", cfg.Commit, cfg.Repo),
		"ProxyStreams":   boolean(cfg.ProxyStreams),
		"ProxyImages":    boolean(cfg.ProxyStreams),
		"Restream":       boolean(cfg.Restream),
		"GetWebProfiles": boolean(cfg.GetWebProfiles),
		"Listening on":   cfg.Addr,
	}
	if cfg.Addr[0] == ':' {
		table["Listening on"] = "127.0.0.1" + cfg.Addr
	}
	longest := ""
	for key := range table {
		if len(key) > len(longest) {
			longest = key
		}
	}
	longest += sep

	for _, key := range [...]string{"Built from", "ProxyStreams", "ProxyImages", "Restream", "GetWebProfiles", "Listening on"} {
		fmt.Print(key)
		fmt.Print(strings.Repeat(" ", len(longest)-len(key)-len(sep)) + sep)
		fmt.Println(table[key])
	}

	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
	if cfg.CodegenConfig {
		log.Println("Warning: you have CodegenConfig enabled, but the config was loaded dynamically.")
	}

	lc := fiber.ListenConfig{EnablePrefork: cfg.Prefork, DisableStartupMessage: true, ListenerNetwork: cfg.Network}
	if cfg.Network == "unix" {
		os.Remove(cfg.Addr)
		lc.BeforeServeFunc = func(*fiber.App) error {
			err := os.Chmod(cfg.Addr, cfg.UnixSocketPerms)
			if err != nil {
				log.Println("failed to chmod socket:", err)
			}

			return nil
		}
	}
	log.Fatal(app.Listen(cfg.Addr, lc))
}
