package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	dbpkg "github.com/mehrnazm/webhookx/go/data-api/db"
	"golang.org/x/crypto/bcrypt"
)

// Authenticate validates the session token and attaches the drop to the Gin context.
// Reads token from Authorization: Bearer header or session_token cookie.
func Authenticate(dbQueries *dbpkg.Queries, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		dropSlug := c.Param("drop_slug")
		drop, err := dbQueries.GetDropBySlug(c.Request.Context(), dropSlug)
		if err != nil {
			logger.Error("failed to get drop from db", "err", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Internal error",
			})
			return
		}
		if drop == nil {
			logger.Error("drop was not found", "slug", dropSlug)
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Drop not found",
			})
			return
		}

		token := extractToken(c)
		if token == "" {
			logger.Error("empty token for authed endpoint", "path", c.Request.URL.Path, "method", c.Request.Method)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "empty_token",
				"message": "Authentication required",
			})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(drop.SessionToken), []byte(token)); err != nil {
			logger.Error("comparing hash and password failed", "err", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_token",
				"message": "Invalid token",
			})
			return
		}

		if time.Now().After(drop.ExpiresAt) {
			logger.Error("expired token", "slug", dropSlug)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "expired_token",
				"message": "Drop has expired",
			})
			return
		}

		c.Set("drop", drop)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	if cookie, err := c.Cookie("session_token"); err == nil {
		return cookie
	}
	return ""
}
