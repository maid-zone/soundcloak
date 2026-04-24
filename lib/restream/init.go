package restream

import (
	"bytes"
	"image"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"git.maid.zone/stuff/soundcloak/lib/preferences"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"github.com/bogem/id3v2/v2"
	"github.com/gcottom/mp4meta"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

var image_httpc *fasthttp.HostClient

func Load(r *fiber.App) {

	image_httpc = &fasthttp.HostClient{
		Addr:                cfg.ImageCDN + ":443",
		IsTLS:               true,
		MaxIdleConnDuration: cfg.MaxIdleConnDuration,
		DialDualStack:       cfg.DialDualStack,
	}

	// glue glue glue
	r.Use("/_/restream", func(c fiber.Ctx) error {
		uri := c.Request().URI()
		p := uri.RequestURI()
		b := make([]byte, 0, len(p)+len("/api"))
		b = append(b, "/_/api/restream/"...)
		b = append(b, p[len("/_/restream/"):]...)
		c.Response().SetStatusCode(fiber.StatusMovedPermanently)
		c.Response().Header.SetBytesV("Location", b)
		return nil
	})

	r.Get("/_/api/restream/:author/:track", func(c fiber.Ctx) error {
		p, err := preferences.Get(c)
		if err != nil {
			return err
		}
		p.ProxyImages = &cfg.False
		p.ProxyStreams = &cfg.False

		t, err := sc.GetTrack(c.Params("author") + "/" + c.Params("track"))
		if err != nil {
			return err
		}

		var isDownload = string(c.RequestCtx().QueryArgs().Peek("metadata")) == "true"
		var forcedQuality = c.RequestCtx().QueryArgs().Peek("audio")
		var quality string
		if len(forcedQuality) != 0 {
			quality = cfg.B2s(forcedQuality)
		} else {
			if isDownload {
				quality = *p.DownloadAudio
			} else {
				quality = *p.RestreamAudio
			}
		}

		if isDownload {
			var s []byte
			if s = c.RequestCtx().QueryArgs().Peek("title"); len(s) > 0 {
				t.Title = cfg.B2s(s)
			}
			if s = c.RequestCtx().QueryArgs().Peek("genre"); len(s) > 0 {
				t.Genre = cfg.B2s(s)
			}
			if s = c.RequestCtx().QueryArgs().Peek("author"); len(s) > 0 {
				t.Author.Username = cfg.B2s(s)
			}
		}

		tr, audio := t.Media.SelectCompatibleRestream(quality)
		if tr == nil {
			return fiber.ErrExpectationFailed
		}

		u, err := tr.GetStream("", t)
		if err != nil {
			return err
		}

		//req := c.Request()
		//rng := req.Header.Peek("Range")
		resp := c.Response()
		resp.Header.SetContentType(tr.Format.MimeType)
		resp.Header.Set("Cache-Control", cfg.RestreamCacheControl)
		resp.Header.Set("Content-Disposition", `attachment; filename="`+t.Permalink+"."+sc.ToExt(audio)+`"`)

		if isDownload {
			if t.Artwork != "" {
				t.Artwork = strings.Replace(t.Artwork, "t500x500", "original", 1)
			}

			switch audio {
			case cfg.AudioMP3:
				req := fasthttp.AcquireRequest()
				resp := fasthttp.AcquireResponse()

				req.Header.SetUserAgent(cfg.UserAgent)

				tag := id3v2.NewEmptyTag()

				tag.SetArtist(t.Author.Username)
				if t.Genre != "" {
					tag.SetGenre(t.Genre)
				}

				tag.SetTitle(t.Title)

				if t.Artwork != "" {
					req.SetRequestURI(t.Artwork)

					err := sc.DoWithRetry(image_httpc, req, resp)
					if err == nil && resp.StatusCode() == 200 {
						//fmt.Println(string(resp.Header.ContentType()), string(resp.Header.Peek("Content-Encoding")), len(resp.Body()))
						tag.AddAttachedPicture(id3v2.PictureFrame{MimeType: cfg.B2s(resp.Header.ContentType()), Picture: resp.Body(), PictureType: id3v2.PTFrontCover, Encoding: id3v2.EncodingUTF8})
					}
				}

				if tr.Format.Protocol == sc.ProtocolProgressive {
					r := acquireInjector()
					tag.WriteTo(r) // write out tag first because the buffers will be overwritten if you reuse the req/resp

					req.SetURI(u.Value.Playlist)
					// enforce streaming here!!
					err := sc.DoWithRetry(misc.HlsStreamingOnlyClient, req, resp)
					if err != nil {
						return err
					}

					r.reader = resp.BodyStream()
					r.resp = resp
					return c.SendStream(r)
				}

				r := acquireReader()
				tag.WriteTo(r)
				r.req = req
				r.resp = resp
				err := r.Setup(u.Value.Playlist, false, nil)
				if err != nil {
					return err
				}

				return c.SendStream(r)
			case cfg.AudioAAC:
				r := acquireReader()
				err := r.Setup(u.Value.Playlist, true, nil)
				if err != nil {
					return err
				}

				r.req.SetRequestURIBytes(r.parts[0])
				err = sc.DoWithRetry(r.client, r.req, r.resp)
				if err != nil {
					return err
				}

				r.index++
				tag, err := mp4meta.ReadMP4(bytes.NewReader(r.resp.Body()))
				if err != nil {
					return err
				}

				tag.SetArtist(t.Author.Username)
				if t.Genre != "" {
					tag.SetGenre(t.Genre)
				}

				tag.SetTitle(t.Title)

				if t.Artwork != "" {
					r.req.SetRequestURI(t.Artwork)

					err := sc.DoWithRetry(misc.ImageStreamingOnlyClient, r.req, r.resp)
					if err == nil && r.resp.StatusCode() == 200 {
						parsed, _, err := image.Decode(r.resp.BodyStream())
						r.resp.CloseBodyStream()
						if err == nil {
							tag.SetCoverArt(&parsed)
						}
					}
				}

				tag.Save(r)
				fixDuration(r.leftover, &t.Duration)

				return c.SendStream(r)
			}
		}

		// just the audio file itself, means less processing overhead for us :)
		if tr.Format.Protocol == sc.ProtocolProgressive {
			misc.Log("use progressive")
			req := fasthttp.AcquireRequest()
			defer fasthttp.ReleaseRequest(req)

			// if len(rng) != 0 {
			// 	req.Header.SetBytesV("Range", rng)
			// }
			req.SetURI(u.Value.Playlist)
			req.Header.SetUserAgent(cfg.UserAgent)

			err = sc.DoWithRetry(misc.HlsStreamingOnlyClient, req, resp)
			resp.Header.Set("Content-Disposition", `attachment; filename="`+t.Permalink+`.mp3"`)
			resp.Header.Del("Accept-Ranges")
			return err
		}

		r := acquireReader()
		if audio == cfg.AudioAAC {
			err = r.Setup(u.Value.Playlist, true, &t.Duration)
		} else {
			err = r.Setup(u.Value.Playlist, false, nil)
		}

		if err != nil {
			return err
		}

		return c.SendStream(r)
	})
}
