package handlers

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"

	"investment-tracker/middleware"
	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type LedgerHandler struct{ service *services.LedgerService }

func NewLedgerHandler(service *services.LedgerService) *LedgerHandler {
	return &LedgerHandler{service: service}
}

func (h *LedgerHandler) GetAll(c *gin.Context) {
	entries, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *LedgerHandler) Save(c *gin.Context) {
	var req models.LedgerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ledger request"})
		return
	}
	userID := middleware.GetEffectiveUserID(c)
	var entry *models.LedgerEntry
	var err error
	if c.Param("id") == "" {
		entry, err = h.service.Create(c.Request.Context(), userID, req)
	} else {
		entry, err = h.service.Update(c.Request.Context(), userID, c.Param("id"), req)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, entry)
}

func ledgerCSVCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if len(trimmed) > 0 && strings.ContainsAny(trimmed[:1], "=+-@") {
		return "'" + value
	}
	return value
}

func (h *LedgerHandler) Export(c *gin.Context) {
	entries, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"Date", "Status", "Kind", "Amount (paise INR)", "Description", "Category", "Bank", "Last four", "Due date"}); err != nil {
		respondError(c, err)
		return
	}
	for _, entry := range entries {
		row := []string{entry.Date, entry.Status, entry.Kind, strconv.FormatInt(entry.AmountPaise, 10), entry.Description, entry.Category, entry.Bank, entry.Account, entry.DueDate}
		for i := range row {
			row[i] = ledgerCSVCell(row[i])
		}
		if err := w.Write(row); err != nil {
			respondError(c, err)
			return
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		respondError(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=income-expenses.csv")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}
