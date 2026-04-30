package handlers

import (
	"net/http"

	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

type GoalHandler struct {
	service *services.GoalService
}

func NewGoalHandler(service *services.GoalService) *GoalHandler {
	return &GoalHandler{service: service}
}

func (h *GoalHandler) Create(c *gin.Context) {
	var req models.CreateGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	goal, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, goal)
}

func (h *GoalHandler) GetAll(c *gin.Context) {
	goals, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if goals == nil {
		goals = []models.Goal{}
	}
	c.JSON(http.StatusOK, goals)
}

func (h *GoalHandler) GetByID(c *gin.Context) {
	goal, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Goal not found"})
		return
	}
	c.JSON(http.StatusOK, goal)
}

func (h *GoalHandler) Update(c *gin.Context) {
	var req models.UpdateGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	goal, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, goal)
}

func (h *GoalHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Goal deleted"})
}

func (h *GoalHandler) LinkInvestment(c *gin.Context) {
	var req models.LinkInvestmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	goal, err := h.service.LinkInvestment(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, goal)
}

func (h *GoalHandler) BatchLinkInvestments(c *gin.Context) {
	var req models.BatchLinkInvestmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	goal, err := h.service.BatchLinkInvestments(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, goal)
}

func (h *GoalHandler) UnlinkInvestment(c *gin.Context) {
	investmentID := c.Param("investmentId")
	goal, err := h.service.UnlinkInvestment(c.Request.Context(), c.Param("id"), investmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, goal)
}
