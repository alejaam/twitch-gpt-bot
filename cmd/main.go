package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"twitch-gpt-bot/internal/bot"
	"twitch-gpt-bot/internal/gemini"
	"twitch-gpt-bot/internal/llm"
	"twitch-gpt-bot/internal/nvidia"
	"twitch-gpt-bot/internal/ratelimit"
	"twitch-gpt-bot/internal/twitch"
)

type config struct {
	port          string
	rateLimit     time.Duration
	commandPrefix string
	provider      string
	twitch        twitch.ClientConfig
	geminiConfig  gemini.Config
	nvidiaConfig  nvidia.Config
}

func main() {
	_ = godotenv.Load()

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	var generator llm.Generator
	switch cfg.provider {
	case "gemini":
		slog.Info("using Google Gemini provider", "model", cfg.geminiConfig.Model)
		generator = gemini.NewClient(cfg.geminiConfig)
	case "nvidia":
		slog.Info("using NVIDIA NIM provider", "model", cfg.nvidiaConfig.Model)
		generator = nvidia.NewClient(cfg.nvidiaConfig)
	default:
		slog.Error("unsupported provider", "provider", cfg.provider)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	twitchClient := twitch.NewClient(cfg.twitch)
	limiter := ratelimit.New(cfg.rateLimit)

	botHandler := bot.New(bot.Config{
		Prefix: cfg.commandPrefix,
	}, limiter, generator, twitchClient)

	twitchClient.OnMessage = func(msg *twitch.ParsedLine) {
		botHandler.HandleMessage(ctx, msg)
	}

	// Start optional health check HTTP server for Dokploy
	var httpServer *http.Server
	if cfg.port != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("ok"))
		})

		httpServer = &http.Server{
			Addr:              ":" + cfg.port,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
		}

		go func() {
			slog.Info("starting health check HTTP server", "port", cfg.port)
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("health HTTP server stopped", "error", err)
			}
		}()
	}

	// Start Twitch IRC connection in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- twitchClient.Run(ctx)
	}()

	slog.Info("twitch-gpt-bot running. Press Ctrl+C to terminate.", "channel", cfg.twitch.Channel, "prefix", cfg.commandPrefix)

	select {
	case <-ctx.Done():
		slog.Info("shutting down bot gracefully")
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("twitch client exited with error", "error", err)
		}
	}

	if httpServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}
}

func loadConfig() (config, error) {
	rateLimitSeconds, err := envInt("RATE_LIMIT_SECONDS", 8)
	if err != nil || rateLimitSeconds <= 0 {
		return config{}, fmt.Errorf("RATE_LIMIT_SECONDS must be a positive integer")
	}

	systemPrompt := envString("SYSTEM_PROMPT", "You are a helpful Twitch chat assistant. Keep every answer concise, single line, and safe for a public stream.")
	commandPrefix := envString("COMMAND_PREFIX", "!gpt")
	provider := strings.ToLower(envString("LLM_PROVIDER", "gemini"))

	twitchUser := strings.TrimSpace(os.Getenv("TWITCH_BOT_USERNAME"))
	if twitchUser == "" {
		return config{}, fmt.Errorf("TWITCH_BOT_USERNAME is required")
	}
	twitchToken := strings.TrimSpace(os.Getenv("TWITCH_OAUTH_TOKEN"))
	if twitchToken == "" {
		return config{}, fmt.Errorf("TWITCH_OAUTH_TOKEN is required")
	}
	twitchChannel := strings.TrimSpace(os.Getenv("TWITCH_CHANNEL"))
	if twitchChannel == "" {
		return config{}, fmt.Errorf("TWITCH_CHANNEL is required")
	}

	var geminiCfg gemini.Config
	var nvidiaCfg nvidia.Config

	switch provider {
	case "gemini":
		apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
		if apiKey == "" {
			return config{}, fmt.Errorf("GEMINI_API_KEY is required when LLM_PROVIDER is gemini")
		}
		model := envString("GEMINI_MODEL", "gemini-2.5-flash")
		maxTokens, err := envInt("GEMINI_MAX_TOKENS", 150)
		if err != nil || maxTokens <= 0 {
			return config{}, fmt.Errorf("GEMINI_MAX_TOKENS must be a positive integer")
		}
		temp, err := envFloat("GEMINI_TEMPERATURE", 0.7)
		if err != nil || temp < 0 || temp > 2 {
			return config{}, fmt.Errorf("GEMINI_TEMPERATURE must be between 0 and 2")
		}

		geminiCfg = gemini.Config{
			APIKey:       apiKey,
			Model:        model,
			MaxTokens:    maxTokens,
			Temperature:  temp,
			SystemPrompt: systemPrompt,
		}

	case "nvidia":
		apiKey := strings.TrimSpace(os.Getenv("NVIDIA_API_KEY"))
		if apiKey == "" {
			return config{}, fmt.Errorf("NVIDIA_API_KEY is required when LLM_PROVIDER is nvidia")
		}
		model := strings.TrimSpace(os.Getenv("NVIDIA_MODEL"))
		if model == "" {
			return config{}, fmt.Errorf("NVIDIA_MODEL is required when LLM_PROVIDER is nvidia")
		}
		maxTokens, err := envInt("NVIDIA_MAX_TOKENS", 120)
		if err != nil || maxTokens <= 0 {
			return config{}, fmt.Errorf("NVIDIA_MAX_TOKENS must be a positive integer")
		}
		temp, err := envFloat("NVIDIA_TEMPERATURE", 0.7)
		if err != nil || temp < 0 || temp > 2 {
			return config{}, fmt.Errorf("NVIDIA_TEMPERATURE must be between 0 and 2")
		}

		nvidiaCfg = nvidia.Config{
			APIKey:       apiKey,
			Model:        model,
			MaxTokens:    maxTokens,
			Temperature:  temp,
			SystemPrompt: systemPrompt,
		}

	default:
		return config{}, fmt.Errorf("invalid LLM_PROVIDER: %q (must be gemini or nvidia)", provider)
	}

	return config{
		port:          envString("PORT", "8080"),
		rateLimit:     time.Duration(rateLimitSeconds) * time.Second,
		commandPrefix: commandPrefix,
		provider:      provider,
		twitch: twitch.ClientConfig{
			Username:   twitchUser,
			OAuthToken: twitchToken,
			Channel:    twitch.NormalizeChannel(twitchChannel),
			Address:    envString("TWITCH_IRC_ADDRESS", ""),
		},
		geminiConfig: geminiCfg,
		nvidiaConfig: nvidiaCfg,
	}, nil
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func envFloat(name string, fallback float64) (float64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(value, 64)
}
