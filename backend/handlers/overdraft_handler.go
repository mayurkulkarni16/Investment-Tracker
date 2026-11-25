package handlers

import (
	"net/http"

	"investment-tracker/middleware"
	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type OverdraftHandler struct {
	service *services.OverdraftService
}

func NewOverdraftHandler(service *services.OverdraftService) *OverdraftHandler {
	return &OverdraftHandler{service: service}
}

func (h *OverdraftHandler) Create(c *gin.Context) {
	var req models.CreateOverdraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	od, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, od)
}

func (h *OverdraftHandler) GetAll(c *gin.Context) {
	items, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	if items == nil {
		items = []models.Overdraft{}
	}
	c.JSON(http.StatusOK, items)
}

func (h *OverdraftHandler) GetByID(c *gin.Context) {
	od, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondNotFound(c, err, "Overdraft not found")
		return
	}
	c.JSON(http.StatusOK, od)
}

func (h *OverdraftHandler) Update(c *gin.Context) {
	var req models.UpdateOverdraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	od, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, od)
}

func (h *OverdraftHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func (h *OverdraftHandler) AddTransaction(c *gin.Context) {
	var req models.AddOverdraftTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	od, err := h.service.AddTransaction(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, od)
}

func (h *OverdraftHandler) DeleteTransaction(c *gin.Context) {
	od, err := h.service.DeleteTransaction(c.Request.Context(), c.Param("id"), c.Param("txnId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, od)
}
