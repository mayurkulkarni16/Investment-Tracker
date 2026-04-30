package main

import (
	"log"

	"investment-tracker/config"
	"investment-tracker/database"
	"investment-tracker/handlers"
	"investment-tracker/repository"
	"investment-tracker/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Repositories
	mfRepo := repository.NewMutualFundRepo(db)
	bondRepo := repository.NewCorporateBondRepo(db)
	fdRepo := repository.NewFixedDepositRepo(db)
	pfRepo := repository.NewProvidentFundRepo(db)
	stockRepo := repository.NewStockRepo(db)
	homeLoanRepo := repository.NewHomeLoanRepo(db)
	personalLoanRepo := repository.NewPersonalLoanRepo(db)
	npsRepo := repository.NewNPSRepo(db)
	creditCardRepo := repository.NewCreditCardRepo(db)
	goalRepo := repository.NewGoalRepo(db)
	netWorthRepo := repository.NewNetWorthRepo(db)
	sipRepo := repository.NewSIPRepo(db)
	notificationRepo := repository.NewNotificationRepo(db)
	profileRepo := repository.NewProfileRepo(db)

	// Services
	navFetcher := services.NewNAVFetcher()
	priceFetcher := services.NewStockPriceFetcher()
	mfService := services.NewMutualFundService(mfRepo, navFetcher)
	bondService := services.NewCorporateBondService(bondRepo)
	fdService := services.NewFixedDepositService(fdRepo)
	pfService := services.NewProvidentFundService(pfRepo)
	stockService := services.NewStockService(stockRepo, priceFetcher)
	homeLoanService := services.NewHomeLoanService(homeLoanRepo)
	personalLoanService := services.NewPersonalLoanService(personalLoanRepo)
	projectionService := services.NewProjectionService(mfRepo, fdRepo, pfRepo, stockRepo, bondRepo)
	dashboardService := services.NewDashboardService(mfRepo, bondRepo, fdRepo, pfRepo, stockRepo, homeLoanRepo, personalLoanRepo, npsRepo, creditCardRepo)
	npsService := services.NewNPSService(npsRepo)
	creditCardService := services.NewCreditCardService(creditCardRepo)
	goalService := services.NewGoalService(goalRepo, mfRepo, stockRepo, fdRepo, pfRepo, npsRepo, bondRepo)
	netWorthService := services.NewNetWorthService(netWorthRepo, mfRepo, stockRepo, fdRepo, pfRepo, npsRepo, bondRepo, homeLoanRepo, personalLoanRepo, creditCardRepo)
	sipService := services.NewSIPService(sipRepo, mfRepo)
	notificationService := services.NewNotificationService(notificationRepo, homeLoanRepo, personalLoanRepo, fdRepo, bondRepo, sipRepo, creditCardRepo, goalService)
	profileService := services.NewProfileService(profileRepo)
	taxService := services.NewTaxService(mfRepo, fdRepo, pfRepo, npsRepo, bondRepo, homeLoanRepo, personalLoanRepo, stockRepo)
	benchmarkService := services.NewBenchmarkService()
	exportService := services.NewExportService(mfRepo, stockRepo, fdRepo, pfRepo, bondRepo, homeLoanRepo, personalLoanRepo, npsRepo)
	backupService := services.NewBackupService(db, mfRepo, stockRepo, fdRepo, pfRepo, bondRepo, homeLoanRepo, personalLoanRepo, npsRepo, creditCardRepo, goalRepo, sipRepo, profileRepo)
	cashflowService := services.NewCashflowService(mfRepo, bondRepo, fdRepo, pfRepo, stockRepo, npsRepo)
	insightsService := services.NewInsightsService(mfService, stockService, bondService, fdService, pfService, homeLoanRepo, personalLoanRepo, npsService, goalService)
	scheduler := services.NewScheduler(mfService, stockService, netWorthService)

	// Handlers
	mfHandler := handlers.NewMutualFundHandler(mfService)
	bondHandler := handlers.NewCorporateBondHandler(bondService)
	fdHandler := handlers.NewFixedDepositHandler(fdService)
	pfHandler := handlers.NewProvidentFundHandler(pfService)
	stockHandler := handlers.NewStockHandler(stockService)
	homeLoanHandler := handlers.NewHomeLoanHandler(homeLoanService)
	personalLoanHandler := handlers.NewPersonalLoanHandler(personalLoanService)
	projectionHandler := handlers.NewProjectionHandler(projectionService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	npsHandler := handlers.NewNPSHandler(npsService)
	creditCardHandler := handlers.NewCreditCardHandler(creditCardService)
	goalHandler := handlers.NewGoalHandler(goalService)
	netWorthHandler := handlers.NewNetWorthHandler(netWorthService)
	sipHandler := handlers.NewSIPHandler(sipService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	profileHandler := handlers.NewProfileHandler(profileService)
	taxHandler := handlers.NewTaxHandler(taxService)
	benchmarkHandler := handlers.NewBenchmarkHandler(benchmarkService)
	exportHandler := handlers.NewExportHandler(exportService)
	backupHandler := handlers.NewBackupHandler(backupService)
	cashflowHandler := handlers.NewCashflowHandler(cashflowService)
	insightsHandler := handlers.NewInsightsHandler(insightsService)

	// Router
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	api := r.Group("/api/v1")
	{
		// Dashboard & Projections
		api.GET("/dashboard", dashboardHandler.GetDashboard)
		api.POST("/dashboard/rebalance", dashboardHandler.GetRebalanceSuggestions)
		api.GET("/projections", projectionHandler.GetProjections)
		api.GET("/cashflow", cashflowHandler.GetMonthlyCashflows)
		api.GET("/cashflow/:month", cashflowHandler.GetMonthDetail)
		api.GET("/insights", insightsHandler.GetInsights)

		// Mutual Funds
		mf := api.Group("/mutual-funds")
		{
			mf.GET("", mfHandler.GetAll)
			mf.POST("", mfHandler.Create)
			mf.POST("/import", mfHandler.ImportFromCAS)
			mf.POST("/recalculate", mfHandler.RecalculateAll)
			mf.GET("/:id", mfHandler.GetByID)
			mf.PUT("/:id", mfHandler.Update)
			mf.DELETE("/:id", mfHandler.Delete)
			mf.POST("/:id/transactions", mfHandler.AddTransaction)
			mf.DELETE("/:id/transactions/:txnId", mfHandler.DeleteTransaction)
			mf.PUT("/:id/transactions/:txnId", mfHandler.UpdateTransaction)
			mf.POST("/:id/refresh-nav", mfHandler.RefreshNAV)
			mf.POST("/refresh-nav", mfHandler.RefreshNAV)
		}

		// Corporate Bonds
		bonds := api.Group("/corporate-bonds")
		{
			bonds.GET("", bondHandler.GetAll)
			bonds.POST("", bondHandler.Create)
			bonds.GET("/:id", bondHandler.GetByID)
			bonds.PUT("/:id", bondHandler.Update)
			bonds.DELETE("/:id", bondHandler.Delete)
			bonds.PUT("/:id/payouts/:payoutId", bondHandler.MarkPayoutReceived)
		}

		// Fixed Deposits
		fd := api.Group("/fixed-deposits")
		{
			fd.GET("", fdHandler.GetAll)
			fd.POST("", fdHandler.Create)
			fd.GET("/:id", fdHandler.GetByID)
			fd.PUT("/:id", fdHandler.Update)
			fd.DELETE("/:id", fdHandler.Delete)
		}

		// Provident Fund
		pf := api.Group("/provident-fund")
		{
			pf.GET("", pfHandler.GetAll)
			pf.POST("", pfHandler.Create)
			pf.GET("/:id", pfHandler.GetByID)
			pf.PUT("/:id", pfHandler.Update)
			pf.DELETE("/:id", pfHandler.Delete)
			pf.POST("/:id/entries", pfHandler.AddContribution)
			pf.POST("/:id/import", pfHandler.ImportFromPDF)
		}

		// Stocks
		stocks := api.Group("/stocks")
		{
			stocks.GET("", stockHandler.GetAll)
			stocks.POST("", stockHandler.Create)
			stocks.GET("/market-status", stockHandler.MarketStatus)
			stocks.POST("/refresh-prices", stockHandler.RefreshPrice)
			stocks.GET("/:id", stockHandler.GetByID)
			stocks.PUT("/:id", stockHandler.Update)
			stocks.DELETE("/:id", stockHandler.Delete)
			stocks.POST("/:id/transactions", stockHandler.AddTransaction)
			stocks.DELETE("/:id/transactions/:txnId", stockHandler.DeleteTransaction)
			stocks.PUT("/:id/transactions/:txnId", stockHandler.UpdateTransaction)
			stocks.POST("/:id/dividends", stockHandler.AddDividend)
			stocks.DELETE("/:id/dividends/:divId", stockHandler.DeleteDividend)
			stocks.POST("/:id/refresh-price", stockHandler.RefreshPrice)
		}

		// Home Loans
		hl := api.Group("/home-loans")
		{
			hl.GET("", homeLoanHandler.GetAll)
			hl.POST("", homeLoanHandler.Create)
			hl.GET("/:id", homeLoanHandler.GetByID)
			hl.PUT("/:id", homeLoanHandler.Update)
			hl.DELETE("/:id", homeLoanHandler.Delete)
			hl.POST("/:id/emi", homeLoanHandler.RecordEMI)
			hl.POST("/:id/prepayment", homeLoanHandler.AddPrepayment)
			hl.POST("/:id/rate-change", homeLoanHandler.ChangeRate)
			hl.GET("/:id/amortization", homeLoanHandler.GetAmortization)
			hl.POST("/:id/disbursement", homeLoanHandler.AddDisbursement)
			hl.POST("/:id/mark-complete", homeLoanHandler.MarkConstructionComplete)
			hl.POST("/:id/recalculate", homeLoanHandler.Recalculate)
		}

		// Personal Loans
		pl := api.Group("/personal-loans")
		{
			pl.GET("", personalLoanHandler.GetAll)
			pl.POST("", personalLoanHandler.Create)
			pl.GET("/:id", personalLoanHandler.GetByID)
			pl.PUT("/:id", personalLoanHandler.Update)
			pl.DELETE("/:id", personalLoanHandler.Delete)
			pl.POST("/:id/emi", personalLoanHandler.RecordEMI)
			pl.POST("/:id/prepayment", personalLoanHandler.AddPrepayment)
			pl.POST("/:id/rate-change", personalLoanHandler.ChangeRate)
			pl.GET("/:id/amortization", personalLoanHandler.GetAmortization)
		}

		// NPS
		nps := api.Group("/nps")
		{
			nps.GET("", npsHandler.GetAll)
			nps.POST("", npsHandler.Create)
			nps.GET("/:id", npsHandler.GetByID)
			nps.PUT("/:id", npsHandler.Update)
			nps.DELETE("/:id", npsHandler.Delete)
			nps.POST("/:id/contributions", npsHandler.AddContribution)
		}

		// Credit Cards
		cc := api.Group("/credit-cards")
		{
			cc.GET("", creditCardHandler.GetAll)
			cc.POST("", creditCardHandler.Create)
			cc.GET("/:id", creditCardHandler.GetByID)
			cc.PUT("/:id", creditCardHandler.Update)
			cc.DELETE("/:id", creditCardHandler.Delete)
			cc.POST("/:id/statements", creditCardHandler.AddStatement)
			cc.POST("/:id/pay-statement", creditCardHandler.PayStatement)
			cc.POST("/:id/transactions", creditCardHandler.AddTransaction)
			cc.POST("/:id/emis", creditCardHandler.AddEMI)
			cc.POST("/:id/credit-score", creditCardHandler.AddCreditScore)
		}

		// Goals
		goals := api.Group("/goals")
		{
			goals.GET("", goalHandler.GetAll)
			goals.POST("", goalHandler.Create)
			goals.GET("/:id", goalHandler.GetByID)
			goals.PUT("/:id", goalHandler.Update)
			goals.DELETE("/:id", goalHandler.Delete)
			goals.POST("/:id/link", goalHandler.LinkInvestment)
			goals.POST("/:id/link-batch", goalHandler.BatchLinkInvestments)
			goals.DELETE("/:id/link/:investmentId", goalHandler.UnlinkInvestment)
		}

		// Net Worth
		nw := api.Group("/net-worth")
		{
			nw.GET("/current", netWorthHandler.GetCurrent)
			nw.POST("/snapshot", netWorthHandler.TakeSnapshot)
			nw.GET("/history", netWorthHandler.GetHistory)
		}

		// SIPs
		sips := api.Group("/sips")
		{
			sips.GET("", sipHandler.GetAll)
			sips.POST("", sipHandler.Create)
			sips.GET("/:id", sipHandler.GetByID)
			sips.PUT("/:id", sipHandler.Update)
			sips.DELETE("/:id", sipHandler.Delete)
			sips.POST("/:id/installments", sipHandler.RecordInstallment)
		}

		// Notifications
		notif := api.Group("/notifications")
		{
			notif.GET("", notificationHandler.GetAll)
			notif.GET("/unread", notificationHandler.GetUnread)
			notif.POST("/generate", notificationHandler.Generate)
			notif.PUT("/:id/read", notificationHandler.MarkRead)
			notif.PUT("/read-all", notificationHandler.MarkAllRead)
		}

		// Profiles
		profiles := api.Group("/profiles")
		{
			profiles.GET("", profileHandler.GetAll)
			profiles.POST("", profileHandler.Create)
			profiles.PUT("/:id", profileHandler.Update)
			profiles.DELETE("/:id", profileHandler.Delete)
		}

		// Tax
		tax := api.Group("/tax")
		{
			tax.GET("/summary", taxHandler.GetTaxSummary)
			tax.GET("/capital-gains", taxHandler.GetCapitalGains)
		}

		// Benchmarks
		api.GET("/benchmarks", benchmarkHandler.GetBenchmarks)

		// Export
		api.GET("/export/csv", exportHandler.ExportCSV)

		// Backup
		api.GET("/backup", backupHandler.Export)
		api.POST("/backup/restore", backupHandler.Restore)
	}

	// Start daily scheduler
	scheduler.Start()
	defer scheduler.Stop()

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
