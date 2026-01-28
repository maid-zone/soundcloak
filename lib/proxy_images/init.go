package proxyimages

import (
	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

var al_httpc *fasthttp.HostClient
var streaming_httpc *fasthttp.HostClient

func Load(r *fiber.App) {

	al_httpc = &fasthttp.HostClient{
		Addr:                "al.sndcdn.com:443",
		IsTLS:               true,
		MaxIdleConnDuration: cfg.MaxIdleConnDuration,
		StreamResponseBody:  true,
		MaxResponseBodySize: 1,
	}

	streaming_httpc = &fasthttp.HostClient{
		Addr:                cfg.ImageCDN + ":443",
		IsTLS:               true,
		MaxIdleConnDuration: cfg.MaxIdleConnDuration,
		StreamResponseBody:  true,
		MaxResponseBodySize: 1,
		DialDualStack:       cfg.DialDualStack,
	}

	r.Get("/_/proxy/images", func(c fiber.Ctx) error {
		url := c.RequestCtx().QueryArgs().Peek("url")
		if len(url) == 0 {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, url)
		if err != nil {
			return err
		}

		const x = ".sndcdn.com"
		if h := parsed.Host(); len(h) > len(x) && string(h[len(h)-len(x):]) != x {
			return fiber.ErrBadRequest
		}

		var cl *fasthttp.HostClient
		if parsed.Host()[0] == 'i' {
			parsed.SetHost(cfg.ImageCDN)
			cl = streaming_httpc
		} else if string(parsed.Host()[:2]) == "al" {
			cl = al_httpc
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)

		resp := fasthttp.AcquireResponse()
		//defer fasthttp.ReleaseResponse(resp) moved to proxyreader!!!

		err = sc.DoWithRetry(cl, req, resp)
		if err != nil {
			return err
		}

		c.Response().Header.SetContentTypeBytes(resp.Header.ContentType())
		c.Set("Cache-Control", cfg.ImageCacheControl)
		//return c.Send(resp.Body())
		pr := misc.AcquireProxyReader()
		pr.Reader = resp.BodyStream()
		pr.Resp = resp
		return c.SendStream(pr, resp.Header.ContentLength())
	})
}
