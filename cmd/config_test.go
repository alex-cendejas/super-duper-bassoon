package main

import (
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("HTTP_PORT", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("NATS_URL", "")
	t.Setenv("LOG_LEVEL", "")
	c, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected defaults: %v", err)
	}
	if c.HTTPPort != 8080 || c.DBPath == "" || c.NatsURL == "" || c.LogLevel != "info" {
		t.Errorf("got: %+v", c)
	}
}

func TestLoadConfig_Validate(t *testing.T) {
	t.Setenv("HTTP_PORT", "99999")
	if _, err := LoadConfig(); err == nil {
		t.Error("expected invalid port")
	}
	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("HEALTH_SUCCESS_THRESHOLD", "150")
	if _, err := LoadConfig(); err == nil {
		t.Error("expected invalid threshold")
	}
	t.Setenv("HEALTH_SUCCESS_THRESHOLD", "80")
	t.Setenv("LOG_LEVEL", "bogus")
	if _, err := LoadConfig(); err == nil {
		t.Error("expected invalid log level")
	}
}

func TestLoadConfig_FromEnv(t *testing.T) {
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("HEALTH_WINDOW_SIZE", "20")
	t.Setenv("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", "75.5")
	t.Setenv("START_NATS", "false")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOOP_THRESHOLD_MS", "2500")
	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTPPort != 9090 {
		t.Errorf("port: %d", c.HTTPPort)
	}
	if c.HealthWindowSize != 20 {
		t.Errorf("window: %d", c.HealthWindowSize)
	}
	if c.CircuitBreakerSuccessThreshold != 75.5 {
		t.Errorf("threshold: %v", c.CircuitBreakerSuccessThreshold)
	}
	if c.StartNATS {
		t.Error("start_nats false")
	}
	if c.LoopThresholdMS != 2500 {
		t.Errorf("loop ms: %d", c.LoopThresholdMS)
	}
}

func TestEnvHelpers_BadValues(t *testing.T) {
	t.Setenv("BOGUS_INT", "not-a-number")
	if envInt("BOGUS_INT", 7) != 7 {
		t.Error("default int on bad")
	}
	t.Setenv("BOGUS_INT64", "x")
	if envInt64("BOGUS_INT64", 9) != 9 {
		t.Error("default int64")
	}
	t.Setenv("BOGUS_FLOAT", "x")
	if envFloat("BOGUS_FLOAT", 1.5) != 1.5 {
		t.Error("default float")
	}
	t.Setenv("BOGUS_BOOL", "x")
	if envBool("BOGUS_BOOL", true) != true {
		t.Error("default bool")
	}
	if envStr("MISSING_STR", "def") != "def" {
		t.Error("default str")
	}
}
