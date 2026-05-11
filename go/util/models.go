package util

import "time"

type Drop struct {
	Slug   string    `json:"slug"`
	Token  string    `json:"token"`
	Expiry time.Time `json:"expiry"`
}

type WebhookMessageMetadata struct {
	Method     string    `json:"http_method"`
	Path       string    `json:"path"`
	IPAddress  string    `json:"ip_address"`
	ReceivedAt time.Time `json:"received_at"`
}

type WebhookMessagePayload struct {
	Headers     map[string]string `json:"headers"`
	QueryParams map[string]string `json:"query_params"`
	Body        interface{}       `json:"body"`
}

// WebhookMessage is what gets enqueued to Redis Streams
type WebhookMessage struct {
	DropSlug string                 `json:"drop_slug"`
	Metadata WebhookMessageMetadata `json:"metadata"`
	Payload  WebhookMessagePayload  `json:"payload"`
}
