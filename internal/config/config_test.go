package config_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
)

func TestLoadConfig(t *testing.T) {
	testYAML := `
routes:
  "gitsense.dyastin.tech": 
    routes:
      "/api": "http://localhost:4001"
      "/": "http://localhost:4001"
    rate_limit:
      burst: 100
      rate: 50
      cooldown: 60000
  "filespace.dyastin.tech":
    routes:
      "/api/v2": "http://localhost:3004"
      "/": "http://localhost:5005"
    rate_limit:
      burst: 100
      rate: 50
      cooldown: 60000
email: "mail@dyastin.tech"
rate_limit:
  burst: 100
  rate: 50
  cooldown: 60000
`

	tmpFile, err := os.CreateTemp("", "test_config_*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(testYAML)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	err = config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("config.Load() failed: %v", err)
	}

	if config.Email != "mail@dyastin.tech" {
		t.Errorf("expected Email to be 'mail@dyastin.tech', got '%s'", config.Email)
	}

	expectedGlobalCooldown := 60 * time.Second
	if config.GlobalRateLimit.Cooldown != expectedGlobalCooldown {
		t.Errorf("expected GlobalRateLimit.Cooldown to be %v, got %v", expectedGlobalCooldown, config.GlobalRateLimit.Cooldown)
	}

	if config.GlobalRateLimit.Burst != 100 {
		t.Errorf("expected GlobalRateLimit.Burst to be 100, got %d", config.GlobalRateLimit.Burst)
	}

	routeConfig, ok := config.Routes["gitsense.dyastin.tech"]
	for domain, cfg := range config.Routes {
		fmt.Printf("domain: %s\n", domain)
		for key, value := range cfg.Routes {
			fmt.Printf("key: %s, value: %s\n", key, value)
		}
		fmt.Printf("rate limit cooldown: %v\n", cfg.RateLimit.Cooldown)
		fmt.Printf("rate limit burst: %d\n", cfg.RateLimit.Burst)
	}

	if !ok {
		t.Fatalf("expected routes for 'gitsense.dyastin.tech' not found")
	}

	if routeConfig.Routes["/api"] != "http://localhost:4001" {
		t.Errorf("expected '/api' route to be 'http://localhost:4001', got '%s'", routeConfig.Routes["/api"])
	}

	if routeConfig.Routes["/"] != "http://localhost:4001" {
		t.Errorf("expected '/' route to be 'http://localhost:4001', got '%s'", routeConfig.Routes["/"])
	}

	domainCooldown := 60 * time.Second
	if routeConfig.RateLimit.Cooldown != domainCooldown {
		t.Errorf("expected cooldown for 'gitsense.dyastin.tech' to be %v, got %v", domainCooldown, routeConfig.RateLimit.Cooldown)
	}

	if routeConfig.RateLimit.Burst != 100 {
		t.Errorf("expected burst for 'gitsense.dyastin.tech' to be 100, got %d", routeConfig.RateLimit.Burst)
	}

	t.Log("config.Load() test passed")
}
