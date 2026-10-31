package d2

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractD2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "fenced with d2",
			input:    "# Header\n```d2\na -> b: test\n```\nFooter",
			expected: "a -> b: test",
		},
		{
			name:     "fenced with sage",
			input:    "```sage\nx -> y: legacy\n```",
			expected: "x -> y: legacy",
		},
		{
			name:     "raw D2 without fences",
			input:    "server -> client: websocket\nclient -> db",
			expected: "server -> client: websocket\nclient -> db",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractD2(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractD2() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCompileBasic(t *testing.T) {
	src := `
server: Web Server {
  shape: rectangle
  style.fill: "#1e3a8a"
}
client: Browser {
  shape: circle
}
client -> server: HTTP Request
`
	res, err := Compile(context.Background(), src, nil)
	if err != nil {
		t.Fatalf("unexpected Compile error: %v", err)
	}

	if len(res.Warnings) > 0 {
		t.Errorf("expected no warnings, got: %v", res.Warnings)
	}

	if !strings.Contains(res.SVG, "<svg") || !strings.Contains(res.SVG, "</svg>") {
		t.Errorf("expected valid SVG output, got: %s", res.SVG)
	}

	if !strings.Contains(res.SVG, "Web Server") {
		t.Errorf("expected SVG to contain node label 'Web Server'")
	}
}

func TestCompileSyntaxErrorHandled(t *testing.T) {
	invalidSrc := `
server: {
  unclosed brace
`
	res, err := Compile(context.Background(), invalidSrc, nil)
	if err != nil {
		t.Fatalf("Compile should return graceful Result on syntax error, got err: %v", err)
	}

	if len(res.Warnings) == 0 {
		t.Errorf("expected warnings on invalid syntax, got none")
	}

	if !strings.Contains(res.SVG, "D2 Diagram Syntax Error") {
		t.Errorf("expected fallback error SVG, got: %s", res.SVG)
	}
}

func TestParseFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.d2")

	content := "a -> b: connects"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	res, err := ParseFile(filePath, nil)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}

	if !strings.Contains(res.SVG, "<svg") {
		t.Errorf("expected SVG from file, got: %s", res.SVG)
	}
}

func TestSAGE_MarkdownCompilesCleanly(t *testing.T) {
	sagePath := filepath.Join("..", "..", "SAGE.md")
	contentBytes, err := os.ReadFile(sagePath)
	if err != nil {
		t.Fatalf("failed to read SAGE.md: %v", err)
	}

	res, err := Compile(context.Background(), string(contentBytes), nil)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	if len(res.Warnings) > 0 {
		t.Errorf("SAGE.md produced warnings: %v", res.Warnings)
	}

	if !strings.Contains(res.SVG, "<svg") {
		t.Errorf("expected valid SVG from SAGE.md, got: %s", res.SVG)
	}
}
