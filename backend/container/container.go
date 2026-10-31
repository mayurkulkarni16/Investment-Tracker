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
	LedgerHandler *handlers.LedgerHandler
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
	VehicleRentalHandler *handlers.VehicleRentalHandler
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

	// Personal finance
	AccountHandler      *handlers.AccountHandler
	InsuranceHandler    *handlers.InsuranceHandler
	GoldHandler         *handlers.GoldHandler
	PropertyHandler     *handlers.PropertyHandler
	PlanningHandler     *handlers.PlanningHandler
	MessagingHandler    *handlers.MessagingHandler
	DocumentHandler     *handlers.DocumentHandler
	BrokerImportHandler *handlers.BrokerImportHandler

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
	vehicleRentalRepo := repository.NewVehicleRentalRepo(db)
	ledgerRepo := repository.NewLedgerRepo(db)
	bankRepo := repository.NewBankAccountRepo(db)
	budgetRepo := repository.NewBudgetRepo(db)
	insuranceRepo := repository.NewInsuranceRepo(db)
	goldRepo := repository.NewGoldRepo(db)
	propertyRepo := repository.NewPropertyRepo(db)
	assignmentRepo := repository.NewProfileAssignmentRepo(db)
	notifSettingsRepo := repository.NewNotificationSettingsRepo(db)
	documentRepo := repository.NewDocumentRepo(db)

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
	projectionService := services.NewProjectionService(mfRepo, fdRepo, pfRepo, stockRepo, bondRepo, npsRepo, sipRepo, homeLoanRepo, personalLoanRepo, vehicleRentalRepo)
	taxService := services.NewTaxService(mfRepo, fdRepo, pfRepo, npsRepo, bondRepo, homeLoanRepo, personalLoanRepo, stockRepo)
	dashboardService := services.NewDashboardService(mfRepo, bondRepo, fdRepo, pfRepo, stockRepo, homeLoanRepo, personalLoanRepo, npsRepo, creditCardRepo, vehicleRentalRepo, taxService)
	npsService := services.NewNPSService(npsRepo)
	vehicleRentalService := services.NewVehicleRentalService(vehicleRentalRepo)
	creditCardService := services.NewCreditCardService(creditCardRepo)
	goalService := services.NewGoalService(goalRepo, mfRepo, stockRepo, fdRepo, pfRepo, npsRepo, bondRepo, sipRepo)
	netWorthService := services.NewNetWorthService(netWorthRepo, mfRepo, stockRepo, fdRepo, pfRepo, npsRepo, bondRepo, homeLoanRepo, personalLoanRepo, creditCardRepo, vehicleRentalRepo)
	sipService := services.NewSIPService(sipRepo, mfRepo)
	notificationService := services.NewNotificationService(notificationRepo, homeLoanRepo, personalLoanRepo, fdRepo, bondRepo, sipRepo, creditCardRepo, vehicleRentalRepo, goalService)
	profileService := services.NewProfileService(profileRepo)
	benchmarkService := services.NewBenchmarkService()
	exportService := services.NewExportService(mfRepo, stockRepo, fdRepo, pfRepo, bondRepo, homeLoanRepo, personalLoanRepo, npsRepo, vehicleRentalRepo)
	backupService := services.NewBackupService(db, mfRepo, stockRepo, fdRepo, pfRepo, bondRepo, homeLoanRepo, personalLoanRepo, npsRepo, creditCardRepo, goalRepo, sipRepo, profileRepo, vehicleRentalRepo)
	cashflowService := services.NewCashflowService(mfRepo, bondRepo, fdRepo, pfRepo, stockRepo, npsRepo, homeLoanRepo, personalLoanRepo, creditCardRepo, vehicleRentalRepo)
	insightsService := services.NewInsightsService(mfService, stockService, bondService, fdService, pfService, homeLoanRepo, personalLoanRepo, npsService, goalService, taxService, vehicleRentalRepo)
	scheduler := services.NewScheduler(userRepo, mfService, stockService, netWorthService, notificationService)

	// Personal finance: cash, budgets, insurance, gold, real estate, family,
	// FIRE planning, reminders by email/Telegram, documents and broker imports.
	accountService := services.NewAccountService(bankRepo, ledgerRepo)
	budgetService := services.NewBudgetService(budgetRepo, ledgerRepo)
	insuranceService := services.NewInsuranceService(insuranceRepo, ledgerRepo, homeLoanRepo, personalLoanRepo)
	goldService := services.NewGoldService(goldRepo, priceFetcher)
	propertyService := services.NewPropertyService(propertyRepo, homeLoanRepo)
	netWorthService.SetPersonalFinanceRepos(bankRepo, goldRepo, propertyRepo, insuranceRepo)
	exportService.SetPersonalFinanceRepos(bankRepo, goldRepo, propertyRepo, insuranceRepo)
	taxService.SetInsuranceRepo(insuranceRepo)
	notificationService.SetFinanceSources(insuranceRepo, budgetService, documentRepo)
	familyService := services.NewFamilyService(netWorthService, profileRepo, assignmentRepo)
	fireService := services.NewFIREService(netWorthService, ledgerRepo, sipRepo)
	financeService := services.NewFinanceService(netWorthService, netWorthRepo, bankRepo, budgetService, insuranceService, ledgerRepo)
	messagingService := services.NewMessagingService(cfg, notifSettingsRepo, notificationRepo, userRepo)
	authService.SetMailer(messagingService)
	scheduler.SetMessaging(messagingService)
	documentService, err := services.NewDocumentService(db, documentRepo)
	if err != nil {
		panic("document storage: " + err.Error())
	}
	brokerImportService := services.NewBrokerImportService(stockService, stockRepo)

	// Handlers
	return &Container{
		LedgerHandler:        handlers.NewLedgerHandler(services.NewLedgerService(ledgerRepo)),
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
		VehicleRentalHandler: handlers.NewVehicleRentalHandler(vehicleRentalService),
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

		AccountHandler:      handlers.NewAccountHandler(accountService, budgetService),
		InsuranceHandler:    handlers.NewInsuranceHandler(insuranceService),
		GoldHandler:         handlers.NewGoldHandler(goldService),
		PropertyHandler:     handlers.NewPropertyHandler(propertyService),
		PlanningHandler:     handlers.NewPlanningHandler(familyService, fireService, financeService),
		MessagingHandler:    handlers.NewMessagingHandler(messagingService),
		DocumentHandler:     handlers.NewDocumentHandler(documentService),
		BrokerImportHandler: handlers.NewBrokerImportHandler(brokerImportService),

		AuthService:     authService,
		MFService:       mfService,
		StockService:    stockService,
		NetWorthService: netWorthService,
		SIPService:      sipService,
		Scheduler:       scheduler,
	}
}
