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
	parts    [][]byte
	leftover []byte
	index    int
	duration *uint32

	req    *fasthttp.Request
	resp   *fasthttp.Response
	client *fasthttp.HostClient
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
	r.req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

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

	data, err := r.resp.BodyUncompressed()
	if err != nil {
		data = r.resp.Body()
	}

	if r.parts == nil {
		misc.Log("make() r.parts")
		r.parts = make([][]byte, 0, defaultPartsCapacity)
	} else {
		misc.Log(cap(r.parts), len(r.parts))
	}
	if aac {
		// clone needed to mitigate memory skill issues here
		for _, s := range bytes.Split(data, []byte{'\n'}) {
			if len(s) == 0 {
				continue
			}
			if s[0] == '#' {
				if bytes.HasPrefix(s, []byte(`#EXT-X-MAP:URI="`)) {
					r.parts = append(r.parts, clone(s[16:len(s)-1]))
				}

				continue
			}

			r.parts = append(r.parts, clone(s))
		}
	} else {
		for _, s := range bytes.Split(data, []byte{'\n'}) {
			if len(s) == 0 || s[0] == '#' {
				continue
			}

			r.parts = append(r.parts, s)
		}
	}

	return nil
}

func (r *reader) Close() error {
	misc.Log("closed :D")
	r.req.Reset()
	r.resp.Reset()

	r.leftover = nil
	r.index = 0
	r.parts = r.parts[:0]

	readerpool.Put(r)
	return nil
}

// you could prob make this a bit faster by concurrency (make a bunch of workers => make them download the parts => temporarily add them to a map => fully assemble the result => make reader.Read() read out the result as the parts are coming in) but whatever, fine for now
func (r *reader) Read(buf []byte) (n int, err error) {
	misc.Log("we read")
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

	err = sc.DoWithRetry(r.client, r.req, r.resp)
	if err != nil {
		return
	}

	data, err := r.resp.BodyUncompressed()
	if err != nil {
		data = r.resp.Body()
	}

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
