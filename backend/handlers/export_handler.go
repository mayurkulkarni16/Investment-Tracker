package handlers

import (
	"net/http"

	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	service *services.ExportService
}

func NewExportHandler(service *services.ExportService) *ExportHandler {
	return &ExportHandler{service: service}
}

func (h *ExportHandler) ExportCSV(c *gin.Context) {
	module := c.DefaultQuery("module", "all")
	data, filename, err := h.service.ExportCSV(c.Request.Context(), module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv", data)
}
