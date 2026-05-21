package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
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
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// PublicDropPost handles POST /drop/:url_slug
func (h *Handler) PublicDropPost(w http.ResponseWriter, r *http.Request) {
	// Extract URL slug from route
	vars := mux.Vars(r)
	dropSlug := vars["url_slug"]

	h.logger.Info("webhook received", "drop_slug", dropSlug, "method", r.Method, "path", r.RequestURI)

	// Rate limit checks (lifetime cap, then sliding-window rate cap).
	outcome, err := checkLimits(r.Context(), h.redisClient, dropSlug)
	if err != nil {
		h.logger.Error("rate limit check failed", "drop_slug", dropSlug, "err", err)
		// fail open: a transient Redis error should not block ingestion
	}
	switch outcome {
	case rateLimitLifetime:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Drop has reached its request limit. Create a new Drop to continue."}`))
		return
	case rateLimitRate:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Drop is receiving requests too quickly. Slow down or create a new Drop."}`))
		return
	}

	// Reject early if the drop doesn't exist or has expired.
	n, err := h.redisClient.Exists(r.Context(), fmt.Sprintf("drop:%s", dropSlug)).Result()
	if err != nil {
		h.logger.Error("redis exists check failed", "drop_slug", dropSlug, "err", err)
		util.WriteError(w, util.InternalError("Service unavailable"))
		return
	}
	if n == 0 {
		util.WriteError(w, util.NotFound("Drop not found"))
		return
	}

	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	reqLogger := h.logger.With("request-ID", requestID)

	// Validate request size
	if err := ValidateRequest(w, r); err != nil {
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

	// Get remote IP (handle X-Forwarded-For for proxies).
	// Strip port from RemoteAddr ("IP:port" → "IP") so the value is a clean INET address.
	remoteIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		remoteIP = strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
	} else if host, _, err := net.SplitHostPort(remoteIP); err == nil {
		remoteIP = host
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
