package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const chatCompletionsURL = "https://api.openai.com/v1/chat/completions"

type Config struct {
	APIKey       string
	Model        string
	MaxTokens    int
	Temperature  float64
	SystemPrompt string
}

type Client struct {
	apiKey       string
	model        string
	maxTokens    int
	temperature  float64
	systemPrompt string
	httpClient   *http.Client
}

func NewClient(config Config) *Client {
	return NewClientWithHTTPClient(config, &http.Client{Timeout: 15 * time.Second})
}

// NewClientWithHTTPClient permits isolated HTTP tests without calling OpenAI.
func NewClientWithHTTPClient(config Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	return &Client{
		apiKey:       config.APIKey,
		model:        config.Model,
		maxTokens:    config.MaxTokens,
		temperature:  config.Temperature,
		systemPrompt: config.SystemPrompt,
		httpClient:   httpClient,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Generate makes one Chat Completions request. Errors deliberately omit request
// and response bodies so user prompts and credentials never end up in logs.
func (c *Client) Generate(ctx context.Context, query string) (string, error) {
	payload, err := json.Marshal(chatCompletionRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: c.systemPrompt},
			{Role: "user", Content: query},
		},
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	})
	if err != nil {
		return "", fmt.Errorf("marshal OpenAI request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionsURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create OpenAI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call OpenAI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return "", fmt.Errorf("OpenAI returned status %d", resp.StatusCode)
	}

	var completion chatCompletionResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&completion); err != nil {
		return "", fmt.Errorf("decode OpenAI response: %w", err)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("OpenAI returned an empty completion")
	}

	return completion.Choices[0].Message.Content, nil
}
