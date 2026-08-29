package webui

import (
	"embed"
	"net/http"
)

//go:embed static/*
var files embed.FS

func Handler() http.Handler { return http.FileServer(http.FS(files)) }
