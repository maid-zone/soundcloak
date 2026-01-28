package misc

import (
	"io"
	"log"
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
	pr.Resp.CloseBodyStream()
	fasthttp.ReleaseResponse(pr.Resp)

	pr.Reader = nil
	pr.Resp = nil

	prpool.Put(pr)
	return nil
}

func Log(what ...any) {
	if cfg.Debug {
		log.Println(what...)
	}
}

var HlsClient *fasthttp.HostClient
var HlsStreamingOnlyClient *fasthttp.HostClient
var HlsAacClient *fasthttp.HostClient

func init() {
	if cfg.Restream || cfg.ProxyStreams {
		HlsClient = &fasthttp.HostClient{
			Addr:                cfg.HLSCDN + ":443",
			IsTLS:               true,
			MaxIdleConnDuration: cfg.MaxIdleConnDuration,
			DialDualStack:       cfg.DialDualStack,
		}

		HlsAacClient = &fasthttp.HostClient{
			Addr:                cfg.HLSAACCDN + ":443",
			IsTLS:               true,
			MaxIdleConnDuration: cfg.MaxIdleConnDuration,
			DialDualStack:       cfg.DialDualStack,
		}

		HlsStreamingOnlyClient = &fasthttp.HostClient{
			Addr:                cfg.HLSCDN + ":443",
			IsTLS:               true,
			MaxIdleConnDuration: cfg.MaxIdleConnDuration,
			StreamResponseBody:  true,
			MaxResponseBodySize: 1,
			DialDualStack:       cfg.DialDualStack,
		}
	}
}
