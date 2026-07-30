package nvidia

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientGenerateSendsNVIDIAChatCompletion(t *testing.T) {
	var gotRequest *http.Request
	var gotBody string

	client := NewClientWithHTTPClient(Config{
		APIKey:       "nvidia-key",
		Model:        "meta/llama-3.1-8b-instruct",
		MaxTokens:    120,
		Temperature:  0.7,
		SystemPrompt: "Keep answers concise.",
	}, &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotRequest = req
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		gotBody = string(body)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"role":"assistant","content":"hello from NVIDIA"}}]}`)),
		}, nil
	})})

	answer, err := client.Generate(context.Background(), "say hello")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if answer != "hello from NVIDIA" {
		t.Fatalf("Generate() = %q, want NVIDIA response", answer)
	}
	if gotRequest.Method != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotRequest.Method)
	}
	if gotRequest.URL.String() != "https://integrate.api.nvidia.com/v1/chat/completions" {
		t.Fatalf("URL = %q, want NVIDIA Chat Completions endpoint", gotRequest.URL)
	}
	if authorization := gotRequest.Header.Get("Authorization"); authorization != "Bearer nvidia-key" {
		t.Fatalf("Authorization = %q, want bearer API key", authorization)
	}
	for _, fragment := range []string{
		`"model":"meta/llama-3.1-8b-instruct"`,
		`"max_tokens":120`,
		`"temperature":0.7`,
		`"role":"system","content":"Keep answers concise."`,
		`"role":"user","content":"say hello"`,
	} {
		if !strings.Contains(gotBody, fragment) {
			t.Fatalf("request body %q does not contain %q", gotBody, fragment)
		}
	}
}
