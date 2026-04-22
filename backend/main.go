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

	// Services
	navFetcher := services.NewNAVFetcher()
	priceFetcher := services.NewStockPriceFetcher()
	mfService := services.NewMutualFundService(mfRepo, navFetcher)
	bondService := services.NewCorporateBondService(bondRepo)
	fdService := services.NewFixedDepositService(fdRepo)
	pfService := services.NewProvidentFundService(pfRepo)
	stockService := services.NewStockService(stockRepo, priceFetcher)
	dashboardService := services.NewDashboardService(mfRepo, bondRepo, fdRepo, pfRepo, stockRepo)

	// Handlers
	mfHandler := handlers.NewMutualFundHandler(mfService)
	bondHandler := handlers.NewCorporateBondHandler(bondService)
	fdHandler := handlers.NewFixedDepositHandler(fdService)
	pfHandler := handlers.NewProvidentFundHandler(pfService)
	stockHandler := handlers.NewStockHandler(stockService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

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
		// Dashboard
		api.GET("/dashboard", dashboardHandler.GetDashboard)

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
			stocks.POST("/:id/refresh-price", stockHandler.RefreshPrice)
		}
	}

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
