package handlers

import (
	"net/http"
	"strconv"

	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type ProjectionHandler struct {
	service *services.ProjectionService
}

func NewProjectionHandler(service *services.ProjectionService) *ProjectionHandler {
	return &ProjectionHandler{service: service}
}

func (h *ProjectionHandler) GetProjections(c *gin.Context) {
	years := 5
	if yStr := c.Query("years"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y > 0 && y <= 30 {
			years = y
		}
	}

	projections, err := h.service.GetProjections(c.Request.Context(), years)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projections)
}
