package main

import "testing"

func TestLoadConfigGeminiDefaults(t *testing.T) {
	t.Setenv("TWITCH_BOT_USERNAME", "mybot")
	t.Setenv("TWITCH_OAUTH_TOKEN", "oauth:abcdef123")
	t.Setenv("TWITCH_CHANNEL", "mychannel")
	t.Setenv("LLM_PROVIDER", "gemini")
	t.Setenv("GEMINI_API_KEY", "gemini-test-key")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("RATE_LIMIT_SECONDS", "10")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.twitch.Username != "mybot" {
		t.Errorf("expected username mybot, got %s", cfg.twitch.Username)
	}
	if cfg.twitch.Channel != "#mychannel" {
		t.Errorf("expected channel #mychannel, got %s", cfg.twitch.Channel)
	}
	if cfg.provider != "gemini" {
		t.Errorf("expected provider gemini, got %s", cfg.provider)
	}
	if cfg.geminiConfig.APIKey != "gemini-test-key" {
		t.Errorf("expected gemini key gemini-test-key, got %s", cfg.geminiConfig.APIKey)
	}
	if cfg.geminiConfig.Model != "gemini-2.5-flash" {
		t.Errorf("expected default model gemini-2.5-flash, got %s", cfg.geminiConfig.Model)
	}
}

func TestLoadConfigNVIDIA(t *testing.T) {
	t.Setenv("TWITCH_BOT_USERNAME", "mybot")
	t.Setenv("TWITCH_OAUTH_TOKEN", "oauth:abcdef123")
	t.Setenv("TWITCH_CHANNEL", "mychannel")
	t.Setenv("LLM_PROVIDER", "nvidia")
	t.Setenv("NVIDIA_API_KEY", "nvidia-key")
	t.Setenv("NVIDIA_MODEL", "meta/llama-3.1-8b-instruct")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.provider != "nvidia" {
		t.Errorf("expected provider nvidia, got %s", cfg.provider)
	}
	if cfg.nvidiaConfig.APIKey != "nvidia-key" {
		t.Errorf("expected nvidia key nvidia-key, got %s", cfg.nvidiaConfig.APIKey)
	}
}

func TestLoadConfigMissingTwitchCredentials(t *testing.T) {
	t.Setenv("TWITCH_BOT_USERNAME", "")
	t.Setenv("TWITCH_OAUTH_TOKEN", "oauth:123")
	t.Setenv("TWITCH_CHANNEL", "mychannel")
	t.Setenv("GEMINI_API_KEY", "key")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for missing TWITCH_BOT_USERNAME, got nil")
	}
}

func TestLoadConfigMissingGeminiAPIKey(t *testing.T) {
	t.Setenv("TWITCH_BOT_USERNAME", "bot")
	t.Setenv("TWITCH_OAUTH_TOKEN", "oauth:123")
	t.Setenv("TWITCH_CHANNEL", "mychannel")
	t.Setenv("LLM_PROVIDER", "gemini")
	t.Setenv("GEMINI_API_KEY", "")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for missing GEMINI_API_KEY, got nil")
	}
}
