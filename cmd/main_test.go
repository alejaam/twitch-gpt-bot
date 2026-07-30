package main

import "testing"

func TestLoadConfigReadsNVIDIAEnvironment(t *testing.T) {
	t.Setenv("NVIDIA_API_KEY", "nvidia-key")
	t.Setenv("NVIDIA_MODEL", "meta/llama-3.1-8b-instruct")
	t.Setenv("NVIDIA_MAX_TOKENS", "96")
	t.Setenv("NVIDIA_TEMPERATURE", "0.4")
	t.Setenv("API_SHARED_SECRET", "shared-secret")
	t.Setenv("RATE_LIMIT_SECONDS", "12")
	t.Setenv("SYSTEM_PROMPT", "Be brief.")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.nvidiaConfig.APIKey != "nvidia-key" {
		t.Fatalf("APIKey = %q, want NVIDIA API key", cfg.nvidiaConfig.APIKey)
	}
	if cfg.nvidiaConfig.Model != "meta/llama-3.1-8b-instruct" {
		t.Fatalf("Model = %q, want configured NVIDIA model", cfg.nvidiaConfig.Model)
	}
	if cfg.nvidiaConfig.MaxTokens != 96 {
		t.Fatalf("MaxTokens = %d, want 96", cfg.nvidiaConfig.MaxTokens)
	}
	if cfg.nvidiaConfig.Temperature != 0.4 {
		t.Fatalf("Temperature = %v, want 0.4", cfg.nvidiaConfig.Temperature)
	}
}

func TestLoadConfigRequiresNVIDIAAPIKey(t *testing.T) {
	t.Setenv("NVIDIA_API_KEY", "")
	t.Setenv("API_SHARED_SECRET", "shared-secret")

	_, err := loadConfig()
	if err == nil || err.Error() != "NVIDIA_API_KEY is required" {
		t.Fatalf("loadConfig() error = %v, want NVIDIA_API_KEY validation error", err)
	}
}
