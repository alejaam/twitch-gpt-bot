package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"twitch-gpt-bot/internal/ratelimit"
)

type stubClient struct {
	response string
	err      error
}

func (s stubClient) Generate(_ context.Context, _ string) (string, error) {
	return s.response, s.err
}

func TestGPTHandler(t *testing.T) {
	longResponse := strings.Repeat("a", 460) + "\nsecond line"

	tests := []struct {
		name       string
		url        string
		client     stubClient
		wantStatus int
		wantBody   string
	}{
		{
			name:       "rejects an invalid shared secret",
			url:        "/gpt?query=hello&user=alice&secret=wrong",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "unauthorized",
		},
		{
			name:       "rejects an empty query",
			url:        "/gpt?query=&user=alice&secret=expected-secret",
			wantStatus: http.StatusBadRequest,
			wantBody:   "query is required",
		},
		{
			name:       "returns a plain normalized response truncated for Twitch",
			url:        "/gpt?query=hello&user=alice&secret=expected-secret",
			client:     stubClient{response: longResponse},
			wantStatus: http.StatusOK,
			wantBody:   strings.Repeat("a", 450),
		},
		{
			name:       "returns a safe fallback when OpenAI fails",
			url:        "/gpt?query=hello&user=alice&secret=expected-secret",
			client:     stubClient{err: errors.New("upstream failure")},
			wantStatus: http.StatusBadGateway,
			wantBody:   "Sorry, I couldn't answer right now. Please try again shortly.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewGPTHandler("expected-secret", ratelimit.New(time.Second), tt.client)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if body := strings.TrimSpace(rec.Body.String()); body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
			if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
				t.Fatalf("Content-Type = %q, want text/plain", contentType)
			}
		})
	}
}

func TestGPTHandlerRateLimit(t *testing.T) {
	client := stubClient{response: "first answer"}
	h := NewGPTHandler("expected-secret", ratelimit.New(time.Hour), client)

	for requestNumber, wantStatus := range []int{http.StatusOK, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodGet, "/gpt?query=hello&user=alice&secret=expected-secret", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != wantStatus {
			t.Fatalf("request %d status = %d, want %d", requestNumber+1, rec.Code, wantStatus)
		}
	}
}
