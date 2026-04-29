package frontendui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var content embed.FS

func Handler() http.Handler {
	staticFiles, err := fs.Sub(content, "static")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(staticFiles))
}
