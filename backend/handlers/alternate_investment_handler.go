package handlers

import (
	"net/http"

	"investment-tracker/middleware"
	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type AlternateInvestmentHandler struct {
	service *services.AlternateInvestmentService
}

func NewAlternateInvestmentHandler(service *services.AlternateInvestmentService) *AlternateInvestmentHandler {
	return &AlternateInvestmentHandler{service: service}
}

func (h *AlternateInvestmentHandler) GetAll(c *gin.Context) {
	items, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	if items == nil {
		items = []models.AlternateInvestment{}
	}
	c.JSON(http.StatusOK, items)
}

func (h *AlternateInvestmentHandler) GetByID(c *gin.Context) {
	v, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondNotFound(c, err, "Alternate investment not found")
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *AlternateInvestmentHandler) Create(c *gin.Context) {
	var req models.CreateAlternateInvestmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *AlternateInvestmentHandler) Update(c *gin.Context) {
	var req models.CreateAlternateInvestmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	respondAlternateInvestment(c, v, err)
}

func (h *AlternateInvestmentHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		respondAlternateInvestment(c, nil, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func (h *AlternateInvestmentHandler) UpdatePayout(c *gin.Context) {
	var req models.UpdateAltPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.service.UpdatePayout(c.Request.Context(), c.Param("id"), c.Param("payoutId"), req)
	respondAlternateInvestment(c, v, err)
}

func (h *AlternateInvestmentHandler) MarkDueReceived(c *gin.Context) {
	v, err := h.service.MarkDueReceived(c.Request.Context(), c.Param("id"))
	respondAlternateInvestment(c, v, err)
}

func (h *AlternateInvestmentHandler) UpdateBuyback(c *gin.Context) {
	var req models.UpdateAltPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.service.UpdateBuyback(c.Request.Context(), c.Param("id"), req)
	respondAlternateInvestment(c, v, err)
}

func respondAlternateInvestment(c *gin.Context, v *models.AlternateInvestment, err error) {
	if err != nil {
		respondNotFound(c, err, "Alternate investment not found")
		return
	}
	c.JSON(http.StatusOK, v)
}
