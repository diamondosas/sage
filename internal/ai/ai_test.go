package ai

import (
	"testing"
)

func TestComputeFileHash(t *testing.T) {
	c1 := []byte("hello world")
	c2 := []byte("hello world")
	c3 := []byte("hello world 2")

	h1 := ComputeFileHash(c1)
	h2 := ComputeFileHash(c2)
	h3 := ComputeFileHash(c3)

	if h1 != h2 {
		t.Fatalf("expected identical hash for identical content, got %s vs %s", h1, h2)
	}
	if h1 == h3 {
		t.Fatalf("expected different hash for different content")
	}
}

func TestHeuristicAnalyzeFile(t *testing.T) {
	code := []byte(`package graph

import (
	"fmt"
	"strings"
	"sage/internal/ai"
)

// Engine manages database operations.
type Engine struct {
	path string
}

func NewEngine() *Engine {
	return &Engine{}
}
`)

	synth := NewSynthesizer("", "", "", "")
	analysis := synth.AnalyzeFile("internal/graph/engine.go", code)

	if analysis.Path != "internal/graph/engine.go" {
		t.Errorf("expected path internal/graph/engine.go, got %s", analysis.Path)
	}
	if analysis.Role != "Knowledge Graph Storage" {
		t.Errorf("expected role 'Knowledge Graph Storage', got %s", analysis.Role)
	}
	if analysis.Hash == "" {
		t.Errorf("expected non-empty hash")
	}
	if len(analysis.Exports) == 0 {
		t.Errorf("expected detected exports, got 0")
	}
	if len(analysis.Dependencies) == 0 {
		t.Errorf("expected detected dependencies, got 0")
	}
}

func TestIsIndexableSourceFile(t *testing.T) {
	cases := []struct {
		path     string
		expected bool
	}{
		{"main.go", true},
		{"internal/graph/sqlite.go", true},
		{"README.md", true},
		{".github/workflows/build.yml", true},
		{".git/config", false},
		{".github/sage/sage.db", false},
		{"bin/sage", false},
		{"node_modules/pkg/index.js", false},
		{"image.png", false},
	}

	for _, tc := range cases {
		got := IsIndexableSourceFile(tc.path)
		if got != tc.expected {
			t.Errorf("IsIndexableSourceFile(%q) = %v, expected %v", tc.path, got, tc.expected)
		}
	}
}
