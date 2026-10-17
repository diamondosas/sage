package parser

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// colorPaletteClasses maps SAGE color names to Mermaid classDef styling properties.
// These match the calibrated dark slate palette variables in ui/js/app.js.
var colorPaletteClasses = map[string]string{
	"blue":   "fill:#1e3a8a,stroke:#60a5fa,color:#dbeafe",
	"green":  "fill:#14532d,stroke:#4ade80,color:#dcfce7",
	"red":    "fill:#7f1d1d,stroke:#f87171,color:#fee2e2",
	"yellow": "fill:#713f12,stroke:#facc15,color:#fef9c3",
	"purple": "fill:#581c87,stroke:#c084fc,color:#f3e8ff",
	"orange": "fill:#7c2d12,stroke:#fb923c,color:#ffedd5",
	"gray":   "fill:#334155,stroke:#94a3b8,color:#f1f5f9",
}


// Configuration variables and default visual attributes.
// Change these default variables if alternate behavior or styling is desired:
// - FallbackNodeShape: shape used when synthesizing undeclared edge endpoint references (default: ShapeRect).
// - FallbackNodeColor: palette color used when synthesizing undeclared edge endpoint references (default: "gray").
// - Default shape dimensions (width/height): configured in the visualizer UI layer (see ui/js/app.js: SHAPE_DIMENSIONS).
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

// extractFenceBlock extracts content inside a specific markdown code fence (e.g., ```sage or ```mermaid).
func extractFenceBlock(input, tag string) (string, bool) {
	fence := "```" + tag
	if !strings.Contains(input, fence) {
		return "", false
	}

	var sb strings.Builder
	lines := strings.Split(input, "\n")
	insideFence := false
	foundFence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, fence) {
			insideFence = true
			foundFence = true
			continue
		}
		if insideFence && strings.HasPrefix(trimmed, "```") {
			insideFence = false
			sb.WriteString("\n")
			break
		}
		if insideFence {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	if !foundFence || strings.TrimSpace(sb.String()) == "" {
		return "", false
	}
	return sb.String(), true
}

// extractContent extracts text within ```sage ... ``` markdown code blocks if present.
// If no code fence is found or the fence contains no statements, it returns the raw input unmodified.
func extractContent(input string) string {
	if content, ok := extractFenceBlock(input, "sage"); ok {
		return content
	}
	return input
}

// extractMermaidBlock extracts Mermaid flowchart definitions from ```mermaid ... ``` code blocks
// or detects raw Mermaid declarations (e.g., lines starting with flowchart or graph, optionally preceded by frontmatter or comments).
func extractMermaidBlock(input string) string {
	if content, ok := extractFenceBlock(input, "mermaid"); ok {
		return strings.TrimSpace(content)
	}

	trimmed := strings.TrimSpace(input)
	checkText := trimmed
	if strings.HasPrefix(checkText, "---") {
		parts := strings.SplitN(checkText[3:], "---", 2)
		if len(parts) == 2 {
			checkText = strings.TrimSpace(parts[1])
		}
	}

	lines := strings.Split(checkText, "\n")
	for _, line := range lines {
		tl := strings.TrimSpace(line)
		if tl == "" || strings.HasPrefix(tl, "%%") {
			continue
		}
		if strings.HasPrefix(tl, "flowchart") || strings.HasPrefix(tl, "graph") {
			return trimmed
		}
		break
	}

	return ""
}

// validateMermaid checks Mermaid markup for valid flowchart headers and balanced delimiters,
// collecting non-fatal syntax errors into a warnings slice.
func validateMermaid(source string) []string {
	var warnings []string
	lines := strings.Split(source, "\n")
	var headerLine string
	inFrontmatter := false
	frontmatterDone := false

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		if trimmed == "---" && !frontmatterDone {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				inFrontmatter = false
				frontmatterDone = true
				continue
			}
		}
		if inFrontmatter {
			continue
		}
		headerLine = trimmed
		break
	}

	if headerLine == "" || (!strings.HasPrefix(headerLine, "flowchart") && !strings.HasPrefix(headerLine, "graph")) {
		warnings = append(warnings, fmt.Sprintf("unsupported or missing flowchart header: expected 'flowchart [direction]' or 'graph [direction]', got %q", headerLine))
	}

	// Delimiter and quote balancing checks
	openBrackets := 0
	openParens := 0
	openBraces := 0

	for lineIdx, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if strings.HasPrefix(trimmed, "%%") || trimmed == "---" {
			continue
		}

		quoteCount := 0
		inQuotes := false
		escaped := false

		for i := 0; i < len(trimmed); i++ {
			ch := trimmed[i]
			if ch == '\\' {
				escaped = !escaped
				continue
			}
			if ch == '"' && !escaped {
				inQuotes = !inQuotes
				quoteCount++
				escaped = false
				continue
			}
			escaped = false

			if inQuotes {
				continue
			}

			// Check for asymmetric node shape opening: e.g., worker>"label"]
			// Where an alphanumeric identifier precedes '>', it acts as opening bracket balanced by ']'
			if ch == '>' && i > 0 && isNodeIdentChar(trimmed[i-1]) {
				openBrackets++
				continue
			}

			switch ch {
			case '[':
				openBrackets++
			case ']':
				openBrackets--
			case '(':
				openParens++
			case ')':
				openParens--
			case '{':
				openBraces++
			case '}':
				openBraces--
			}
		}

		if quoteCount%2 != 0 {
			warnings = append(warnings, fmt.Sprintf("unclosed quote on line %d: %s", lineIdx+1, trimmed))
		}
	}

	if openBrackets != 0 {
		warnings = append(warnings, fmt.Sprintf("unbalanced bracket delimiter '[': difference count %d", openBrackets))
	}
	if openParens != 0 {
		warnings = append(warnings, fmt.Sprintf("unbalanced parenthesis delimiter '(': difference count %d", openParens))
	}
	if openBraces != 0 {
		warnings = append(warnings, fmt.Sprintf("unbalanced brace delimiter '{': difference count %d", openBraces))
	}

	return warnings
}

func isNodeIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// Parse extracts the architectural Graph from raw or markdown-fenced SAGE DSL content.
// It parses shapes, edges, groups, and paths, collecting non-fatal syntax errors into Graph.Warnings.
func Parse(input string) *Graph {
	g := &Graph{
		Nodes:    []Node{},
		Edges:    []Edge{},
		Warnings: []string{},
	}

	// If the document contains Mermaid markup, validate and return it.
	if mermaidSrc := extractMermaidBlock(input); mermaidSrc != "" {
		g.Mermaid = mermaidSrc
		g.Warnings = validateMermaid(mermaidSrc)
		return g
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

	// Transpile legacy DSL AST into valid Mermaid flowchart markup
	if len(g.Nodes) > 0 || len(g.Edges) > 0 {
		g.Mermaid = transpileToMermaid(g)
	}

	return g
}

// formatMermaidNode renders a Node into its corresponding Mermaid flowchart shape declaration.
func formatMermaidNode(node Node) string {
	cleanLabel := node.Label
	if cleanLabel == "" {
		cleanLabel = node.ID
	}
	var shapeFmt string
	switch node.Shape {
	case ShapeCircle:
		shapeFmt = fmt.Sprintf("%s((\"%s\"))", node.ID, cleanLabel)
	case ShapeDiamond:
		shapeFmt = fmt.Sprintf("%s{\"%s\"}", node.ID, cleanLabel)
	case ShapeTriangle:
		shapeFmt = fmt.Sprintf("%s>\"%s\"]", node.ID, cleanLabel)
	case ShapeCylinder:
		shapeFmt = fmt.Sprintf("%s[(\"%s\")]", node.ID, cleanLabel)
	case ShapeRect:
		fallthrough
	default:
		shapeFmt = fmt.Sprintf("%s[\"%s\"]", node.ID, cleanLabel)
	}

	if node.Color != "" {
		shapeFmt += fmt.Sprintf(":::%s", strings.ToLower(node.Color))
	}
	return shapeFmt
}

// formatMermaidEdge renders an Edge into Mermaid connection syntax.
func formatMermaidEdge(edge Edge) string {
	arrow := "-->"
	if edge.Bidirectional {
		arrow = "<-->"
	}
	if edge.Label != "" {
		return fmt.Sprintf("%s %s|\"%s\"| %s", edge.From, arrow, edge.Label, edge.To)
	}
	return fmt.Sprintf("%s %s %s", edge.From, arrow, edge.To)
}

// sanitizeSubgraphID produces a clean identifier for a Mermaid subgraph.
func sanitizeSubgraphID(name string, index int) string {
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	cleanBase := strings.Trim(sb.String(), "_")
	if cleanBase == "" {
		return fmt.Sprintf("subgraph_%d", index)
	}
	return fmt.Sprintf("sg_%s", cleanBase)
}

// transpileToMermaid converts the parsed Graph AST into standard Mermaid flowchart markup.
func transpileToMermaid(g *Graph) string {
	if len(g.Nodes) == 0 && len(g.Edges) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("flowchart LR\n")

	// Group nodes by architectural domain clusters
	var groupOrder []string
	groupedNodes := make(map[string][]Node)
	var ungroupedNodes []Node
	usedColors := make(map[string]bool)

	for _, node := range g.Nodes {
		if node.Color != "" {
			usedColors[strings.ToLower(node.Color)] = true
		}
		if node.Group == "" {
			ungroupedNodes = append(ungroupedNodes, node)
		} else {
			if _, exists := groupedNodes[node.Group]; !exists {
				groupOrder = append(groupOrder, node.Group)
			}
			groupedNodes[node.Group] = append(groupedNodes[node.Group], node)
		}
	}

	for idx, groupName := range groupOrder {
		sgID := sanitizeSubgraphID(groupName, idx)
		sb.WriteString(fmt.Sprintf("    subgraph %s [\"%s\"]\n", sgID, groupName))
		for _, node := range groupedNodes[groupName] {
			sb.WriteString("        ")
			sb.WriteString(formatMermaidNode(node))
			sb.WriteString("\n")
		}
		sb.WriteString("    end\n")
	}

	for _, node := range ungroupedNodes {
		sb.WriteString("    ")
		sb.WriteString(formatMermaidNode(node))
		sb.WriteString("\n")
	}

	for _, edge := range g.Edges {
		sb.WriteString("    ")
		sb.WriteString(formatMermaidEdge(edge))
		sb.WriteString("\n")
	}

	// Emit classDef definitions for colors used in diagram
	var sortedColors []string
	for c := range usedColors {
		sortedColors = append(sortedColors, c)
	}
	sort.Strings(sortedColors)

	for _, colorName := range sortedColors {
		if style, ok := colorPaletteClasses[colorName]; ok {
			sb.WriteString(fmt.Sprintf("    classDef %s %s\n", colorName, style))
		}
	}

	return strings.TrimSpace(sb.String())
}

// ParseFile reads a SAGE DSL file from disk and parses its content into a Graph.
func ParseFile(filePath string) (*Graph, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return Parse(string(bytes)), nil
}

