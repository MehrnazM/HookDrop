package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// WebhookMessage is what gets enqueued to Redis Streams
type WebhookMessage struct {
	DropSlug    string            `json:"drop_slug"`
	Method      string            `json:"http_method"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	QueryParams map[string]string `json:"query_params"`
	Body        interface{}       `json:"body"`
	IPAddress   string            `json:"ip_address"`
	ReceivedAt  string            `json:"received_at"`
}

// EnqueueWebhook publishes a webhook message to Redis Streams
func EnqueueWebhook(ctx context.Context, rc RedisClient, msg WebhookMessage, logger *slog.Logger) error {
	// Marshal to JSON

	payload, err := json.Marshal(msg)
	if err != nil {
		logger.Error("failed to marshal webhook message", "err", err)
		return err
	}

	// Enqueue to Redis Streams with a 5-second timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = rc.XAdd(ctx, &redis.XAddArgs{
		Stream: "webhooks",
		Values: []interface{}{"payload", string(payload)},
	}).Result()
	if err != nil {
		logger.Error("failed to enqueue webhook to redis", "err", err, "drop_slug", msg.DropSlug)
		return err
	}

	logger.Debug("webhook enqueued", "drop_slug", msg.DropSlug)
	return nil
}
