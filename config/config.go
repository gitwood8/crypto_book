package config

import (
	"log"
	"os"
)

type Config struct {
	TelegramBotToken string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
}

func Load() *Config {
	cfg := &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DBHost:           os.Getenv("DB_HOST"),
		DBPort:           os.Getenv("DB_PORT"),
		DBUser:           os.Getenv("DB_USER"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		DBName:           os.Getenv("DB_NAME"),
	}

	if cfg.TelegramBotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}
	return cfg
}

