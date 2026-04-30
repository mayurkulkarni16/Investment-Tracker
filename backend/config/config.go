package config

import (
	"os"
)

type Config struct {
	MongoURI      string
	DBName        string
	ServerPort    string
	JWTSecret     string
	AdminEmail    string
	AdminPassword string
	FrontendURL   string
}

func Load() *Config {
	return &Config{
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:        getEnv("DB_NAME", "investment_tracker"),
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-production-32chars!"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@investtrack.com"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		FrontendURL:   getEnv("FRONTEND_URL", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
