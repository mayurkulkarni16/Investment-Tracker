package handlers

import (
	"net/http"

	"investment-tracker/middleware"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type TaxHandler struct {
	service *services.TaxService
}

func NewTaxHandler(service *services.TaxService) *TaxHandler {
	return &TaxHandler{service: service}
}

func (h *TaxHandler) GetTaxSummary(c *gin.Context) {
	fy := c.DefaultQuery("fy", "")
	summary, err := h.service.GetTaxSummary(c.Request.Context(), middleware.GetEffectiveUserID(c), fy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *TaxHandler) GetCapitalGains(c *gin.Context) {
	fy := c.DefaultQuery("fy", "")
	cg, err := h.service.GetCapitalGains(c.Request.Context(), middleware.GetEffectiveUserID(c), fy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cg)
}







