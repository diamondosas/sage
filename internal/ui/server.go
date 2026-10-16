package ui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"sage/internal/graph"
)

//go:embed assets/*
var embeddedAssets embed.FS

// FileInspectionDetails contains comprehensive file inspection information.
type FileInspectionDetails struct {
	Path         string   `json:"path"`
	Role         string   `json:"role"`
	Summary      string   `json:"summary"`
	Exports      []string `json:"exports"`
	Dependencies []string `json:"dependencies"`
	Owner        string   `json:"owner"`
	Experts      []string `json:"experts"`
	Why          []string `json:"why"`
}

// ServerConfig configures the embedded UI web server.
type ServerConfig struct {
	Port    int
	Host    string
	Engine  *graph.Engine
	OnReady func(addr string)
}

// Server provides the embedded UI HTTP server.
type Server struct {
	config     ServerConfig
	mux        *http.ServeMux
	httpServer *http.Server
	listener   net.Listener
	addr       string
	mu         sync.RWMutex
}

// NewServer creates a new UI server instance.
func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		config: cfg,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// Handler returns the underlying http.Handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Addr returns the network address the server is listening on (e.g. "127.0.0.1:8080").
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

// Port returns the bound port number the server is listening on.
func (s *Server) Port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return s.config.Port
	}
	tcpAddr, ok := s.listener.Addr().(*net.TCPAddr)
	if ok {
		return tcpAddr.Port
	}
	return s.config.Port
}

// Start begins listening on the configured host and port and serves HTTP requests.
// It gracefully shuts down when ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	host := s.config.Host
	if host == "" {
		host = "127.0.0.1"
	}
	bindAddr := fmt.Sprintf("%s:%d", host, s.config.Port)

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return fmt.Errorf("failed to bind to %s: %w", bindAddr, err)
	}

	s.mu.Lock()
	s.listener = ln
	s.addr = ln.Addr().String()
	s.httpServer = &http.Server{
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	s.mu.Unlock()

	// Handle graceful shutdown via context cancellation
	stopDone := make(chan struct{})
	defer close(stopDone)

	go func() {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.Shutdown(shutdownCtx)
		case <-stopDone:
		}
	}()

	if err := s.httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server error: %w", err)
	}

	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	httpSrv := s.httpServer
	s.mu.RUnlock()

	if httpSrv == nil {
		return nil
	}
	return httpSrv.Shutdown(ctx)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/graph", s.handleGraph)
	s.mux.HandleFunc("/api/file", s.handleFile)

	assetsSub, err := fs.Sub(embeddedAssets, "assets")
	if err == nil {
		fileServer := http.FileServer(http.FS(assetsSub))
		s.mux.Handle("/", fileServer)
	}
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.config.Engine == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "database engine not initialized"})
		return
	}

	archGraph, err := s.config.Engine.QueryArchitectureGraph()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to query architecture graph: %v", err)})
		return
	}

	if err := json.NewEncoder(w).Encode(archGraph); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to encode json: %v", err)})
		return
	}
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "path query parameter is required"})
		return
	}

	if s.config.Engine == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "database engine not initialized"})
		return
	}

	cleanPath := strings.TrimPrefix(path, "/")

	analysis, err := s.config.Engine.GetFileAnalysis(cleanPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to query file analysis: %v", err)})
		return
	}

	owner, experts, err := s.config.Engine.QueryOwner(cleanPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to query file owner: %v", err)})
		return
	}

	why, err := s.config.Engine.QueryWhy(cleanPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to query file why lineage: %v", err)})
		return
	}

	isUnassignedOwner := owner == "" || owner == "Unassigned (Check CODEOWNERS)"
	if analysis == nil && len(why) == 0 && isUnassignedOwner && len(experts) == 0 {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("file %s not found in knowledge base", path)})
		return
	}

	details := FileInspectionDetails{
		Path:         path,
		Owner:        owner,
		Exports:      make([]string, 0),
		Dependencies: make([]string, 0),
		Experts:      make([]string, 0),
		Why:          make([]string, 0),
	}

	if analysis != nil {
		details.Role = analysis.Role
		details.Summary = analysis.Summary
		if len(analysis.Exports) > 0 {
			details.Exports = analysis.Exports
		}
		if len(analysis.Dependencies) > 0 {
			details.Dependencies = analysis.Dependencies
		}
	}

	if len(experts) > 0 {
		details.Experts = experts
	}

	if len(why) > 0 {
		details.Why = why
	}

	if err := json.NewEncoder(w).Encode(details); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to encode json: %v", err)})
		return
	}
}
