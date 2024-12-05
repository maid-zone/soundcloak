package proxyimages

import (
	"bytes"

	"github.com/gofiber/fiber/v2"
	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/maid-zone/soundcloak/lib/sc"
	"github.com/valyala/fasthttp"
)

func Load(r fiber.Router) {
	r.Get("/_/proxy/images", func(c *fiber.Ctx) error {
		url := c.Query("url")
		if url == "" {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, []byte(url))
		if err != nil {
			return err
		}

		if !bytes.HasSuffix(parsed.Host(), []byte(".sndcdn.com")) {
			return fiber.ErrBadRequest
		}

		parsed.SetHost(cfg.ImageCDN)

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.Set("User-Agent", cfg.UserAgent)
		//req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd") images not big enough to be compressed

		resp := fasthttp.AcquireResponse()
		//defer fasthttp.ReleaseResponse(resp) moved to proxyreader!!!

		err = sc.DoWithRetry(sc.ImageClient, req, resp)
		if err != nil {
			return err
		}

		c.Set("Content-Type", "image/jpeg")
		c.Set("Cache-Control", cfg.ImageCacheControl)
		//return c.Send(resp.Body())
		pr := cfg.AcquireProxyReader()
		pr.Reader = resp.BodyStream()
		pr.Resp = resp
		return c.SendStream(pr)
	})
}
