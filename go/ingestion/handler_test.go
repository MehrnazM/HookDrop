package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

// evalCall records a single call to MockRedisClient.Eval for assertion in tests.
type evalCall struct {
	keys []string
	args []interface{}
}

// MockRedisClient is a simple mock for Redis
type MockRedisClient struct {
	addedMessages  []util.WebhookMessage
	shouldFail     bool
	dropExists     bool
	existsErr      error
	// Seeded starting values for rate-limit counters.
	// Eval increments these and returns the new value.
	lifetimeCounts map[string]int64
	rateCounts     map[string]int64
	evalCalls      []evalCall
	evalErr        error
}

func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "EXISTS")
	if m.existsErr != nil {
		cmd.SetErr(m.existsErr)
		return cmd
	}
	if m.dropExists {
		cmd.SetVal(1)
	} else {
		cmd.SetVal(0)
	}
	return cmd
}

func (m *MockRedisClient) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, "XADD")
	if m.shouldFail {
		cmd.SetVal("")
		cmd.SetErr(redis.Nil)
	} else {
		cmd.SetVal("1234567890-0")
	}
	return cmd
}

func (m *MockRedisClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	m.evalCalls = append(m.evalCalls, evalCall{keys: keys, args: args})

	cmd := redis.NewCmd(ctx, "eval")
	if m.evalErr != nil {
		cmd.SetErr(m.evalErr)
		return cmd
	}
	if len(keys) == 0 {
		cmd.SetVal(int64(1))
		return cmd
	}

	if m.lifetimeCounts == nil {
		m.lifetimeCounts = make(map[string]int64)
	}
	if m.rateCounts == nil {
		m.rateCounts = make(map[string]int64)
	}

	key := keys[0]
	if strings.HasSuffix(key, ":lifetime_count") {
		m.lifetimeCounts[key]++
		cmd.SetVal(m.lifetimeCounts[key])
	} else if strings.HasSuffix(key, ":rate_window") {
		m.rateCounts[key]++
		cmd.SetVal(m.rateCounts[key])
	} else {
		cmd.SetVal(int64(1))
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
	mockRedis := &MockRedisClient{shouldFail: false, dropExists: true}
	return NewHandler(mockRedis, logger)
}

func newTestHandlerWithRedis(mock *MockRedisClient) *Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return NewHandler(mock, logger)
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

func TestPublicDropPost_DropNotFound(t *testing.T) {
	handler := newTestHandlerWithRedis(&MockRedisClient{dropExists: false})

	req := httptest.NewRequest("POST", "/drop/unknownslug", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"url_slug": "unknownslug"})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown drop, got %d", w.Code)
	}
}

func TestPublicDropPost_RedisExistsError(t *testing.T) {
	handler := newTestHandlerWithRedis(&MockRedisClient{existsErr: errors.New("redis unavailable")})

	req := httptest.NewRequest("POST", "/drop/test123", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"url_slug": "test123"})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on redis error, got %d", w.Code)
	}
}

// TestRateLimit_LifetimeCap verifies that the 10,001st request is rejected with 429.
func TestRateLimit_LifetimeCap(t *testing.T) {
	slug := "lifeslug"
	lifetimeKey := fmt.Sprintf("drop:%s:lifetime_count", slug)

	mock := &MockRedisClient{
		dropExists:     true,
		lifetimeCounts: map[string]int64{lifetimeKey: 10_000}, // next Eval returns 10_001
	}
	handler := newTestHandlerWithRedis(mock)

	req := httptest.NewRequest("POST", "/drop/"+slug, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"url_slug": slug})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on lifetime cap, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "rate_limit_exceeded") {
		t.Errorf("expected rate_limit_exceeded in body, got: %s", body)
	}
	if strings.Contains(body, "too quickly") {
		t.Errorf("lifetime rejection must not use the rate-cap message, got: %s", body)
	}
	if w.Header().Get("Retry-After") != "" {
		t.Errorf("lifetime cap must not set Retry-After header")
	}
}

// TestRateLimit_SustainedRate verifies that the 101st request within one second is rejected with 429.
func TestRateLimit_SustainedRate(t *testing.T) {
	slug := "rateslug"
	rateKey := fmt.Sprintf("drop:%s:rate_window", slug)

	mock := &MockRedisClient{
		dropExists: true,
		rateCounts: map[string]int64{rateKey: 100}, // next Eval returns 101
	}
	handler := newTestHandlerWithRedis(mock)

	req := httptest.NewRequest("POST", "/drop/"+slug, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"url_slug": slug})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on rate cap, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "rate_limit_exceeded") {
		t.Errorf("expected rate_limit_exceeded in body, got: %s", body)
	}
	if !strings.Contains(body, "too quickly") {
		t.Errorf("rate cap must use the 'too quickly' message, got: %s", body)
	}
	if w.Header().Get("Retry-After") != "1" {
		t.Errorf("expected Retry-After: 1 on rate cap, got %q", w.Header().Get("Retry-After"))
	}
}

// TestRateLimit_LifetimeCounter_MirrorsDopTTL verifies that the lifetime Lua script is called
// with KEYS = [lifetimeKey, dropKey] so it can mirror the drop's TTL onto the counter.
func TestRateLimit_LifetimeCounter_MirrorsDopTTL(t *testing.T) {
	slug := "ttlslug"
	mock := &MockRedisClient{dropExists: true}
	handler := newTestHandlerWithRedis(mock)

	req := httptest.NewRequest("POST", "/drop/"+slug, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"url_slug": slug})

	w := httptest.NewRecorder()
	handler.PublicDropPost(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(mock.evalCalls) < 1 {
		t.Fatal("expected at least one Eval call for the lifetime script")
	}
	first := mock.evalCalls[0]
	if len(first.keys) != 2 {
		t.Fatalf("lifetime script must receive 2 keys, got %d: %v", len(first.keys), first.keys)
	}
	wantLifetimeKey := fmt.Sprintf("drop:%s:lifetime_count", slug)
	wantDropKey := fmt.Sprintf("drop:%s", slug)
	if first.keys[0] != wantLifetimeKey {
		t.Errorf("KEYS[0] = %q, want %q", first.keys[0], wantLifetimeKey)
	}
	if first.keys[1] != wantDropKey {
		t.Errorf("KEYS[1] = %q, want %q — drop key needed for TTL mirroring", first.keys[1], wantDropKey)
	}
}
