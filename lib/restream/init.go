package restream

import (
	"bytes"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maid-zone/soundcloak/lib/cfg"
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
	r.Get("/_/restream", func(c *fiber.Ctx) error {
		// uncomment this to automatically get playlist of a track for easy testing
		// t, err := sc.GetTrack("homelocked/por-mais-alguem-prod-homelocked")
		// if err != nil {
		// 	return err
		// }

		// u, err := t.GetStream()
		// if err != nil {
		// 	return err
		// }

		// and comment this
		u := c.Query("playlist")

		if !strings.HasPrefix(u, "https://"+cdn+"/") {
			return fiber.ErrBadRequest
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
