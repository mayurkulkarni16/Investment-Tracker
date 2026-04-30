package handlers

import (
	"net/http"
	"strconv"

	"investment-tracker/middleware"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type CashflowHandler struct {
	service *services.CashflowService
}

func NewCashflowHandler(service *services.CashflowService) *CashflowHandler {
	return &CashflowHandler{service: service}
}

func (h *CashflowHandler) GetMonthlyCashflows(c *gin.Context) {
	months := 12
	if m := c.Query("months"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed > 0 {
			months = parsed
		}
	}

	cashflows, err := h.service.GetMonthlyCashflows(c.Request.Context(), middleware.GetEffectiveUserID(c), months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cashflows)
}

func (h *CashflowHandler) GetMonthDetail(c *gin.Context) {
	month := c.Param("month")
	details, err := h.service.GetMonthDetail(c.Request.Context(), middleware.GetEffectiveUserID(c), month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, details)
}







