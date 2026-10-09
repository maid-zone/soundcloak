package static

// I had to move the folders to here due to go limitation. You can't embed from relative (e.g. parent) paths

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

//go:embed */*
var All embed.FS

var assetFS fs.FS = All
var versions sync.Map
var embedded = true

// configure uses the same filesystem and instance overrides as the asset server
// call once during startup, before serving requests
func Configure(filesystem fs.FS) {
	assetFS = filesystem
	_, embedded = filesystem.(embed.FS)
	versions = sync.Map{}
}

func Version(name string) string {
	if embedded {
		if version, ok := versions.Load(name); ok {
			return version.(string)
		}
	}
	var data []byte
	var err error
	if strings.HasPrefix(name, "external/") {
		data, err = fs.ReadFile(assetFS, name)
	} else {
		data, err = fs.ReadFile(assetFS, "instance/"+name)
		if err != nil {
			data, err = fs.ReadFile(assetFS, "assets/"+name)
		}
	}
	if err != nil {
		return ""
	}
	version := fmt.Sprintf("%x", sha256.Sum256(data))[:16]
	if embedded {
		versions.Store(name, version)
	}
	return version
}

func URL(name string) string {
	version := Version(name)
	if version == "" {
		return "/_/static/" + name
	}
	return "/_/static/" + name + "?v=" + version
}
