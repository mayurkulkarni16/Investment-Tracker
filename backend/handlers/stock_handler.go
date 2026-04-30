package handlers

import (
	"net/http"

	"investment-tracker/models"
	"investment-tracker/middleware"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type StockHandler struct {
	service *services.StockService
}

func NewStockHandler(service *services.StockService) *StockHandler {
	return &StockHandler{service: service}
}

func (h *StockHandler) Create(c *gin.Context) {
	var req models.CreateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stock, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, stock)
}

func (h *StockHandler) GetAll(c *gin.Context) {
	stocks, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if stocks == nil {
		stocks = []models.Stock{}
	}
	c.JSON(http.StatusOK, stocks)
}

func (h *StockHandler) GetByID(c *gin.Context) {
	stock, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock not found"})
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *StockHandler) AddTransaction(c *gin.Context) {
	var req models.AddStockTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stock, err := h.service.AddTransaction(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *StockHandler) DeleteTransaction(c *gin.Context) {
	stock, err := h.service.DeleteTransaction(c.Request.Context(), c.Param("id"), c.Param("txnId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *StockHandler) UpdateTransaction(c *gin.Context) {
	var req models.AddStockTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	stock, err := h.service.UpdateTransaction(c.Request.Context(), c.Param("id"), c.Param("txnId"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *StockHandler) RefreshPrice(c *gin.Context) {
	id := c.Param("id")
	if id != "" {
		stock, err := h.service.RefreshPrice(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, stock)
		return
	}

	stocks, err := h.service.RefreshAllPrices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stocks)
}

func (h *StockHandler) Update(c *gin.Context) {
	var req models.UpdateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stock, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *StockHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func (h *StockHandler) MarketStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"is_market_open": services.IsMarketOpen()})
}

func (h *StockHandler) AddDividend(c *gin.Context) {
	var req models.AddStockDividendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	stock, err := h.service.AddDividend(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *StockHandler) DeleteDividend(c *gin.Context) {
	stock, err := h.service.DeleteDividend(c.Request.Context(), c.Param("id"), c.Param("divId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stock)
}




