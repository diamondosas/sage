package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sage/internal/ai"
	"sage/internal/git"
	"sage/internal/github"
	"sage/internal/graph"
)

func TestStaticAssetDelivery(t *testing.T) {
	tmpDir := t.TempDir()
	engine, err := graph.NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test engine: %v", err)
	}
	defer engine.Close()

	srv := NewServer(ServerConfig{
		Engine: engine,
	})

	// 1. Verify index.html
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Sage Architecture Visualizer") {
		t.Fatalf("expected body to contain visualizer title, got: %s", body)
	}
	requiredElements := []string{
		"theme-toggle",
		"search-input",
		"arch-graph",
		"cluster-view",
		"detail-drawer",
	}
	for _, el := range requiredElements {
		if !strings.Contains(body, el) {
			t.Errorf("expected index.html to contain element id %q", el)
		}
	}

	// 2. Verify style.css
	recCSS := httptest.NewRecorder()
	reqCSS := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	srv.Handler().ServeHTTP(recCSS, reqCSS)

	if recCSS.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /style.css, got %d", recCSS.Code)
	}
	cssBody := recCSS.Body.String()
	if !strings.Contains(cssBody, "--bg-primary") || !strings.Contains(cssBody, "--text-primary") {
		t.Errorf("expected style.css to define theme custom properties")
	}

	// 3. Verify app.js
	recJS := httptest.NewRecorder()
	reqJS := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	srv.Handler().ServeHTTP(recJS, reqJS)

	if recJS.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /app.js, got %d", recJS.Code)
	}
	jsBody := recJS.Body.String()
	if !strings.Contains(jsBody, "/api/graph") || !strings.Contains(jsBody, "/api/file") {
		t.Errorf("expected app.js to fetch from /api/graph and /api/file")
	}
}

func TestVisualizerFrontendFeatures(t *testing.T) {
	srv := NewServer(ServerConfig{
		Engine: nil,
	})

	// Test CSS features: Dark mode, drawer transitions, and responsive grid
	recCSS := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recCSS, httptest.NewRequest(http.MethodGet, "/style.css", nil))
	css := recCSS.Body.String()

	cssFeatures := []string{
		`[data-theme="dark"]`,
		`#detail-drawer`,
		`.cluster-node-group`,
		`.files-grid`,
		`.file-card`,
		`.role-badge`,
	}
	for _, feat := range cssFeatures {
		if !strings.Contains(css, feat) {
			t.Errorf("expected style.css to contain %q", feat)
		}
	}

	// Test JS features: SVG layout, click-to-drill-down, search filtering, why lineage, drawer rendering
	recJS := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recJS, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	js := recJS.Body.String()

	jsFeatures := []string{
		"openClusterView",
		"openFileDrawer",
		"performSearch",
		"highlightConnections",
		"renderDrawerDetails",
		"Why Lineage",
	}
	for _, feat := range jsFeatures {
		if !strings.Contains(js, feat) {
			t.Errorf("expected app.js to contain %q", feat)
		}
	}
}

func TestGetGraphEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	engine, err := graph.NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test engine: %v", err)
	}
	defer engine.Close()

	// Seed files into the SQLite database
	analyses := []ai.FileAnalysis{
		{
			Path:         "cmd/sage/main.go",
			Summary:      "CLI entrypoint",
			Role:         "CLI Handler",
			Exports:      []string{"main"},
			Dependencies: []string{"internal/graph"},
			Hash:         "hash1",
			UpdatedAt:    time.Now(),
		},
		{
			Path:         "internal/graph/sqlite.go",
			Summary:      "Graph storage engine",
			Role:         "Knowledge Graph Storage",
			Exports:      []string{"NewEngine", "QueryArchitectureGraph"},
			Dependencies: []string{"internal/ai"},
			Hash:         "hash2",
			UpdatedAt:    time.Now(),
		},
	}
	if err := engine.IngestFileAnalysis(analyses); err != nil {
		t.Fatalf("failed to ingest file analyses: %v", err)
	}

	srv := NewServer(ServerConfig{
		Engine: engine,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected application/json Content-Type, got %s", contentType)
	}

	var res graph.ArchitectureGraph
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if res.TotalFiles != 2 {
		t.Fatalf("expected 2 total files, got %d", res.TotalFiles)
	}

	if len(res.Clusters) == 0 {
		t.Fatalf("expected at least one cluster, got none")
	}

	// Verify edges were extracted
	if len(res.Edges) == 0 {
		t.Fatalf("expected dependency edges between clusters, got 0")
	}
}

func TestGetGraphEndpoint_NilEngine(t *testing.T) {
	srv := NewServer(ServerConfig{
		Engine: nil,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 when engine is nil, got %d", rec.Code)
	}
}

func TestGetFileEndpoint_Success(t *testing.T) {
	tmpDir := t.TempDir()
	engine, err := graph.NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test engine: %v", err)
	}
	defer engine.Close()

	targetFile := "internal/graph/sqlite.go"

	// 1. Ingest File Analysis
	analyses := []ai.FileAnalysis{
		{
			Path:         targetFile,
			Summary:      "Graph persistence layer for SQLite",
			Role:         "Knowledge Graph Storage",
			Exports:      []string{"NewEngine", "QueryWhy", "QueryArchitectureGraph"},
			Dependencies: []string{"database/sql", "sage/internal/ai"},
			Hash:         "hash123",
			UpdatedAt:    time.Now(),
		},
	}
	if err := engine.IngestFileAnalysis(analyses); err != nil {
		t.Fatalf("failed to ingest file analysis: %v", err)
	}

	// 2. Ingest Codeowners
	rules := []git.OwnershipRule{
		{
			Pattern: "internal/graph/*",
			Owners:  []string{"@graph-maintainers"},
		},
	}
	if err := engine.IngestCodeowners(rules); err != nil {
		t.Fatalf("failed to ingest codeowners: %v", err)
	}

	// 3. Ingest Git History & Experts
	commits := []git.Commit{
		{
			SHA:         "c1234567890abcdef",
			Message:     "feat(graph): add sqlite engine",
			AuthorName:  "Grace Hopper",
			AuthorEmail: "grace@navy.mil",
			Timestamp:   time.Now(),
			Files:       []string{targetFile},
		},
	}
	expertise := map[string][]git.ContributorExpertise{
		targetFile: {
			{
				AuthorName:  "Grace Hopper",
				AuthorEmail: "grace@navy.mil",
				Score:       42.5,
				CommitCount: 5,
			},
		},
	}
	if err := engine.IngestGitHistory(commits, expertise); err != nil {
		t.Fatalf("failed to ingest git history: %v", err)
	}

	// 4. Ingest Decisions
	prs := []github.PullRequest{
		{Number: 42, Title: "Adopt SQLite embedded store", URL: "https://github.com/pull/42", Author: "grace"},
	}
	decisions := []ai.Decision{
		{
			ID:        "DEC-042",
			Title:     "SQLite Embedded Architecture",
			Rationale: "Eliminates Docker/network daemon dependencies",
			AppliesTo: []string{targetFile},
		},
	}
	if err := engine.IngestDecisions(prs, decisions); err != nil {
		t.Fatalf("failed to ingest decisions: %v", err)
	}

	srv := NewServer(ServerConfig{
		Engine: engine,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/file?path="+targetFile, nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected application/json Content-Type, got %s", contentType)
	}

	var details FileInspectionDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &details); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if details.Path != targetFile {
		t.Errorf("expected path %s, got %s", targetFile, details.Path)
	}
	if details.Role != "Knowledge Graph Storage" {
		t.Errorf("expected role Knowledge Graph Storage, got %s", details.Role)
	}
	if !strings.Contains(details.Summary, "Graph persistence") {
		t.Errorf("expected summary to contain 'Graph persistence', got %s", details.Summary)
	}
	if len(details.Exports) != 3 {
		t.Errorf("expected 3 exports, got %d", len(details.Exports))
	}
	if len(details.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(details.Dependencies))
	}
	if details.Owner != "@graph-maintainers" {
		t.Errorf("expected owner @graph-maintainers, got %s", details.Owner)
	}
	if len(details.Experts) == 0 || !strings.Contains(details.Experts[0], "Grace Hopper") {
		t.Errorf("expected Grace Hopper in experts, got %v", details.Experts)
	}
	if len(details.Why) == 0 {
		t.Errorf("expected why lineage entries, got empty")
	}
}

func TestGetFileEndpoint_MissingPath(t *testing.T) {
	srv := NewServer(ServerConfig{
		Engine: nil,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/file", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing path, got %d", rec.Code)
	}
}

func TestGetFileEndpoint_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	engine, err := graph.NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test engine: %v", err)
	}
	defer engine.Close()

	srv := NewServer(ServerConfig{
		Engine: engine,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/file?path=nonexistent/file.go", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for nonexistent file, got %d", rec.Code)
	}
}

func TestServer_LifecycleAndGracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	engine, err := graph.NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test engine: %v", err)
	}
	defer engine.Close()

	srv := NewServer(ServerConfig{
		Port:   0, // Ephemeral port
		Host:   "127.0.0.1",
		Engine: engine,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait for server to bind port
	var addr string
	for i := 0; i < 50; i++ {
		addr = srv.Addr()
		if addr != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if addr == "" {
		t.Fatalf("server failed to bind address in time")
	}

	if srv.Port() <= 0 {
		t.Fatalf("expected positive port number, got %d", srv.Port())
	}

	// Verify server responds to live HTTP request
	resp, err := http.Get(fmt.Sprintf("http://%s/api/graph", addr))
	if err != nil {
		t.Fatalf("failed to make HTTP GET request to server: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from running server, got %d", resp.StatusCode)
	}

	// Trigger graceful shutdown via context cancellation
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected clean shutdown, got error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("server did not shut down gracefully within timeout")
	}
}

