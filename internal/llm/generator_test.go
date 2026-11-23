package llm_test

import (
	"testing"

	"twitch-gpt-bot/internal/gemini"
	"twitch-gpt-bot/internal/llm"
	"twitch-gpt-bot/internal/nvidia"
)

func TestGeneratorsImplementInterface(t *testing.T) {
	var _ llm.Generator = (*gemini.Client)(nil)
	var _ llm.Generator = (*nvidia.Client)(nil)
}
