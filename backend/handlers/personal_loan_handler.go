package handlers

import (
	"net/http"

	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type PersonalLoanHandler struct {
	service *services.PersonalLoanService
}

func NewPersonalLoanHandler(service *services.PersonalLoanService) *PersonalLoanHandler {
	return &PersonalLoanHandler{service: service}
}

func (h *PersonalLoanHandler) Create(c *gin.Context) {
	var req models.CreatePersonalLoanRequest
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

func (h *PersonalLoanHandler) GetAll(c *gin.Context) {
	loans, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if loans == nil {
		loans = []models.PersonalLoan{}
	}
	c.JSON(http.StatusOK, loans)
}

func (h *PersonalLoanHandler) GetByID(c *gin.Context) {
	loan, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Personal loan not found"})
		return
	}
	c.JSON(http.StatusOK, loan)
}

func (h *PersonalLoanHandler) Update(c *gin.Context) {
	var req models.UpdatePersonalLoanRequest
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

func (h *PersonalLoanHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func (h *PersonalLoanHandler) RecordEMI(c *gin.Context) {
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

func (h *PersonalLoanHandler) AddPrepayment(c *gin.Context) {
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

func (h *PersonalLoanHandler) ChangeRate(c *gin.Context) {
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

func (h *PersonalLoanHandler) GetAmortization(c *gin.Context) {
	schedule, err := h.service.GetAmortizationSchedule(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schedule)
}
