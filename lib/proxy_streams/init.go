package proxystreams

import (
	"bytes"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

const sndcdn = ".sndcdn.com"
const soundcloudcloud = "soundcloud.cloud"

var newline = []byte{'\n'}

var hls_aac_streaming_httpc *fasthttp.HostClient

func Load(a *fiber.App) {
	hls_aac_streaming_httpc = &fasthttp.HostClient{
		Addr:                cfg.HLSAACCDN + ":443",
		IsTLS:               true,
		MaxIdleConnDuration: cfg.MaxIdleConnDuration,
		StreamResponseBody:  true,
		MaxResponseBodySize: 1,
		DialDualStack:       cfg.DialDualStack,
	}

	r := a.Group("/_/proxy/streams")

	r.Get("/", func(c fiber.Ctx) error {
		ur := c.RequestCtx().QueryArgs().Peek("url")
		if len(ur) == 0 {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, ur)
		if err != nil {
			return err
		}

		if h := parsed.Host(); len(h) > len(sndcdn) && string(h[len(h)-len(sndcdn):]) != sndcdn {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)

		resp := fasthttp.AcquireResponse()
		//defer fasthttp.ReleaseResponse(resp)

		err = sc.DoWithRetry(misc.HlsStreamingOnlyClient, req, resp)
		if err != nil {
			return err
		}
		//return c.Send(resp.Body())
		pr := misc.AcquireProxyReader()
		pr.Reader = resp.BodyStream()
		pr.Resp = resp
		return c.SendStream(pr)
	})

	r.Get("/aac", func(c fiber.Ctx) error {
		ur := c.RequestCtx().QueryArgs().Peek("url")
		if len(ur) == 0 {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, ur)
		if err != nil {
			return err
		}

		if h := parsed.Host(); len(h) > len(soundcloudcloud) && string(h[len(h)-len(soundcloudcloud):]) != soundcloudcloud {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)

		resp := fasthttp.AcquireResponse()

		err = sc.DoWithRetry(hls_aac_streaming_httpc, req, resp)
		if err != nil {
			return err
		}

		pr := misc.AcquireProxyReader()
		pr.Reader = resp.BodyStream()
		pr.Resp = resp
		return c.SendStream(pr)
	})

	r.Get("/playlist", func(c fiber.Ctx) error {
		ur := c.RequestCtx().QueryArgs().Peek("url")
		if len(ur) == 0 {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, ur)
		if err != nil {
			return err
		}

		const x = ".sndcdn.com"
		if h := parsed.Host(); len(h) > len(x) && string(h[len(h)-len(x):]) != x {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)

		err = sc.DoWithRetry(misc.HlsClient, req, resp)
		if err != nil {
			return err
		}

		for l := range bytes.SplitSeq(resp.Body(), newline) {
			if len(l) == 0 || l[0] == '#' {
				c.Response().AppendBody(l)
				c.Response().AppendBody(newline)
				continue
			}

			c.Response().AppendBodyString("/_/proxy/streams?url=")
			c.Response().AppendBody(fasthttp.AppendQuotedArg(nil, l))
			c.Response().AppendBody(newline)
		}

		return nil
	})

	r.Get("/playlist/aac", func(c fiber.Ctx) error {
		ur := c.RequestCtx().QueryArgs().Peek("url")
		if len(ur) == 0 {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, ur)
		if err != nil {
			return err
		}

		const x = ".soundcloud.cloud"
		if h := parsed.Host(); len(h) > len(x) && string(h[len(h)-len(x):]) != x {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)

		err = sc.DoWithRetry(misc.HlsAacClient, req, resp)
		if err != nil {
			return err
		}

		for l := range bytes.SplitSeq(resp.Body(), newline) {
			if len(l) == 0 {
				c.Response().AppendBody(l)
				c.Response().AppendBody(newline)
				continue
			}

			if l[0] == '#' {
				// #EXT-X-MAP:URI="..."
				const x = `#EXT-X-MAP:URI="`
				if len(l) > len(x) && string(l[:len(x)]) == x {
					c.Response().AppendBodyString(`#EXT-X-MAP:URI="/_/proxy/streams/aac?url=`)
					c.Response().AppendBody(fasthttp.AppendQuotedArg(nil, l[16:len(l)-1]))
					c.Response().AppendBodyString(`"`)
				} else {
					c.Response().AppendBody(l)
				}
				c.Response().AppendBody(newline)
				continue
			}

			c.Response().AppendBodyString("/_/proxy/streams/aac?url=")
			c.Response().AppendBody(fasthttp.AppendQuotedArg(nil, l))
			c.Response().AppendBody(newline)
		}

		return nil
	})
}
