package routes

import (
	"investment-tracker/config"
	"investment-tracker/container"
	"investment-tracker/middleware"

	"github.com/gin-gonic/gin"
)

// Register sets up all API routes on the given router.
func Register(r *gin.Engine, c *container.Container, cfg *config.Config) {
	// Public auth routes
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/signup", c.AuthHandler.Signup)
		auth.POST("/login", c.AuthHandler.Login)
		auth.POST("/forgot-password", c.AuthHandler.ForgotPassword)
		auth.POST("/reset-password", c.AuthHandler.ResetPassword)
	}

	api := r.Group("/api/v1")
	api.Use(middleware.AuthRequired(cfg))
	api.Use(middleware.ViewOnlyCheck())
	{
		// Authenticated auth routes
		api.GET("/auth/me", c.AuthHandler.GetProfile)
		api.PUT("/auth/me", c.AuthHandler.UpdateProfile)
		api.PUT("/auth/change-password", c.AuthHandler.ChangePassword)
		api.GET("/auth/users", middleware.AdminOnly(), c.AuthHandler.GetAllUsers)

		// Dashboard & Projections
		api.GET("/dashboard", c.DashboardHandler.GetDashboard)
		api.POST("/dashboard/rebalance", c.DashboardHandler.GetRebalanceSuggestions)
		api.GET("/projections", c.ProjectionHandler.GetProjections)
		api.GET("/cashflow", c.CashflowHandler.GetMonthlyCashflows)
		api.GET("/cashflow/:month", c.CashflowHandler.GetMonthDetail)
		api.GET("/insights", c.InsightsHandler.GetInsights)

		// Mutual Funds
		registerMutualFundRoutes(api, c)

		// Corporate Bonds
		registerCorporateBondRoutes(api, c)

		// Fixed Deposits
		registerFixedDepositRoutes(api, c)

		// Provident Fund
		registerProvidentFundRoutes(api, c)

		// Stocks
		registerStockRoutes(api, c)

		// Home Loans
		registerHomeLoanRoutes(api, c)

		// Personal Loans
		registerPersonalLoanRoutes(api, c)

		// NPS
		registerNPSRoutes(api, c)

		// Credit Cards
		registerCreditCardRoutes(api, c)

		// Goals
		registerGoalRoutes(api, c)

		// Net Worth
		registerNetWorthRoutes(api, c)

		// SIPs
		registerSIPRoutes(api, c)

		// Notifications
		registerNotificationRoutes(api, c)

		// Profiles
		registerProfileRoutes(api, c)

		// Tax
		registerTaxRoutes(api, c)

		// Benchmarks
		api.GET("/benchmarks", c.BenchmarkHandler.GetBenchmarks)

		// Export
		api.GET("/export/csv", c.ExportHandler.ExportCSV)

		// Backup
		api.GET("/backup", c.BackupHandler.Export)
		api.POST("/backup/restore", c.BackupHandler.Restore)
	}
}

func registerMutualFundRoutes(api *gin.RouterGroup, c *container.Container) {
	mf := api.Group("/mutual-funds")
	{
		mf.GET("", c.MutualFundHandler.GetAll)
		mf.POST("", c.MutualFundHandler.Create)
		mf.POST("/import", c.MutualFundHandler.ImportFromCAS)
		mf.POST("/recalculate", c.MutualFundHandler.RecalculateAll)
		mf.GET("/:id", c.MutualFundHandler.GetByID)
		mf.PUT("/:id", c.MutualFundHandler.Update)
		mf.DELETE("/:id", c.MutualFundHandler.Delete)
		mf.POST("/:id/transactions", c.MutualFundHandler.AddTransaction)
		mf.DELETE("/:id/transactions/:txnId", c.MutualFundHandler.DeleteTransaction)
		mf.PUT("/:id/transactions/:txnId", c.MutualFundHandler.UpdateTransaction)
		mf.POST("/:id/refresh-nav", c.MutualFundHandler.RefreshNAV)
		mf.POST("/refresh-nav", c.MutualFundHandler.RefreshNAV)
	}
}

func registerCorporateBondRoutes(api *gin.RouterGroup, c *container.Container) {
	bonds := api.Group("/corporate-bonds")
	{
		bonds.GET("", c.CorporateBondHandler.GetAll)
		bonds.POST("", c.CorporateBondHandler.Create)
		bonds.GET("/:id", c.CorporateBondHandler.GetByID)
		bonds.PUT("/:id", c.CorporateBondHandler.Update)
		bonds.DELETE("/:id", c.CorporateBondHandler.Delete)
		bonds.PUT("/:id/payouts/:payoutId", c.CorporateBondHandler.MarkPayoutReceived)
	}
}

func registerFixedDepositRoutes(api *gin.RouterGroup, c *container.Container) {
	fd := api.Group("/fixed-deposits")
	{
		fd.GET("", c.FixedDepositHandler.GetAll)
		fd.POST("", c.FixedDepositHandler.Create)
		fd.GET("/:id", c.FixedDepositHandler.GetByID)
		fd.PUT("/:id", c.FixedDepositHandler.Update)
		fd.DELETE("/:id", c.FixedDepositHandler.Delete)
	}
}

func registerProvidentFundRoutes(api *gin.RouterGroup, c *container.Container) {
	pf := api.Group("/provident-fund")
	{
		pf.GET("", c.ProvidentFundHandler.GetAll)
		pf.POST("", c.ProvidentFundHandler.Create)
		pf.GET("/:id", c.ProvidentFundHandler.GetByID)
		pf.PUT("/:id", c.ProvidentFundHandler.Update)
		pf.DELETE("/:id", c.ProvidentFundHandler.Delete)
		pf.POST("/:id/entries", c.ProvidentFundHandler.AddContribution)
		pf.POST("/:id/import", c.ProvidentFundHandler.ImportFromPDF)
	}
}

func registerStockRoutes(api *gin.RouterGroup, c *container.Container) {
	stocks := api.Group("/stocks")
	{
		stocks.GET("", c.StockHandler.GetAll)
		stocks.POST("", c.StockHandler.Create)
		stocks.GET("/market-status", c.StockHandler.MarketStatus)
		stocks.POST("/refresh-prices", c.StockHandler.RefreshPrice)
		stocks.GET("/:id", c.StockHandler.GetByID)
		stocks.PUT("/:id", c.StockHandler.Update)
		stocks.DELETE("/:id", c.StockHandler.Delete)
		stocks.POST("/:id/transactions", c.StockHandler.AddTransaction)
		stocks.DELETE("/:id/transactions/:txnId", c.StockHandler.DeleteTransaction)
		stocks.PUT("/:id/transactions/:txnId", c.StockHandler.UpdateTransaction)
		stocks.POST("/:id/dividends", c.StockHandler.AddDividend)
		stocks.DELETE("/:id/dividends/:divId", c.StockHandler.DeleteDividend)
		stocks.POST("/:id/refresh-price", c.StockHandler.RefreshPrice)
	}
}

func registerHomeLoanRoutes(api *gin.RouterGroup, c *container.Container) {
	hl := api.Group("/home-loans")
	{
		hl.GET("", c.HomeLoanHandler.GetAll)
		hl.POST("", c.HomeLoanHandler.Create)
		hl.GET("/:id", c.HomeLoanHandler.GetByID)
		hl.PUT("/:id", c.HomeLoanHandler.Update)
		hl.DELETE("/:id", c.HomeLoanHandler.Delete)
		hl.POST("/:id/emi", c.HomeLoanHandler.RecordEMI)
		hl.POST("/:id/prepayment", c.HomeLoanHandler.AddPrepayment)
		hl.POST("/:id/rate-change", c.HomeLoanHandler.ChangeRate)
		hl.GET("/:id/amortization", c.HomeLoanHandler.GetAmortization)
		hl.POST("/:id/disbursement", c.HomeLoanHandler.AddDisbursement)
		hl.POST("/:id/mark-complete", c.HomeLoanHandler.MarkConstructionComplete)
		hl.POST("/:id/recalculate", c.HomeLoanHandler.Recalculate)
	}
}

func registerPersonalLoanRoutes(api *gin.RouterGroup, c *container.Container) {
	pl := api.Group("/personal-loans")
	{
		pl.GET("", c.PersonalLoanHandler.GetAll)
		pl.POST("", c.PersonalLoanHandler.Create)
		pl.GET("/:id", c.PersonalLoanHandler.GetByID)
		pl.PUT("/:id", c.PersonalLoanHandler.Update)
		pl.DELETE("/:id", c.PersonalLoanHandler.Delete)
		pl.POST("/:id/emi", c.PersonalLoanHandler.RecordEMI)
		pl.POST("/:id/prepayment", c.PersonalLoanHandler.AddPrepayment)
		pl.POST("/:id/rate-change", c.PersonalLoanHandler.ChangeRate)
		pl.GET("/:id/amortization", c.PersonalLoanHandler.GetAmortization)
	}
}

func registerNPSRoutes(api *gin.RouterGroup, c *container.Container) {
	nps := api.Group("/nps")
	{
		nps.GET("", c.NPSHandler.GetAll)
		nps.POST("", c.NPSHandler.Create)
		nps.GET("/:id", c.NPSHandler.GetByID)
		nps.PUT("/:id", c.NPSHandler.Update)
		nps.DELETE("/:id", c.NPSHandler.Delete)
		nps.POST("/:id/contributions", c.NPSHandler.AddContribution)
	}
}

func registerCreditCardRoutes(api *gin.RouterGroup, c *container.Container) {
	cc := api.Group("/credit-cards")
	{
		cc.GET("", c.CreditCardHandler.GetAll)
		cc.POST("", c.CreditCardHandler.Create)
		cc.GET("/:id", c.CreditCardHandler.GetByID)
		cc.PUT("/:id", c.CreditCardHandler.Update)
		cc.DELETE("/:id", c.CreditCardHandler.Delete)
		cc.POST("/:id/statements", c.CreditCardHandler.AddStatement)
		cc.POST("/:id/pay-statement", c.CreditCardHandler.PayStatement)
		cc.POST("/:id/transactions", c.CreditCardHandler.AddTransaction)
		cc.POST("/:id/emis", c.CreditCardHandler.AddEMI)
		cc.POST("/:id/credit-score", c.CreditCardHandler.AddCreditScore)
	}
}

func registerGoalRoutes(api *gin.RouterGroup, c *container.Container) {
	goals := api.Group("/goals")
	{
		goals.GET("", c.GoalHandler.GetAll)
		goals.POST("", c.GoalHandler.Create)
		goals.GET("/:id", c.GoalHandler.GetByID)
		goals.PUT("/:id", c.GoalHandler.Update)
		goals.DELETE("/:id", c.GoalHandler.Delete)
		goals.POST("/:id/link", c.GoalHandler.LinkInvestment)
		goals.POST("/:id/link-batch", c.GoalHandler.BatchLinkInvestments)
		goals.DELETE("/:id/link/:investmentId", c.GoalHandler.UnlinkInvestment)
	}
}

func registerNetWorthRoutes(api *gin.RouterGroup, c *container.Container) {
	nw := api.Group("/net-worth")
	{
		nw.GET("/current", c.NetWorthHandler.GetCurrent)
		nw.POST("/snapshot", c.NetWorthHandler.TakeSnapshot)
		nw.GET("/history", c.NetWorthHandler.GetHistory)
	}
}

func registerSIPRoutes(api *gin.RouterGroup, c *container.Container) {
	sips := api.Group("/sips")
	{
		sips.GET("", c.SIPHandler.GetAll)
		sips.POST("", c.SIPHandler.Create)
		sips.GET("/:id", c.SIPHandler.GetByID)
		sips.PUT("/:id", c.SIPHandler.Update)
		sips.DELETE("/:id", c.SIPHandler.Delete)
		sips.POST("/:id/installments", c.SIPHandler.RecordInstallment)
	}
}

func registerNotificationRoutes(api *gin.RouterGroup, c *container.Container) {
	notif := api.Group("/notifications")
	{
		notif.GET("", c.NotificationHandler.GetAll)
		notif.GET("/unread", c.NotificationHandler.GetUnread)
		notif.POST("/generate", c.NotificationHandler.Generate)
		notif.PUT("/:id/read", c.NotificationHandler.MarkRead)
		notif.PUT("/read-all", c.NotificationHandler.MarkAllRead)
	}
}

func registerProfileRoutes(api *gin.RouterGroup, c *container.Container) {
	profiles := api.Group("/profiles")
	{
		profiles.GET("", c.ProfileHandler.GetAll)
		profiles.POST("", c.ProfileHandler.Create)
		profiles.PUT("/:id", c.ProfileHandler.Update)
		profiles.DELETE("/:id", c.ProfileHandler.Delete)
	}
}

func registerTaxRoutes(api *gin.RouterGroup, c *container.Container) {
	tax := api.Group("/tax")
	{
		tax.GET("/summary", c.TaxHandler.GetTaxSummary)
		tax.GET("/capital-gains", c.TaxHandler.GetCapitalGains)
	}
}
