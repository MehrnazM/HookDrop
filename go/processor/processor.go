package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	util "github.com/mehrnazm/webhookx/go/util"
	"github.com/redis/go-redis/v9"
)

const (
	streamName    = "webhooks"
	consumerGroup = "processor"
	consumerName  = "processor-1"
)

func ConsumeRedisStreams(ctx context.Context, rc *redis.Client, apiClient DataAPIClient, sseServer *SSEServer, logger *slog.Logger) {
	err := rc.XGroupCreateMkStream(ctx, streamName, consumerGroup, "0").Err()
	if err != nil && !errors.Is(err, redis.Nil) && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		logger.Error("failed to create consumer group", "error", err)
		return
	}

	args := redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: consumerName,
		Streams:  []string{streamName, ">"},
		Count:    10,
		Block:    5 * time.Second,
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		streams, err := rc.XReadGroup(ctx, &args).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			logger.Error("xreadgroup failed", "error", err)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				processMessage(ctx, rc, apiClient, sseServer, logger, msg)
			}
		}
	}
}

func processMessage(ctx context.Context, rc *redis.Client, apiClient DataAPIClient, sseServer *SSEServer, logger *slog.Logger, msg redis.XMessage) {
	ack := func() { rc.XAck(ctx, streamName, consumerGroup, msg.ID) }

	raw, ok := msg.Values["drop"].(string)
	if !ok {
		logger.Warn("message missing drop field", "message_id", msg.ID)
		ack()
		return
	}

	var wm util.WebhookMessage
	if err := json.Unmarshal([]byte(raw), &wm); err != nil {
		logger.Warn("failed to unmarshal webhook message", "message_id", msg.ID, "error", err)
		ack()
		return
	}

	if wm.DropSlug == "" || wm.Metadata.Method == "" || wm.Metadata.Path == "" {
		logger.Warn("webhook message missing required fields", "message_id", msg.ID)
		ack()
		return
	}

	if wm.Metadata.ReceivedAt.IsZero() {
		wm.Metadata.ReceivedAt = time.Now().UTC()
	}

	if err := apiClient.StoreWebhookEvent(ctx, wm); err != nil {
		logger.Error("failed to store webhook event, message will be retried", "drop_slug", wm.DropSlug, "error", err)
		return
	}

	updateJSON, err := json.Marshal(map[string]interface{}{
		"type":  "webhook_received",
		"event": wm,
	})
	if err != nil {
		logger.Error("failed to marshal update payload", "drop_slug", wm.DropSlug, "error", err)
	} else {
		channel := fmt.Sprintf("drops:%s:updates", wm.DropSlug)
		if err := rc.Publish(ctx, channel, updateJSON).Err(); err != nil {
			logger.Error("failed to publish to redis pubsub", "drop_slug", wm.DropSlug, "error", err)
		}
	}

	ack()
}
