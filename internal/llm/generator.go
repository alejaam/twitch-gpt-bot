package llm

import "context"

// Generator represents any LLM client capable of producing a text completion.
type Generator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
