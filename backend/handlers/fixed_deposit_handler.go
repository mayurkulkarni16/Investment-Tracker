package handlers

import (
	"net/http"

	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type FixedDepositHandler struct {
	service *services.FixedDepositService
}

func NewFixedDepositHandler(service *services.FixedDepositService) *FixedDepositHandler {
	return &FixedDepositHandler{service: service}
}

func (h *FixedDepositHandler) Create(c *gin.Context) {
	var req models.CreateFixedDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fd, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, fd)
}

func (h *FixedDepositHandler) GetAll(c *gin.Context) {
	fds, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if fds == nil {
		fds = []models.FixedDeposit{}
	}
	c.JSON(http.StatusOK, fds)
}

func (h *FixedDepositHandler) GetByID(c *gin.Context) {
	fd, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fixed deposit not found"})
		return
	}
	c.JSON(http.StatusOK, fd)
}

func (h *FixedDepositHandler) Update(c *gin.Context) {
	var req models.CreateFixedDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fd, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fd)
}

func (h *FixedDepositHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}
