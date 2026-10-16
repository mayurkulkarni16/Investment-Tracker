package container

import (
	"investment-tracker/config"
	"investment-tracker/handlers"
	"investment-tracker/repository"
	"investment-tracker/services"

	"go.mongodb.org/mongo-driver/mongo"
)

// Container holds all application dependencies wired together.
type Container struct {
	// Handlers
	AuthHandler          *handlers.AuthHandler
	MutualFundHandler    *handlers.MutualFundHandler
	CorporateBondHandler *handlers.CorporateBondHandler
	FixedDepositHandler  *handlers.FixedDepositHandler
	ProvidentFundHandler *handlers.ProvidentFundHandler
	StockHandler         *handlers.StockHandler
	HomeLoanHandler      *handlers.HomeLoanHandler
	PersonalLoanHandler  *handlers.PersonalLoanHandler
	ProjectionHandler    *handlers.ProjectionHandler
	DashboardHandler     *handlers.DashboardHandler
	NPSHandler           *handlers.NPSHandler
	CreditCardHandler    *handlers.CreditCardHandler
	GoalHandler          *handlers.GoalHandler
	NetWorthHandler      *handlers.NetWorthHandler
	SIPHandler           *handlers.SIPHandler
	NotificationHandler  *handlers.NotificationHandler
	ProfileHandler       *handlers.ProfileHandler
	TaxHandler           *handlers.TaxHandler
	BenchmarkHandler     *handlers.BenchmarkHandler
	ExportHandler        *handlers.ExportHandler
	BackupHandler        *handlers.BackupHandler
	CashflowHandler      *handlers.CashflowHandler
	InsightsHandler      *handlers.InsightsHandler

	// Services exposed for use in main (seeding, scheduling, etc.)
	AuthService     *services.AuthService
	MFService       *services.MutualFundService
	StockService    *services.StockService
	NetWorthService *services.NetWorthService
	SIPService      *services.SIPService
	Scheduler       *services.Scheduler
}

// New creates and wires all repositories, services, and handlers.
func New(db *mongo.Database, cfg *config.Config) *Container {
	// Repositories
	userRepo := repository.NewUserRepo(db)
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
	authService := services.NewAuthService(userRepo, cfg)
	navFetcher := services.NewNAVFetcher()
	priceFetcher := services.NewStockPriceFetcher()
	mfService := services.NewMutualFundService(mfRepo, navFetcher)
	bondService := services.NewCorporateBondService(bondRepo)
	fdService := services.NewFixedDepositService(fdRepo)
	pfService := services.NewProvidentFundService(pfRepo)
	stockService := services.NewStockService(stockRepo, priceFetcher)
	homeLoanService := services.NewHomeLoanService(homeLoanRepo)
	personalLoanService := services.NewPersonalLoanService(personalLoanRepo)
	projectionService := services.NewProjectionService(mfRepo, fdRepo, pfRepo, stockRepo, bondRepo, npsRepo, sipRepo)
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
	cashflowService := services.NewCashflowService(mfRepo, bondRepo, fdRepo, pfRepo, stockRepo, npsRepo, homeLoanRepo, personalLoanRepo, creditCardRepo)
	insightsService := services.NewInsightsService(mfService, stockService, bondService, fdService, pfService, homeLoanRepo, personalLoanRepo, npsService, goalService)
	scheduler := services.NewScheduler(mfService, stockService, netWorthService)

	// Handlers
	return &Container{
		AuthHandler:          handlers.NewAuthHandler(authService),
		MutualFundHandler:    handlers.NewMutualFundHandler(mfService),
		CorporateBondHandler: handlers.NewCorporateBondHandler(bondService),
		FixedDepositHandler:  handlers.NewFixedDepositHandler(fdService),
		ProvidentFundHandler: handlers.NewProvidentFundHandler(pfService),
		StockHandler:         handlers.NewStockHandler(stockService),
		HomeLoanHandler:      handlers.NewHomeLoanHandler(homeLoanService),
		PersonalLoanHandler:  handlers.NewPersonalLoanHandler(personalLoanService),
		ProjectionHandler:    handlers.NewProjectionHandler(projectionService),
		DashboardHandler:     handlers.NewDashboardHandler(dashboardService),
		NPSHandler:           handlers.NewNPSHandler(npsService),
		CreditCardHandler:    handlers.NewCreditCardHandler(creditCardService),
		GoalHandler:          handlers.NewGoalHandler(goalService),
		NetWorthHandler:      handlers.NewNetWorthHandler(netWorthService),
		SIPHandler:           handlers.NewSIPHandler(sipService),
		NotificationHandler:  handlers.NewNotificationHandler(notificationService),
		ProfileHandler:       handlers.NewProfileHandler(profileService),
		TaxHandler:           handlers.NewTaxHandler(taxService),
		BenchmarkHandler:     handlers.NewBenchmarkHandler(benchmarkService),
		ExportHandler:        handlers.NewExportHandler(exportService),
		BackupHandler:        handlers.NewBackupHandler(backupService),
		CashflowHandler:      handlers.NewCashflowHandler(cashflowService),
		InsightsHandler:      handlers.NewInsightsHandler(insightsService),

		AuthService:     authService,
		MFService:       mfService,
		StockService:    stockService,
		NetWorthService: netWorthService,
		SIPService:      sipService,
		Scheduler:       scheduler,
	}
}
