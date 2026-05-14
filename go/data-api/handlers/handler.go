package handlers

import (
	"log/slog"

	"github.com/redis/go-redis/v9"
	dbpkg "github.com/mehrnazm/webhookx/go/data-api/db"
)

type Handler struct {
	logger  *slog.Logger
	queries *dbpkg.Queries
	redis   *redis.Client
}

func NewHandler(queries *dbpkg.Queries, logger *slog.Logger, rc *redis.Client) *Handler {
	return &Handler{queries: queries, logger: logger, redis: rc}
}
