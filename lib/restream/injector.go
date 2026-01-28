package restream

import (
	"io"
	"sync"

	"github.com/valyala/fasthttp"
)

// inject some bytes before the real reader
type injector struct {
	leftover []byte
	reader   io.Reader
	resp     *fasthttp.Response
}

var injectorpool = sync.Pool{
	New: func() any {
		return &injector{}
	},
}

func acquireInjector() *injector {
	return injectorpool.Get().(*injector)
}

func (r *injector) Read(buf []byte) (n int, err error) {
	if len(r.leftover) != 0 {
		h := min(len(buf), len(r.leftover))

		n = copy(buf, r.leftover[:h])

		if n > len(r.leftover) {
			r.leftover = nil
		} else {
			r.leftover = r.leftover[n:]
		}

		return
	}

	return r.reader.Read(buf)
}

func (r *injector) Close() error {
	r.resp.CloseBodyStream()
	fasthttp.ReleaseResponse(r.resp)

	r.reader = nil
	r.resp = nil
	r.leftover = r.leftover[:0]

	injectorpool.Put(r)
	return nil
}

func (c *injector) Write(data []byte) (n int, err error) {
	c.leftover = append(c.leftover, data...)
	return len(data), nil
}
