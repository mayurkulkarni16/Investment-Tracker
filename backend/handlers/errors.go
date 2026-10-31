package handlers

import (
	"net/http"

	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

// respondError maps service errors to HTTP status codes: not found (including
// records owned by another user and malformed IDs) -> 404, invalid input -> 400,
// anything else -> 500.
func respondError(c *gin.Context, err error) {
	switch {
	case services.IsNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	case services.IsInvalidInput(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// respondNotFound is respondError with a resource-specific not-found message.
func respondNotFound(c *gin.Context, err error, msg string) {
	if services.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": msg})
		return
	}
	respondError(c, err)
}
