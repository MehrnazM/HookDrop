package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	dbpkg "github.com/mehrnazm/hookdrop/go/data-api/db"
)

func (h *Handler) ListEvents(c *gin.Context) {
	drop := c.MustGet("drop").(*dbpkg.Drop)

	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	events, total, err := h.queries.ListEventsForDrop(c.Request.Context(), drop.ID, page, limit)
	if err != nil {
		h.logger.Error("failed to list events", "drop_id", drop.ID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Internal error",
		})
		return
	}

	if events == nil {
		events = []dbpkg.EventPreview{}
	}

	c.JSON(http.StatusOK, gin.H{
		"events":      events,
		"total_count": total,
		"page":        page,
		"limit":       limit,
	})
}

func (h *Handler) GetEvent(c *gin.Context) {
	drop := c.MustGet("drop").(*dbpkg.Drop)
	eventID := c.Param("event_id")

	event, err := h.queries.GetEventDetail(c.Request.Context(), drop.ID, eventID)
	if err != nil {
		h.logger.Error("failed to get event", "drop_id", drop.ID, "event_id", eventID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Internal error",
		})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Event not found",
		})
		return
	}

	c.JSON(http.StatusOK, event)
}

func queryInt(c *gin.Context, key string, def int) int {
	v, err := strconv.Atoi(c.DefaultQuery(key, strconv.Itoa(def)))
	if err != nil {
		return def
	}
	return v
}
