package ui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"sage/internal/parser"
	"sage/ui"
)

// DefaultServerConfig defines standard parameters for the embedded UI web server.
// Variables here can be adjusted to change default file paths and polling intervals.
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

// Server provides the embedded web server, SSE live-reload, and JSON graph endpoints.
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

	// API endpoint: returns the parsed AST Graph as JSON
	s.mux.HandleFunc("/api/graph", s.handleAPIGraph)

	// Live-reload endpoint: Server-Sent Events stream alerting browser on file modification
	s.mux.HandleFunc("/api/events", s.handleSSE)

	// Static assets: serves embedded CSS, Rough.js, Dagre.js, and app.js
	// Change the prefix "/static/" here if assets need to be hosted under a different path.
	s.mux.Handle("/static/", http.StripPrefix("/static/", ui.StaticFileServer()))
}

// loadGraph reads the diagram file from disk and parses it into a Graph AST.
// If the file does not exist, it synthesizes an initial empty graph with a warning.
func (s *Server) loadGraph() *parser.Graph {
	g, err := parser.ParseFile(s.filePath)
	if err != nil {
		return &parser.Graph{
			Nodes:    []parser.Node{},
			Edges:    []parser.Edge{},
			Warnings: []string{fmt.Sprintf("Diagram file '%s' could not be read: %v", s.filePath, err)},
		}
	}
	return g
}

// handleIndex serves the base HTML template populated with parsed diagram data.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	graph := s.loadGraph()
	graphBytes, err := json.Marshal(graph)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode graph: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl, err := ui.ParseBaseTemplate()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse template: %v", err), http.StatusInternalServerError)
		return
	}

	data := struct {
		ActiveFile string
		FilePath   string
		Graph      *parser.Graph
		GraphJSON  template.JS
		Warnings   []string
	}{
		ActiveFile: s.filePath,
		FilePath:   s.filePath,
		Graph:      graph,
		GraphJSON:  template.JS(graphBytes),
		Warnings:   graph.Warnings,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to render page: %v", err), http.StatusInternalServerError)
	}
}

// handleAPIGraph returns the parsed architecture graph as JSON.
func (s *Server) handleAPIGraph(w http.ResponseWriter, r *http.Request) {
	graph := s.loadGraph()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(graph); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode graph JSON: %v", err), http.StatusInternalServerError)
	}
}

// handleSSE manages the Server-Sent Events stream, pinging the client whenever SAGE.md is touched.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// SSE response headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	// Initial file modification time
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

			// If file was modified, notify client to re-fetch and re-render
			if fi.ModTime().After(lastMod) {
				lastMod = fi.ModTime()
				_, _ = fmt.Fprintf(w, "event: reload\ndata: {}\n\n")
				flusher.Flush()
			}
		}
	}
}
