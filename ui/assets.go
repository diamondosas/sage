package ui

import (
	"embed"
	"html/template"
	"mime"
	"net/http"
)

func init() {
	// Register .tpl files as JavaScript so http.FileServer serves embedded .tpl scripts with the correct MIME type.
	// VARIABLE TWEAK: Change "text/javascript; charset=utf-8" if an alternate MIME type is needed.
	_ = mime.AddExtensionType(".tpl", "text/javascript; charset=utf-8")
}

// Assets embeds all static files (CSS, JS) and Go templates (*.tpl).
//
//go:embed html/* css/* js/* fonts/*
var Assets embed.FS

// StaticFileServer returns an http.Handler that serves embedded static assets (css, js, etc.).
func StaticFileServer() http.Handler {
	return http.FileServer(http.FS(Assets))
}

// ParseBaseTemplate parses ui/html/base.tpl from the embedded filesystem.
func ParseBaseTemplate() (*template.Template, error) {
	return template.ParseFS(Assets, "html/base.tpl")
}
