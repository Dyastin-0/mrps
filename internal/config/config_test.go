package config_test

import (
	"os"
	"testing"

	"github.com/Dyastin-0/mrps/internal/config"
)

func TestLoadConfig(t *testing.T) {
	testYAML := `
routes:
  "*.example.com":
    routes:
      "/wildcard": "http://localhost:5003"

  "example.com":
    routes:
      "/api": "http://localhost:5001"
      "/home": "http://localhost:5002"

  "another.com":
    routes:
      "/": "http://localhost:6001"

  "*.another.com":
    routes:
      "/wild": "http://localhost:6002"
      "/api": "http://localhost:6003"

rate_limit:
  burst: 200
  rate: 100
  cooldown: 120000

misc:
  email: "admin@example.com"
  metrics_port: 8080
`

	tmpFile, err := os.CreateTemp("", "test_config_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Errorf("Failed to remove temp file: %v", err)
		}
	}()

	if _, err := tmpFile.Write([]byte(testYAML)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	if err := config.Load(tmpFile.Name()); err != nil {
		t.Fatalf("config.Load() failed: %v", err)
	}

	t.Run("Match Exact Domains", func(t *testing.T) {
		tests := []struct {
			domain       string
			expectedPath string
			expectedURL  string
		}{
			{"example.com", "/api", "http://localhost:5001"},
			{"another.com", "/", "http://localhost:6001"},
		}

		for _, test := range tests {
			routeConfigPtr := config.DomainTrie.Match(test.domain)
			if routeConfigPtr == nil {
				t.Fatalf("Domain not found in trie: %s", test.domain)
			}
			routeConfig := *routeConfigPtr
			assertEqual(t, routeConfig.Routes[test.expectedPath], test.expectedURL, "Routes")
		}
	})

	t.Run("Match Wildcard Domains", func(t *testing.T) {
		tests := []struct {
			domain       string
			expectedPath string
			expectedURL  string
		}{
			{"sub.example.com", "/wildcard", "http://localhost:5003"},
			{"test.example.com", "/wildcard", "http://localhost:5003"},
			{"sub.another.com", "/wild", "http://localhost:6002"},
			{"api.another.com", "/api", "http://localhost:6003"},
		}

		for _, test := range tests {
			routeConfigPtr := config.DomainTrie.Match(test.domain)
			if routeConfigPtr == nil {
				t.Fatalf("Domain not found in trie: %s", test.domain)
			}
			routeConfig := *routeConfigPtr
			assertEqual(t, routeConfig.Routes[test.expectedPath], test.expectedURL, "Routes")
		}
	})

}

func assertEqual[T comparable](t *testing.T, actual, expected T, fieldName string) {
	if actual != expected {
		t.Errorf("Expected %s to be %v, but got %v", fieldName, expected, actual)
	}
}
