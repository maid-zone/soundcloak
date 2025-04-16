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
		var quality *string
		if isDownload {
			quality = p.DownloadAudio
		} else {
			quality = p.RestreamAudio
		}

		tr, audio := t.Media.SelectCompatible(*quality, true)
		if tr == nil {
			return fiber.ErrExpectationFailed
		}

		u, err := tr.GetStream(cid, p, t.Authorization)
		if err != nil {
			return err
		}

		c.Request().Header.SetContentType(tr.Format.MimeType)
		c.Set("Cache-Control", cfg.RestreamCacheControl)

		r := acquireReader()
		if isDownload {
			if t.Artwork != "" {
				t.Artwork = strings.Replace(t.Artwork, "t500x500", "original", 1)
			}

			switch audio {
			case cfg.AudioMP3:
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
					r.req.Header.Del("Accept-Encoding")

					err := sc.DoWithRetry(misc.ImageClient, r.req, r.resp)
					if err != nil {
						return err
					}

					tag.AddAttachedPicture(id3v2.PictureFrame{MimeType: cfg.B2s(r.req.Header.ContentType()), Picture: r.req.Body(), PictureType: id3v2.PTFrontCover, Encoding: id3v2.EncodingUTF8})
					r.req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
				}

				var col collector
				tag.WriteTo(&col)
				r.leftover = col.data

				return c.SendStream(r)
			case cfg.AudioOpus:
				err := r.Setup(u, false, nil)
				if err != nil {
					return err
				}

				r.req.SetRequestURIBytes(r.parts[0])
				err = sc.DoWithRetry(r.client, r.req, r.resp)
				if err != nil {
					return err
				}

				r.index++
				tag, err := oggmeta.ReadOGG(bytes.NewReader(r.resp.Body()))
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
					r.req.Header.Del("Accept-Encoding")

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
					r.req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
				}

				var col collector
				tag.Save(&col)
				r.leftover = col.data

				return c.SendStream(r)
			case cfg.AudioAAC:
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
					r.req.Header.Del("Accept-Encoding")

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
					r.req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
				}

				var col collector
				tag.Save(&col)
				r.leftover = col.data

				return c.SendStream(r)
			}
		}

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
