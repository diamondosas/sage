package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnvDefaults(t *testing.T) {
	// Clean environment variables that could affect defaults
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("GROQ_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")

	cfg := LoadFromEnv()

	if cfg.MaxCommits != 500 {
		t.Errorf("expected default max commits 500, got %d", cfg.MaxCommits)
	}
	if cfg.DecayHalfLife != 60*24*time.Hour {
		t.Errorf("expected default decay 60 days, got %v", cfg.DecayHalfLife)
	}
	if cfg.DBPath != ".github/sage" {
		t.Errorf("expected default DBPath '.github/sage', got %s", cfg.DBPath)
	}
	if cfg.LLMProvider != "google" {
		t.Errorf("expected default provider 'google', got %s", cfg.LLMProvider)
	}
}

func TestLoadFromEnvOverrides(t *testing.T) {
	os.Setenv("SAGE_MAX_COMMITS", "123")
	os.Setenv("SAGE_DB_PATH", "custom/sage.db")
	defer func() {
		os.Unsetenv("SAGE_MAX_COMMITS")
		os.Unsetenv("SAGE_DB_PATH")
	}()

	cfg := LoadFromEnv()

	if cfg.MaxCommits != 123 {
		t.Errorf("expected max commits 123, got %d", cfg.MaxCommits)
	}
	if cfg.DBPath != "custom/sage.db" {
		t.Errorf("expected DBPath 'custom/sage.db', got %s", cfg.DBPath)
	}
}

func TestLoadFromEnvGroqAutoDetect(t *testing.T) {
	os.Unsetenv("LLM_PROVIDER")
	os.Setenv("GROQ_API_KEY", "gsk_test123456")
	defer os.Unsetenv("GROQ_API_KEY")

	cfg := LoadFromEnv()
	if cfg.LLMProvider != "groq" {
		t.Errorf("expected provider 'groq', got %s", cfg.LLMProvider)
	}
	if cfg.LLMModel != "openai/gpt-oss-120b" {
		t.Errorf("expected model 'openai/gpt-oss-120b', got %s", cfg.LLMModel)
	}
	if cfg.LLMKey != "gsk_test123456" {
		t.Errorf("expected key 'gsk_test123456', got %s", cfg.LLMKey)
	}
}

func TestLoadFromEnvGrok(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "grok")
	os.Setenv("GROK_API_KEY", "xai-testkey")
	defer func() {
		os.Unsetenv("LLM_PROVIDER")
		os.Unsetenv("GROK_API_KEY")
	}()

	cfg := LoadFromEnv()
	if cfg.LLMProvider != "grok" {
		t.Errorf("expected provider 'grok', got %s", cfg.LLMProvider)
	}
	if cfg.LLMModel != "grok-beta" {
		t.Errorf("expected model 'grok-beta', got %s", cfg.LLMModel)
	}
	if cfg.LLMKey != "xai-testkey" {
		t.Errorf("expected key 'xai-testkey', got %s", cfg.LLMKey)
	}
}

func TestLoadFromEnvExplicitProviderWithSpecificKey(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "openai")
	os.Setenv("OPENAI_API_KEY", "sk-openai-key")
	os.Setenv("GROQ_API_KEY", "gsk_groq_key")
	defer func() {
		os.Unsetenv("LLM_PROVIDER")
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("GROQ_API_KEY")
	}()

	cfg := LoadFromEnv()
	if cfg.LLMProvider != "openai" {
		t.Errorf("expected provider 'openai', got %s", cfg.LLMProvider)
	}
	if cfg.LLMKey != "sk-openai-key" {
		t.Errorf("expected key 'sk-openai-key', got %s", cfg.LLMKey)
	}
	if cfg.LLMModel != "gpt-4o-mini" {
		t.Errorf("expected model 'gpt-4o-mini', got %s", cfg.LLMModel)
	}
}
