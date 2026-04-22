package handlers

import (
	"net/http"

	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type MutualFundHandler struct {
	service *services.MutualFundService
}

func NewMutualFundHandler(service *services.MutualFundService) *MutualFundHandler {
	return &MutualFundHandler{service: service}
}

func (h *MutualFundHandler) Create(c *gin.Context) {
	var req models.CreateMutualFundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mf, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, mf)
}

func (h *MutualFundHandler) GetAll(c *gin.Context) {
	funds, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if funds == nil {
		funds = []models.MutualFund{}
	}
	c.JSON(http.StatusOK, funds)
}

func (h *MutualFundHandler) GetByID(c *gin.Context) {
	mf, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mutual fund not found"})
		return
	}
	c.JSON(http.StatusOK, mf)
}

func (h *MutualFundHandler) AddTransaction(c *gin.Context) {
	var req models.AddMFTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mf, err := h.service.AddTransaction(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mf)
}

func (h *MutualFundHandler) RefreshNAV(c *gin.Context) {
	id := c.Param("id")
	if id != "" {
		mf, err := h.service.RefreshNAV(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, mf)
		return
	}

	funds, err := h.service.RefreshAllNAVs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, funds)
}

func (h *MutualFundHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func (h *MutualFundHandler) Update(c *gin.Context) {
	var req models.UpdateMutualFundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mf, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mf)
}

func (h *MutualFundHandler) ImportFromCAS(c *gin.Context) {
	var req models.ImportCASRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.ImportFromCAS(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *MutualFundHandler) RecalculateAll(c *gin.Context) {
	count, err := h.service.RecalculateAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Recalculated all funds", "funds_updated": count})
}
