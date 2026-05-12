package restream

import (
	"io"
	"sync"

	"github.com/valyala/fasthttp"
)

// inject some bytes before the real reader
type injector struct {
	reader   io.Reader
	resp     *fasthttp.Response
	leftover []byte
	offset   int
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
		n = copy(buf, r.leftover[r.offset:])

		if r.offset+n >= len(r.leftover) {
			r.leftover = r.leftover[:0]
			r.offset = 0
		} else {
			r.offset += n
		}

		if n == len(buf) {
			return
		}
	}

	m, err := r.reader.Read(buf[n:])
	n += m
	return
}

func (r *injector) Close() error {
	r.resp.CloseBodyStream()
	fasthttp.ReleaseResponse(r.resp)

	r.reader = nil
	r.resp = nil
	r.leftover = r.leftover[:0]
	r.offset = 0

	injectorpool.Put(r)
	return nil
}

func (r *injector) Write(data []byte) (n int, err error) {
	r.leftover = append(r.leftover, data...)
	return len(data), nil
}
