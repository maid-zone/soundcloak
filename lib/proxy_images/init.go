package proxyimages

import (
	"bytes"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

var httpc *fasthttp.HostClient
var al_httpc *fasthttp.HostClient

func Load(r fiber.Router) {
	httpc = &fasthttp.HostClient{
		Addr:                cfg.ImageCDN + ":443",
		IsTLS:               true,
		DialDualStack:       true,
		Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
		MaxIdleConnDuration: 1<<63 - 1,
		StreamResponseBody:  true,
	}

	al_httpc = &fasthttp.HostClient{
		Addr:                "al.sndcdn.com:443",
		IsTLS:               true,
		DialDualStack:       true,
		Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
		MaxIdleConnDuration: 1<<63 - 1,
		StreamResponseBody:  true,
	}

	r.Get("/_/proxy/images", func(c fiber.Ctx) error {
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

		var cl *fasthttp.HostClient
		if parsed.Host()[0] == 'i' {
			parsed.SetHost(cfg.ImageCDN)
			cl = httpc
		} else if string(parsed.Host()[:2]) == "al" {
			cl = al_httpc
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)
		//req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd") images not big enough to be compressed

		resp := fasthttp.AcquireResponse()
		//defer fasthttp.ReleaseResponse(resp) moved to proxyreader!!!

		err = sc.DoWithRetry(cl, req, resp)
		if err != nil {
			return err
		}

		c.Set("Content-Type", "image/jpeg")
		c.Set("Cache-Control", cfg.ImageCacheControl)
		//return c.Send(resp.Body())
		pr := misc.AcquireProxyReader()
		pr.Reader = resp.BodyStream()
		pr.Resp = resp
		return c.SendStream(pr)
	})
}
