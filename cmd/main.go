package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"twitch-gpt-bot/internal/handler"
	"twitch-gpt-bot/internal/nvidia"
	"twitch-gpt-bot/internal/ratelimit"
)

type config struct {
	port             string
	apiSharedSecret  string
	rateLimit         time.Duration
	nvidiaConfig     nvidia.Config
}

func main() {
	// .env is for local development. A missing file is normal in Dokploy, where
	// environment variables are injected directly into the container.
	_ = godotenv.Load()

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	client := nvidia.NewClient(cfg.nvidiaConfig)
	gptHandler := handler.NewGPTHandler(cfg.apiSharedSecret, ratelimit.New(cfg.rateLimit), client)

	mux := http.NewServeMux()
	mux.Handle("GET /gpt", gptHandler)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("starting HTTP server", "port", cfg.port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("HTTP server stopped", "error", err)
		os.Exit(1)
	}
}

func loadConfig() (config, error) {
	rateLimitSeconds, err := envInt("RATE_LIMIT_SECONDS", 8)
	if err != nil || rateLimitSeconds <= 0 {
		return config{}, fmt.Errorf("RATE_LIMIT_SECONDS must be a positive integer")
	}

	maxTokens, err := envInt("NVIDIA_MAX_TOKENS", 120)
	if err != nil || maxTokens <= 0 {
		return config{}, fmt.Errorf("NVIDIA_MAX_TOKENS must be a positive integer")
	}

	temperature, err := envFloat("NVIDIA_TEMPERATURE", 0.7)
	if err != nil || temperature < 0 || temperature > 2 {
		return config{}, fmt.Errorf("NVIDIA_TEMPERATURE must be between 0 and 2")
	}

	apiKey := strings.TrimSpace(os.Getenv("NVIDIA_API_KEY"))
	if apiKey == "" {
		return config{}, fmt.Errorf("NVIDIA_API_KEY is required")
	}
	model := strings.TrimSpace(os.Getenv("NVIDIA_MODEL"))
	if model == "" {
		return config{}, fmt.Errorf("NVIDIA_MODEL is required")
	}
	sharedSecret := strings.TrimSpace(os.Getenv("API_SHARED_SECRET"))
	if sharedSecret == "" {
		return config{}, fmt.Errorf("API_SHARED_SECRET is required")
	}

	return config{
		port:            envString("PORT", "8080"),
		apiSharedSecret: sharedSecret,
		rateLimit:        time.Duration(rateLimitSeconds) * time.Second,
		nvidiaConfig: nvidia.Config{
			APIKey:       apiKey,
			Model:        model,
			MaxTokens:    maxTokens,
			Temperature:  temperature,
			SystemPrompt: envString("SYSTEM_PROMPT", "You are a helpful Twitch chat assistant. Keep every answer concise and safe for a public chat."),
		},
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
