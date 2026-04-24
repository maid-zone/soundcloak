package misc

import (
	"log"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"github.com/valyala/fasthttp"
)

func Log(what ...any) {
	if cfg.Debug {
		log.Println(what...)
	}
}

var HlsClient *fasthttp.HostClient
var HlsStreamingOnlyClient *fasthttp.HostClient
var HlsAacClient *fasthttp.HostClient
var ImageStreamingOnlyClient *fasthttp.HostClient

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

	if cfg.Restream || cfg.ProxyImages {
		ImageStreamingOnlyClient = &fasthttp.HostClient{
			Addr:                cfg.ImageCDN + ":443",
			IsTLS:               true,
			MaxIdleConnDuration: cfg.MaxIdleConnDuration,
			StreamResponseBody:  true,
			MaxResponseBodySize: 1,
			DialDualStack:       cfg.DialDualStack,
		}
	}
}
