package handlers

import (
	"net/http"
	"time"

	"investment-tracker/middleware"
	"investment-tracker/models"
	"investment-tracker/services"
	"investment-tracker/taxengine"

	"github.com/gin-gonic/gin"
)

type TaxHandler struct {
	service *services.TaxService
	family  *services.FamilyTaxService
}

func NewTaxHandler(service *services.TaxService, family *services.FamilyTaxService) *TaxHandler {
	return &TaxHandler{service: service, family: family}
}

// GetTaxSummary returns household deductions, or one member's when profile_id is set.
func (h *TaxHandler) GetTaxSummary(c *gin.Context) {
	fy := c.DefaultQuery("fy", "")
	ctx, uid := c.Request.Context(), middleware.GetEffectiveUserID(c)
	var (
		summary *models.TaxSummary
		err     error
	)
	if pid := c.Query("profile_id"); pid != "" {
		summary, err = h.family.MemberSummary(ctx, uid, fy, pid)
	} else {
		summary, err = h.service.GetTaxSummary(ctx, uid, fy)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *TaxHandler) GetCapitalGains(c *gin.Context) {
	fy := c.DefaultQuery("fy", "")
	ctx, uid := c.Request.Context(), middleware.GetEffectiveUserID(c)
	var (
		cg  *models.CapitalGainsSummary
		err error
	)
	if pid := c.Query("profile_id"); pid != "" {
		cg, err = h.family.MemberCapitalGains(ctx, uid, fy, pid)
	} else {
		cg, err = h.service.GetCapitalGains(ctx, uid, fy)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, cg)
}

// Meta lists supported years and the option values used by the tax forms.
func (h *TaxHandler) Meta(c *gin.Context) {
	start := time.Date(time.Now().Year(), 4, 1, 0, 0, 0, 0, time.Local)
	if time.Now().Before(start) {
		start = start.AddDate(-1, 0, 0)
	}
	c.JSON(http.StatusOK, gin.H{
		"supported_fys": taxengine.SupportedFYs(),
		"current_fy":    taxengine.NormalizeFY(start.Format("2006")),
		"occupations":   models.Occupations,
		"instruments":   models.PlacementInstruments,
	})
}

func (h *TaxHandler) FamilyReport(c *gin.Context) {
	rep, err := h.family.Report(c.Request.Context(), middleware.GetEffectiveUserID(c), c.Query("fy"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, rep)
}

func (h *TaxHandler) GetSettings(c *gin.Context) {
	v, err := h.family.GetSettings(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *TaxHandler) SaveSettings(c *gin.Context) {
	var req models.TaxSettings
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.family.SaveSettings(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *TaxHandler) ListIncomes(c *gin.Context) {
	v, err := h.family.ListIncomes(c.Request.Context(), middleware.GetEffectiveUserID(c), c.Query("fy"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *TaxHandler) GetIncome(c *gin.Context) {
	v, err := h.family.GetIncome(c.Request.Context(), middleware.GetEffectiveUserID(c), c.Param("profileId"), c.Query("fy"))
	if err != nil {
		respondNotFound(c, err, "Family member not found")
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *TaxHandler) SaveIncome(c *gin.Context) {
	var req models.MemberIncomeRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.family.SaveIncome(c.Request.Context(), middleware.GetEffectiveUserID(c), c.Param("profileId"), c.Query("fy"), req)
	if err != nil {
		respondNotFound(c, err, "Family member not found")
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *TaxHandler) DeleteIncome(c *gin.Context) {
	err := h.family.DeleteIncome(c.Request.Context(), middleware.GetEffectiveUserID(c), c.Param("profileId"), c.Query("fy"))
	respondDeleted(c, err, "Income record not found")
}

// Placement ranks family members for holding a new investment (GET so it
// also works in view-only mode).
func (h *TaxHandler) Placement(c *gin.Context) {
	var req models.PlacementRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.family.Placement(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		respondNotFound(c, err, "Family member not found")
		return
	}
	c.JSON(http.StatusOK, v)
}
