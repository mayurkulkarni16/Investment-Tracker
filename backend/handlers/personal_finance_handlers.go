package handlers

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"investment-tracker/middleware"
	"investment-tracker/models"
	"investment-tracker/services"

	"github.com/gin-gonic/gin"
)

func bindJSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func respond(c *gin.Context, v interface{}, err error, notFoundMsg string) {
	if err != nil {
		respondNotFound(c, err, notFoundMsg)
		return
	}
	c.JSON(http.StatusOK, v)
}

func respondDeleted(c *gin.Context, err error, notFoundMsg string) {
	respond(c, gin.H{"message": "Deleted"}, err, notFoundMsg)
}

// ---------------------------------------------------------------------------
// Bank accounts, budgets and the ledger summary

type AccountHandler struct {
	accounts *services.AccountService
	budgets  *services.BudgetService
}

func NewAccountHandler(accounts *services.AccountService, budgets *services.BudgetService) *AccountHandler {
	return &AccountHandler{accounts: accounts, budgets: budgets}
}

func (h *AccountHandler) GetAll(c *gin.Context) {
	v, err := h.accounts.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	respond(c, v, err, "Account not found")
}

func (h *AccountHandler) GetByID(c *gin.Context) {
	v, err := h.accounts.GetByID(c.Request.Context(), c.Param("id"))
	respond(c, v, err, "Account not found")
}

func (h *AccountHandler) Create(c *gin.Context) {
	var req models.BankAccountRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.accounts.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *AccountHandler) Update(c *gin.Context) {
	var req models.BankAccountRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.accounts.Update(c.Request.Context(), c.Param("id"), req)
	respond(c, v, err, "Account not found")
}

func (h *AccountHandler) UpdateBalance(c *gin.Context) {
	var req models.UpdateBalanceRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.accounts.UpdateBalance(c.Request.Context(), c.Param("id"), req)
	respond(c, v, err, "Account not found")
}

func (h *AccountHandler) Delete(c *gin.Context) {
	respondDeleted(c, h.accounts.Delete(c.Request.Context(), c.Param("id")), "Account not found")
}

func (h *AccountHandler) GetBudgets(c *gin.Context) {
	v, err := h.budgets.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if v == nil {
		v = []models.Budget{}
	}
	respond(c, v, err, "Budget not found")
}

func (h *AccountHandler) SaveBudget(c *gin.Context) {
	var req models.BudgetRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.budgets.Upsert(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	respond(c, v, err, "Budget not found")
}

func (h *AccountHandler) DeleteBudget(c *gin.Context) {
	respondDeleted(c, h.budgets.Delete(c.Request.Context(), c.Param("id")), "Budget not found")
}

func (h *AccountHandler) LedgerSummary(c *gin.Context) {
	months, _ := strconv.Atoi(c.DefaultQuery("months", "12"))
	v, err := h.budgets.Summary(c.Request.Context(), middleware.GetEffectiveUserID(c), months)
	respond(c, v, err, "Not found")
}

func (h *AccountHandler) DeleteLedgerEntry(c *gin.Context) {
	respondDeleted(c, h.budgets.DeleteLedgerEntry(c.Request.Context(), middleware.GetEffectiveUserID(c), c.Param("id")), "Entry not found")
}

// ---------------------------------------------------------------------------
// Insurance

type InsuranceHandler struct{ service *services.InsuranceService }

func NewInsuranceHandler(s *services.InsuranceService) *InsuranceHandler {
	return &InsuranceHandler{service: s}
}

func (h *InsuranceHandler) GetAll(c *gin.Context) {
	v, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	respond(c, v, err, "Policy not found")
}

func (h *InsuranceHandler) GetByID(c *gin.Context) {
	v, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	respond(c, v, err, "Policy not found")
}

func (h *InsuranceHandler) Create(c *gin.Context) {
	var req models.InsurancePolicyRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *InsuranceHandler) Update(c *gin.Context) {
	var req models.InsurancePolicyRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	respond(c, v, err, "Policy not found")
}

func (h *InsuranceHandler) Delete(c *gin.Context) {
	respondDeleted(c, h.service.Delete(c.Request.Context(), c.Param("id")), "Policy not found")
}

func (h *InsuranceHandler) Summary(c *gin.Context) {
	income, _ := strconv.ParseFloat(c.Query("annual_income"), 64)
	if income < 0 {
		income = 0
	}
	v, err := h.service.Summary(c.Request.Context(), middleware.GetEffectiveUserID(c), income)
	respond(c, v, err, "Not found")
}

// ---------------------------------------------------------------------------
// Gold and real estate

type GoldHandler struct{ service *services.GoldService }

func NewGoldHandler(s *services.GoldService) *GoldHandler { return &GoldHandler{service: s} }

func (h *GoldHandler) GetAll(c *gin.Context) {
	v, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	respond(c, v, err, "Holding not found")
}

func (h *GoldHandler) GetByID(c *gin.Context) {
	v, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	respond(c, v, err, "Holding not found")
}

func (h *GoldHandler) Create(c *gin.Context) {
	var req models.GoldHoldingRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *GoldHandler) Update(c *gin.Context) {
	var req models.GoldHoldingRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	respond(c, v, err, "Holding not found")
}

func (h *GoldHandler) Delete(c *gin.Context) {
	respondDeleted(c, h.service.Delete(c.Request.Context(), c.Param("id")), "Holding not found")
}

func (h *GoldHandler) SetRate(c *gin.Context) {
	var req models.GoldRateRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.service.SetRate(c.Request.Context(), middleware.GetEffectiveUserID(c), req.Rate24KPerGram)
	respond(c, v, err, "Not found")
}

func (h *GoldHandler) RefreshPrices(c *gin.Context) {
	v, err := h.service.RefreshPrices(c.Request.Context(), middleware.GetEffectiveUserID(c))
	respond(c, v, err, "Not found")
}

type PropertyHandler struct{ service *services.PropertyService }

func NewPropertyHandler(s *services.PropertyService) *PropertyHandler {
	return &PropertyHandler{service: s}
}

func (h *PropertyHandler) GetAll(c *gin.Context) {
	v, err := h.service.GetAll(c.Request.Context(), middleware.GetEffectiveUserID(c))
	respond(c, v, err, "Property not found")
}

func (h *PropertyHandler) GetByID(c *gin.Context) {
	v, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	respond(c, v, err, "Property not found")
}

func (h *PropertyHandler) Create(c *gin.Context) {
	var req models.PropertyRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.service.Create(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *PropertyHandler) Update(c *gin.Context) {
	var req models.PropertyRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	respond(c, v, err, "Property not found")
}

func (h *PropertyHandler) Delete(c *gin.Context) {
	respondDeleted(c, h.service.Delete(c.Request.Context(), c.Param("id")), "Property not found")
}

// ---------------------------------------------------------------------------
// Family, FIRE planner and the finance overview

type PlanningHandler struct {
	family  *services.FamilyService
	fire    *services.FIREService
	finance *services.FinanceService
}

func NewPlanningHandler(family *services.FamilyService, fire *services.FIREService, finance *services.FinanceService) *PlanningHandler {
	return &PlanningHandler{family: family, fire: fire, finance: finance}
}

func (h *PlanningHandler) FamilySummary(c *gin.Context) {
	v, err := h.family.Summary(c.Request.Context(), middleware.GetEffectiveUserID(c))
	respond(c, v, err, "Not found")
}

func (h *PlanningHandler) Assign(c *gin.Context) {
	var req models.AssignProfileRequest
	if !bindJSON(c, &req) {
		return
	}
	err := h.family.Assign(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	respond(c, gin.H{"message": "Saved"}, err, "Profile not found")
}

func (h *PlanningHandler) FIREDefaults(c *gin.Context) {
	v, err := h.fire.Defaults(c.Request.Context(), middleware.GetEffectiveUserID(c))
	respond(c, v, err, "Not found")
}

func (h *PlanningHandler) FIREPlan(c *gin.Context) {
	var req models.FIREPlanRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.fire.Plan(req)
	respond(c, v, err, "Not found")
}

func (h *PlanningHandler) Overview(c *gin.Context) {
	v, err := h.finance.Overview(c.Request.Context(), middleware.GetEffectiveUserID(c))
	respond(c, v, err, "Not found")
}

// ---------------------------------------------------------------------------
// Email / Telegram reminder settings

type MessagingHandler struct{ service *services.MessagingService }

func NewMessagingHandler(s *services.MessagingService) *MessagingHandler {
	return &MessagingHandler{service: s}
}

func (h *MessagingHandler) GetSettings(c *gin.Context) {
	v, err := h.service.GetSettings(c.Request.Context(), middleware.GetEffectiveUserID(c))
	respond(c, v, err, "Not found")
}

func (h *MessagingHandler) UpdateSettings(c *gin.Context) {
	var req models.NotificationSettingsRequest
	if !bindJSON(c, &req) {
		return
	}
	v, err := h.service.UpdateSettings(c.Request.Context(), middleware.GetEffectiveUserID(c), req)
	respond(c, v, err, "Not found")
}

func (h *MessagingHandler) Test(c *gin.Context) {
	results, err := h.service.SendTest(c.Request.Context(), middleware.GetEffectiveUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	sent := []string{}
	var failures []string
	for channel, outcome := range results {
		if outcome == "sent" {
			sent = append(sent, channel)
		} else {
			failures = append(failures, channel+": "+outcome)
		}
	}
	if len(sent) == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": strings.Join(failures, "; ")})
		return
	}
	msg := "Test message sent via " + strings.Join(sent, " and ")
	if len(failures) > 0 {
		msg += " (failed: " + strings.Join(failures, "; ") + ")"
	}
	c.JSON(http.StatusOK, gin.H{"message": msg, "sent": sent})
}

// ---------------------------------------------------------------------------
// Document vault

type DocumentHandler struct{ service *services.DocumentService }

func NewDocumentHandler(s *services.DocumentService) *DocumentHandler {
	return &DocumentHandler{service: s}
}

func (h *DocumentHandler) List(c *gin.Context) {
	v, err := h.service.List(c.Request.Context(), middleware.GetEffectiveUserID(c), c.Query("asset_type"), c.Query("asset_id"))
	respond(c, v, err, "Not found")
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxDocumentBytes+1<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required (max 10 MB)"})
		return
	}
	f, err := fh.Open()
	if err != nil {
		respondError(c, err)
		return
	}
	defer f.Close()
	doc, err := h.service.Upload(c.Request.Context(), middleware.GetEffectiveUserID(c), services.DocumentUpload{
		Title: c.PostForm("title"), FileName: fh.Filename, Category: c.PostForm("category"),
		AssetType: c.PostForm("asset_type"), AssetID: c.PostForm("asset_id"),
		ExpiryDate: c.PostForm("expiry_date"), Notes: c.PostForm("notes"), Size: fh.Size, Reader: f,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, doc)
}

func (h *DocumentHandler) Download(c *gin.Context) {
	doc, stream, err := h.service.Open(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondNotFound(c, err, "Document not found")
		return
	}
	defer stream.Close()
	name := filepath.Base(doc.FileName)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Length", strconv.FormatInt(doc.Size, 10))
	c.Status(http.StatusOK)
	c.Header("Content-Type", doc.ContentType)
	_, _ = io.Copy(c.Writer, stream)
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	respondDeleted(c, h.service.Delete(c.Request.Context(), c.Param("id")), "Document not found")
}

// ---------------------------------------------------------------------------
// Broker CSV import

type BrokerImportHandler struct{ service *services.BrokerImportService }

func NewBrokerImportHandler(s *services.BrokerImportService) *BrokerImportHandler {
	return &BrokerImportHandler{service: s}
}

func (h *BrokerImportHandler) Import(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 6<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is required (max 5 MB)"})
		return
	}
	if ext := strings.ToLower(filepath.Ext(fh.Filename)); ext != ".csv" && ext != ".txt" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upload the broker export as a .csv file"})
		return
	}
	f, err := fh.Open()
	if err != nil {
		respondError(c, err)
		return
	}
	defer f.Close()
	dryRun := c.PostForm("dry_run") != "false"
	res, err := h.service.Import(c.Request.Context(), middleware.GetEffectiveUserID(c), f, c.PostForm("broker"), dryRun, c.PostForm("as_of"))
	respond(c, res, err, "Not found")
}

// Refresh issues a new token for the signed-in user (mobile apps call this before expiry).
func (h *AuthHandler) Refresh(c *gin.Context) {
	uid, _ := c.Get("user_id")
	userID, _ := uid.(string)
	res, err := h.service.Refresh(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
