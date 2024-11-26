package restream

import (
	"bytes"
	"io"

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
}

// Needed for restream to work even if prefs.Player != RestreamPlayer
var stubPrefs = cfg.Preferences{}

func init() {
	p := cfg.RestreamPlayer

	stubPrefs.Player = &p

	f := false

	stubPrefs.ProxyStreams = &f
	stubPrefs.ProxyImages = &f
}

type reader struct {
	parts    [][]byte
	leftover []byte
	index    int

	req  *fasthttp.Request
	resp *fasthttp.Response
}

func (r *reader) Setup(url string) error {
	r.req = fasthttp.AcquireRequest()
	r.resp = fasthttp.AcquireResponse()

	r.req.SetRequestURI(url)
	r.req.Header.Set("User-Agent", cfg.UserAgent)
	r.req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	err := httpc.Do(r.req, r.resp)
	if err != nil {
		return err
	}

	data, err := r.resp.BodyUncompressed()
	if err != nil {
		data = r.resp.Body()
	}

	for _, s := range bytes.Split(data, []byte{'\n'}) {
		if len(s) == 0 || s[0] == '#' {
			continue
		}

		r.parts = append(r.parts, s)
	}

	return nil
}

// you could prob make this a bit faster by concurrency (make a bunch of workers => make them download the parts => temporarily add them to a map => fully assemble the result => make reader.Read() read out the result as the parts are coming in) but whatever, fine for now
func (r *reader) Read(buf []byte) (n int, err error) {
	if len(r.leftover) != 0 {
		h := len(buf)
		if h > len(r.leftover) {
			h = len(r.leftover)
		}

		n = copy(buf, r.leftover[:h])

		if n > len(r.leftover) {
			r.leftover = r.leftover[:0]
		} else {
			r.leftover = r.leftover[n:]
		}

		if n < len(buf) && r.index == len(r.parts) {
			err = io.EOF
		}

		return
	}

	if r.index == len(r.parts) {
		fasthttp.ReleaseRequest(r.req)
		fasthttp.ReleaseResponse(r.resp)
		err = io.EOF
		return
	}

	r.req.SetRequestURIBytes(r.parts[r.index])

	err = httpc.Do(r.req, r.resp)
	if err != nil {
		return
	}

	data, err := r.resp.BodyUncompressed()
	if err != nil {
		data = r.resp.Body()
	}

	n = copy(buf, data[:len(buf)])

	r.leftover = data[n:]
	r.index++

	if n < len(buf) && r.index == len(r.parts) {
		err = io.EOF
	}

	return
}

func Load(r fiber.Router) {
	r.Get("/_/restream/:author/:track", func(c *fiber.Ctx) error {
		t, err := sc.GetTrack(stubPrefs, c.Params("author")+"/"+c.Params("track"))
		if err != nil {
			return err
		}

		tr := t.Media.SelectCompatible()
		if tr == nil {
			return fiber.ErrExpectationFailed
		}

		u, err := tr.GetStream(stubPrefs, t.Authorization)
		if err != nil {
			return err
		}

		c.Set("Content-Type", "audio/mpeg")
		c.Set("Cache-Control", cfg.RestreamCacheControl)

		r := reader{}
		if err := r.Setup(u); err != nil {
			return err
		}

		return c.SendStream(&r)
	})
}
