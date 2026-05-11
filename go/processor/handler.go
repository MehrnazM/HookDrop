package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type Handler struct {
	apiClient DataAPIClient
	sseServer *SSEServer
	logger    *slog.Logger
}

func NewHandler(apiClient DataAPIClient, sseServer *SSEServer, logger *slog.Logger) *Handler {
	return &Handler{
		apiClient: apiClient,
		sseServer: sseServer,
		logger:    logger,
	}
}

func (h *Handler) HandleSSEStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dropSlug := vars["url_slug"]

	h.logger.Info("sse stream received", "drop_slug", dropSlug, "method", r.Method, "path", r.RequestURI)

	bearerToken := r.Header.Get("Authorization")
	if bearerToken == "" || !strings.HasPrefix(bearerToken, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(bearerToken, "Bearer ")

	drop, err := h.apiClient.ValidateDropAndToken(r.Context(), dropSlug, token)
	if err != nil {
		h.logger.Error("failed to validate drop and token", "error", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if drop == nil {
		h.logger.Warn("drop not found or token invalid", "drop_slug", dropSlug)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sseClient := &SSEClient{
		DropSlug: dropSlug,
		Channel:  make(chan []byte, 10),
		Done:     make(chan string, 1),
	}
	h.sseServer.RegisterClient(sseClient)
	defer h.sseServer.UnregisterClient(sseClient)

	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case data := <-sseClient.Channel:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Second):
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case reason := <-sseClient.Done:
			fmt.Fprintf(w, "data: {\"type\":\"%s\"}\n\n", reason)
			flusher.Flush()
			return
		}

	}
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
