package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port            string
	LogLevel        string
	Symbols         []string
	Workers         int
	MaxPositionSize int64
	MaxOrderSize    int64
	MaxDailyLoss    float64
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		Symbols:         getEnvSlice("SYMBOLS", []string{"AAPL", "GOOGL", "MSFT", "AMZN", "TSLA"}),
		Workers:         getEnvInt("WORKERS", 4),
		MaxPositionSize: int64(getEnvInt("MAX_POSITION_SIZE", 10000)),
		MaxOrderSize:    int64(getEnvInt("MAX_ORDER_SIZE", 1000)),
		MaxDailyLoss:    getEnvFloat("MAX_DAILY_LOSS", 50000.0),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
