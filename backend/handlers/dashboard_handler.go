package handlers

import (
	"net/http"

	"investment-tracker/middleware"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	service *services.DashboardService
}

func NewDashboardHandler(service *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	userID := middleware.GetEffectiveUserID(c)
	dashboard, err := h.service.GetDashboard(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dashboard)
}

func (h *DashboardHandler) GetRebalanceSuggestions(c *gin.Context) {
	userID := middleware.GetEffectiveUserID(c)
	var req struct {
		Targets map[string]float64 `json:"targets"` // category -> target percentage
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	suggestions, err := h.service.GetRebalanceSuggestions(c.Request.Context(), userID, req.Targets)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, suggestions)
}
