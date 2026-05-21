package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	util "github.com/mehrnazm/hookdrop/go/util"
	"github.com/redis/go-redis/v9"
)

// EnqueueWebhook publishes a webhook message to Redis Streams
func EnqueueWebhook(ctx context.Context, rc RedisClient, msg util.WebhookMessage, logger *slog.Logger) error {
	// Marshal to JSON

	body, err := json.Marshal(msg)
	if err != nil {
		logger.Error("failed to marshal webhook message", "err", err)
		return err
	}

	// Enqueue to Redis Streams with a 5-second timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = rc.XAdd(ctx, &redis.XAddArgs{
		Stream: "webhooks",
		MaxLen: 10_000, // cap stream size; processor should drain fast enough that this is never hit
		Approx: true,   // ~ prefix: efficient approximate trimming
		Values: []interface{}{"drop", string(body)},
	}).Result()
	if err != nil {
		logger.Error("failed to enqueue webhook to redis", "err", err, "drop_slug", msg.DropSlug)
		return err
	}

	logger.Debug("webhook enqueued", "drop_slug", msg.DropSlug)
	return nil
}
