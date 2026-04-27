package handlers

import (
	"net/http"

	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type BenchmarkHandler struct {
	service *services.BenchmarkService
}

func NewBenchmarkHandler(service *services.BenchmarkService) *BenchmarkHandler {
	return &BenchmarkHandler{service: service}
}

func (h *BenchmarkHandler) GetBenchmarks(c *gin.Context) {
	period := c.DefaultQuery("period", "1y")
	data, err := h.service.GetBenchmarks(c.Request.Context(), period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}
