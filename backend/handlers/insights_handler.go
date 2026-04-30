package handlers

import (
	"net/http"

	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type InsightsHandler struct {
	service *services.InsightsService
}

func NewInsightsHandler(service *services.InsightsService) *InsightsHandler {
	return &InsightsHandler{service: service}
}

func (h *InsightsHandler) GetInsights(c *gin.Context) {
	resp, err := h.service.GetInsights(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
