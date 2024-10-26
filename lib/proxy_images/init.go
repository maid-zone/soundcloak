package proxyimages

import (
	"bytes"

	"github.com/gofiber/fiber/v2"
	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/maid-zone/soundcloak/lib/sc"
	"github.com/valyala/fasthttp"
)

var sndcdn = []byte(".sndcdn.com")

// seems soundcloud has 4 of these (i1, i2, i3, i4)
// they point to the same ip from my observations, and they all serve the same files
const cdn = "i1.sndcdn.com"

var httpc = &fasthttp.HostClient{
	Addr:                cdn + ":443",
	IsTLS:               true,
	DialDualStack:       true,
	Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
	MaxIdleConnDuration: 1<<63 - 1,
}

func Load(r fiber.Router) {
	r.Get("/proxy/images", func(c *fiber.Ctx) error {
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

		if !bytes.HasSuffix(parsed.Host(), sndcdn) {
			return fiber.ErrBadRequest
		}

		parsed.SetHost(cdn)

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.Set("User-Agent", cfg.UserAgent)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)

		err = sc.DoWithRetry(httpc, req, resp)
		if err != nil {
			return err
		}

		c.Response().Header.SetBytesV("Content-Type", resp.Header.Peek("Content-Type"))
		c.Set("Cache-Control", cfg.ImageCacheControl)
		return c.Send(resp.Body())
	})
}
