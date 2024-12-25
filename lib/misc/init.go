package misc

import (
	"fmt"
	"io"
	"sync"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"github.com/valyala/fasthttp"
)

var prpool = sync.Pool{
	New: func() any {
		return &ProxyReader{}
	},
}

func AcquireProxyReader() *ProxyReader {
	return prpool.Get().(*ProxyReader)
}

type ProxyReader struct {
	Reader io.Reader
	Resp   *fasthttp.Response
}

func (pr *ProxyReader) Read(p []byte) (int, error) {
	return pr.Reader.Read(p)
}

func (pr *ProxyReader) Close() error {
	defer fasthttp.ReleaseResponse(pr.Resp)
	defer prpool.Put(pr)
	return pr.Resp.CloseBodyStream()
}

func Log(what ...any) {
	if cfg.Debug {
		fmt.Println(what...)
	}
}
