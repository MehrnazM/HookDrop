package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	util "github.com/mehrnazm/webhookx/go/util"
	"golang.org/x/crypto/bcrypt"
)

// DataAPIClient is the port. Processor business logic depends only on this interface.
// Swap the adapter (PostgresDataAPIClient today, HTTPDataAPIClient if needed later)
// without touching processor.go, handler.go, or sse.go.
type DataAPIClient interface {
	ValidateDropAndToken(ctx context.Context, dropSlug, sessionToken string) (*util.Drop, error)
	StoreWebhookEvent(ctx context.Context, event util.WebhookMessage) error
	DropExists(ctx context.Context, dropSlug string) bool
}

// PostgresDataAPIClient is the adapter that queries Postgres directly.
type PostgresDataAPIClient struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewPostgresDataAPIClient(db *sql.DB, logger *slog.Logger) *PostgresDataAPIClient {
	return &PostgresDataAPIClient{db: db, logger: logger}
}

func (c *PostgresDataAPIClient) ValidateDropAndToken(ctx context.Context, dropSlug, sessionToken string) (*util.Drop, error) {
	var id, urlSlug, hashedToken string
	var createdAt, expiresAt time.Time

	err := c.db.QueryRowContext(ctx,
		`SELECT id, url_slug, session_token, created_at, expires_at
		 FROM drops WHERE url_slug = $1`,
		dropSlug,
	).Scan(&id, &urlSlug, &hashedToken, &createdAt, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedToken), []byte(sessionToken)); err != nil {
		return nil, nil
	}

	if time.Now().After(expiresAt) {
		return nil, nil
	}

	return &util.Drop{
		Slug:      urlSlug,
		Expiry:    expiresAt,
		CreatedAt: createdAt,
	}, nil
}

func (c *PostgresDataAPIClient) StoreWebhookEvent(ctx context.Context, event util.WebhookMessage) error {
	var dropID string
	err := c.db.QueryRowContext(ctx,
		`SELECT id FROM drops WHERE url_slug = $1`,
		event.DropSlug,
	).Scan(&dropID)
	if err == sql.ErrNoRows {
		c.logger.Warn("drop not found when storing event", "slug", event.DropSlug)
		return nil // drop was deleted between ingestion and processing; discard silently
	}
	if err != nil {
		return err
	}

	headersJSON, _ := json.Marshal(event.Payload.Headers)
	qpJSON, _ := json.Marshal(event.Payload.QueryParams)
	bodyJSON, _ := json.Marshal(event.Payload.Body)

	var ip interface{}
	if event.Metadata.IPAddress != "" {
		ip = event.Metadata.IPAddress
	}

	_, err = c.db.ExecContext(ctx,
		`INSERT INTO webhook_events
		 (drop_id, http_method, path, headers, query_params, body, received_at, ip_address)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		dropID,
		event.Metadata.Method,
		event.Metadata.Path,
		headersJSON, qpJSON, bodyJSON,
		event.Metadata.ReceivedAt,
		ip,
	)
	return err
}

func (c *PostgresDataAPIClient) DropExists(ctx context.Context, dropSlug string) bool {
	var exists bool
	err := c.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM drops WHERE url_slug = $1 AND expires_at > NOW())`,
		dropSlug,
	).Scan(&exists)
	if err != nil {
		c.logger.Error("drop exists check failed", "slug", dropSlug, "err", err)
		return false
	}
	return exists
}
