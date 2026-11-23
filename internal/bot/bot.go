package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"twitch-gpt-bot/internal/llm"
	"twitch-gpt-bot/internal/twitch"
)

const (
	maxTwitchMessageLength = 450
	defaultFallbackMessage = "Sorry, I couldn't answer right now. Please try again shortly."
	defaultTimeout         = 15 * time.Second
)

type MessageSender interface {
	SendMessage(ctx context.Context, text string) error
}

type RateLimiter interface {
	Allow(user string, now time.Time) bool
}

type Config struct {
	Prefix          string
	FallbackMessage string
	Timeout         time.Duration
}

type Bot struct {
	prefix          string
	fallbackMessage string
	timeout         time.Duration
	limiter         RateLimiter
	generator       llm.Generator
	sender          MessageSender
}

func New(cfg Config, limiter RateLimiter, generator llm.Generator, sender MessageSender) *Bot {
	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" {
		prefix = "!gpt"
	}
	fallback := strings.TrimSpace(cfg.FallbackMessage)
	if fallback == "" {
		fallback = defaultFallbackMessage
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &Bot{
		prefix:          prefix,
		fallbackMessage: fallback,
		timeout:         timeout,
		limiter:         limiter,
		generator:       generator,
		sender:          sender,
	}
}

// HandleMessage inspects incoming chat messages, checks rate limits, and triggers LLM replies.
func (b *Bot) HandleMessage(ctx context.Context, msg *twitch.ParsedLine) {
	if msg == nil || !msg.IsPrivmsg {
		return
	}

	trimmedText := strings.TrimSpace(msg.Text)
	lowerText := strings.ToLower(trimmedText)
	lowerPrefix := strings.ToLower(b.prefix)

	if !strings.HasPrefix(lowerText, lowerPrefix) {
		return
	}

	// Must match prefix boundary (e.g. "!gpt hello", not "!gpto")
	rest := trimmedText[len(b.prefix):]
	if len(rest) > 0 && rest[0] != ' ' {
		return
	}

	query := strings.TrimSpace(rest)
	if query == "" {
		return
	}

	user := strings.TrimSpace(msg.User)
	if user == "" {
		return
	}

	if !b.limiter.Allow(user, time.Now()) {
		slog.Debug("user request rate limited", "user", user)
		return
	}

	go b.processQuery(ctx, user, query)
}

func (b *Bot) processQuery(ctx context.Context, user, query string) {
	reqCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	slog.Info("processing prompt for user", "user", user, "query_length", len(query))
	reply, err := b.generator.Generate(reqCtx, query)
	if err != nil {
		slog.Error("failed to generate response", "user", user, "error", err)
		_ = b.sender.SendMessage(ctx, fmt.Sprintf("@%s %s", user, b.fallbackMessage))
		return
	}

	outMessage := NormalizeForTwitch(user, reply)
	if err := b.sender.SendMessage(ctx, outMessage); err != nil {
		slog.Error("failed to send twitch message", "user", user, "error", err)
	}
}

// NormalizeForTwitch cleans whitespace, prepends @user mention, and truncates to 450 characters.
func NormalizeForTwitch(user, text string) string {
	cleaned := strings.Join(strings.Fields(text), " ")
	full := fmt.Sprintf("@%s %s", user, cleaned)
	if len(full) > maxTwitchMessageLength {
		return full[:maxTwitchMessageLength]
	}
	return full
}
