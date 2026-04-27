package handlers

import (
	"net/http"

	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type SIPHandler struct {
	service *services.SIPService
}

func NewSIPHandler(service *services.SIPService) *SIPHandler {
	return &SIPHandler{service: service}
}

func (h *SIPHandler) Create(c *gin.Context) {
	var req models.CreateSIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sip, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sip)
}

func (h *SIPHandler) GetAll(c *gin.Context) {
	sips, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sips == nil {
		sips = []models.SIP{}
	}
	c.JSON(http.StatusOK, sips)
}

func (h *SIPHandler) GetByID(c *gin.Context) {
	sip, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SIP not found"})
		return
	}
	c.JSON(http.StatusOK, sip)
}

func (h *SIPHandler) Update(c *gin.Context) {
	var req models.UpdateSIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sip, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sip)
}

func (h *SIPHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SIP deleted"})
}

func (h *SIPHandler) RecordInstallment(c *gin.Context) {
	var req models.RecordSIPInstallmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sip, err := h.service.RecordInstallment(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sip)
}
