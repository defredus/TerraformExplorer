package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist/* dist/assets/*
var embedded embed.FS

func Assets() fs.FS {
	assets, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return assets
}
