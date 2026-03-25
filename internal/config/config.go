package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL      string
	DatabasePort     int
	DatabaseUsername string
	DatabasePassword string
	RedisAddr        string
	Port             string
}

func Load() (*Config, error) {
	port, err := strconv.Atoi(getEnvOrDefault("DATABASE_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_PORT: %w", err)
	}

	return &Config{
		DatabaseURL:      getEnvOrDefault("DATABASE_URL", ""),
		DatabasePort:     port,
		DatabaseUsername: getEnvOrDefault("DATABASE_USERNAME", ""),
		DatabasePassword: getEnvOrDefault("DATABASE_PASSWORD", ""),
		RedisAddr:        getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		Port:             getEnvOrDefault("PORT", "8080"),
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
