package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist fallback/index.html
var files embed.FS

func FS() fs.FS {
	root := "dist"
	if _, err := fs.Stat(files, "dist/index.html"); err != nil {
		root = "fallback"
	}
	sub, err := fs.Sub(files, root)
	if err != nil {
		panic(err)
	}
	return sub
}
