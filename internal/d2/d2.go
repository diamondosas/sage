package d2

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/d2lang/d2/d2graph"
	"github.com/d2lang/d2/d2layouts/d2dagrelayout"
	"github.com/d2lang/d2/d2lib"
	"github.com/d2lang/d2/d2renderers/d2svg"
	"github.com/d2lang/d2/d2themes/d2themescatalog"
	"github.com/d2lang/d2/lib/textmeasure"
	"github.com/d2lang/util-go/go2"
)

// =========================================================================
// ACTIONABLE CONFIGURATION VARIABLES
// Tweak these variables to change default rendering behavior:
// - DefaultSketch: set to false for crisp straight lines instead of hand-drawn look.
// - DefaultThemeID: light theme ID (d2themescatalog.NeutralDefault = 0).
// - DefaultDarkThemeID: dark theme ID (d2themescatalog.DarkMauve = 100).
// - DefaultPadding: padding around diagram in pixels.
// =========================================================================
const (
	DefaultSketch  = true
	DefaultPadding = int64(24)
)

var (
	DefaultThemeID     = d2themescatalog.NeutralDefault.ID // Neutral Default (0)
	DefaultDarkThemeID = d2themescatalog.DarkMauve.ID      // Dark Mauve (100)
)

// Options allows configuring the D2 rendering pipeline.
type Options struct {
	Sketch      *bool
	ThemeID     *int64
	DarkThemeID *int64
	Padding     *int64
}

// Result holds the compiled diagram SVG and any non-fatal syntax or layout warnings.
type Result struct {
	SVG      string   `json:"svg"`
	Warnings []string `json:"warnings,omitempty"`
}

// ExtractD2 extracts D2 diagram markup from a markdown document.
// It prioritizes ```d2 blocks, falls back to ```sage blocks, and defaults
// to returning the raw input if no fences are found.
func ExtractD2(input string) string {
	for _, fence := range []string{"```d2", "```sage"} {
		if idx := strings.Index(input, fence); idx != -1 {
			start := idx + len(fence)
			if end := strings.Index(input[start:], "```"); end != -1 {
				extracted := strings.TrimSpace(input[start : start+end])
				if extracted != "" {
					return extracted
				}
			}
		}
	}
	// No fenced code block found; return raw input
	return strings.TrimSpace(input)
}

// Compile compiles D2 source script into an interactive SVG string.
func Compile(ctx context.Context, source string, opts *Options) (*Result, error) {
	d2Source := ExtractD2(source)
	if d2Source == "" {
		return &Result{
			SVG:      "",
			Warnings: []string{"Diagram source is empty."},
		}, nil
	}

	// Initialize text ruler for font measurements
	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize D2 text ruler: %w", err)
	}

	// Resolve compile options with Dagre layout engine
	compileOpts := &d2lib.CompileOptions{
		Ruler: ruler,
		LayoutResolver: func(engine string) (d2graph.LayoutGraph, error) {
			return d2dagrelayout.DefaultLayout, nil
		},
	}

	// Resolve rendering options
	sketch := DefaultSketch
	padding := DefaultPadding
	themeID := DefaultThemeID
	darkThemeID := DefaultDarkThemeID

	if opts != nil {
		if opts.Sketch != nil {
			sketch = *opts.Sketch
		}
		if opts.Padding != nil {
			padding = *opts.Padding
		}
		if opts.ThemeID != nil {
			themeID = *opts.ThemeID
		}
		if opts.DarkThemeID != nil {
			darkThemeID = *opts.DarkThemeID
		}
	}

	renderOpts := &d2svg.RenderOpts{
		Pad:         go2.Pointer(padding),
		Sketch:      go2.Pointer(sketch),
		ThemeID:     go2.Pointer(themeID),
		DarkThemeID: go2.Pointer(darkThemeID),
	}

	// Compile D2 graph AST
	diagram, _, err := d2lib.Compile(ctx, d2Source, compileOpts, renderOpts)
	if err != nil {
		return &Result{
			SVG:      renderErrorSVG(err.Error()),
			Warnings: []string{fmt.Sprintf("D2 compile error: %v", err)},
		}, nil
	}

	// Render diagram to SVG
	svgBytes, err := d2svg.Render(diagram, renderOpts)
	if err != nil {
		return &Result{
			SVG:      renderErrorSVG(err.Error()),
			Warnings: []string{fmt.Sprintf("D2 render error: %v", err)},
		}, nil
	}

	return &Result{
		SVG: string(svgBytes),
	}, nil
}

// ParseFile reads a diagram file from disk and compiles it into an SVG Result.
func ParseFile(filePath string, opts *Options) (*Result, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return &Result{
			SVG:      renderErrorSVG(fmt.Sprintf("Could not read file %s: %v", filePath, err)),
			Warnings: []string{fmt.Sprintf("Could not read file '%s': %v", filePath, err)},
		}, nil
	}

	return Compile(context.Background(), string(data), opts)
}

// renderErrorSVG creates a fallback SVG notification when compilation encounters an error.
func renderErrorSVG(msg string) string {
	escaped := strings.ReplaceAll(msg, "<", "&lt;")
	escaped = strings.ReplaceAll(escaped, ">", "&gt;")
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="600" height="120" viewBox="0 0 600 120">
  <rect width="100%%" height="100%%" fill="#2d1515" rx="8" stroke="#f87171" stroke-width="2"/>
  <text x="20" y="40" fill="#f87171" font-family="sans-serif" font-weight="bold" font-size="16">D2 Diagram Syntax Error</text>
  <text x="20" y="70" fill="#fca5a5" font-family="monospace" font-size="13">%s</text>
</svg>`, escaped)
}
