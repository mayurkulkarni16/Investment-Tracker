package handlers

import (
	"net/http"

	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type HomeLoanHandler struct {
	service *services.HomeLoanService
}

func NewHomeLoanHandler(service *services.HomeLoanService) *HomeLoanHandler {
	return &HomeLoanHandler{service: service}
}

func (h *HomeLoanHandler) Create(c *gin.Context) {
	var req models.CreateHomeLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loan, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, loan)
}

func (h *HomeLoanHandler) GetAll(c *gin.Context) {
	loans, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if loans == nil {
		loans = []models.HomeLoan{}
	}
	c.JSON(http.StatusOK, loans)
}

func (h *HomeLoanHandler) GetByID(c *gin.Context) {
	loan, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Home loan not found"})
		return
	}
	c.JSON(http.StatusOK, loan)
}

func (h *HomeLoanHandler) Update(c *gin.Context) {
	var req models.UpdateHomeLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loan, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loan)
}

func (h *HomeLoanHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func (h *HomeLoanHandler) RecordEMI(c *gin.Context) {
	var req models.AddEMIPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loan, err := h.service.RecordEMIPayment(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loan)
}

func (h *HomeLoanHandler) AddPrepayment(c *gin.Context) {
	var req models.AddPrepaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loan, err := h.service.AddPrepayment(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loan)
}

func (h *HomeLoanHandler) ChangeRate(c *gin.Context) {
	var req models.ChangeRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loan, err := h.service.ChangeRate(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loan)
}

func (h *HomeLoanHandler) GetAmortization(c *gin.Context) {
	schedule, err := h.service.GetAmortizationSchedule(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schedule)
}

func (h *HomeLoanHandler) AddDisbursement(c *gin.Context) {
	var req models.AddDisbursementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	loan, err := h.service.AddDisbursement(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loan)
}

func (h *HomeLoanHandler) MarkConstructionComplete(c *gin.Context) {
	var req models.MarkConstructionCompleteRequest
	_ = c.ShouldBindJSON(&req) // optional body
	loan, err := h.service.MarkConstructionComplete(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loan)
}

func (h *HomeLoanHandler) Recalculate(c *gin.Context) {
	loan, err := h.service.Recalculate(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loan)
}
