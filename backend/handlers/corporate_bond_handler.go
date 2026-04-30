package handlers

import (
	"net/http"
	"time"

	"investment-tracker/models"
	"investment-tracker/middleware"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type CorporateBondHandler struct {
	service *services.CorporateBondService
}

func NewCorporateBondHandler(service *services.CorporateBondService) *CorporateBondHandler {
	return &CorporateBondHandler{service: service}
}

func (h *CorporateBondHandler) Create(c *gin.Context) {
	var req models.CreateCorporateBondRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bond, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bond)
}

func (h *CorporateBondHandler) GetAll(c *gin.Context) {
	bonds, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if bonds == nil {
		bonds = []models.CorporateBond{}
	}
	c.JSON(http.StatusOK, bonds)
}

func (h *CorporateBondHandler) GetByID(c *gin.Context) {
	bond, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bond not found"})
		return
	}
	c.JSON(http.StatusOK, bond)
}

type MarkPayoutRequest struct {
	ReceivedDate time.Time `json:"received_date" binding:"required"`
}

func (h *CorporateBondHandler) MarkPayoutReceived(c *gin.Context) {
	var req MarkPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bond, err := h.service.MarkPayoutReceived(
		c.Request.Context(),
		c.Param("id"),
		c.Param("payoutId"),
		req.ReceivedDate,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bond)
}

func (h *CorporateBondHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func (h *CorporateBondHandler) Update(c *gin.Context) {
	var req models.CreateCorporateBondRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bond, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bond)
}




