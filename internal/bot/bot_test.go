package bot

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"twitch-gpt-bot/internal/ratelimit"
	"twitch-gpt-bot/internal/twitch"
)

type mockSender struct {
	mu       sync.Mutex
	messages []string
}

func (m *mockSender) SendMessage(ctx context.Context, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, text)
	return nil
}

func (m *mockSender) LastMessage() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[len(m.messages)-1]
}

type mockGenerator struct {
	reply string
	err   error
}

func (g *mockGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	return g.reply, g.err
}

func TestBotHandleMessage_ValidCommand(t *testing.T) {
	sender := &mockSender{}
	gen := &mockGenerator{reply: "This is a great answer!"}
	limiter := ratelimit.New(5 * time.Second)

	b := New(Config{
		Prefix: "!gpt",
	}, limiter, gen, sender)

	msg := &twitch.ParsedLine{
		IsPrivmsg: true,
		User:      "aleca",
		Channel:   "#streamer",
		Text:      "!gpt tell me something cool",
	}

	b.HandleMessage(context.Background(), msg)

	// Wait briefly for asynchronous execution
	time.Sleep(50 * time.Millisecond)

	last := sender.LastMessage()
	expected := "@aleca This is a great answer!"
	if last != expected {
		t.Fatalf("expected %q, got %q", expected, last)
	}
}

func TestBotHandleMessage_IgnoresNonCommand(t *testing.T) {
	sender := &mockSender{}
	gen := &mockGenerator{reply: "should not be called"}
	limiter := ratelimit.New(5 * time.Second)

	b := New(Config{Prefix: "!gpt"}, limiter, gen, sender)

	msg := &twitch.ParsedLine{
		IsPrivmsg: true,
		User:      "viewer",
		Text:      "hello everyone",
	}

	b.HandleMessage(context.Background(), msg)
	time.Sleep(20 * time.Millisecond)

	if sender.LastMessage() != "" {
		t.Fatalf("expected no messages sent, got %q", sender.LastMessage())
	}
}

func TestBotHandleMessage_CooldownEnforced(t *testing.T) {
	sender := &mockSender{}
	gen := &mockGenerator{reply: "First answer"}
	limiter := ratelimit.New(10 * time.Second)

	b := New(Config{Prefix: "!gpt"}, limiter, gen, sender)

	msg := &twitch.ParsedLine{
		IsPrivmsg: true,
		User:      "spammer",
		Text:      "!gpt first",
	}

	b.HandleMessage(context.Background(), msg)
	time.Sleep(50 * time.Millisecond)

	if sender.LastMessage() != "@spammer First answer" {
		t.Fatalf("unexpected first message: %s", sender.LastMessage())
	}

	// Second immediate message should be dropped by rate limiter
	msg2 := &twitch.ParsedLine{
		IsPrivmsg: true,
		User:      "spammer",
		Text:      "!gpt second",
	}
	b.HandleMessage(context.Background(), msg2)
	time.Sleep(50 * time.Millisecond)

	if len(sender.messages) != 1 {
		t.Fatalf("expected only 1 message sent due to cooldown, got %d", len(sender.messages))
	}
}

func TestBotHandleMessage_GeneratorError(t *testing.T) {
	sender := &mockSender{}
	gen := &mockGenerator{err: errors.New("api failure")}
	limiter := ratelimit.New(5 * time.Second)

	b := New(Config{Prefix: "!gpt"}, limiter, gen, sender)

	msg := &twitch.ParsedLine{
		IsPrivmsg: true,
		User:      "user1",
		Text:      "!gpt broken",
	}

	b.HandleMessage(context.Background(), msg)
	time.Sleep(50 * time.Millisecond)

	last := sender.LastMessage()
	if !strings.Contains(last, "@user1") || !strings.Contains(last, "Sorry") {
		t.Fatalf("expected fallback message with user mention, got %q", last)
	}
}

func TestNormalizeForTwitch(t *testing.T) {
	input := "Line 1\nLine 2\n\nLine 3    with extra   spaces"
	normalized := NormalizeForTwitch("user1", input)
	expected := "@user1 Line 1 Line 2 Line 3 with extra spaces"
	if normalized != expected {
		t.Fatalf("expected %q, got %q", expected, normalized)
	}
}
