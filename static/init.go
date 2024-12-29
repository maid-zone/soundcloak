package static

// I had to move the folders to here due to go limitation. You can't embed from relative (e.g. parent) paths

import "embed"

//go:embed assets/*
var Assets embed.FS

//go:embed instance/*
var Instance embed.FS

//go:embed external/*
var External embed.FS
