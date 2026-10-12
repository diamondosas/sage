package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime parameters for Sage.
type Config struct {
	// FalkorDB Connection Settings
	FalkorHost string // Host for FalkorDB (e.g., "127.0.0.1" or "falkordb")
	FalkorPort int    // Port for FalkorDB (default: 6379)
	GraphName  string // Key name for the graph (default: "sage")

	// Git Ingestion Settings
	MaxCommits    int           // Max commits to scan backwards (default: 500)
	DecayHalfLife time.Duration // Half-life for expertise scoring decay (default: 60 days)

	// GitHub Integration
	GitHubToken string // Token for GitHub REST API (reads from GITHUB_TOKEN)

	// AI / LLM Settings via GoAI (github.com/zendev-sh/goai)
	LLMProvider string // Provider: "google", "openai", "anthropic", or "ollama"
	LLMModel    string // Model name (e.g., "gemini-2.5-flash", "gpt-4o-mini", "llama3")
	LLMKey      string // API key for selected provider
	OllamaHost  string // Custom Ollama host if using local models (default: "http://localhost:11434")

	// Badger Embedded Storage
	BadgerPath string // Path to Badger DB directory (default: ".github/sage")

	// Storage Paths
	SageDir string // Local cache directory (default: ".github/sage")
}

// LoadFromEnv loads configuration with sensible defaults and environment overrides.
func LoadFromEnv() *Config {
	// Provider selection
	// CHANGE HERE: Change default provider to "groq", "grok", "openai", "anthropic", or "ollama"
	provider := getEnv("LLM_PROVIDER", "")

	// Check API keys
	apiKey := getEnvMulti("GROQ_API_KEY", "GROK_API_KEY", "XAI_API_KEY", "GEMINI_API_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY")

	// Auto-detect Groq keys (starts with "gsk_")
	if provider == "" {
		if strings.HasPrefix(apiKey, "gsk_") {
			provider = "groq"
		} else {
			provider = "google"
		}
	}

	// Default model selection based on provider
	// CHANGE HERE: Modify default model per provider
	defaultModel := "gemini-2.5-flash"
	switch strings.ToLower(provider) {
	case "groq":
		defaultModel = "openai/gpt-oss-120b" // Fast Groq model, can also be "qwen/qwen3.8-27b"
	case "grok":
		if strings.HasPrefix(apiKey, "gsk_") {
			// If Groq key provided under 'grok', route to Groq model
			provider = "groq"
			defaultModel = "openai/gpt-oss-120b"
		} else {
			defaultModel = "grok-beta"
		}
	case "openai":
		defaultModel = "gpt-4o-mini"
	case "anthropic":
		defaultModel = "claude-3-5-haiku-20241022"
	case "ollama":
		defaultModel = "llama3"
	}

	return &Config{
		// FalkorDB settings
		// CHANGE HERE: Modify host if FalkorDB is on another machine or container
		FalkorHost: getEnv("FALKORDB_HOST", "127.0.0.1"),
		FalkorPort: getEnvInt("FALKORDB_PORT", 6379),
		GraphName:  getEnv("FALKORDB_GRAPH", "sage"),

		// Git scanning settings
		// CHANGE HERE: Tune max commits to scan (e.g. 200 for fast test, 1000 for deep history)
		MaxCommits: getEnvInt("SAGE_MAX_COMMITS", 500),
		// CHANGE HERE: Tune how quickly past contributor activity decays (in days)
		DecayHalfLife: time.Duration(getEnvInt("SAGE_DECAY_DAYS", 60)) * 24 * time.Hour,

		// GitHub & GoAI settings
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		LLMProvider: provider,
		// CHANGE HERE: Set custom model via LLM_MODEL env var
		LLMModel:   getEnv("LLM_MODEL", defaultModel),
		LLMKey:     apiKey,
		OllamaHost: getEnv("OLLAMA_HOST", "http://localhost:11434"),

		BadgerPath: getEnv("SAGE_DB_PATH", ".github/sage"),
		SageDir:    ".github/sage",
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvMulti(keys ...string) string {
	for _, k := range keys {
		if val := os.Getenv(k); val != "" {
			return val
		}
	}
	return ""
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultVal
}
