package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/mehrnazm/webhookx/go/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock ---

type MockDataAPIClient struct {
	mock.Mock
}

func (m *MockDataAPIClient) ValidateDropAndToken(ctx context.Context, dropSlug, sessionToken string) (*util.Drop, error) {
	args := m.Called(ctx, dropSlug, sessionToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*util.Drop), args.Error(1)
}

func (m *MockDataAPIClient) StoreWebhookEvent(ctx context.Context, event util.WebhookMessage) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockDataAPIClient) DropExists(ctx context.Context, dropID string) bool {
	args := m.Called(ctx, dropID)
	return args.Bool(0)
}

// --- Helpers ---

func newTestSSEServer() *SSEServer {
	return &SSEServer{
		clients: make(map[string]map[*SSEClient]bool),
		logger:  slog.Default(),
	}
}

func newTestRequest(method, url, slug, authHeader string) *http.Request {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so the SSE loop exits right after the handshake
	req := httptest.NewRequest(method, url, nil).WithContext(ctx)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	// inject gorilla/mux vars so the handler can read {url_slug}
	req = mux.SetURLVars(req, map[string]string{"url_slug": slug})
	return req
}

// --- Tests ---

func TestHandleSSEStream_ValidConnection(t *testing.T) {
	mockClient := new(MockDataAPIClient)
	sseServer := newTestSSEServer()
	handler := NewHandler(mockClient, sseServer, slog.Default())

	drop := &util.Drop{Slug: "abc12345"}
	mockClient.On("ValidateDropAndToken", mock.Anything, "abc12345", "valid-token").
		Return(drop, nil)

	req := newTestRequest(http.MethodGet, "/api/drops/abc12345/stream", "abc12345", "Bearer valid-token")
	w := httptest.NewRecorder()

	handler.HandleSSEStream(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no", w.Header().Get("X-Accel-Buffering"))
	assert.Contains(t, w.Body.String(), `data: {"type":"connected"}`)
	mockClient.AssertExpectations(t)
}

func TestHandleSSEStream_MissingToken(t *testing.T) {
	mockClient := new(MockDataAPIClient)
	sseServer := newTestSSEServer()
	handler := NewHandler(mockClient, sseServer, slog.Default())

	req := newTestRequest(http.MethodGet, "/api/drops/abc12345/stream", "abc12345", "")
	w := httptest.NewRecorder()

	handler.HandleSSEStream(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockClient.AssertNotCalled(t, "ValidateDropAndToken")
}

func TestHandleSSEStream_InvalidToken(t *testing.T) {
	mockClient := new(MockDataAPIClient)
	sseServer := newTestSSEServer()
	handler := NewHandler(mockClient, sseServer, slog.Default())

	mockClient.On("ValidateDropAndToken", mock.Anything, "abc12345", "bad-token").
		Return(nil, errors.New("unauthorized"))

	req := newTestRequest(http.MethodGet, "/api/drops/abc12345/stream", "abc12345", "Bearer bad-token")
	w := httptest.NewRecorder()

	handler.HandleSSEStream(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockClient.AssertExpectations(t)
}

func TestBroadcastToClients_DeliversData(t *testing.T) {
	sseServer := newTestSSEServer()

	client := &SSEClient{
		DropSlug: "abc12345",
		Channel:  make(chan []byte, 10),
		Done:     make(chan string, 1),
	}
	sseServer.RegisterClient(client)

	payload := []byte(`{"type":"webhook_received"}`)
	sseServer.BroadcastToClients("abc12345", payload)

	select {
	case received := <-client.Channel:
		assert.Equal(t, payload, received)
	default:
		t.Fatal("expected data on client channel but got nothing")
	}
}

func TestHandleSSEStream_ClientUnregisteredOnDisconnect(t *testing.T) {
	mockClient := new(MockDataAPIClient)
	sseServer := newTestSSEServer()
	handler := NewHandler(mockClient, sseServer, slog.Default())

	drop := &util.Drop{Slug: "abc12345"}
	mockClient.On("ValidateDropAndToken", mock.Anything, "abc12345", "valid-token").
		Return(drop, nil)

	req := newTestRequest(http.MethodGet, "/api/drops/abc12345/stream", "abc12345", "Bearer valid-token")
	w := httptest.NewRecorder()

	handler.HandleSSEStream(w, req)

	// after handler returns the client map entry should be cleaned up
	sseServer.mu.RLock()
	_, exists := sseServer.clients["abc12345"]
	sseServer.mu.RUnlock()

	assert.False(t, exists, "client map entry should be removed after disconnect")
	mockClient.AssertExpectations(t)
}

func TestHandleSSEStream_DropExpired(t *testing.T) {
	mockClient := new(MockDataAPIClient)
	sseServer := newTestSSEServer()
	handler := NewHandler(mockClient, sseServer, slog.Default())

	drop := &util.Drop{Slug: "abc12345"}
	mockClient.On("ValidateDropAndToken", mock.Anything, "abc12345", "valid-token").
		Return(drop, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/drops/abc12345/stream", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req = mux.SetURLVars(req, map[string]string{"url_slug": "abc12345"})
	w := httptest.NewRecorder()

	handlerDone := make(chan struct{})
	go func() {
		defer close(handlerDone)
		handler.HandleSSEStream(w, req)
	}()

	// Give the handler time to register and enter the select loop
	time.Sleep(10 * time.Millisecond)
	sseServer.EvictDrop("abc12345")

	<-handlerDone

	assert.Contains(t, w.Body.String(), `"type":"drop_expired"`)
	mockClient.AssertExpectations(t)
}

func TestUnregisterClient_AfterEviction_NoPanic(t *testing.T) {
	sseServer := newTestSSEServer()
	client := &SSEClient{
		DropSlug: "abc12345",
		Channel:  make(chan []byte, 10),
		Done:     make(chan string, 1),
	}
	sseServer.RegisterClient(client)
	sseServer.EvictDrop("abc12345")

	// deferred UnregisterClient fires after eviction — should not panic
	assert.NotPanics(t, func() {
		sseServer.UnregisterClient(client)
	})
}
