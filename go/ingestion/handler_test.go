package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	util "github.com/mehrnazm/webhookx/go/util"
	"github.com/redis/go-redis/v9"
)

// MockRedisClient is a simple mock for Redis
type MockRedisClient struct {
	addedMessages []util.WebhookMessage
	shouldFail    bool
}

func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "EXISTS")
	cmd.SetVal(1) // drop always exists in tests
	return cmd
}

func (m *MockRedisClient) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	// Mock implementation - just return a dummy cmd
	cmd := redis.NewStringCmd(ctx, "XADD")
	if m.shouldFail {
		cmd.SetVal("")
		cmd.SetErr(redis.Nil)
	} else {
		cmd.SetVal("1234567890-0")
	}
	return cmd
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusCmd(ctx, "PING")
}

func (m *MockRedisClient) Close() error {
	return nil
}

// newTestHandler creates a handler with a mock Redis client and test logger
func newTestHandler() *Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mockRedis := &MockRedisClient{shouldFail: false}
	return NewHandler(mockRedis, logger)
}

func TestPublicDropPostValid(t *testing.T) {
	handler := newTestHandler()

	body := `{"id": "evt_123", "type": "payment.success"}`
	req := httptest.NewRequest("POST", "/drop/test123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "custom-value")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "test123"})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	respBody, _ := io.ReadAll(w.Body)
	if len(respBody) > 0 {
		t.Errorf("Expected empty response body, got %s", string(respBody))
	}
}

func TestPublicDropPostPayloadTooLarge(t *testing.T) {
	handler := newTestHandler()

	largebody := bytes.NewReader(make([]byte, 2*1024*1024))
	req := httptest.NewRequest("POST", "/drop/test123", largebody)
	req.ContentLength = int64(2 * 1024 * 1024)
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "test123"})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413, got %d", w.Code)
	}
}

func TestPublicDropPostJSONParseError(t *testing.T) {
	handler := newTestHandler()

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/drop/test123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "test123"})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestPublicDropPostMissingHeaders(t *testing.T) {
	handler := newTestHandler()

	body := `{"data": "test"}`
	req := httptest.NewRequest("POST", "/drop/minimal", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "minimal"})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 even with minimal headers, got %d", w.Code)
	}
}

func TestPublicDropPostNonJSONBody(t *testing.T) {
	handler := newTestHandler()

	body := `order_id=12345&status=paid&amount=49.99`
	req := httptest.NewRequest("POST", "/drop/shopify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "shopify"})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for form data, got %d", w.Code)
	}
}

func TestPublicDropPostEmptyBody(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest("POST", "/drop/test", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "test"})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for empty body, got %d", w.Code)
	}
}
