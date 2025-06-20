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
	"github.com/gcottom/oggmeta"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

type collector struct {
	data []byte
}

func (c *collector) Write(data []byte) (n int, err error) {
	c.data = append(c.data, data...)
	return len(data), nil
}

func Load(r *fiber.App) {
	r.Get("/_/restream/:author/:track", func(c fiber.Ctx) error {
		p, err := preferences.Get(c)
		if err != nil {
			return err
		}
		p.ProxyImages = &cfg.False
		p.ProxyStreams = &cfg.False

		cid, err := sc.GetClientID()
		if err != nil {
			return err
		}

		t, err := sc.GetTrack(cid, c.Params("author")+"/"+c.Params("track"))
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

		tr, audio := t.Media.SelectCompatible(quality, true)
		if tr == nil {
			return fiber.ErrExpectationFailed
		}

		u, err := tr.GetStream(cid, p, t.Authorization)
		if err != nil {
			return err
		}

		c.Request().Header.SetContentType(tr.Format.MimeType)
		c.Set("Cache-Control", cfg.RestreamCacheControl)

		if isDownload {
			if t.Artwork != "" {
				t.Artwork = strings.Replace(t.Artwork, "t500x500", "original", 1)
			}

			switch audio {
			case cfg.AudioMP3:
				r := acquireReader()
				err := r.Setup(u, false, nil)
				if err != nil {
					return err
				}

				tag := id3v2.NewEmptyTag()

				tag.SetArtist(t.Author.Username)
				if t.Genre != "" {
					tag.SetGenre(t.Genre)
				}

				tag.SetTitle(t.Title)

				if t.Artwork != "" {
					r.req.SetRequestURI(t.Artwork)

					err := sc.DoWithRetry(misc.ImageClient, r.req, r.resp)
					if err != nil {
						return err
					}

					tag.AddAttachedPicture(id3v2.PictureFrame{MimeType: cfg.B2s(r.req.Header.ContentType()), Picture: r.req.Body(), PictureType: id3v2.PTFrontCover, Encoding: id3v2.EncodingUTF8})
				}

				var col collector
				tag.WriteTo(&col)
				r.leftover = col.data

				c.Response().Header.SetContentType("audio/mpeg")
				return c.SendStream(r)
			case cfg.AudioOpus: // might try to fuck around with metadata injection. Dynamically injecting metadata for opus wasn't really good idea as it breaks some things :P
				req := fasthttp.AcquireRequest()
				resp := fasthttp.AcquireResponse()

				req.SetRequestURI(u)
				req.Header.SetUserAgent(cfg.UserAgent)

				err := sc.DoWithRetry(misc.HlsClient, req, resp)
				if err != nil {
					return err
				}

				data := resp.Body()

				res := make([]byte, 0, 1024*1024*1)
				for _, s := range bytes.Split(data, []byte{'\n'}) {
					if len(s) == 0 || s[0] == '#' {
						continue
					}

					req.SetRequestURIBytes(s)
					err := sc.DoWithRetry(misc.HlsClient, req, resp)
					if err != nil {
						return err
					}

					data = resp.Body()

					res = append(res, data...)
				}

				tag, err := oggmeta.ReadOGG(bytes.NewReader(res))
				if err != nil {
					return err
				}

				tag.SetArtist(t.Author.Username)
				if t.Genre != "" {
					tag.SetGenre(t.Genre)
				}

				tag.SetTitle(t.Title)

				if t.Artwork != "" {
					req.SetRequestURI(t.Artwork)

					err := sc.DoWithRetry(misc.ImageClient, req, resp)
					if err != nil {
						return err
					}

					parsed, _, err := image.Decode(resp.BodyStream())
					resp.CloseBodyStream()
					if err != nil {
						return err
					}

					tag.SetCoverArt(&parsed)
				}

				c.Response().Header.SetContentType(`audio/ogg; codecs="opus"`)
				return tag.Save(c.Response().BodyWriter())
			case cfg.AudioAAC:
				r := acquireReader()
				err := r.Setup(u, true, &t.Duration)
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

					err := sc.DoWithRetry(misc.ImageClient, r.req, r.resp)
					if err != nil {
						return err
					}

					parsed, _, err := image.Decode(r.resp.BodyStream())
					r.resp.CloseBodyStream()
					if err != nil {
						return err
					}

					tag.SetCoverArt(&parsed)
				}

				var col collector
				tag.Save(&col)
				fixDuration(col.data, &t.Duration)
				r.leftover = col.data

				c.Response().Header.SetContentType("audio/mp4")
				return c.SendStream(r)
			}
		}

		r := acquireReader()
		if audio == cfg.AudioAAC {
			err = r.Setup(u, true, &t.Duration)
		} else {
			err = r.Setup(u, false, nil)
		}

		if err != nil {
			return err
		}

		return c.SendStream(r)
	})
}
