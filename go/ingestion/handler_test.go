package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

// MockRedisClient is a simple mock for Redis
type MockRedisClient struct {
	addedMessages []WebhookMessage
	shouldFail    bool
}

func (m *MockRedisClient) XAdd(ctx context.Context, stream string, nomkstream string, values ...interface{}) *redis.StringCmd {
	// This is a simplification - in real code you'd use testify/mock
	return nil
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return nil
}

func (m *MockRedisClient) Close() error {
	return nil
}

func TestPublicDropPostValid(t *testing.T) {
	// Create a request
	body := `{"id": "evt_123", "type": "payment.success"}`
	req := httptest.NewRequest("POST", "/drop/test123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "custom-value")

	// Add mux vars (simulating gorilla/mux routing)
	req = mux.SetURLVars(req, map[string]string{"url_slug": "test123"})

	w := httptest.NewRecorder()

	// Call handler
	PublicDropPost(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response is empty
	reqBody, _ := io.ReadAll(w.Body)
	if len(reqBody) > 0 {
		t.Errorf("Expected empty response body, got %s", string(reqBody))
	}
}

func TestPublicDropPostPayloadTooLarge(t *testing.T) {
	// Create a request with Content-Length > 1MB
	largebody := bytes.NewReader(make([]byte, 2*1024*1024))
	req := httptest.NewRequest("POST", "/drop/test123", largebody)
	req.ContentLength = int64(2 * 1024 * 1024)
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "test123"})

	w := httptest.NewRecorder()

	PublicDropPost(w, req)

	// Check response is 413
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413, got %d", w.Code)
	}
}

func TestPublicDropPostJSONParseError(t *testing.T) {
	// Invalid JSON body
	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/drop/test123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "test123"})

	w := httptest.NewRecorder()

	PublicDropPost(w, req)

	// Should still return 200 (we store invalid JSON as raw)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestPublicDropPostMissingHeaders(t *testing.T) {
	// Request with minimal headers (no X-Custom-Header, User-Agent, etc)
	body := `{"data": "test"}`
	req := httptest.NewRequest("POST", "/drop/minimal", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Deliberately omit optional headers

	req = mux.SetURLVars(req, map[string]string{"url_slug": "minimal"})

	w := httptest.NewRecorder()
	PublicDropPost(w, req)

	// Should handle gracefully with minimal headers
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 even with minimal headers, got %d", w.Code)
	}
}

func TestPublicDropPostNonJSONBody(t *testing.T) {
	// Form-encoded body (from Shopify, PayPal, etc)
	body := `order_id=12345&status=paid&amount=49.99`
	req := httptest.NewRequest("POST", "/drop/shopify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "shopify"})

	w := httptest.NewRecorder()
	PublicDropPost(w, req)

	// Should return 200 and store as raw
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for form data, got %d", w.Code)
	}
}

func TestPublicDropPostEmptyBody(t *testing.T) {
	// Some webhook services send empty body
	req := httptest.NewRequest("POST", "/drop/test", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{"url_slug": "test"})

	w := httptest.NewRecorder()
	PublicDropPost(w, req)

	// Should handle empty body gracefully
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for empty body, got %d", w.Code)
	}
}
