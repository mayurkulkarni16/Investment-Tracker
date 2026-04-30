package handlers

import (
	"net/http"

	"investment-tracker/models"
	"investment-tracker/middleware"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type ProvidentFundHandler struct {
	service *services.ProvidentFundService
}

func NewProvidentFundHandler(service *services.ProvidentFundService) *ProvidentFundHandler {
	return &ProvidentFundHandler{service: service}
}

func (h *ProvidentFundHandler) Create(c *gin.Context) {
	var req models.CreateProvidentFundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pf, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, pf)
}

func (h *ProvidentFundHandler) GetAll(c *gin.Context) {
	pfs, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if pfs == nil {
		pfs = []models.ProvidentFund{}
	}
	c.JSON(http.StatusOK, pfs)
}

func (h *ProvidentFundHandler) GetByID(c *gin.Context) {
	pf, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provident fund not found"})
		return
	}
	c.JSON(http.StatusOK, pf)
}

func (h *ProvidentFundHandler) AddContribution(c *gin.Context) {
	var req models.AddMonthlyContributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pf, err := h.service.AddContribution(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pf)
}

func (h *ProvidentFundHandler) Update(c *gin.Context) {
	var req models.CreateProvidentFundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pf, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pf)
}

func (h *ProvidentFundHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func (h *ProvidentFundHandler) ImportFromPDF(c *gin.Context) {
	var req models.ImportPFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.ImportFromPDF(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}




