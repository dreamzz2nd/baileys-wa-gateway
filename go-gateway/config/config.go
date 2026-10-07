package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	APIKey        string
	WebhookURL    string
	DBPath        string
	LogLevel      string
}

func LoadConfig() *Config {
	_ = godotenv.Load() // Memuat .env jika ada

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		apiKey = "mysecretkey123"
	}

	webhookURL := os.Getenv("WEBHOOK_URL")

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "whatsapp.db"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	return &Config{
		Port:       port,
		APIKey:     apiKey,
		WebhookURL: webhookURL,
		DBPath:     dbPath,
		LogLevel:   logLevel,
	}
}
