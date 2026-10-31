package handlers

import (
	"net/http"

	"investment-tracker/middleware"
	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type VehicleRentalHandler struct {
	service *services.VehicleRentalService
}

func NewVehicleRentalHandler(service *services.VehicleRentalService) *VehicleRentalHandler {
	return &VehicleRentalHandler{service: service}
}

func (h *VehicleRentalHandler) GetAll(c *gin.Context) {
	items, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	if items == nil {
		items = []models.VehicleRental{}
	}
	c.JSON(http.StatusOK, items)
}

func (h *VehicleRentalHandler) GetByID(c *gin.Context) {
	v, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondNotFound(c, err, "Vehicle rental investment not found")
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *VehicleRentalHandler) Create(c *gin.Context) {
	var req models.CreateVehicleRentalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *VehicleRentalHandler) Update(c *gin.Context) {
	var req models.CreateVehicleRentalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	respondVehicleRental(c, v, err)
}

func (h *VehicleRentalHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		respondVehicleRental(c, nil, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func (h *VehicleRentalHandler) UpdatePayout(c *gin.Context) {
	var req models.UpdateRentalPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.service.UpdatePayout(c.Request.Context(), c.Param("id"), c.Param("payoutId"), req)
	respondVehicleRental(c, v, err)
}

func (h *VehicleRentalHandler) MarkDueReceived(c *gin.Context) {
	v, err := h.service.MarkDueReceived(c.Request.Context(), c.Param("id"))
	respondVehicleRental(c, v, err)
}

func (h *VehicleRentalHandler) UpdateBuyback(c *gin.Context) {
	var req models.UpdateRentalPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.service.UpdateBuyback(c.Request.Context(), c.Param("id"), req)
	respondVehicleRental(c, v, err)
}

func respondVehicleRental(c *gin.Context, v *models.VehicleRental, err error) {
	if err != nil {
		respondNotFound(c, err, "Vehicle rental investment not found")
		return
	}
	c.JSON(http.StatusOK, v)
}
