package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime parameters for Sage.
type Config struct {
	// Git Ingestion Settings
	MaxCommits    int           // Max commits to scan backwards (default: 500)
	DecayHalfLife time.Duration // Half-life for expertise scoring decay (default: 60 days)

	// GitHub Integration
	GitHubToken string // Token for GitHub REST API (reads from GITHUB_TOKEN)

	// AI / LLM Settings via GoAI (github.com/zendev-sh/goai)
	LLMProvider string // Provider: "google", "openai", "anthropic", "groq", "grok", or "ollama"
	LLMModel    string // Model name (e.g., "gemini-2.5-flash", "gpt-4o-mini", "llama3")
	LLMKey      string // API key for selected provider
	OllamaHost  string // Custom Ollama host if using local models (default: "http://localhost:11434")

	// Storage Paths
	DBPath  string // Path to SQLite database file or directory (default: ".github/sage")
	SageDir string // Local cache directory (default: ".github/sage")
}

// LoadFromEnv loads configuration with sensible defaults and environment overrides.
func LoadFromEnv() *Config {
	provider := strings.ToLower(getEnv("LLM_PROVIDER", ""))

	groqKey := os.Getenv("GROQ_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	grokKey := getEnvMulti("GROK_API_KEY", "XAI_API_KEY")

	// Auto-detect provider if not explicitly specified
	if provider == "" {
		switch {
		case groqKey != "" || strings.HasPrefix(grokKey, "gsk_"):
			provider = "groq"
		case geminiKey != "":
			provider = "google"
		case openaiKey != "":
			provider = "openai"
		case anthropicKey != "":
			provider = "anthropic"
		case grokKey != "":
			provider = "grok"
		default:
			provider = "google"
		}
	}

	// Select key matching the resolved provider
	var apiKey string
	switch provider {
	case "groq":
		if groqKey != "" {
			apiKey = groqKey
		} else if strings.HasPrefix(grokKey, "gsk_") {
			apiKey = grokKey
		}
	case "google", "gemini":
		provider = "google"
		apiKey = geminiKey
	case "openai":
		apiKey = openaiKey
	case "anthropic":
		apiKey = anthropicKey
	case "grok":
		if strings.HasPrefix(grokKey, "gsk_") {
			provider = "groq"
			apiKey = grokKey
		} else {
			apiKey = grokKey
		}
	case "ollama":
		apiKey = ""
	}

	// Fallback to any available key if provider key is still empty
	if apiKey == "" && provider != "ollama" {
		apiKey = getEnvMulti("GROQ_API_KEY", "GEMINI_API_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GROK_API_KEY", "XAI_API_KEY")
	}

	// Default model selection based on provider
	defaultModel := "gemini-2.5-flash"
	switch provider {
	case "groq":
		defaultModel = "openai/gpt-oss-120b"
	case "grok":
		defaultModel = "grok-beta"
	case "openai":
		defaultModel = "gpt-4o-mini"
	case "anthropic":
		defaultModel = "claude-3-5-haiku-20241022"
	case "ollama":
		defaultModel = "llama3"
	}

	return &Config{
		// Git scanning settings
		MaxCommits:    getEnvInt("SAGE_MAX_COMMITS", 500),
		DecayHalfLife: time.Duration(getEnvInt("SAGE_DECAY_DAYS", 60)) * 24 * time.Hour,

		// GitHub & GoAI settings
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		LLMProvider: provider,
		LLMModel:    getEnv("LLM_MODEL", defaultModel),
		LLMKey:      apiKey,
		OllamaHost:  getEnv("OLLAMA_HOST", "http://localhost:11434"),

		DBPath:  getEnv("SAGE_DB_PATH", ".github/sage"),
		SageDir: ".github/sage",
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
