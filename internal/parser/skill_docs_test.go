package parser_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"sage/internal/parser"
)

// extractMermaidBlocks extracts all ```mermaid ... ``` code blocks from markdown content.
// ACTIONABLE: Modify regex pattern if custom fence identifiers (e.g. "mermaid-preview") are supported.
func extractMermaidBlocks(content string) []string {
	re := regexp.MustCompile("(?s)```mermaid\\s*\\n(.*?)\\n```")
	matches := re.FindAllStringSubmatch(content, -1)
	var blocks []string
	for _, m := range matches {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			blocks = append(blocks, strings.TrimSpace(m[1]))
		}
	}
	return blocks
}

// assertValidMermaidBlocks verifies each extracted block produces a valid Mermaid AST with zero parser warnings.
// ACTIONABLE: Adjust warning or node assertions if diagrams are expected to produce specific node counts.
func assertValidMermaidBlocks(t *testing.T, blocks []string, sourceName string) {
	t.Helper()
	if len(blocks) == 0 {
		t.Fatalf("%s: expected at least one ```mermaid block", sourceName)
	}
	for i, block := range blocks {
		g := parser.Parse(block)
		if len(g.Warnings) > 0 {
			t.Errorf("%s: mermaid block #%d generated warnings: %v\nContent:\n%s", sourceName, i, g.Warnings, block)
		}
		if g.Mermaid == "" {
			t.Errorf("%s: mermaid block #%d produced empty Mermaid AST", sourceName, i)
		}
		if !strings.Contains(g.Mermaid, "flowchart") && !strings.Contains(g.Mermaid, "graph") {
			t.Errorf("%s: mermaid block #%d missing flowchart/graph header: %s", sourceName, i, g.Mermaid)
		}
	}
}

func TestSkillMarkdown_ConformsToStandardsAndMermaidSpec(t *testing.T) {
	// ACTIONABLE: Change relative path if skills directory is relocated or symlinked
	skillPath := filepath.Join("..", "..", "skills", "sage", "SKILL.md")
	contentBytes, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", skillPath, err)
	}
	content := string(contentBytes)

	// 1. YAML frontmatter validation
	if !strings.HasPrefix(content, "---\n") {
		t.Error("SKILL.md must begin with YAML frontmatter delimiter '---'")
	}
	if !strings.Contains(content, "name: sage") {
		t.Error("SKILL.md frontmatter must contain 'name: sage'")
	}
	if !strings.Contains(content, "description:") {
		t.Error("SKILL.md frontmatter must contain 'description:'")
	}

	// 2. Mission & Author guidelines preservation
	// ACTIONABLE: Add or update required rules if author specifications or install endpoints change
	requiredMissionRules := []string{
		"Never guess",
		"Concise explanations",
		"Incremental Update Protocol",
		"Never regenerate the whole file",
		"Locate corresponding `// group:` or node IDs in `SAGE.md`.",
		"curl -fsSL https://raw.githubusercontent.com/diamondosas/sage/main/assets/install.sh | sh",
		"npx sage-codebase",
	}
	for _, rule := range requiredMissionRules {
		if !strings.Contains(content, rule) {
			t.Errorf("SKILL.md is missing required guideline/rule: %q", rule)
		}
	}

	// 3. Quick reference must use mermaid flowchart syntax
	if !strings.Contains(content, "```mermaid") || !strings.Contains(content, "flowchart") {
		t.Errorf("SKILL.md quick reference must contain ```mermaid flowchart block, got:\n%s", content)
	}

	// 4. Validate all Mermaid blocks in SKILL.md parse cleanly with zero parser warnings
	blocks := extractMermaidBlocks(content)
	assertValidMermaidBlocks(t, blocks, "SKILL.md")
}

func TestLanguageSyntaxReference_MermaidCheatSheetAndParserCompatibility(t *testing.T) {
	// ACTIONABLE: Change relative path if reference directory is moved
	refPath := filepath.Join("..", "..", "skills", "sage", "references", "language-syntax.md")
	contentBytes, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", refPath, err)
	}
	content := string(contentBytes)

	// 1. Golden rule preservation
	if !strings.Contains(content, "Golden Rule: No Assumptions") && !strings.Contains(content, "Never speculate") {
		t.Error("language-syntax.md must preserve the Golden Rule: No Assumptions")
	}

	// 2. Subgraph directory clustering, shapes, and color classes documentation
	// ACTIONABLE: Expand required sections if new palette colors or shape categories are introduced
	requiredSections := []string{
		"subgraph",
		":::blue",
		":::green",
		":::purple",
		":::yellow",
		":::orange",
		"Incremental Maintenance",
	}
	for _, req := range requiredSections {
		if !strings.Contains(content, req) {
			t.Errorf("language-syntax.md missing required section/element: %q", req)
		}
	}

	// 3. Validate all Mermaid blocks in language-syntax.md parse cleanly
	blocks := extractMermaidBlocks(content)
	assertValidMermaidBlocks(t, blocks, "language-syntax.md")
}
