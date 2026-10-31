package routes

import (
	"investment-tracker/container"

	"github.com/gin-gonic/gin"
)

func registerPersonalFinanceRoutes(api *gin.RouterGroup, c *container.Container) {
	accounts := api.Group("/accounts")
	{
		accounts.GET("", c.AccountHandler.GetAll)
		accounts.POST("", c.AccountHandler.Create)
		accounts.GET("/:id", c.AccountHandler.GetByID)
		accounts.PUT("/:id", c.AccountHandler.Update)
		accounts.DELETE("/:id", c.AccountHandler.Delete)
		accounts.PUT("/:id/balance", c.AccountHandler.UpdateBalance)
	}

	budgets := api.Group("/budgets")
	{
		budgets.GET("", c.AccountHandler.GetBudgets)
		budgets.POST("", c.AccountHandler.SaveBudget)
		budgets.DELETE("/:id", c.AccountHandler.DeleteBudget)
	}

	insurance := api.Group("/insurance")
	{
		insurance.GET("", c.InsuranceHandler.GetAll)
		insurance.POST("", c.InsuranceHandler.Create)
		insurance.GET("/summary", c.InsuranceHandler.Summary)
		insurance.GET("/:id", c.InsuranceHandler.GetByID)
		insurance.PUT("/:id", c.InsuranceHandler.Update)
		insurance.DELETE("/:id", c.InsuranceHandler.Delete)
	}

	gold := api.Group("/gold")
	{
		gold.GET("", c.GoldHandler.GetAll)
		gold.POST("", c.GoldHandler.Create)
		gold.POST("/rate", c.GoldHandler.SetRate)
		gold.POST("/refresh-prices", c.GoldHandler.RefreshPrices)
		gold.GET("/:id", c.GoldHandler.GetByID)
		gold.PUT("/:id", c.GoldHandler.Update)
		gold.DELETE("/:id", c.GoldHandler.Delete)
	}

	properties := api.Group("/properties")
	{
		properties.GET("", c.PropertyHandler.GetAll)
		properties.POST("", c.PropertyHandler.Create)
		properties.GET("/:id", c.PropertyHandler.GetByID)
		properties.PUT("/:id", c.PropertyHandler.Update)
		properties.DELETE("/:id", c.PropertyHandler.Delete)
	}

	api.GET("/family/summary", c.PlanningHandler.FamilySummary)
	api.PUT("/family/assign", c.PlanningHandler.Assign)
	api.GET("/fire/defaults", c.PlanningHandler.FIREDefaults)
	api.POST("/fire/plan", c.PlanningHandler.FIREPlan)
	api.GET("/finance/overview", c.PlanningHandler.Overview)

	docs := api.Group("/documents")
	{
		docs.GET("", c.DocumentHandler.List)
		docs.POST("", c.DocumentHandler.Upload)
		docs.GET("/:id/download", c.DocumentHandler.Download)
		docs.DELETE("/:id", c.DocumentHandler.Delete)
	}
}
