package handlers

import (
	"net/http"

	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type NetWorthHandler struct {
	service *services.NetWorthService
}

func NewNetWorthHandler(service *services.NetWorthService) *NetWorthHandler {
	return &NetWorthHandler{service: service}
}

func (h *NetWorthHandler) GetCurrent(c *gin.Context) {
	nw, err := h.service.GetCurrent(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nw)
}

func (h *NetWorthHandler) TakeSnapshot(c *gin.Context) {
	snapshot, err := h.service.TakeSnapshot(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

func (h *NetWorthHandler) GetHistory(c *gin.Context) {
	history, err := h.service.GetHistory(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, history)
}
