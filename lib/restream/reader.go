package restream

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"git.maid.zone/stuff/soundcloak/lib/misc"
	"git.maid.zone/stuff/soundcloak/lib/sc"
	"github.com/valyala/fasthttp"
)

const defaultPartsCapacity = 24

type reader struct {
	duration *uint32

	req      *fasthttp.Request
	resp     *fasthttp.Response
	client   *fasthttp.HostClient
	parts    [][]byte
	leftover []byte
	index    int
}

var readerpool = sync.Pool{
	New: func() any {
		return &reader{}
	},
}

func acquireReader() *reader {
	return readerpool.Get().(*reader)
}

func clone(buf []byte) []byte {
	out := make([]byte, len(buf))
	copy(out, buf)
	return out
}

var mvhd = []byte("mvhd")
var newline = []byte{'\n'}

func fixDuration(data []byte, duration *uint32) {
	i := bytes.Index(data, mvhd)
	if i != -1 {
		i += 20

		bt := make([]byte, 4)
		binary.BigEndian.PutUint32(bt, *duration)
		copy(data[i:], bt)
		// timescale is already 1000 in the files
	}
}

func (r *reader) Setup(url string, aac bool, duration *uint32) error {
	if r.req == nil {
		r.req = fasthttp.AcquireRequest()
	}

	if r.resp == nil {
		r.resp = fasthttp.AcquireResponse()
	}

	r.req.SetRequestURI(url)
	r.req.Header.SetUserAgent(cfg.UserAgent)

	if aac {
		r.client = misc.HlsAacClient
		r.duration = duration
	} else {
		r.client = misc.HlsClient
	}

	err := sc.DoWithRetry(r.client, r.req, r.resp)
	if err != nil {
		return err
	}

	if r.parts == nil {
		misc.Log("make() r.parts")
		r.parts = make([][]byte, 0, defaultPartsCapacity)
	} else {
		misc.Log(cap(r.parts), len(r.parts))
	}
	// clone needed to mitigate memory skill issues smh
	if aac {
		for s := range bytes.SplitSeq(r.resp.Body(), newline) {
			if len(s) == 0 {
				continue
			}
			if s[0] == '#' {
				// #EXT-X-MAP:URI="..."
				const x = `#EXT-X-MAP:URI="`
				if len(s) > len(x) && string(s[:len(x)]) == x {
					r.parts = append(r.parts, clone(s[len(x):len(s)-1]))
				}

				continue
			}

			r.parts = append(r.parts, clone(s))
		}
	} else {
		for s := range bytes.SplitSeq(r.resp.Body(), newline) {
			if len(s) == 0 || s[0] == '#' {
				continue
			}

			r.parts = append(r.parts, clone(s))
		}
	}

	return nil
}

func (r *reader) Close() error {
	misc.Log("closed :D")
	r.req.Reset()
	r.resp.Reset()

	r.leftover = r.leftover[:0]
	r.index = 0
	r.parts = r.parts[:0]

	readerpool.Put(r)
	return nil
}

// I have no idea what this truly even does anymore. Maybe a rewrite/refactor would be good?
func (r *reader) Read(buf []byte) (n int, err error) {
	misc.Log("we read")
	if len(r.leftover) != 0 {
		h := min(len(buf), len(r.leftover))

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

	err = sc.DoWithRetry(r.client, r.req, r.resp)
	if err != nil {
		return
	}

	data := r.resp.Body()
	if r.index == 0 && r.duration != nil {
		fixDuration(data, r.duration) // I'm guessing that mvhd will always be in first part
	}

	if len(data) > len(buf) {
		n = copy(buf, data[:len(buf)])
	} else {
		n = copy(buf, data)
	}

	r.leftover = data[n:]
	r.index++

	if n < len(buf) && r.index == len(r.parts) {
		err = io.EOF
	}

	return
}

func (c *reader) Write(data []byte) (n int, err error) {
	c.leftover = append(c.leftover, data...)
	return len(data), nil
}
