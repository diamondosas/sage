package d2_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"sage/internal/d2"
)

// extractD2Blocks extracts all ```d2 ... ``` code blocks from markdown content.
func extractD2Blocks(content string) []string {
	re := regexp.MustCompile("(?s)```d2\\s*\\n(.*?)\\n```")
	matches := re.FindAllStringSubmatch(content, -1)
	var blocks []string
	for _, m := range matches {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			blocks = append(blocks, strings.TrimSpace(m[1]))
		}
	}
	return blocks
}

// assertValidD2Blocks verifies each extracted block compiles to valid SVG with zero errors.
func assertValidD2Blocks(t *testing.T, blocks []string, sourceName string) {
	t.Helper()
	if len(blocks) == 0 {
		t.Fatalf("%s: expected at least one ```d2 block", sourceName)
	}
	for i, block := range blocks {
		res, err := d2.Compile(context.Background(), block, nil)
		if err != nil {
			t.Fatalf("%s: d2 block #%d returned unexpected error: %v", sourceName, i, err)
		}
		if len(res.Warnings) > 0 {
			t.Errorf("%s: d2 block #%d generated warnings: %v\nContent:\n%s", sourceName, i, res.Warnings, block)
		}
		if !strings.Contains(res.SVG, "<svg") {
			t.Errorf("%s: d2 block #%d produced invalid or missing SVG", sourceName, i)
		}
	}
}

func TestSkillMarkdown_ConformsToStandardsAndD2Spec(t *testing.T) {
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

	// 3. Quick reference must use D2 syntax
	if !strings.Contains(content, "```d2") {
		t.Errorf("SKILL.md quick reference must contain ```d2 block, got:\n%s", content)
	}

	// 4. Validate all D2 blocks in SKILL.md compile cleanly with zero warnings
	blocks := extractD2Blocks(content)
	assertValidD2Blocks(t, blocks, "SKILL.md")
}

func TestLanguageSyntaxReference_D2CheatSheetAndCompilation(t *testing.T) {
	refPath := filepath.Join("..", "..", "skills", "sage", "references", "language-syntax.md")
	contentBytes, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", refPath, err)
	}
	content := string(contentBytes)

	blocks := extractD2Blocks(content)
	assertValidD2Blocks(t, blocks, "language-syntax.md")
}
