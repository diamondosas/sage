package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnvDefaults(t *testing.T) {
	cfg := LoadFromEnv()

	if cfg.FalkorHost != "127.0.0.1" {
		t.Errorf("expected default host 127.0.0.1, got %s", cfg.FalkorHost)
	}
	if cfg.FalkorPort != 6379 {
		t.Errorf("expected default port 6379, got %d", cfg.FalkorPort)
	}
	if cfg.GraphName != "sage" {
		t.Errorf("expected default graph 'sage', got %s", cfg.GraphName)
	}
	if cfg.MaxCommits != 500 {
		t.Errorf("expected default max commits 500, got %d", cfg.MaxCommits)
	}
	if cfg.DecayHalfLife != 60*24*time.Hour {
		t.Errorf("expected default decay 60 days, got %v", cfg.DecayHalfLife)
	}
}

func TestLoadFromEnvOverrides(t *testing.T) {
	os.Setenv("FALKORDB_HOST", "falkor-cluster")
	os.Setenv("FALKORDB_PORT", "9999")
	os.Setenv("SAGE_MAX_COMMITS", "123")
	defer func() {
		os.Unsetenv("FALKORDB_HOST")
		os.Unsetenv("FALKORDB_PORT")
		os.Unsetenv("SAGE_MAX_COMMITS")
	}()

	cfg := LoadFromEnv()

	if cfg.FalkorHost != "falkor-cluster" {
		t.Errorf("expected host 'falkor-cluster', got %s", cfg.FalkorHost)
	}
	if cfg.FalkorPort != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.FalkorPort)
	}
	if cfg.MaxCommits != 123 {
		t.Errorf("expected max commits 123, got %d", cfg.MaxCommits)
	}
}

func TestLoadFromEnvGroqAutoDetect(t *testing.T) {
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
}
