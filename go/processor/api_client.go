package main

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/mehrnazm/webhookx/go/util"
)

type DataAPIClient interface {
	// Get drop metadata and validate session
	ValidateDropAndToken(ctx context.Context, dropSlug, sessionToken string) (*util.Drop, error)

	// Store webhook event
	StoreWebhookEvent(ctx context.Context, event util.WebhookMessage) error

	// Check if drop still exists (for cleanup)
	DropExists(ctx context.Context, dropSlug string) bool
}

// V1: Stub that queries PostgreSQL directly
type StubDataAPIClient struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewStubDataAPIClient(db *sql.DB, logger *slog.Logger) *StubDataAPIClient {
	return &StubDataAPIClient{
		db:     db,
		logger: logger,
	}
}

func (c *StubDataAPIClient) ValidateDropAndToken(ctx context.Context, dropSlug, sessionToken string) (*util.Drop, error) {
	// Query: SELECT id, expires_at FROM drops WHERE id = $1
	// Check: if expires_at < now(), return error (drop expired)
	// Compare: bcrypt.CompareHashAndPassword(hashedToken, []byte(sessionToken))
	// Return: &Drop{ID, ExpiresAt} or error
	return nil, nil
}

func (c *StubDataAPIClient) StoreWebhookEvent(ctx context.Context, event util.WebhookMessage) error {
	// INSERT INTO webhook_events (id, drop_id, http_method, path, headers, query_params, body, received_at, ip_address)
	// VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	// Return error if fails
	return nil
}

func (c *StubDataAPIClient) DropExists(ctx context.Context, dropSlug string) bool {
	// Query: SELECT 1 FROM drops WHERE id = $1 AND expires_at > now()
	// Return true/false
	return false
}
