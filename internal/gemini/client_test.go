package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientGenerateSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "test-model:generateContent") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "secret-key" {
			t.Fatalf("expected key secret-key, got %s", r.URL.Query().Get("key"))
		}

		var reqBody geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(reqBody.Contents) == 0 || len(reqBody.Contents[0].Parts) == 0 {
			t.Fatalf("expected non-empty prompt parts")
		}
		if reqBody.Contents[0].Parts[0].Text != "hello" {
			t.Fatalf("expected hello, got %s", reqBody.Contents[0].Parts[0].Text)
		}

		resp := geminiResponse{
			Candidates: []candidate{
				{
					Content: contentData{
						Parts: []part{
							{Text: "world"},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(Config{
		APIKey:       "secret-key",
		Model:        "test-model",
		MaxTokens:    100,
		Temperature:  0.5,
		SystemPrompt: "Be brief.",
	}, ts.URL, ts.Client())

	reply, err := client.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "world" {
		t.Fatalf("expected world, got %s", reply)
	}
}

func TestClientGenerateErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"quota exceeded"}}`))
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(Config{
		APIKey: "secret-key",
		Model:  "test-model",
	}, ts.URL, ts.Client())

	_, err := client.Generate(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Fatalf("expected 429 in error, got %v", err)
	}
}

func TestClientGenerateEmptyCandidates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Candidates: []candidate{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(Config{
		APIKey: "secret-key",
		Model:  "test-model",
	}, ts.URL, ts.Client())

	_, err := client.Generate(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for empty candidates, got nil")
	}
}
