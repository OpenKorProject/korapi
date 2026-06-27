package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler holds gateway HTTP handlers.
type Handler struct{}

// NewHandler creates a new Handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Healthz handles the liveness probe.
func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// Readyz handles the readiness probe.
// TODO: check Redis and downstream service connectivity.
func (h *Handler) Readyz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// NotFound handles unmatched routes.
func (h *Handler) NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"code":       "NOT_FOUND",
			"message":    "Endpoint not found",
			"request_id": c.GetString("request_id"),
		},
	})
}
