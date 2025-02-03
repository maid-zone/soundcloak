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
	pr.Resp.CloseBodyStream()
	fasthttp.ReleaseResponse(pr.Resp)

	pr.Reader = nil
	pr.Resp = nil

	prpool.Put(pr)
	return nil
}

func Log(what ...any) {
	if cfg.Debug {
		fmt.Println(what...)
	}
}

var ImageClient *fasthttp.HostClient
var HlsClient *fasthttp.HostClient
var HlsAacClient *fasthttp.HostClient

func init() {
	if cfg.Restream || cfg.ProxyImages {
		ImageClient = &fasthttp.HostClient{
			Addr:                cfg.ImageCDN + ":443",
			IsTLS:               true,
			Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
			MaxIdleConnDuration: cfg.MaxIdleConnDuration,
			StreamResponseBody:  true,
		}
	}

	if cfg.Restream || cfg.ProxyStreams {
		HlsClient = &fasthttp.HostClient{
			Addr:                cfg.HLSCDN + ":443",
			IsTLS:               true,
			Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
			MaxIdleConnDuration: cfg.MaxIdleConnDuration,
			StreamResponseBody:  true,
		}

		HlsAacClient = &fasthttp.HostClient{
			Addr:                cfg.HLSAACCDN + ":443",
			IsTLS:               true,
			Dial:                (&fasthttp.TCPDialer{DNSCacheDuration: cfg.DNSCacheTTL}).Dial,
			MaxIdleConnDuration: cfg.MaxIdleConnDuration,
			StreamResponseBody:  true,
		}
	}
}
