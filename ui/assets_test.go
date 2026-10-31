package ui_test

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sage/ui"
)

// TestStaticAssetsServed tests that all static assets and .tpl files
// are embedded and served over HTTP with 200 OK and non-empty content.
func TestStaticAssetsServed(t *testing.T) {
	server := httptest.NewServer(ui.StaticFileServer())
	defer server.Close()

	assetPaths := []struct {
		path            string
		expectedTypeSub string
	}{
		{"/css/tailwind.min.css", "text/css"},
		{"/fonts/Caveat-Regular.ttf", "font"},
		{"/js/panzoom.tpl", "javascript"},
		{"/js/app.tpl", "javascript"},
	}

	for _, asset := range assetPaths {
		t.Run(asset.path, func(t *testing.T) {
			resp, err := http.Get(server.URL + asset.path)
			if err != nil {
				t.Fatalf("failed to fetch %s: %v", asset.path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200 OK for %s, got %d", asset.path, resp.StatusCode)
			}

			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), asset.expectedTypeSub) {
				t.Errorf("expected Content-Type containing %q for %s, got %q", asset.expectedTypeSub, asset.path, contentType)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body of %s: %v", asset.path, err)
			}

			if len(body) == 0 {
				t.Errorf("expected non-empty content for %s", asset.path)
			}
		})
	}
}

// TestBaseTemplateRendering verifies that ui/html/base.tpl parses cleanly
// and renders all structural elements and D2 SVG injections.
func TestBaseTemplateRendering(t *testing.T) {
	tmpl, err := ui.ParseBaseTemplate()
	if err != nil {
		t.Fatalf("failed to parse base.tpl: %v", err)
	}

	data := struct {
		ActiveFile string
		DiagramSVG template.HTML
		Warnings   []string
	}{
		ActiveFile: "SAGE.md",
		DiagramSVG: template.HTML(`<svg id="test-d2"><circle cx="10" cy="10" r="5"/></svg>`),
		Warnings:   []string{"Test warning"},
	}

	var w strings.Builder
	if err := tmpl.Execute(&w, data); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	rendered := w.String()

	requiredSubstrings := []string{
		`<!DOCTYPE html>`,
		`/static/css/tailwind.min.css`,
		`/static/js/panzoom.tpl`,
		`/static/js/app.tpl`,
		`<div id="diagram-stage"`,
		`<svg id="test-d2">`,
		`Test warning`,
		`id="btn-theme-toggle"`,
		`id="btn-zoom-in"`,
		`id="btn-zoom-out"`,
		`id="btn-reset-view"`,
		`id="btn-export-svg"`,
	}

	for _, sub := range requiredSubstrings {
		if !strings.Contains(rendered, sub) {
			t.Errorf("rendered HTML missing required element %q", sub)
		}
	}
}
