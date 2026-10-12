package ai

import (
	"context"
	"fmt"
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

// SynthesizeDecision turns a PR into a structured decision using GoAI.
func (s *Synthesizer) SynthesizeDecision(pr github.PullRequest) Decision {
	// If no API key is provided and not using local Ollama, fallback to heuristic extraction
	if s.APIKey == "" && s.Provider != "ollama" {
		return s.heuristicExtract(pr)
	}

	prompt := fmt.Sprintf(`Analyze this merged Pull Request and summarize the core decision in 2 sentences.
Title: %s
Author: %s
Description:
%s`, pr.Title, pr.Author, pr.Body)

	prov := strings.ToLower(s.Provider)
	if strings.HasPrefix(s.APIKey, "gsk_") && (prov == "" || prov == "grok") {
		prov = "groq"
	}

	// Select model based on provider using GoAI
	// CHANGE HERE: Easily switch providers or customize model options
	var model provider.LanguageModel
	switch prov {
	case "groq":
		// Groq provides ultra-fast OpenAI-compatible endpoints
		// CHANGE HERE: Change Groq base URL or model options if using a custom gateway
		baseURL := "https://api.groq.com/openai/v1"
		modelName := s.Model
		if modelName == "" {
			modelName = "openai/gpt-oss-120b"
		}
		model = openai.Chat(modelName, openai.WithBaseURL(baseURL), openai.WithAPIKey(s.APIKey))
	case "grok":
		// xAI Grok provides OpenAI-compatible endpoints
		// CHANGE HERE: Change Grok base URL or model
		baseURL := "https://api.x.ai/v1"
		modelName := s.Model
		if modelName == "" {
			modelName = "grok-beta"
		}
		model = openai.Chat(modelName, openai.WithBaseURL(baseURL), openai.WithAPIKey(s.APIKey))
	case "openai":
		if s.APIKey != "" {
			model = openai.Chat(s.Model, openai.WithAPIKey(s.APIKey))
		} else {
			model = openai.Chat(s.Model)
		}
	case "anthropic":
		if s.APIKey != "" {
			model = anthropic.Chat(s.Model, anthropic.WithAPIKey(s.APIKey))
		} else {
			model = anthropic.Chat(s.Model)
		}
	case "ollama":
		if s.OllamaHost != "" {
			model = ollama.Chat(s.Model, ollama.WithBaseURL(s.OllamaHost))
		} else {
			model = ollama.Chat(s.Model)
		}
	case "google", "gemini":
		fallthrough
	default:
		if s.APIKey != "" {
			model = google.Chat(s.Model, google.WithAPIKey(s.APIKey))
		} else {
			model = google.Chat(s.Model)
		}
	}

	// CHANGE HERE: Modify AI generation timeout for larger models or slower connections
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Generate structured object directly using Go generics in GoAI
	result, err := goai.GenerateObject[DecisionOutput](ctx, model, goai.WithPrompt(prompt))
	if err != nil || result.Object.Title == "" {
		// Gracefully fall back to local heuristic extraction on any error
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
