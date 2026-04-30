package handlers

import (
	"net/http"

	"investment-tracker/models"
	"investment-tracker/middleware"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type CreditCardHandler struct {
	service *services.CreditCardService
}

func NewCreditCardHandler(service *services.CreditCardService) *CreditCardHandler {
	return &CreditCardHandler{service: service}
}

func (h *CreditCardHandler) Create(c *gin.Context) {
	var req models.CreateCreditCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	card, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, card)
}

func (h *CreditCardHandler) GetAll(c *gin.Context) {
	cards, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cards == nil {
		cards = []models.CreditCard{}
	}
	c.JSON(http.StatusOK, cards)
}

func (h *CreditCardHandler) GetByID(c *gin.Context) {
	card, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Credit card not found"})
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *CreditCardHandler) Update(c *gin.Context) {
	var req models.UpdateCreditCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	card, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *CreditCardHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Credit card deleted"})
}

func (h *CreditCardHandler) AddStatement(c *gin.Context) {
	var req models.AddCardStatementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	card, err := h.service.AddStatement(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *CreditCardHandler) PayStatement(c *gin.Context) {
	var req models.PayStatementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	card, err := h.service.PayStatement(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *CreditCardHandler) AddTransaction(c *gin.Context) {
	var req models.AddCardTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	card, err := h.service.AddTransaction(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *CreditCardHandler) AddEMI(c *gin.Context) {
	var req models.AddCardEMIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	card, err := h.service.AddEMI(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *CreditCardHandler) AddCreditScore(c *gin.Context) {
	var req models.AddCreditScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	card, err := h.service.AddCreditScore(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, card)
}




