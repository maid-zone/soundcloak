package restream

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
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
var crcTable [256]uint32

func Load(r *fiber.App) {
	for i := range crcTable {
		r := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if r&0x80000000 != 0 {
				r = (r << 1) ^ 0x04c11db7
			} else {
				r <<= 1
			}
		}
		crcTable[i] = r
	}

	image_httpc = &fasthttp.HostClient{
		Addr:                cfg.ImageCDN + ":443",
		IsTLS:               true,
		MaxIdleConnDuration: cfg.MaxIdleConnDuration,
		DialDualStack:       cfg.DialDualStack,
	}

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

		tr, audio := t.Media.SelectCompatible(quality, true, true)
		if tr == nil {
			return fiber.ErrExpectationFailed
		}

		u, err := tr.GetStream(cid, p, t.Authorization)
		if err != nil {
			return err
		}

		c.Response().Header.SetContentType(tr.Format.MimeType)
		c.Set("Cache-Control", cfg.RestreamCacheControl)
		c.Set("Content-Disposition", `attachment; filename="`+t.Permalink+"."+sc.ToExt(audio)+`"`)

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

					req.SetRequestURI(u)
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
				err := r.Setup(u, false, nil)
				if err != nil {
					return err
				}

				return c.SendStream(r)
			case cfg.AudioOpus:
				r := acquireReader()
				err := r.Setup(u, false, nil)
				if err != nil {
					return err
				}

				// insane tech im crine brah
				const opustags_prelude = "OpusTags\x04\x00\x00\x00maid" // "OpusTags", uint32 for length, then <length> bytes for vendor string
				const artist = "ARTIST="
				const title = "TITLE="
				const genre = "GENRE="
				// METADATA_BLOCK_PICTURE comes from the FLAC format (https://www.rfc-editor.org/rfc/rfc9639.html#section-8.8), but base64 encoded
				const picture = "METADATA_BLOCK_PICTURE="
				// we need to put actual content somewhere else to segment it properly

				var num byte = 2
				ln := len(opustags_prelude) + // opustags hdr
					4 + // number of fields
					4 + len(artist) + len(t.Author.Username) + // ARTIST=...
					4 + len(title) + len(t.Title) // TITLE=...

				if t.Genre != "" {
					ln += 4 + len(genre) + len(t.Genre) // GENRE=...
					num++
				}

				var pic []byte
				var newLen int
				if t.Artwork != "" {
					r.req.SetRequestURI(t.Artwork)

					err := sc.DoWithRetry(image_httpc, r.req, r.resp)
					if err == nil && r.resp.StatusCode() == 200 {
						const emptyshits2 = "\x00\x00\x00\x00" + // desc length
							"\x00\x00\x00\x00" + // width
							"\x00\x00\x00\x00" + // height
							"\x00\x00\x00\x00" + // color depth
							"\x00\x00\x00\x00" // indexed color count
						pic = make([]byte, 0, 4+ // picture type
							4+ // mime len
							len(r.resp.Header.ContentType())+ // mime
							len(emptyshits2)+ // blah blah look above
							4+ // body len
							len(r.resp.Body()))
						pic = append(pic, 0, 0, 0, 3) // picture type (3, Front cover)
						pic = binary.BigEndian.AppendUint32(pic, uint32(len(r.resp.Header.ContentType())))
						pic = append(pic, r.resp.Header.ContentType()...)
						pic = pic[:len(pic)+len(emptyshits2)]
						pic = binary.BigEndian.AppendUint32(pic, uint32(len(r.resp.Body())))
						pic = append(pic, r.resp.Body()...)

						newLen = base64.StdEncoding.EncodedLen(len(pic))
						ln += 4 + len(picture) + newLen // METADATA_BLOCK_PICTURE=...
						num++
					}
				}

				leftover := make([]byte, 0, ln)
				misc.Log("alloc leftover", ln)

				// here come the metadata
				leftover = append(leftover, opustags_prelude...)
				// number of fields
				leftover = append(leftover, num, 0, 0, 0)
				// each field in the format of SOME_KEY=SOME_VALUE
				// same approach here, first the field length, then the field itself
				leftover = binary.LittleEndian.AppendUint32(leftover, uint32(len(artist)+len(t.Author.Username)))
				leftover = append(leftover, artist...)
				leftover = append(leftover, t.Author.Username...)

				leftover = binary.LittleEndian.AppendUint32(leftover, uint32(len(title)+len(t.Title)))
				leftover = append(leftover, title...)
				leftover = append(leftover, t.Title...)
				if t.Genre != "" {
					leftover = binary.LittleEndian.AppendUint32(leftover, uint32(len(genre)+len(t.Genre)))
					leftover = append(leftover, genre...)
					leftover = append(leftover, t.Genre...)
				}

				if pic != nil {
					leftover = binary.LittleEndian.AppendUint32(leftover, uint32(len(picture)+newLen))
					leftover = append(leftover, picture...)
					base64.StdEncoding.Encode(leftover[len(leftover):len(leftover)+newLen], pic)
					leftover = leftover[:len(leftover)+newLen]
				}
				misc.Log("ended leftover", cap(leftover))

				// now its safe to fuck shit up
				r.req.SetRequestURIBytes(r.parts[0])
				err = sc.DoWithRetry(r.client, r.req, r.resp)
				if err != nil {
					return err
				}

				r.index++
				res := r.resp.Body()
				const until_hdr = len("OggS") + 1 /* ver */ + 0
				const until_seq = until_hdr + 1 /* hdr */ + 8 /* granule */ + 4 /* bitstream */ + 0
				const until_checksum = until_seq + 4 /* seq */ + 0
				const until_segments = until_checksum + 4 /* checksum */ + 1 /* segments num */ + 0

				// this expects first page to only have 1 segment
				second_page := until_segments + 1 + int(res[until_segments])

				const (
					Continuation byte = 0x01
					BOS          byte = 0x02
					EOS          byte = 0x04
				)
				const max_possible_page = 255 * 255     // 255 segments, each can have 255 bytes
				if len(leftover) <= max_possible_page { // happy path :) it all fits in one page
					// lets segment it using ceil division
					segments_num := (len(leftover) + 254) / 255
					// allocate
					r.leftover = make([]byte, second_page, second_page+until_segments+segments_num+len(leftover)+len(res)-(second_page+until_segments+int(res[second_page+until_segments])))
					misc.Log("alloc r.leftover", cap(r.leftover))
					copy(r.leftover, res[:second_page])
					r.leftover = append(r.leftover, res[second_page:second_page+until_checksum]...)
					r.leftover = append(r.leftover, 0, 0, 0, 0) // checksum, to be filled in

					r.leftover = append(r.leftover, byte(segments_num))

					r.leftover = r.leftover[:len(r.leftover)+segments_num]
					for i := range segments_num - 1 {
						r.leftover[len(r.leftover)-i-2] = 255
					}

					if n := byte(len(leftover) % 255); n != 0 {
						r.leftover[len(r.leftover)-1] = n
					} else {
						r.leftover[len(r.leftover)-1] = 255
					}

					r.leftover = append(r.leftover, leftover...)

					// checksum is calculated for entire page including header
					var crc uint32
					for _, b := range r.leftover[second_page:] {
						crc = (crc << 8) ^ crcTable[(crc>>24)^uint32(b)]
					}

					binary.LittleEndian.PutUint32(r.leftover[second_page+until_checksum:], crc)
				} else { // sad path :(
					pages_num := (len(leftover) + max_possible_page - 1) / max_possible_page

					// allocate exactly as much as we need
					r.leftover = make([]byte, second_page,
						second_page+pages_num*until_segments+ // headers
							((len(leftover)+254)/255)+ // needed segments
							len(leftover)+ // contents
							len(res)-(second_page+until_segments+int(res[second_page+until_segments])), // the rest
					)
					copy(r.leftover, res[:second_page])
					for i := range pages_num {
						ptr := len(r.leftover)
						r.leftover = append(r.leftover, res[second_page:second_page+until_checksum]...)
						binary.LittleEndian.PutUint32(r.leftover[ptr+until_seq:], uint32(i))
						r.leftover = append(r.leftover, 0, 0, 0, 0) // checksum, to be filled in

						var segments_num int
						var sl []byte
						if i+1 == pages_num {
							r.leftover[ptr+until_hdr] = EOS
							sl = leftover[i*max_possible_page:]
							segments_num = (len(sl) + 254) / 255
						} else {
							if i != 0 {
								r.leftover[ptr+until_hdr] = Continuation
							} else {
								r.leftover[ptr+until_hdr] = BOS
							}
							segments_num = 255
							sl = leftover[i*max_possible_page : (i+1)*max_possible_page]
						}
						r.leftover = append(r.leftover, byte(segments_num))

						r.leftover = r.leftover[:len(r.leftover)+segments_num]
						for i := range segments_num - 1 {
							r.leftover[len(r.leftover)-i-2] = 255
						}

						if n := byte(len(sl) % 255); n != 0 {
							r.leftover[len(r.leftover)-1] = n
						} else {
							r.leftover[len(r.leftover)-1] = 255
						}

						r.leftover = append(r.leftover, sl...)

						// checksum is calculated for entire page including header
						var crc uint32
						for _, b := range r.leftover[ptr:] {
							crc = (crc << 8) ^ crcTable[(crc>>24)^uint32(b)]
						}

						binary.LittleEndian.PutUint32(r.leftover[ptr+until_checksum:], crc)
					}
				}
				// dump the rest after original 2nd page
				r.leftover = append(r.leftover, res[second_page+until_segments+int(res[second_page+until_segments]):]...)
				misc.Log("ended r.leftover", cap(r.leftover))
				return c.SendStream(r)
			case cfg.AudioAAC:
				r := acquireReader()
				err := r.Setup(u, true, nil)
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

			resp := fasthttp.AcquireResponse()

			req.SetRequestURI(u)
			req.Header.SetUserAgent(cfg.UserAgent)

			// enforce streaming here!!
			err := sc.DoWithRetry(misc.HlsStreamingOnlyClient, req, resp)
			if err != nil {
				return err
			}

			r := misc.AcquireProxyReader()
			r.Reader = resp.BodyStream()
			r.Resp = resp
			return c.SendStream(r)
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
