package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	dbpkg "github.com/mehrnazm/hookdrop/go/data-api/db"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) CreateDrop(c *gin.Context) {
	raw, hashed, err := generateSessionToken()
	if err != nil {
		h.logger.Error("failed to generate session token", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate session token",
		})
		return
	}

	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	var insertErr error
	for attempt := range 3 {
		slug := generateSlug()
		insertErr = h.queries.InsertDrop(c.Request.Context(), slug, hashed, expiresAt)
		if insertErr == nil {
			// Cache slug in Redis so ingestion can validate drop existence cheaply.
			// Best-effort: drop is created even if Redis write fails, but log loudly.
			key := fmt.Sprintf("drop:%s", slug)
			if err := h.redis.Set(c.Request.Context(), key, 1, time.Until(expiresAt)).Err(); err != nil {
				h.logger.Error("failed to cache drop slug in redis — ingestion will reject webhooks for this drop", "slug", slug, "err", err)
			}
			h.logger.Info("drop created", "url_slug", slug)
			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie("session_token", raw, int(24*time.Hour/time.Second), "/", "", true, true)
			c.JSON(http.StatusCreated, gin.H{
				"url_slug":      slug,
				"session_token": raw,
				"expires_at":    expiresAt,
			})
			return
		}
		if pqErr, ok := insertErr.(*pq.Error); ok && pqErr.Code == "23505" {
			h.logger.Warn("slug collision, retrying", "attempt", attempt+1)
			continue
		}
		break
	}

	h.logger.Error("failed to insert drop", "err", insertErr)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "internal_error",
		"message": "Failed to create drop",
	})
}

func (h *Handler) GetDrop(c *gin.Context) {
	drop := c.MustGet("drop").(*dbpkg.Drop)

	count, err := h.queries.CountEventsForDrop(c.Request.Context(), drop.ID)
	if err != nil {
		h.logger.Error("failed to count events", "drop_id", drop.ID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Internal error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url_slug":    drop.URLSlug,
		"created_at":  drop.CreatedAt,
		"expires_at":  drop.ExpiresAt,
		"event_count": count,
	})
}

func (h *Handler) DeleteDrop(c *gin.Context) {
	drop := c.MustGet("drop").(*dbpkg.Drop)

	if err := h.queries.DeleteDrop(c.Request.Context(), drop.ID); err != nil {
		h.logger.Error("failed to delete drop", "drop_id", drop.ID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Internal error",
		})
		return
	}

	// Remove Redis cache entry so ingestion immediately stops accepting webhooks.
	key := fmt.Sprintf("drop:%s", drop.URLSlug)
	if err := h.redis.Del(c.Request.Context(), key).Err(); err != nil {
		h.logger.Error("failed to remove drop slug from redis cache", "slug", drop.URLSlug, "err", err)
	}

	h.logger.Info("drop deleted", "url_slug", drop.URLSlug)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func generateSlug() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	for i, v := range b {
		b[i] = chars[v%byte(len(chars))]
	}
	return string(b)
}

func generateSessionToken() (raw, hashed string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw = hex.EncodeToString(b)
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	hashed = string(hash)
	return
}
