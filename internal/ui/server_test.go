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

	"sage/internal/d2"
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

func TestServer_APIDiagram(t *testing.T) {
	content := `
direction: right
api: API Gateway
auth: Auth Service
api -> auth: validates token
`
	filePath := setupTestDiagram(t, content)
	server := internalui.NewServer(internalui.Config{
		FilePath: filePath,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/diagram", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected application/json, got %s", contentType)
	}

	var res d2.Result
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if !strings.Contains(res.SVG, "<svg") {
		t.Fatalf("expected SVG in response, got: %s", res.SVG)
	}
	if !strings.Contains(res.SVG, "API Gateway") {
		t.Fatalf("expected SVG to contain 'API Gateway'")
	}
}

func TestServer_APIGraph_BackwardCompatibility(t *testing.T) {
	content := "```d2\na -> b: connects\n```"
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

	var res d2.Result
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if !strings.Contains(res.SVG, "<svg") {
		t.Fatalf("expected SVG in response, got: %s", res.SVG)
	}
}

func TestServer_Index(t *testing.T) {
	content := `core: Core Service { shape: rectangle }`
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
	if !strings.Contains(body, "diagram-stage") {
		t.Errorf("expected HTML to contain diagram-stage, got:\n%s", body)
	}
	if !strings.Contains(body, "Core Service") {
		t.Errorf("expected HTML to contain 'Core Service', got:\n%s", body)
	}
}

func TestServer_StaticAssets(t *testing.T) {
	filePath := setupTestDiagram(t, "")
	server := internalui.NewServer(internalui.Config{
		FilePath: filePath,
	})

	staticPaths := []string{
		"/static/css/tailwind.min.css",
		"/static/fonts/Caveat-Regular.ttf",
		"/static/js/panzoom.tpl",
		"/static/js/app.tpl",
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
	filePath := setupTestDiagram(t, `init: Initial Node`)
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
	newContent := "init: Initial Node\nnew: New Node\ninit -> new"
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

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/diagram", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res d2.Result
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(res.Warnings) == 0 {
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
