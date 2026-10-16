package main

import (
	"context"
	"log"
	"os"

	"investment-tracker/config"
	"investment-tracker/container"
	"investment-tracker/database"
	"investment-tracker/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Wire all dependencies
	c := container.New(db, cfg)

	// Seed admin account
	c.AuthService.SeedAdmin(context.Background())

	// Router
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-View-As-User"},
		AllowCredentials: true,
	}))

	// Register all routes
	routes.Register(r, c, cfg)

	// Start daily scheduler
	c.Scheduler.Start()
	defer c.Scheduler.Stop()

	// Serve frontend static files in production
	if _, err := os.Stat("./static"); err == nil {
		r.Static("/assets", "./static/assets")
		r.StaticFile("/favicon.ico", "./static/favicon.ico")
		r.NoRoute(func(c *gin.Context) {
			c.File("./static/index.html")
		})
		log.Println("Serving frontend from ./static")
	}

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
