package proxystreams

import (
	"bytes"
	"net/url"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

var sndcdn = []byte(".sndcdn.com")
var soundcloudcloud = []byte(".soundcloud.cloud")
var newline = []byte{'\n'}
var extxmap = []byte(`#EXT-X-MAP:URI="`)

func Load(a *fiber.App) {
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

		if !bytes.HasSuffix(parsed.Host(), sndcdn) {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)

		resp := fasthttp.AcquireResponse()
		//defer fasthttp.ReleaseResponse(resp)

		err = sc.DoWithRetry(misc.HlsClient, req, resp)
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

		if !bytes.HasSuffix(parsed.Host(), soundcloudcloud) {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)

		resp := fasthttp.AcquireResponse()

		err = sc.DoWithRetry(misc.HlsAacClient, req, resp)
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

		if !bytes.HasSuffix(parsed.Host(), sndcdn) {
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

		data, err := resp.BodyUncompressed()
		if err != nil {
			data = resp.Body()
		}

		var sp = bytes.Split(data, newline)
		for i, l := range sp {
			if len(l) == 0 || l[0] == '#' {
				continue
			}

			l = []byte("/_/proxy/streams?url=" + url.QueryEscape(cfg.B2s(l)))
			sp[i] = l
		}

		return c.Send(bytes.Join(sp, newline))
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

		if !bytes.HasSuffix(parsed.Host(), soundcloudcloud) {
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

		var sp = bytes.Split(resp.Body(), newline)
		for i, l := range sp {
			if len(l) == 0 {
				continue
			}

			if l[0] == '#' {
				if bytes.HasPrefix(l, extxmap) {
					l = []byte(`#EXT-X-MAP:URI="/_/proxy/streams/aac?url=` + url.QueryEscape(cfg.B2s(l[16:len(l)-1])) + `"`)
					sp[i] = l
				}

				continue
			}

			l = []byte("/_/proxy/streams/aac?url=" + url.QueryEscape(cfg.B2s(l)))
			sp[i] = l
		}

		return c.Send(bytes.Join(sp, newline))
	})
}
