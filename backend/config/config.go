package config

import (
	"os"
)

type Config struct {
	MongoURI   string
	DBName     string
	ServerPort string
}

func Load() *Config {
	return &Config{
		MongoURI:   getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:     getEnv("DB_NAME", "investment_tracker"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
