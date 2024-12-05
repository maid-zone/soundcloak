package proxystreams

import (
	"bytes"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/maid-zone/soundcloak/lib/cfg"
	"github.com/maid-zone/soundcloak/lib/sc"
	"github.com/valyala/fasthttp"
)

const cdn = "cf-hls-media.sndcdn.com"

var httpc = &fasthttp.HostClient{
	Addr:                cdn + ":443",
	IsTLS:               true,
	DialDualStack:       true,
	Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
	MaxIdleConnDuration: 1<<63 - 1,
	StreamResponseBody:  true,
}

func Load(r fiber.Router) {
	r.Get("/_/proxy/streams", func(c *fiber.Ctx) error {
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

		if !bytes.Equal(parsed.Host(), []byte(cdn)) {
			return fiber.ErrBadRequest
		}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetURI(parsed)
		req.Header.Set("User-Agent", cfg.UserAgent)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

		resp := fasthttp.AcquireResponse()
		//defer fasthttp.ReleaseResponse(resp)

		err = sc.DoWithRetry(httpc, req, resp)
		if err != nil {
			return err
		}
		//return c.Send(resp.Body())
		pr := cfg.AcquireProxyReader()
		pr.Reader = resp.BodyStream()
		pr.Resp = resp
		return c.SendStream(pr)
	})

	r.Get("/_/proxy/streams/playlist", func(c *fiber.Ctx) error {
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

		if !bytes.Equal(parsed.Host(), []byte(cdn)) {
			return fiber.ErrBadRequest
		}

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
}
