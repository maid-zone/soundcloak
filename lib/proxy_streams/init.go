package proxystreams

import (
	"bytes"
	"strings"
	"time"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"git.maid.zone/stuff/soundcloak/lib/preferences"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

var newline = []byte{'\n'}
var redirect_parts = []byte("?redirect=true")

var hls_aac_streaming_httpc *fasthttp.HostClient

// https://playback.media-streaming.soundcloud.cloud/5qxs2zYpVFDq/aac_160k/67bfdb94-662d-40e3-8297-08ae5a507f61/playlist.m3u8?expires=...&Policy=...&Key-Pair-Id=...
// =>
// /playlist.m3u8
func lastSegment(p []byte) []byte {
	i := bytes.LastIndexByte(p, '/')
	if i == -1 {
		i = 0
	}
	i2 := bytes.IndexByte(p[i:], '?')
	if i2 == -1 {
		i2 = len(p)
	}
	return p[i : i+i2]
}

func Load(app *fiber.App) {
	hls_aac_streaming_httpc = &fasthttp.HostClient{
		Addr:                cfg.HLSAACCDN + ":443",
		IsTLS:               true,
		MaxIdleConnDuration: cfg.MaxIdleConnDuration,
		StreamResponseBody:  true,
		MaxResponseBodySize: 1,
		DialDualStack:       cfg.DialDualStack,
	}

	// the new proxy :)
	// it now caches the retrieved streams, and also automatically renews them,
	// since they expire after track.Duration + 105s
	// this also hides away the long ephemeral URLs behind the server, like restream does
	// but still gives you an option to get redirected there (for example if you got ProxyStreams disabled)
	app.Get("/_/api/hls/*", func(c fiber.Ctx) error {
		s := c.Path()[len("/_/api/hls/"):]
		t, err := sc.GetTrack(s)
		if err != nil {
			return err
		}

		var forcedQuality = c.RequestCtx().QueryArgs().Peek("audio")
		var quality string
		if len(forcedQuality) != 0 {
			quality = cfg.B2s(forcedQuality)
		} else {
			p, err := preferences.Get(c)
			if err != nil {
				return err
			}
			quality = *p.HLSAudio
		}

		tr, audio := t.Media.SelectCompatibleHLS(quality)
		if tr == nil {
			return fiber.ErrExpectationFailed
		}

		s += "/" + tr.Quality + "/" + tr.Preset + "/" + string(tr.Format.Protocol)
		cl, err := tr.GetStream(s, t)
		if err != nil {
			return err
		}

		req := c.Request()
		if string(req.URI().QueryArgs().Peek("redirect")) == "true" {
			c.Response().SetStatusCode(fiber.StatusFound)
			c.Response().Header.SetBytesV("Location", cl.Value.Playlist.FullURI())
			return nil
		}
		var params []byte
		if !cfg.ProxyStreams || string(req.URI().QueryArgs().Peek("redirect_parts")) == "true" {
			params = redirect_parts
		}
		req.Reset()
		req.SetURI(cl.Value.Playlist)
		req.Header.SetUserAgent(cfg.UserAgent)

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)

		httpc := misc.HlsClient
		if audio == cfg.AudioAAC {
			httpc = misc.HlsAacClient
		}
		err = sc.DoWithRetry(httpc, req, resp)
		if err != nil {
			return err
		}

		s2 := s[:len(s)-len("/hls")]
		ln := 0
		if cl.Value.Base != nil {
			ln = len(cl.Value.Base.Scheme()) + len("://") + len(cl.Value.Base.Host()) + len(cl.Value.Base.Path())
		}
		r := c.Response()
		if httpc == misc.HlsClient {
			for l := range bytes.SplitSeq(resp.Body(), newline) {
				if len(l) == 0 {
					r.AppendBody(newline)
					continue
				}

				if l[0] == '#' {
					r.AppendBody(l)
					r.AppendBody(newline)
					continue
				}

				if cl.Value.FreshBase {
					if cl.Value.Base == nil {
						cl.Value.Base = fasthttp.AcquireURI()
					}
					if cl.Value.Base.Parse(nil, l) == nil {
						// /media/159660/0/31762/KwmxqcPQKkEL.128.mp3
						// ^^^^^ const | ^ part | ^^^^^^ const, same on playlist
						p := cl.Value.Base.Path()
						i := bytes.IndexByte(p[len("/media/"):], '/')
						if i != -1 {
							// only get first const
							// /media/159660/
							cl.Value.Base.SetPathBytes(p[:len("/media/")+i+1])
							cl.Value.FreshBase = false
							ln = len(cl.Value.Base.Scheme()) + len("://") + len(cl.Value.Base.Host()) + len(cl.Value.Base.Path())
							sc.StreamCacheMut.Lock()
							sc.StreamCache[s] = cl
							sc.StreamCacheMut.Unlock()
						}
					}
				}

				// https://cf-hls-media.sndcdn.com/media/159660/0/31762/KwmxqcPQKkEL.128.mp3?Policy=...&Key-Pair-Id=...
				l = l[ln-1:]
				// /0/31762/KwmxqcPQKkEL.128.mp3?Policy=...&Key-Pair-Id=...
				i := bytes.LastIndexByte(l, '/')
				if i == -1 {
					continue
				}
				l = l[:i]
				// /0/31762

				r.AppendBodyString("/_/proxy/hls/")
				r.AppendBodyString(s2)
				r.AppendBody(l)
				r.AppendBody(params)
				r.AppendBody(newline)
			}
		} else {
			for l := range bytes.SplitSeq(resp.Body(), newline) {
				if len(l) == 0 {
					r.AppendBody(newline)
					continue
				}

				if l[0] == '#' {
					// #EXT-X-MAP:URI="..."
					const x = `#EXT-X-MAP:URI="`
					if len(l) > len(x) && string(l[:len(x)]) == x {
						r.AppendBodyString(`#EXT-X-MAP:URI="/_/proxy/hls/`)
						r.AppendBodyString(s2)
						r.AppendBody(lastSegment(l[len(x) : len(l)-1]))
						r.AppendBody(params)
						r.AppendBodyString(`"`)
					} else {
						r.AppendBody(l)
					}
					r.AppendBody(newline)
					continue
				}

				if cl.Value.FreshBase {
					if cl.Value.Base == nil {
						cl.Value.Base = fasthttp.AcquireURI()
					}
					if cl.Value.Base.Parse(nil, l) == nil {
						p := cl.Value.Base.Path()
						cl.Value.Base.SetPathBytes(p[:len(p)-len(cl.Value.Base.LastPathSegment())])
						cl.Value.FreshBase = false
						sc.StreamCacheMut.Lock()
						sc.StreamCache[s] = cl
						sc.StreamCacheMut.Unlock()
					}
				}

				r.AppendBodyString("/_/proxy/hls/")
				r.AppendBodyString(s2)
				r.AppendBody(lastSegment(l))
				r.AppendBody(params)
				r.AppendBody(newline)
			}
		}
		r.Header.SetContentType("audio/mpegurl")
		return nil
	})

	app.Get("/_/proxy/hls/:author/:track/:quality/:preset/*", func(c fiber.Ctx) error {
		req := c.Request()
		p := c.Params("preset")
		aac := strings.HasPrefix(p, "aac_")
		s := c.Params("author") + "/" + c.Params("track") + "/" + c.Params("quality") + "/" + p + "/hls"
		_s := c.Request().URI().Path()
		fp := string(_s[len("/_/proxy/hls/")+len(s)-len("/hls")+1:])
		//fmt.Println(s, string(_s), fp)
		sc.StreamCacheMut.RLock()
		cl, ok := sc.StreamCache[s]
		sc.StreamCacheMut.RUnlock()
		if !ok || cl.Expires.Before(time.Now()) {
			t, err := sc.GetTrack(c.Params("author") + "/" + c.Params("track"))
			if err != nil {
				return err
			}

			q := c.Params("quality")
			var transcoding *sc.Transcoding
			for _, tr := range t.Media.Transcodings {
				if tr.Format.Protocol == sc.ProtocolHLS && tr.Quality == q && tr.Preset == p {
					transcoding = &tr
					break
				}
			}
			if transcoding == nil {
				return fiber.ErrExpectationFailed
			}
			cl, err = transcoding.GetStream(s, t)
			if err != nil {
				return err
			}
		}

		var httpc *fasthttp.HostClient
		if aac {
			httpc = misc.HlsAacClient
		} else {
			httpc = misc.HlsClient
		}

		redir := !cfg.ProxyStreams || string(req.URI().QueryArgs().Peek("redirect")) == "true"
		req.Reset()
		req.Header.SetUserAgent(cfg.UserAgent)
		resp := c.Response()
		if cl.Value.FreshBase || cl.Value.Base == nil {
			if cl.Value.Base == nil {
				cl.Value.Base = fasthttp.AcquireURI()
			}
			req.SetURI(cl.Value.Playlist)
			err := sc.DoWithRetry(httpc, req, resp)
			if err != nil {
				return err
			}
			for l := range bytes.SplitSeq(resp.Body(), newline) {
				if len(l) == 0 || l[0] == '#' {
					continue
				}

				if cl.Value.Base.Parse(nil, l) == nil {
					p := cl.Value.Base.Path()
					if aac {
						cl.Value.Base.SetPathBytes(p[:len(p)-len(cl.Value.Base.LastPathSegment())])
					} else {
						i := bytes.IndexByte(p[len("/media/"):], '/')
						if i != -1 {
							// only get first const
							// /media/159660/
							cl.Value.Base.SetPathBytes(p[:len("/media/")+i+1])
						}
					}
					cl.Value.FreshBase = false
					sc.StreamCacheMut.Lock()
					sc.StreamCache[s] = cl
					sc.StreamCacheMut.Unlock()
					break
				}
			}
		}

		req.SetURI(cl.Value.Base)
		if aac {
			req.URI().SetPathBytes(append(req.URI().Path(), fp...))
		} else {
			p := cl.Value.Playlist.Path()
			req.URI().SetPathBytes(append(append(req.URI().Path(), fp...), p[len("/playlist"):len(p)-len("/playlist.m3u8")]...))
		}

		if redir {
			resp.SetStatusCode(fiber.StatusFound)
			resp.Header.SetBytesV("Location", req.URI().FullURI())
			return nil
		}

		if aac {
			httpc = hls_aac_streaming_httpc
		} else {
			httpc = misc.HlsStreamingOnlyClient
		}
		return sc.DoWithRetry(httpc, req, resp)
	})

	app.Get("/_/api/progressive/*", func(c fiber.Ctx) error {
		s := c.Path()[len("/_/api/progressive/"):]
		t, err := sc.GetTrack(s)
		if err != nil {
			return err
		}

		tr := t.Media.SelectCompatibleProgressive()
		if tr == nil {
			return fiber.ErrExpectationFailed
		}

		s += "/" + tr.Quality + "/" + tr.Preset + "/" + string(tr.Format.Protocol)
		cl, err := tr.GetStream(s, t)
		if err != nil {
			return err
		}

		req := c.Request()
		resp := c.Response()
		if !cfg.ProxyStreams || string(req.URI().QueryArgs().Peek("redirect")) == "true" {
			resp.SetStatusCode(fiber.StatusFound)
			resp.Header.SetBytesV("Location", cl.Value.Playlist.FullURI())
			return nil
		}
		// rng := req.Header.Peek("Range")
		req.Reset()
		// if len(rng) != 0 {
		// 	req.Header.SetBytesV("Range", rng)
		// }

		req.SetURI(cl.Value.Playlist)
		req.Header.SetUserAgent(cfg.UserAgent)
		err = sc.DoWithRetry(misc.HlsStreamingOnlyClient, req, resp)
		resp.Header.Set("Content-Disposition", `attachment; filename="`+t.Permalink+`.mp3"`)
		resp.Header.Del("Accept-Ranges")
		return err
	})

	// deprecated kind of, will remove at some point
	if cfg.ProxyStreams {
		legacy(app.Group("/_/proxy/streams"))
	}
}
