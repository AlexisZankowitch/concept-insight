package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SlackToken string
}

var AppConfig Config

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	AppConfig = Config{
		SlackToken: getEnvOrFatal("SLACK_TOKEN"),
	}
}

func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return value
}
