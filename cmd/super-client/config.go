package main

import (
	"fmt"
	"os"
	"strconv"
)

type config struct {
	NATSUrl      string
	PoolSize     int
	ClientPrefix string
	LogLevel     string
}

func loadConfig() (config, error) {
	natsURL := getenv("NATS_URL", "nats://localhost:4222")
	poolSizeStr := getenv("POOL_SIZE", "5")
	poolSize, err := strconv.Atoi(poolSizeStr)
	if err != nil || poolSize <= 0 {
		return config{}, fmt.Errorf("invalid POOL_SIZE=%q: must be a positive integer", poolSizeStr)
	}
	return config{
		NATSUrl:      natsURL,
		PoolSize:     poolSize,
		ClientPrefix: getenv("CLIENT_PREFIX", "client"),
		LogLevel:     getenv("LOG_LEVEL", "info"),
	}, nil
}

func getenv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
