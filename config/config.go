package config

import (
	"os"
	"strconv"
)

type Config struct {
	MaxConcurrentClaims int
	MaxConcurrentTransfers int
	FloodingGoroutines int
	ClaimingFee int64  // In stroops
	TransferFee int64  // In stroops
	MaxRetries int
	RetryDelay int // milliseconds
}

func LoadConfig() *Config {
	return &Config{
		MaxConcurrentClaims: getEnvInt("MAX_CONCURRENT_CLAIMS", 50),
		MaxConcurrentTransfers: getEnvInt("MAX_CONCURRENT_TRANSFERS", 30),
		FloodingGoroutines: getEnvInt("FLOODING_GOROUTINES", 100),
		ClaimingFee: getEnvInt64("CLAIMING_FEE", 32000000), // 3.2 PI in stroops
		TransferFee: getEnvInt64("TRANSFER_FEE", 94000000), // 9.4 PI in stroops
		MaxRetries: getEnvInt("MAX_RETRIES", 20),
		RetryDelay: getEnvInt("RETRY_DELAY", 50),
	}
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}