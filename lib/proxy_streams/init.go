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

func Load(r *fiber.App) {
	r.Get("/_/proxy/streams", func(c fiber.Ctx) error {
		ur := c.Query("url")
		if ur == "" {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, []byte(ur))
		if err != nil {
			return err
		}

		if !bytes.HasSuffix(parsed.Host(), []byte(".sndcdn.com")) {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

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

	r.Get("/_/proxy/streams/aac", func(c fiber.Ctx) error {
		ur := c.Query("url")
		if ur == "" {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, []byte(ur))
		if err != nil {
			return err
		}

		if !bytes.HasSuffix(parsed.Host(), []byte(".soundcloud.cloud")) {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

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

	r.Get("/_/proxy/streams/playlist", func(c fiber.Ctx) error {
		ur := c.Query("url")
		if ur == "" {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, []byte(ur))
		if err != nil {
			return err
		}

		if !bytes.HasSuffix(parsed.Host(), []byte(".sndcdn.com")) {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

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

		var sp = bytes.Split(data, []byte("\n"))
		for i, l := range sp {
			if len(l) == 0 || l[0] == '#' {
				continue
			}

			l = []byte("/_/proxy/streams?url=" + url.QueryEscape(string(l)))
			sp[i] = l
		}

		return c.Send(bytes.Join(sp, []byte("\n")))
	})

	r.Get("/_/proxy/streams/playlist/aac", func(c fiber.Ctx) error {
		ur := c.Query("url")
		if ur == "" {
			return fiber.ErrBadRequest
		}

		parsed := fasthttp.AcquireURI()
		defer fasthttp.ReleaseURI(parsed)

		err := parsed.Parse(nil, []byte(ur))
		if err != nil {
			return err
		}

		if !bytes.HasSuffix(parsed.Host(), []byte(".soundcloud.cloud")) {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.SetUserAgent(cfg.UserAgent)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)

		err = sc.DoWithRetry(misc.HlsAacClient, req, resp)
		if err != nil {
			return err
		}

		data, err := resp.BodyUncompressed()
		if err != nil {
			data = resp.Body()
		}

		var sp = bytes.Split(data, []byte("\n"))
		for i, l := range sp {
			if len(l) == 0 {
				continue
			}

			if l[0] == '#' {
				if bytes.HasPrefix(l, []byte(`#EXT-X-MAP:URI="`)) {
					l = []byte(`#EXT-X-MAP:URI="/_/proxy/streams/aac?url=` + url.QueryEscape(string(l[16:len(l)-1])) + `"`)
					sp[i] = l
				}

				continue
			}

			l = []byte("/_/proxy/streams/aac?url=" + url.QueryEscape(string(l)))
			sp[i] = l
		}

		return c.Send(bytes.Join(sp, []byte("\n")))
	})
}
