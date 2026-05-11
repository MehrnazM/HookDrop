package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	util "github.com/mehrnazm/webhookx/go/util"
	"github.com/redis/go-redis/v9"
)

// RedisClient interface defines the methods we use from redis.Client
type RedisClient interface {
	XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

type Handler struct {
	redisClient RedisClient
	logger      *slog.Logger
}

func NewHandler(redis RedisClient, logger *slog.Logger) *Handler {
	return &Handler{
		redisClient: redis,
		logger:      logger,
	}
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// PublicDropPost handles POST /drop/:url_slug
func (h *Handler) PublicDropPost(w http.ResponseWriter, r *http.Request) {
	// Extract URL slug from route
	vars := mux.Vars(r)
	dropSlug := vars["url_slug"]

	h.logger.Info("webhook received", "drop_slug", dropSlug, "method", r.Method, "path", r.RequestURI)

	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	reqLogger := h.logger.With("request-ID", requestID)

	// Validate request size
	if err := ValidateRequest(r); err != nil {
		reqLogger.Error("request validation failed", "err", err)
		util.WriteError(w, err)
		return
	}

	// Read body
	body, validErr := ReadRequestBody(r)
	if validErr != nil {
		reqLogger.Error("reading request body failed", "err", validErr)
		util.WriteError(w, validErr)
		return
	}

	// Parse body as JSON if content-type is application/json
	var bodyJSON interface{}
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.Unmarshal(body, &bodyJSON); err != nil {
			// Store as raw string if JSON parsing fails
			bodyJSON = map[string]string{
				"_raw":          string(body),
				"_content_type": contentType,
				"_parse_error":  err.Error(),
			}
		}
	} else if len(body) > 0 {
		// Non-JSON body
		bodyJSON = map[string]string{
			"_raw":          string(body),
			"_content_type": contentType,
		}
	}

	// Capture headers (skip hop-by-hop headers)
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Capture query params
	queryParams := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	// Get remote IP (handle X-Forwarded-For for proxies)
	remoteIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		remoteIP = strings.TrimSpace(parts[0])
	}

	// Build webhook message
	msg := util.WebhookMessage{
		DropSlug: dropSlug,
		Metadata: util.WebhookMessageMetadata{
			Method:     r.Method,
			Path:       r.RequestURI,
			IPAddress:  remoteIP,
			ReceivedAt: time.Now().UTC(),
		},
		Payload: util.WebhookMessagePayload{
			Headers:     headers,
			QueryParams: queryParams,
			Body:        bodyJSON,
		},
	}

	reqLogger.Debug("webhook message to publish to redis", "msg", fmt.Sprintf("%+v", msg))

	// Enqueue to Redis
	if err := EnqueueWebhook(r.Context(), h.redisClient, msg, reqLogger); err != nil {
		reqLogger.Error("enqueue to redis faild", "err", err)
		util.WriteError(w, util.InternalError("Failed to enqueue webhook"))
		return
	}

	// Return 200 OK with empty body
	w.WriteHeader(http.StatusOK)
}
