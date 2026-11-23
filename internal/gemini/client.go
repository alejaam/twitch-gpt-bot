package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com"

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
	baseURL      string
	httpClient   *http.Client
}

func NewClient(config Config) *Client {
	return NewClientWithBaseURL(config, defaultBaseURL, &http.Client{Timeout: 15 * time.Second})
}

func NewClientWithBaseURL(config Config, baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	model := strings.TrimSpace(config.Model)
	if model == "" {
		model = "gemini-2.5-flash"
	}
	maxTokens := config.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 150
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		apiKey:       config.APIKey,
		model:        model,
		maxTokens:    maxTokens,
		temperature:  config.Temperature,
		systemPrompt: config.SystemPrompt,
		baseURL:      baseURL,
		httpClient:   httpClient,
	}
}

type part struct {
	Text string `json:"text"`
}

type contentData struct {
	Parts []part `json:"parts"`
	Role  string `json:"role,omitempty"`
}

type systemInstructionData struct {
	Parts []part `json:"parts"`
}

type generationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type geminiRequest struct {
	Contents          []contentData          `json:"contents"`
	SystemInstruction *systemInstructionData `json:"systemInstruction,omitempty"`
	GenerationConfig  generationConfig       `json:"generationConfig"`
}

type candidate struct {
	Content contentData `json:"content"`
}

type geminiResponse struct {
	Candidates []candidate `json:"candidates"`
}

// Generate sends a prompt to Google Gemini generateContent endpoint.
func (c *Client) Generate(ctx context.Context, query string) (string, error) {
	reqBody := geminiRequest{
		Contents: []contentData{
			{
				Role:  "user",
				Parts: []part{{Text: query}},
			},
		},
		GenerationConfig: generationConfig{
			MaxOutputTokens: c.maxTokens,
			Temperature:     c.temperature,
		},
	}

	if strings.TrimSpace(c.systemPrompt) != "" {
		reqBody.SystemInstruction = &systemInstructionData{
			Parts: []part{{Text: c.systemPrompt}},
		}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal Gemini request: %w", err)
	}

	endpointURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s",
		c.baseURL,
		url.PathEscape(c.model),
		url.QueryEscape(c.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create Gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call Gemini: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return "", fmt.Errorf("Gemini returned status %d", resp.StatusCode)
	}

	var result geminiResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("decode Gemini response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini returned an empty candidate list")
	}

	reply := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
	if reply == "" {
		return "", fmt.Errorf("Gemini returned empty text")
	}

	return reply, nil
}
