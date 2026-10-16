package parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Fallback configuration for undefined node references.
// Change these default variables if a different fallback shape or accent color is preferred.
const (
	FallbackNodeShape = ShapeRect
	FallbackNodeColor = "gray"
)

// Regex patterns for parsing SAGE DSL statements.
// You can adjust these regular expressions to tweak token rules or add custom syntax.
var (
	// Matches: <shape> <id> "<label>" [optional_color]
	nodeRegex = regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s+([a-zA-Z0-9_-]+)\s+"([^"]*)"(?:\s+\[([a-zA-Z0-9_-]+)\])?$`)
	// Matches: <from_id> -> <to_id> [optional "label"] OR <from_id> <-> <to_id> [optional "label"]
	// Change or expand this regex if new arrow types (e.g., dashed "-->" or dotted "..>") are needed.
	edgeRegex = regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s*(<->|->)\s*([a-zA-Z0-9_-]+)(?:\s+"([^"]*)")?$`)
)

// normalizeShape maps shape aliases to their canonical ShapeType.
// Change or expand this map if you want to support new shape aliases (e.g., "storage" -> ShapeCylinder).
func normalizeShape(raw string) (ShapeType, bool) {
	switch strings.ToLower(raw) {
	case "rect", "box":
		return ShapeRect, true
	case "circle":
		return ShapeCircle, true
	case "diamond":
		return ShapeDiamond, true
	case "triangle":
		return ShapeTriangle, true
	case "cylinder", "db":
		return ShapeCylinder, true
	default:
		return "", false
	}
}

// extractContent extracts text within ```sage ... ``` markdown code blocks if present.
// If no code fence is found, it returns the raw input unmodified.
func extractContent(input string) string {
	if !strings.Contains(input, "```sage") {
		return input
	}

	var sb strings.Builder
	lines := strings.Split(input, "\n")
	insideFence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```sage") {
			insideFence = true
			continue
		}
		if insideFence && strings.HasPrefix(trimmed, "```") {
			insideFence = false
			sb.WriteString("\n")
			continue
		}
		if insideFence {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// Parse extracts the architectural Graph from raw or markdown-fenced SAGE DSL content.
// It parses shapes, edges, groups, and paths, collecting non-fatal syntax errors into Graph.Warnings.
func Parse(input string) *Graph {
	g := &Graph{
		Nodes:    []Node{},
		Edges:    []Edge{},
		Warnings: []string{},
	}

	content := extractContent(input)
	lines := strings.Split(content, "\n")

	// State tracking across lines.
	// currentGroup assigns the active architectural cluster for subsequent nodes.
	currentGroup := ""
	// currentPath assigns the active file/directory path for subsequent nodes.
	currentPath := ""

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		// Check for group directives: e.g., "// group: Backend" or "# group: Core"
		// Change the prefix here if you want alternative directive tokens like "@group".
		if strings.HasPrefix(line, "// group:") || strings.HasPrefix(line, "# group:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentGroup = strings.TrimSpace(parts[1])
			}
			continue
		}

		// Check for path directives: e.g., "// path: internal/api" or "# path: cmd/cli"
		// Change the prefix here if you want alternative directive tokens like "@path".
		if strings.HasPrefix(line, "// path:") || strings.HasPrefix(line, "# path:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentPath = strings.TrimSpace(parts[1])
			}
			continue
		}

		// Ignore single-line comments.
		// Change comment prefixes here if alternative comment styles (like "--" or ";") are desired.
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse node shape definitions
		if match := nodeRegex.FindStringSubmatch(line); match != nil {
			shapeRaw := match[1]
			id := match[2]
			label := match[3]
			color := match[4]

			shape, ok := normalizeShape(shapeRaw)
			if !ok {
				g.Warnings = append(g.Warnings, fmt.Sprintf("unsupported shape '%s' on line: %s", shapeRaw, line))
				continue
			}

			g.Nodes = append(g.Nodes, Node{
				ID:       id,
				Shape:    shape,
				Label:    label,
				Color:    color,
				Group:    currentGroup,
				FilePath: currentPath,
			})
			continue
		}

		// Parse connection statements
		if match := edgeRegex.FindStringSubmatch(line); match != nil {
			from := match[1]
			arrow := match[2]
			to := match[3]
			label := match[4]

			g.Edges = append(g.Edges, Edge{
				From:          from,
				To:            to,
				Label:         label,
				Bidirectional: arrow == "<->",
			})
			continue
		}

		// Collect any unrecognized lines into warnings without crashing
		g.Warnings = append(g.Warnings, fmt.Sprintf("unrecognized line: %s", line))
	}

	// Validate edge endpoints and synthesize fallback nodes for undeclared references.
	nodeExists := make(map[string]bool)
	for _, node := range g.Nodes {
		nodeExists[node.ID] = true
	}

	ensureNode := func(id string) {
		if !nodeExists[id] {
			nodeExists[id] = true
			g.Nodes = append(g.Nodes, Node{
				ID:    id,
				Shape: FallbackNodeShape,
				Label: id,
				Color: FallbackNodeColor,
			})
			g.Warnings = append(g.Warnings, fmt.Sprintf("synthesized fallback node for undefined reference: %s", id))
		}
	}

	for _, edge := range g.Edges {
		ensureNode(edge.From)
		ensureNode(edge.To)
	}

	return g
}

// ParseFile reads a SAGE DSL file from disk and parses its content into a Graph.
func ParseFile(filePath string) (*Graph, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return Parse(string(bytes)), nil
}
