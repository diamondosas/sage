package ui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"sage/internal/d2"
	"sage/ui"
)

// DefaultServerConfig defines standard parameters for the embedded UI web server.
// ACTIONABLE: Adjust these constants to change default diagram file name or polling interval.
const (
	DefaultFilePath      = "SAGE.md"
	DefaultCheckInterval = 500 * time.Millisecond // SSE file watch polling frequency
)

// Config configures the embedded HTTP UI server.
type Config struct {
	// FilePath is the path to the SAGE.md diagram file. Defaults to "SAGE.md".
	FilePath string
	// CheckInterval controls how often the SSE watcher checks for file modifications.
	CheckInterval time.Duration
}

// Server provides the embedded web server, SSE live-reload, and D2 diagram endpoints.
type Server struct {
	filePath      string
	checkInterval time.Duration
	mux           *http.ServeMux
}

// NewServer initializes a new Server with the provided configuration and registers all routes.
func NewServer(cfg Config) *Server {
	filePath := cfg.FilePath
	if filePath == "" {
		filePath = DefaultFilePath
	}

	checkInterval := cfg.CheckInterval
	if checkInterval <= 0 {
		checkInterval = DefaultCheckInterval
	}

	s := &Server{
		filePath:      filePath,
		checkInterval: checkInterval,
		mux:           http.NewServeMux(),
	}

	s.routes()
	return s
}

// Handler returns the underlying http.Handler with all registered routes.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// routes registers the application endpoints.
func (s *Server) routes() {
	// Root endpoint: renders the full diagram page using ui/html/base.tpl
	s.mux.HandleFunc("/", s.handleIndex)

	// API endpoint: returns the parsed D2 Result as JSON
	s.mux.HandleFunc("/api/diagram", s.handleAPIDiagram)
	s.mux.HandleFunc("/api/graph", s.handleAPIDiagram) // backward compatibility

	// Live-reload endpoint: Server-Sent Events stream alerting browser on file modification
	s.mux.HandleFunc("/api/events", s.handleSSE)

	// Static assets: serves embedded CSS, fonts, and template scripts
	s.mux.Handle("/static/", http.StripPrefix("/static/", ui.StaticFileServer()))

	// Screenshot synchronization barrier:
	// Coordinates between browser render completion and headless browser screenshot capture.
	var (
		renderReadyMu sync.Mutex
		renderReadyCh = make(chan struct{})
	)

	// Endpoint called by app.tpl as soon as diagram render is complete
	s.mux.HandleFunc("/api/ready", func(w http.ResponseWriter, r *http.Request) {
		renderReadyMu.Lock()
		select {
		case <-renderReadyCh:
		default:
			close(renderReadyCh)
		}
		renderReadyMu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	// Script endpoint loaded at the bottom of base.tpl that holds response until render is complete
	// ACTIONABLE: Maximum timeout before fallback (default 4 seconds)
	s.mux.HandleFunc("/api/wait", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-renderReadyCh:
			time.Sleep(100 * time.Millisecond)
		case <-time.After(4 * time.Second):
		}
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write([]byte("// render sync complete\n"))
	})

	// Debug SVG capture endpoint
	s.mux.HandleFunc("/api/debug-svg", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			_ = os.WriteFile("/tmp/sage_rendered.svg", body, 0644)
		}
		w.WriteHeader(http.StatusOK)
	})
}

// loadDiagram reads the diagram file from disk and compiles it with D2.
func (s *Server) loadDiagram() *d2.Result {
	res, err := d2.ParseFile(s.filePath, nil)
	if err != nil {
		return &d2.Result{
			SVG:      "",
			Warnings: []string{fmt.Sprintf("Diagram file '%s' could not be read: %v", s.filePath, err)},
		}
	}
	return res
}

// handleIndex serves the base HTML template populated with parsed D2 diagram SVG.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	result := s.loadDiagram()

	tmpl, err := ui.ParseBaseTemplate()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse template: %v", err), http.StatusInternalServerError)
		return
	}

	data := struct {
		ActiveFile string
		FilePath   string
		DiagramSVG template.HTML
		Warnings   []string
	}{
		ActiveFile: s.filePath,
		FilePath:   s.filePath,
		DiagramSVG: template.HTML(result.SVG),
		Warnings:   result.Warnings,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to render page: %v", err), http.StatusInternalServerError)
	}
}

// handleAPIDiagram returns the parsed D2 result as JSON.
func (s *Server) handleAPIDiagram(w http.ResponseWriter, r *http.Request) {
	result := s.loadDiagram()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode JSON: %v", err), http.StatusInternalServerError)
	}
}

// handleSSE manages the Server-Sent Events stream, pinging the client whenever SAGE.md is touched.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	var lastMod time.Time
	if fi, err := os.Stat(s.filePath); err == nil {
		lastMod = fi.ModTime()
	}

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fi, err := os.Stat(s.filePath)
			if err != nil {
				continue
			}

			if fi.ModTime().After(lastMod) {
				lastMod = fi.ModTime()
				_, _ = fmt.Fprintf(w, "event: reload\ndata: {}\n\n")
				flusher.Flush()
			}
		}
	}
}
