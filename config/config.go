package config

import (
	"log"
	"os"
)

type Config struct {
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     string
}

func NewConfig() *Config {
	config := &Config{
		DBHost:     getEnvOrDefault("DB_HOST", "chat-application-postgres-1"),
		DBUser:     getEnvOrDefault("DB_USER", "postgres"),
		DBPassword: getEnvOrDefault("DB_PASSWORD", "123321"),
		DBName:     getEnvOrDefault("DB_NAME", "chat_app"),
		DBPort:     getEnvOrDefault("DB_PORT", "5432"),
	}

	// Validate critical fields
	if config.DBHost == "" || config.DBUser == "" || config.DBPassword == "" || config.DBName == "" || config.DBPort == "" {
		log.Fatal("One or more required database configuration values are missing")
	}

	return config
}

// Helper function to get env var or fallback to default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	log.Printf("Environment variable %s not set, using default: %s", key, defaultValue)
	return defaultValue
}
