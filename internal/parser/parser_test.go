package parser

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestParseNodes verifies parsing of shape declarations, aliases, labels, and color tags.
func TestParseNodes(t *testing.T) {
	input := `
rect auth "Authentication Service" [blue]
circle cache "Redis Cache"
diamond gateway "API Gateway" [red]
triangle worker "Background Worker"
cylinder db1 "Main Database" [green]
box service2 "Order Service"
db db2 "Analytics Database"
`
	expected := []Node{
		{ID: "auth", Shape: ShapeRect, Label: "Authentication Service", Color: "blue"},
		{ID: "cache", Shape: ShapeCircle, Label: "Redis Cache"},
		{ID: "gateway", Shape: ShapeDiamond, Label: "API Gateway", Color: "red"},
		{ID: "worker", Shape: ShapeTriangle, Label: "Background Worker"},
		{ID: "db1", Shape: ShapeCylinder, Label: "Main Database", Color: "green"},
		{ID: "service2", Shape: ShapeRect, Label: "Order Service"},
		{ID: "db2", Shape: ShapeCylinder, Label: "Analytics Database"},
	}

	g := Parse(input)
	if !reflect.DeepEqual(g.Nodes, expected) {
		t.Fatalf("expected nodes:\n%+v\ngot:\n%+v", expected, g.Nodes)
	}
}

// TestParseFencedBlock tests extracting DSL statements inside markdown ```sage blocks.
func TestParseFencedBlock(t *testing.T) {
	input := `# Architecture Overview
rect outside "Should Not Be Parsed"

Here is our system architecture:

` + "```sage" + `
rect api "API"
rect db "DB"
` + "```" + `

rect alsoOutside "Should Not Be Parsed"
`
	expected := []Node{
		{ID: "api", Shape: ShapeRect, Label: "API"},
		{ID: "db", Shape: ShapeRect, Label: "DB"},
	}

	g := Parse(input)
	if !reflect.DeepEqual(g.Nodes, expected) {
		t.Fatalf("expected nodes:\n%+v\ngot:\n%+v", expected, g.Nodes)
	}
	// Verify non-diagram text outside the block was not recorded as warnings
	if len(g.Warnings) > 0 {
		t.Fatalf("expected no warnings for markdown outside fence, got: %v", g.Warnings)
	}
}

// TestParseMermaidBlock tests extracting Mermaid flowchart blocks.
func TestParseMermaidBlock(t *testing.T) {
	input := `# Architecture Overview
Here is the system:

` + "```mermaid" + `
flowchart LR
    frontend[Web App] --> api[API Gateway]
    api --> db[(PostgreSQL)]
` + "```" + `
`
	g := Parse(input)
	if g.Mermaid == "" {
		t.Fatalf("expected Mermaid content to be extracted, got empty")
	}
	if !strings.Contains(g.Mermaid, "flowchart LR") {
		t.Errorf("expected Mermaid content to contain 'flowchart LR', got: %s", g.Mermaid)
	}
	if !strings.Contains(g.Mermaid, "frontend[Web App] --> api[API Gateway]") {
		t.Errorf("expected Mermaid content to contain node connection, got: %s", g.Mermaid)
	}
	if len(g.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got: %v", g.Warnings)
	}
}

// TestParseEdges verifies directed and bidirectional connections with or without labels.
func TestParseEdges(t *testing.T) {
	input := `
rect a "Service A"
rect b "Service B"
rect c "Service C"

a -> b "calls RPC"
b <-> c "syncs data"
a -> c
`
	expectedEdges := []Edge{
		{From: "a", To: "b", Label: "calls RPC", Bidirectional: false},
		{From: "b", To: "c", Label: "syncs data", Bidirectional: true},
		{From: "a", To: "c", Label: "", Bidirectional: false},
	}

	g := Parse(input)
	if !reflect.DeepEqual(g.Edges, expectedEdges) {
		t.Fatalf("expected edges:\n%+v\ngot:\n%+v", expectedEdges, g.Edges)
	}
}

// TestParseGroupsAndComments verifies group and path context propagation and comment skipping.
func TestParseGroupsAndComments(t *testing.T) {
	input := `
# System Level Architecture
// Group A definition
// group: Backend Core
// path: internal/backend

rect api "API Service" [blue]
circle worker "Worker Service"

# Switch group
// group: Frontend
// path: ui/web

rect client "Web Client"

// regular comment
# another comment
`
	expectedNodes := []Node{
		{ID: "api", Shape: ShapeRect, Label: "API Service", Color: "blue", Group: "Backend Core", FilePath: "internal/backend"},
		{ID: "worker", Shape: ShapeCircle, Label: "Worker Service", Group: "Backend Core", FilePath: "internal/backend"},
		{ID: "client", Shape: ShapeRect, Label: "Web Client", Group: "Frontend", FilePath: "ui/web"},
	}

	g := Parse(input)
	if !reflect.DeepEqual(g.Nodes, expectedNodes) {
		t.Fatalf("expected nodes:\n%+v\ngot:\n%+v", expectedNodes, g.Nodes)
	}
	if len(g.Warnings) > 0 {
		t.Fatalf("expected no warnings for comments and groups, got: %v", g.Warnings)
	}
}

// TestParseFallbacksAndWarnings verifies synthesis of missing edge nodes and logging of malformed lines.
func TestParseFallbacksAndWarnings(t *testing.T) {
	input := `
rect auth "Auth"
this is an invalid line that cannot be parsed
auth -> unknownService "authenticates"
unknownService -> anotherMissing
`
	g := Parse(input)

	// Ensure auth, unknownService, and anotherMissing all exist in Nodes
	nodeMap := make(map[string]Node)
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
	}

	if _, ok := nodeMap["auth"]; !ok {
		t.Errorf("expected node 'auth' to exist")
	}

	fallback1, ok := nodeMap["unknownService"]
	if !ok {
		t.Fatalf("expected synthesized fallback node for 'unknownService'")
	}
	if fallback1.Shape != ShapeRect || fallback1.Label != "unknownService" || fallback1.Color != "gray" {
		t.Errorf("unexpected fallback node attributes: %+v", fallback1)
	}

	fallback2, ok := nodeMap["anotherMissing"]
	if !ok {
		t.Fatalf("expected synthesized fallback node for 'anotherMissing'")
	}
	if fallback2.Shape != ShapeRect || fallback2.Label != "anotherMissing" || fallback2.Color != "gray" {
		t.Errorf("unexpected fallback node attributes: %+v", fallback2)
	}

	// Verify warnings were emitted for the malformed line and the missing references
	if len(g.Warnings) < 3 {
		t.Fatalf("expected at least 3 warnings (1 malformed, 2 synthesized), got: %v", g.Warnings)
	}
}

// TestParseFile verifies parsing from a filesystem file and handling non-existent paths.
func TestParseFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sage-test-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := "rect svc \"Service\"\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	g, err := ParseFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error parsing file: %v", err)
	}
	if len(g.Nodes) != 1 || g.Nodes[0].ID != "svc" {
		t.Fatalf("unexpected parsed nodes from file: %+v", g.Nodes)
	}

	// Test non-existent file error
	_, err = ParseFile("/non/existent/path/to/sage.md")
	if err == nil {
		t.Fatalf("expected error for non-existent file, got nil")
	}
}

// TestParseUnsupportedShape verifies that unknown shape keywords trigger a warning.
func TestParseUnsupportedShape(t *testing.T) {
	input := `hexagon poly "Hexagon Service" [blue]`
	g := Parse(input)

	if len(g.Nodes) != 0 {
		t.Fatalf("expected 0 valid nodes for unsupported shape, got: %+v", g.Nodes)
	}
	if len(g.Warnings) == 0 {
		t.Fatalf("expected warning for unsupported shape, got none")
	}
}

// TestParseInlineFenceMention verifies that mentioning ```sage inline doesn't drop content.
func TestParseInlineFenceMention(t *testing.T) {
	input := "Note: you can write ```sage in documents.\nrect svc \"Service\"\n"
	g := Parse(input)
	if len(g.Nodes) != 1 || g.Nodes[0].ID != "svc" {
		t.Fatalf("expected raw parsing to extract node when no actual fence exists, got: %+v", g.Nodes)
	}
}

// TestTranspileLegacyDSLToMermaid_ShapesAndEdges verifies that legacy DSL declarations
// (rect, circle, diamond, triangle, cylinder, and connections) are converted into Mermaid markup.
func TestTranspileLegacyDSLToMermaid_ShapesAndEdges(t *testing.T) {
	input := `
rect svc "User Service"
circle client "Web Client"
diamond gate "API Gate"
triangle worker "Job Worker"
cylinder db "Main DB"

client -> gate "requests"
gate -> svc
svc <-> db "syncs"
`
	g := Parse(input)
	if g.Mermaid == "" {
		t.Fatalf("expected g.Mermaid to be populated from legacy DSL transpilation, got empty")
	}

	expectedSnippets := []string{
		"flowchart LR",
		`svc["User Service"]`,
		`client(("Web Client"))`,
		`gate{"API Gate"}`,
		`worker>"Job Worker"]`,
		`db[("Main DB")]`,
		`client -->|"requests"| gate`,
		`gate --> svc`,
		`svc <-->|"syncs"| db`,
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(g.Mermaid, snippet) {
			t.Errorf("expected transpiled Mermaid to contain %q, but got:\n%s", snippet, g.Mermaid)
		}
	}
}

// TestTranspileLegacyDSLToMermaid_Subgraphs verifies that // group: headers are transpiled
// into Mermaid subgraphs containing their respective member nodes.
func TestTranspileLegacyDSLToMermaid_Subgraphs(t *testing.T) {
	input := `
// group: Frontend Layer
// path: ui/web
circle browser "Web Browser"

// group: Backend Services
// path: internal/api
rect gateway "API Gateway"
rect auth "Auth Service"

// group:
cylinder shared_db "Database"

browser -> gateway "HTTPS"
gateway -> auth
auth -> shared_db
`
	g := Parse(input)
	if g.Mermaid == "" {
		t.Fatalf("expected g.Mermaid to be populated, got empty")
	}

	expectedSnippets := []string{
		"flowchart LR",
		"subgraph",
		`Frontend Layer`,
		`browser(("Web Browser"))`,
		`Backend Services`,
		`gateway["API Gateway"]`,
		`auth["Auth Service"]`,
		`shared_db[("Database")]`,
		`browser -->|"HTTPS"| gateway`,
		`gateway --> auth`,
		`auth --> shared_db`,
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(g.Mermaid, snippet) {
			t.Errorf("expected transpiled Mermaid to contain %q, but got:\n%s", snippet, g.Mermaid)
		}
	}
}

// TestTranspileLegacyDSLToMermaid_ColorsAndFallbacks verifies palette class tags, classDef emissions,
// and synthesized fallback nodes for undeclared edge endpoints.
func TestTranspileLegacyDSLToMermaid_ColorsAndFallbacks(t *testing.T) {
	input := `
rect api "API Gateway" [blue]
circle client "Web Client" [green]
api -> unknownService "routes to"
`
	g := Parse(input)
	if g.Mermaid == "" {
		t.Fatalf("expected g.Mermaid to be populated, got empty")
	}

	// Verify class tags on nodes
	if !strings.Contains(g.Mermaid, `api["API Gateway"]:::blue`) {
		t.Errorf("expected api node to have :::blue class tag, got:\n%s", g.Mermaid)
	}
	if !strings.Contains(g.Mermaid, `client(("Web Client")):::green`) {
		t.Errorf("expected client node to have :::green class tag, got:\n%s", g.Mermaid)
	}

	// Verify classDef palette rules
	if !strings.Contains(g.Mermaid, "classDef blue") || !strings.Contains(g.Mermaid, "classDef green") {
		t.Errorf("expected Mermaid to contain classDef rules for blue and green, got:\n%s", g.Mermaid)
	}

	// Verify synthesized fallback node
	if !strings.Contains(g.Mermaid, `unknownService["unknownService"]:::gray`) {
		t.Errorf("expected synthesized fallback node in Mermaid, got:\n%s", g.Mermaid)
	}
	if len(g.Warnings) == 0 {
		t.Errorf("expected warning for synthesized fallback node, got none")
	}
}

// TestParse_ExistingRootSAGEMD verifies that existing SAGE.md files render cleanly.
func TestParse_ExistingRootSAGEMD(t *testing.T) {
	bytes, err := os.ReadFile("../../SAGE.md")
	if err != nil {
		t.Skip("root SAGE.md not found relative to test, skipping")
	}

	g := Parse(string(bytes))
	if g.Mermaid == "" {
		t.Fatalf("expected root SAGE.md to transpile to Mermaid, got empty")
	}

	if !strings.Contains(g.Mermaid, "flowchart LR") {
		t.Errorf("expected SAGE.md transpiled output to contain 'flowchart LR'")
	}
	if !strings.Contains(g.Mermaid, "CLI Entrypoint") {
		t.Errorf("expected SAGE.md transpiled output to contain 'CLI Entrypoint' subgraph")
	}
	if !strings.Contains(g.Mermaid, `user(("Terminal Developer")):::green`) {
		t.Errorf("expected SAGE.md transpiled output to contain user node")
	}
}

// TestValidateMermaid_ValidAndMalformed verifies zero warnings for clean Mermaid
// and non-fatal warnings for malformed syntax or missing headers.
func TestValidateMermaid_ValidAndMalformed(t *testing.T) {
	validMermaid := "```mermaid\nflowchart LR\n    a[Alpha] --> b[(Beta)]\n```"
	gValid := Parse(validMermaid)
	if len(gValid.Warnings) != 0 {
		t.Errorf("expected 0 warnings for valid Mermaid, got: %v", gValid.Warnings)
	}

	malformedHeader := "```mermaid\nsequenceDiagram\n    Alice->>Bob: Hello\n```"
	gHeader := Parse(malformedHeader)
	if len(gHeader.Warnings) == 0 {
		t.Errorf("expected warning for non-flowchart Mermaid diagram")
	}

	unbalancedDelim := "```mermaid\nflowchart LR\n    a[Unclosed box --> b\n```"
	gDelim := Parse(unbalancedDelim)
	if len(gDelim.Warnings) == 0 {
		t.Errorf("expected warning for unbalanced delimiter in Mermaid")
	}
}
