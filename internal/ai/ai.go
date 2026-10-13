package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/anthropic"
	"github.com/zendev-sh/goai/provider/google"
	"github.com/zendev-sh/goai/provider/ollama"
	"github.com/zendev-sh/goai/provider/openai"
	"sage/internal/github"
)

// Synthesizer uses GoAI (github.com/zendev-sh/goai) to summarize PR discussions into decisions.
type Synthesizer struct {
	Provider   string // "google", "openai", "anthropic", or "ollama"
	Model      string // Model name (e.g., "gemini-2.5-flash", "gpt-4o-mini", "llama3")
	APIKey     string // Provider API Key
	OllamaHost string // Base URL for local Ollama
}

// NewSynthesizer initializes the GoAI decision synthesizer.
func NewSynthesizer(provider, model, apiKey, ollamaHost string) *Synthesizer {
	return &Synthesizer{
		Provider:   provider,
		Model:      model,
		APIKey:     apiKey,
		OllamaHost: ollamaHost,
	}
}

// DecisionOutput defines the structured schema for LLM generation.
type DecisionOutput struct {
	Title     string `json:"title"`
	Rationale string `json:"rationale"`
}

// getModel initializes the provider language model using GoAI.
// CHANGE HERE: Modify default model or endpoint per provider if needed
func (s *Synthesizer) getModel() provider.LanguageModel {
	prov := strings.ToLower(s.Provider)
	if strings.HasPrefix(s.APIKey, "gsk_") && (prov == "" || prov == "grok") {
		prov = "groq"
	}

	switch prov {
	case "groq":
		baseURL := "https://api.groq.com/openai/v1"
		modelName := s.Model
		if modelName == "" {
			modelName = "openai/gpt-oss-120b"
		}
		return openai.Chat(modelName, openai.WithBaseURL(baseURL), openai.WithAPIKey(s.APIKey))
	case "grok":
		baseURL := "https://api.x.ai/v1"
		modelName := s.Model
		if modelName == "" {
			modelName = "grok-beta"
		}
		return openai.Chat(modelName, openai.WithBaseURL(baseURL), openai.WithAPIKey(s.APIKey))
	case "openai":
		if s.APIKey != "" {
			return openai.Chat(s.Model, openai.WithAPIKey(s.APIKey))
		}
		return openai.Chat(s.Model)
	case "anthropic":
		if s.APIKey != "" {
			return anthropic.Chat(s.Model, anthropic.WithAPIKey(s.APIKey))
		}
		return anthropic.Chat(s.Model)
	case "ollama":
		if s.OllamaHost != "" {
			return ollama.Chat(s.Model, ollama.WithBaseURL(s.OllamaHost))
		}
		return ollama.Chat(s.Model)
	case "google", "gemini":
		fallthrough
	default:
		if s.APIKey != "" {
			return google.Chat(s.Model, google.WithAPIKey(s.APIKey))
		}
		return google.Chat(s.Model)
	}
}

// SynthesizeDecision turns a PR into a structured decision using GoAI.
func (s *Synthesizer) SynthesizeDecision(pr github.PullRequest) Decision {
	if s.APIKey == "" && s.Provider != "ollama" {
		return s.heuristicExtract(pr)
	}

	prompt := fmt.Sprintf(`Analyze this merged Pull Request and summarize the core decision in 2 sentences.
Title: %s
Author: %s
Description:
%s`, pr.Title, pr.Author, pr.Body)

	model := s.getModel()

	// CHANGE HERE: Modify AI generation timeout for PR summarization
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := goai.GenerateObject[DecisionOutput](ctx, model, goai.WithPrompt(prompt))
	if err != nil || result.Object.Title == "" {
		return s.heuristicExtract(pr)
	}

	return Decision{
		ID:         fmt.Sprintf("DEC-%d", pr.Number),
		Title:      result.Object.Title,
		Rationale:  result.Object.Rationale,
		ApprovedBy: pr.Author,
		PRNumber:   pr.Number,
	}
}

// FileAnalysisOutput defines the structured schema for AI file understanding.
type FileAnalysisOutput struct {
	Summary      string   `json:"summary"`
	Role         string   `json:"role"`
	Exports      []string `json:"exports"`
	Dependencies []string `json:"dependencies"`
}

// AnalyzeFile reads source code and synthesizes architectural knowledge.
func (s *Synthesizer) AnalyzeFile(path string, content []byte) FileAnalysis {
	hash := ComputeFileHash(content)

	// If no AI key is provided or offline, use smart AST/heuristic extraction
	if s.APIKey == "" && s.Provider != "ollama" {
		return s.heuristicAnalyzeFile(path, content, hash)
	}

	// CHANGE HERE: Max bytes of file content sent to AI prompt (default 4KB to save tokens)
	maxPromptBytes := 4000
	text := string(content)
	if len(text) > maxPromptBytes {
		text = text[:maxPromptBytes] + "\n...(truncated for analysis)"
	}

	prompt := fmt.Sprintf(`Analyze this codebase file and describe its role concisely:
File: %s
Content:
%s

Return JSON with:
- "summary": 1-2 sentence description of file responsibility.
- "role": Architectural role (e.g. Storage Engine, CLI Command, HTTP Client, Model, Config, Test).
- "exports": List of top exported symbols, functions, or types (max 5).
- "dependencies": List of key imported packages or components (max 5).`, path, text)

	model := s.getModel()

	// CHANGE HERE: Timeout for file analysis generation
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	result, err := goai.GenerateObject[FileAnalysisOutput](ctx, model, goai.WithPrompt(prompt))
	if err != nil || result.Object.Summary == "" {
		// Fallback gracefully on any error or timeout
		return s.heuristicAnalyzeFile(path, content, hash)
	}

	return FileAnalysis{
		Path:         path,
		Summary:      result.Object.Summary,
		Role:         result.Object.Role,
		Exports:      result.Object.Exports,
		Dependencies: result.Object.Dependencies,
		Hash:         hash,
		UpdatedAt:    time.Now(),
	}
}

// ComputeFileHash returns the SHA-256 hex digest of file bytes.
func ComputeFileHash(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

// heuristicAnalyzeFile parses code symbols offline without external LLM calls.
func (s *Synthesizer) heuristicAnalyzeFile(path string, content []byte, hash string) FileAnalysis {
	var exports []string
	var dependencies []string
	var pkgName string
	var firstComment string

	// For Go files, use accurate standard library AST parser
	if strings.HasSuffix(path, ".go") {
		fset := token.NewFileSet()
		if node, err := parser.ParseFile(fset, path, content, parser.ParseComments); err == nil {
			pkgName = node.Name.Name
			if node.Doc != nil {
				firstComment = strings.TrimSpace(node.Doc.Text())
			}
			for _, imp := range node.Imports {
				if len(dependencies) < 5 && imp.Path != nil {
					dep := strings.Trim(imp.Path.Value, `"`)
					if strings.Contains(dep, "/") {
						dependencies = append(dependencies, dep)
					}
				}
			}
			for _, decl := range node.Decls {
				if len(exports) >= 5 {
					break
				}
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Name.IsExported() {
						exports = append(exports, d.Name.Name)
					}
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.IsExported() {
							exports = append(exports, ts.Name.Name)
						}
					}
				}
			}
		}
	}

	// Fallback line parsing for non-Go or unparseable files
	if pkgName == "" && firstComment == "" && len(exports) == 0 {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if (strings.HasPrefix(trimmed, "// ") || strings.HasPrefix(trimmed, "/* ") || strings.HasPrefix(trimmed, "# ")) && firstComment == "" {
				firstComment = strings.TrimLeft(trimmed, "/*# ")
			}
		}
	}

	// Classify role based on path heuristics
	role := "Source Module"
	cleanLower := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(cleanLower, "_test.") || strings.HasPrefix(cleanLower, "test/") || strings.Contains(cleanLower, "/test/"):
		role = "Automated Test Suite"
	case strings.Contains(cleanLower, "cmd/") || strings.HasSuffix(cleanLower, "/main.go") || cleanLower == "main.go" || strings.Contains(cleanLower, "cli/"):
		role = "CLI Command"
	case strings.Contains(cleanLower, "internal/graph") || strings.Contains(cleanLower, "graph/"):
		role = "Knowledge Graph Storage"
	case strings.Contains(cleanLower, "internal/ai") || strings.Contains(cleanLower, "/ai/"):
		role = "AI Synthesis Engine"
	case strings.Contains(cleanLower, "internal/git") || strings.Contains(cleanLower, "/git/"):
		role = "Git Extractor"
	case strings.Contains(cleanLower, "internal/config") || strings.Contains(cleanLower, "config"):
		role = "Configuration"
	case strings.Contains(cleanLower, "mcp/") || strings.Contains(cleanLower, "server/"):
		role = "Protocol Server"
	case strings.Contains(cleanLower, "store") || strings.Contains(cleanLower, "db/") || strings.Contains(cleanLower, "database/"):
		role = "Data Storage / Models"
	case strings.Contains(cleanLower, "api/") || strings.Contains(cleanLower, "http/") || strings.Contains(cleanLower, "handler/"):
		role = "API / HTTP Handler"
	case strings.HasSuffix(cleanLower, ".md"):
		role = "Documentation"
	case strings.HasSuffix(cleanLower, ".yml") || strings.HasSuffix(cleanLower, ".yaml"):
		role = "Workflow / CI Config"
	}

	summary := firstComment
	if summary == "" {
		if pkgName != "" {
			summary = fmt.Sprintf("Package %s module managing %s operations.", pkgName, role)
		} else {
			summary = fmt.Sprintf("Codebase file responsible for %s.", role)
		}
	}

	return FileAnalysis{
		Path:         path,
		Summary:      summary,
		Role:         role,
		Exports:      exports,
		Dependencies: dependencies,
		Hash:         hash,
		UpdatedAt:    time.Now(),
	}
}

// IndexableExtensions contains file extensions indexed by Sage.
// CHANGE HERE: Add more programming language file extensions to index
var IndexableExtensions = map[string]bool{
	".go": true, ".ts": true, ".js": true, ".py": true, ".rs": true,
	".java": true, ".c": true, ".cpp": true, ".h": true, ".cs": true,
	".md": true, ".json": true, ".yml": true, ".yaml": true, ".sh": true,
}

// IsIndexableSourceFile checks if a file path is a source file eligible for AI indexing.
func IsIndexableSourceFile(path string) bool {
	clean := filepath.ToSlash(path)
	// Ignore internal Badger DB, git metadata, and common build artifacts
	if strings.HasPrefix(clean, ".git/") || strings.Contains(clean, "/.git/") {
		return false
	}
	if strings.HasPrefix(clean, ".github/sage/") || strings.Contains(clean, "/.github/sage/") {
		return false
	}
	if strings.Contains(clean, "node_modules/") || strings.Contains(clean, "vendor/") || strings.Contains(clean, "bin/") || strings.Contains(clean, "dist/") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(clean))
	return IndexableExtensions[ext]
}

// heuristicExtract provides zero-dependency offline fallback extraction.
func (s *Synthesizer) heuristicExtract(pr github.PullRequest) Decision {
	lines := strings.Split(pr.Body, "\n")
	var rationaleParts []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			rationaleParts = append(rationaleParts, trimmed)
			if len(rationaleParts) >= 2 {
				break
			}
		}
	}

	rationale := strings.Join(rationaleParts, " ")
	if rationale == "" {
		rationale = "Merged in PR #" + fmt.Sprintf("%d", pr.Number)
	}

	return Decision{
		ID:         fmt.Sprintf("DEC-%d", pr.Number),
		Title:      pr.Title,
		Rationale:  rationale,
		ApprovedBy: pr.Author,
		PRNumber:   pr.Number,
	}
}

