package handler

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"
)

const (
	maxTwitchMessageLength = 450
	fallbackMessage        = "Sorry, I couldn't answer right now. Please try again shortly."
)

type Generator interface {
	Generate(context.Context, string) (string, error)
}

type RateLimiter interface {
	Allow(user string, now time.Time) bool
}

type GPTHandler struct {
	sharedSecret string
	limiter      RateLimiter
	client       Generator
}

func NewGPTHandler(sharedSecret string, limiter RateLimiter, client Generator) *GPTHandler {
	return &GPTHandler{
		sharedSecret: sharedSecret,
		limiter:      limiter,
		client:       client,
	}
}

func (h *GPTHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method != http.MethodGet {
		writePlainText(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	user := strings.TrimSpace(r.URL.Query().Get("user"))
	if !validSecret(r.URL.Query().Get("secret"), h.sharedSecret) {
		writePlainText(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if query == "" {
		writePlainText(w, http.StatusBadRequest, "query is required")
		return
	}
	if user == "" {
		writePlainText(w, http.StatusBadRequest, "user is required")
		return
	}
	if !h.limiter.Allow(user, time.Now()) {
		writePlainText(w, http.StatusTooManyRequests, "Please wait before using !gpt again.")
		return
	}

	response, err := h.client.Generate(r.Context(), query)
	if err != nil {
		writePlainText(w, http.StatusBadGateway, fallbackMessage)
		return
	}

	writePlainText(w, http.StatusOK, normalizeForTwitch(response))
}

func validSecret(provided, expected string) bool {
	if len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func writePlainText(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func normalizeForTwitch(response string) string {
	response = strings.Join(strings.Fields(response), " ")
	if len(response) > maxTwitchMessageLength {
		return response[:maxTwitchMessageLength]
	}
	return response
}
