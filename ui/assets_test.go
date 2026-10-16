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

// TestStaticAssetsServed tests that all required static assets (tailwind, rough, dagre)
// are embedded and served over HTTP with 200 OK and non-empty content.
func TestStaticAssetsServed(t *testing.T) {
	server := httptest.NewServer(ui.StaticFileServer())
	defer server.Close()

	// To test additional static assets, add their paths to this slice:
	assetPaths := []string{
		"/css/tailwind.min.css",
		"/js/libs/rough.js",
		"/js/libs/dagre.min.js",
	}

	for _, path := range assetPaths {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(server.URL + path)
			if err != nil {
				t.Fatalf("failed to fetch %s: %v", path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200 OK for %s, got %d", path, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body of %s: %v", path, err)
			}

			if len(body) == 0 {
				t.Errorf("expected non-empty content for %s", path)
			}
		})
	}
}

// TestBaseTemplateRendering verifies that ui/html/base.tpl parses cleanly
// and renders all required structural elements and data injections.
func TestBaseTemplateRendering(t *testing.T) {
	tmpl, err := ui.ParseBaseTemplate()
	if err != nil {
		t.Fatalf("failed to parse base.tpl: %v", err)
	}

	// Modify these test variables to test alternative outputs:
	// - ActiveFile: change to test different file names or fallback labels
	// - GraphJSON: modify JSON structure to test different initial payloads
	// - Warnings: add or empty out warnings to test notification banner visibility
	data := struct {
		ActiveFile string
		GraphJSON  template.JS
		Warnings   []string
	}{
		ActiveFile: "SAGE.md",
		GraphJSON:  template.JS(`{"nodes":[{"id":"A","label":"Service A"}]}`),
		Warnings:   []string{"Fallback node created for missing reference B"},
	}
	var w strings.Builder
	if err := tmpl.Execute(&w, data); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	rendered := w.String()

	// Modify or expand requiredSubstrings to assert on new DOM elements,
	// updated component IDs, or adjusted CSS classes.
	requiredSubstrings := []string{
		`<!DOCTYPE html>`,
		`/static/css/tailwind.min.css`,
		`/static/js/libs/rough.js`,
		`/static/js/libs/dagre.min.js`,
		`SAGE.md`,
		`Live`,
		`sage-canvas`,
		`Fallback node created for missing reference B`,
		`window.__INITIAL_GRAPH__ = {"nodes":[{"id":"A","label":"Service A"}]};`,
		`Zoom In`,
		`Zoom Out`,
		`Reset`,
		`Export SVG`,
	}

	for _, sub := range requiredSubstrings {
		if !strings.Contains(rendered, sub) {
			t.Errorf("rendered template missing expected substring: %q", sub)
		}
	}
}

