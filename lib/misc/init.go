package misc

import (
	"log"
	"strconv"

	"git.maid.zone/stuff/soundcloak/lib/cfg"
	"github.com/valyala/fasthttp"
)

func Log(what ...any) {
	if cfg.Debug {
		log.Println(what...)
	}
}

func n(b byte) byte {
	return b/10 + '0'
}

func n2(b byte) byte {
	return b%10 + '0'
}

func FormatTimecode(timecode int) string {
	timecode /= 1000
	seconds := byte(timecode % 60)
	minutes := byte((timecode / 60) % 60)
	hours := byte(timecode / 3600)
	if hours != 0 {
		return string([]byte{n(hours), n2(hours), ':', n(minutes), n2(minutes), ':', n(seconds), n2(seconds)})
	} else {
		return string([]byte{n(minutes), n2(minutes), ':', n(seconds), n2(seconds)})
	}
}

func HumanizeNumber(input int64) string {
	if input >= 1_000_000_000 {
		return strconv.FormatInt(input/1_000_000_000, 10) + "B"
	} else if input >= 1_000_000 {
		return strconv.FormatInt(input/1_000_000, 10) + "M"
	} else if input >= 10_000 {
		return strconv.FormatInt(input/1_000, 10) + "K"
	} else {
		return strconv.FormatInt(input, 10)
	}
}

var HlsClient = &fasthttp.HostClient{
	Addr:                cfg.HLSCDN + ":443",
	IsTLS:               true,
	MaxIdleConnDuration: cfg.MaxIdleConnDuration,
	DialDualStack:       cfg.DialDualStack,
}

var HlsAacClient = &fasthttp.HostClient{
	Addr:                cfg.HLSAACCDN + ":443",
	IsTLS:               true,
	MaxIdleConnDuration: cfg.MaxIdleConnDuration,
	DialDualStack:       cfg.DialDualStack,
}
var HlsStreamingOnlyClient *fasthttp.HostClient
var ImageStreamingOnlyClient *fasthttp.HostClient

func init() {
	if cfg.Restream || cfg.ProxyStreams {
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
