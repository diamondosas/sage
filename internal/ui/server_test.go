package ui_test

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sage/internal/parser"
	internalui "sage/internal/ui"
)

// setupTestDiagram creates a temporary SAGE.md file for testing.
func setupTestDiagram(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "SAGE.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test diagram file: %v", err)
	}
	return filePath
}

func TestServer_APIGraph(t *testing.T) {
	content := `
rect auth "Auth Service" [blue]
rect api "API Gateway" [green]
api -> auth "validates token"
`
	filePath := setupTestDiagram(t, content)
	server := internalui.NewServer(internalui.Config{
		FilePath: filePath,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected application/json, got %s", contentType)
	}

	var graph parser.Graph
	if err := json.NewDecoder(rec.Body).Decode(&graph); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if len(graph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(graph.Edges))
	}
	if graph.Nodes[0].ID != "auth" || graph.Nodes[1].ID != "api" {
		t.Fatalf("unexpected nodes: %+v", graph.Nodes)
	}
	if graph.Edges[0].From != "api" || graph.Edges[0].To != "auth" {
		t.Fatalf("unexpected edges: %+v", graph.Edges)
	}
}

// TestServer_APIGraph_MermaidContent verifies that raw Mermaid markup within markdown
// code blocks is cleanly parsed and served via the /api/graph endpoint.
// TWEAK HERE: Modify the Mermaid flowchart snippet below to test alternative diagram
// structures, custom subgraphs, or different directional flows (e.g., flowchart TD).
func TestServer_APIGraph_MermaidContent(t *testing.T) {
	content := "# Architecture\n\n```mermaid\nflowchart LR\n    api[API Gateway] --> auth[Auth Service]\n```"
	filePath := setupTestDiagram(t, content)
	server := internalui.NewServer(internalui.Config{
		FilePath: filePath,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var graph parser.Graph
	if err := json.NewDecoder(rec.Body).Decode(&graph); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if !strings.Contains(graph.Mermaid, "flowchart LR") || !strings.Contains(graph.Mermaid, "api[API Gateway]") {
		t.Fatalf("expected graph.Mermaid to contain flowchart markup, got %q", graph.Mermaid)
	}
}

func TestServer_Index(t *testing.T) {
	content := `rect core "Core Service" [yellow]`
	filePath := setupTestDiagram(t, content)
	server := internalui.NewServer(internalui.Config{
		FilePath: filePath,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "sage-canvas") {
		t.Errorf("expected HTML to contain sage-canvas element, got:\n%s", body)
	}
	if !strings.Contains(body, "Core Service") && !strings.Contains(body, "core") {
		t.Errorf("expected HTML to contain diagram data, got:\n%s", body)
	}
}

func TestServer_StaticAssets(t *testing.T) {
	filePath := setupTestDiagram(t, "")
	server := internalui.NewServer(internalui.Config{
		FilePath: filePath,
	})

	staticPaths := []string{
		"/static/css/tailwind.min.css",
		"/static/js/libs/rough.js",
		"/static/js/libs/dagre.min.js",
		"/static/js/libs/mermaid.min.js",
		"/static/js/sketchify.js",
		"/static/js/panzoom.js",
		"/static/js/annotations.js",
		"/static/js/export.js",
		"/static/fonts/PatrickHand-Regular.ttf",
		"/static/js/app.js",
	}

	for _, path := range staticPaths {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			server.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200 for %s, got %d", path, rec.Code)
			}
			if rec.Body.Len() == 0 {
				t.Fatalf("expected non-empty body for %s", path)
			}
		})
	}
}

func TestServer_SSE_LiveReload(t *testing.T) {
	filePath := setupTestDiagram(t, `rect init "Initial Node"`)
	// Use small watch ticker for fast test execution
	server := internalui.NewServer(internalui.Config{
		FilePath:      filePath,
		CheckInterval: 50 * time.Millisecond,
	})

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	client := ts.Client()
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("failed to create SSE request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SSE request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for SSE, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream Content-Type, got %s", ct)
	}

	// Trigger file update by modifying the file
	time.Sleep(75 * time.Millisecond)
	newContent := `rect init "Initial Node"` + "\n" + `circle new "New Node"`
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		t.Fatalf("failed to update test diagram: %v", err)
	}

	// Read SSE response stream
	reader := bufio.NewReader(resp.Body)
	receivedReload := false

	readChan := make(chan string, 10)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			readChan <- line
		}
	}()

	timeout := time.After(2 * time.Second)
	for !receivedReload {
		select {
		case line := <-readChan:
			if strings.Contains(line, "event: reload") || strings.Contains(line, "reload") {
				receivedReload = true
			}
		case <-timeout:
			t.Fatal("timed out waiting for SSE reload event")
		}
	}
}

func TestServer_NonExistentFile(t *testing.T) {
	server := internalui.NewServer(internalui.Config{
		FilePath: "/path/to/nonexistent/SAGE.md",
	})

	// /api/graph should return 200 with warnings
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var g parser.Graph
	if err := json.NewDecoder(rec.Body).Decode(&g); err != nil {
		t.Fatalf("failed to decode graph: %v", err)
	}
	if len(g.Warnings) == 0 {
		t.Errorf("expected warning for missing file")
	}

	// Index should also serve without crashing
	recIndex := httptest.NewRecorder()
	reqIndex := httptest.NewRequest(http.MethodGet, "/", nil)
	server.Handler().ServeHTTP(recIndex, reqIndex)
	if recIndex.Code != http.StatusOK {
		t.Fatalf("expected status 200 for index, got %d", recIndex.Code)
	}
}

func TestServer_NotFound(t *testing.T) {
	server := internalui.NewServer(internalui.Config{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/non-existent-page", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown path, got %d", rec.Code)
	}
}

