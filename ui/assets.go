package ui

import (
	"embed"
	"html/template"
	"net/http"
)

// Assets embeds all static files (CSS, JS) and Go templates (*.tpl).
// Modify this embed pattern if additional asset folders are introduced.
//
//go:embed html/* css/* js/*
var Assets embed.FS

// StaticFileServer returns an http.Handler that serves embedded static assets (css, js, etc.).
// Customizations: Wrap with http.StripPrefix (e.g., http.StripPrefix("/static/", ui.StaticFileServer()))
// if serving behind a route prefix like /static/.
func StaticFileServer() http.Handler {
	return http.FileServer(http.FS(Assets))
}

// ParseBaseTemplate parses ui/html/base.tpl from the embedded filesystem.
// Customizations: Add template.FuncMap here if custom formatting or helper functions are needed in templates.
func ParseBaseTemplate() (*template.Template, error) {
	return template.ParseFS(Assets, "html/base.tpl")
}
