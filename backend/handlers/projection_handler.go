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

	inflation := 0.0
	if iStr := c.Query("inflation"); iStr != "" {
		if v, err := strconv.ParseFloat(iStr, 64); err == nil && v >= 0 && v <= 20 {
			inflation = v
		}
	}

	projections, err := h.service.GetProjections(c.Request.Context(), middleware.GetEffectiveUserID(c), services.ProjectionOptions{
		Years: years, Scenario: scenario, InflationRate: inflation,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, projections)
}
