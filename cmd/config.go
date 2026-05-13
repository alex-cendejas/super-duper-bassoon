package main

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPHost                       string
	HTTPPort                       int
	HTTPReadTimeoutMS              int64
	DBPath                         string
	NatsURL                        string
	TriggerCheckIntervalMS         int64
	HealthAggregationIntervalMS    int64
	HealthWindowSize               int
	HealthSuccessThreshold         float64
	CircuitBreakerCheckIntervalMS  int64
	CircuitBreakerSuccessThreshold float64
	CircuitBreakerCooldownMS       int64
	LoopThresholdMS                int64
	LogLevel                       string
	ShutdownTimeoutMS              int64
	StartNATS                      bool
}

func envStr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func envInt64(key string, def int64) int64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func LoadConfig() (*Config, error) {
	c := &Config{
		HTTPHost:                       envStr("HTTP_HOST", "0.0.0.0"),
		HTTPPort:                       envInt("HTTP_PORT", 8080),
		HTTPReadTimeoutMS:              envInt64("HTTP_READ_TIMEOUT_MS", 30000),
		DBPath:                         envStr("DB_PATH", "./data/server.db"),
		NatsURL:                        envStr("NATS_URL", "nats://localhost:4222"),
		TriggerCheckIntervalMS:         envInt64("TRIGGER_CHECK_INTERVAL_MS", 5000),
		HealthAggregationIntervalMS:    envInt64("HEALTH_AGGREGATION_INTERVAL_MS", 5000),
		HealthWindowSize:               envInt("HEALTH_WINDOW_SIZE", 10),
		HealthSuccessThreshold:         envFloat("HEALTH_SUCCESS_THRESHOLD", 80),
		CircuitBreakerCheckIntervalMS:  envInt64("CIRCUIT_BREAKER_CHECK_INTERVAL_MS", 10000),
		CircuitBreakerSuccessThreshold: envFloat("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 80),
		CircuitBreakerCooldownMS:       envInt64("CIRCUIT_BREAKER_COOLDOWN_MS", 300000),
		LoopThresholdMS:                envInt64("LOOP_THRESHOLD_MS", 5000),
		LogLevel:                       envStr("LOG_LEVEL", "info"),
		ShutdownTimeoutMS:              envInt64("SHUTDOWN_TIMEOUT_MS", 30000),
		StartNATS:                      envBool("START_NATS", true),
	}
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return nil, fmt.Errorf("invalid HTTP_PORT: %d", c.HTTPPort)
	}
	if c.HealthSuccessThreshold < 0 || c.HealthSuccessThreshold > 100 {
		return nil, fmt.Errorf("invalid HEALTH_SUCCESS_THRESHOLD")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return nil, fmt.Errorf("invalid LOG_LEVEL: %s", c.LogLevel)
	}
	return c, nil
}
