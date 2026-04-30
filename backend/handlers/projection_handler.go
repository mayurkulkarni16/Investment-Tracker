package handlers

import (
	"net/http"
	"strconv"

	"investment-tracker/middleware"
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

	scenario := c.DefaultQuery("scenario", "base")
	if scenario != "bull" && scenario != "bear" && scenario != "base" {
		scenario = "base"
	}

	projections, err := h.service.GetProjectionsWithScenario(c.Request.Context(), middleware.GetEffectiveUserID(c), years, scenario)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projections)
}
