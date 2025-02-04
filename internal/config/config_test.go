package config

import (
	"os"
	"testing"
	"time"

	"github.com/Dyastin-0/mrps/internal/common"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"golang.org/x/time/rate"
)

func TestLoadComplexConfig(t *testing.T) {
	testYAML := `
domains:
  dyastin.tech:
    enabled: false
    routes:
      /:
        dest: http://localhost:4002
        rewrite:
          type: ""
          value: ""
          replace_val: ""
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
  filespace.dyastin.tech:
    enabled: false
    routes:
      /:
        dest: http://localhost:5005
        rewrite:
          type: ""
          value: ""
          replace_val: ""
      /api/v2:
        dest: http://localhost:3004
        rewrite:
          type: ""
          value: ""
          replace_val: ""
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
  filmpin.dyastin.tech:
    enabled: false
    routes:
      /:
        dest: http://localhost:5002
        rewrite:
          type: ""
          value: ""
          replace_val: ""
      /api:
        dest: http://localhost:5001
        rewrite:
          type: ""
          value: ""
          replace_val: ""
      /socket.io:
        dest: http://localhost:5001
        rewrite:
          type: ""
          value: ""
          replace_val: ""
    rate_limit:
      burst: 100
      rate: 50
      cooldown: 60000
  gitsense.dyastin.tech:
    enabled: false
    routes:
      /:
        dest: http://localhost:4001
        rewrite:
          type: ""
          value: ""
          replace_val: ""
      /api/v1:
        dest: http://localhost:4000
        rewrite:
          type: regex
          value: ^/api/v1/(.*)$
          replace_val: /$1
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
  metrics.dyastin.tech:
    enabled: false
    routes:
      /:
        dest: http://localhost:3000
        rewrite:
          type: ""
          value: ""
          replace_val: ""
    rate_limit:
      burst: 100
      rate: 50
      cooldown: 60000
  mrps.dyastin.tech:
    enabled: true
    routes:
      /:
        dest: http://localhost:5050
        rewrite:
          type: ""
          value: ""
          replace_val: ""
      /api:
        dest: http://localhost:6060
        rewrite:
          type: regex
          value: ^/api/(.*)$
          replace_val: /$1
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
  omnisense.dyastin.tech:
    enabled: false
    routes:
      /:
        dest: http://localhost:4004
        rewrite:
          type: ""
          value: ""
          replace_val: ""
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
misc:
  email: mail@dyastin.tech
  metrics_port: "7070"
  config_api_port: "6060"
  allowed_origins:
  - https://mrps.dyastin.tech
  - http://localhost:5173
rate_limit:
  burst: 100
  rate: 50
  cooldown: 60000
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

	if err := Load(tmpFile.Name()); err != nil {
		t.Fatalf("config.Load() failed: %v", err)
	}

	t.Run("Validate Domain Configurations", func(t *testing.T) {
		tests := []struct {
			domain       string
			path         string
			expectedDest string
		}{
			{"gitsense.dyastin.tech", "/api/v1", "http://localhost:4000"},
			{"gitsense.dyastin.tech", "/", "http://localhost:4001"},
			{"filespace.dyastin.tech", "/api/v2", "http://localhost:3004"},
			{"filespace.dyastin.tech", "/", "http://localhost:5005"},
			{"omnisense.dyastin.tech", "/", "http://localhost:4004"},
			{"filmpin.dyastin.tech", "/socket.io", "http://localhost:5001"},
			{"filmpin.dyastin.tech", "/api", "http://localhost:5001"},
			{"filmpin.dyastin.tech", "/", "http://localhost:5002"},
			{"metrics.dyastin.tech", "/", "http://localhost:3000"},
			{"dyastin.tech", "/", "http://localhost:4002"},
		}

		for _, test := range tests {
			routeConfigPtr := DomainTrie.Match(test.domain)
			if routeConfigPtr == nil {
				t.Fatalf("Domain not found in trie: %s", test.domain)
			}
			routeConfig := *routeConfigPtr
			dest := routeConfig.Routes[test.path].Dest
			assertEqual(t, dest, test.expectedDest, "Destinations")
		}
	})

	t.Run("Validate Rate Limits", func(t *testing.T) {
		tests := []struct {
			domain           string
			expectedBurst    int
			expectedRate     rate.Limit
			expectedCooldown time.Duration
		}{
			{"gitsense.dyastin.tech", 15, 10, 60000},
			{"filespace.dyastin.tech", 15, 10, 60000},
			{"omnisense.dyastin.tech", 15, 10, 60000},
			{"filmpin.dyastin.tech", 100, 50, 60000},
			{"metrics.dyastin.tech", 100, 50, 60000},
			{"dyastin.tech", 15, 10, 60000},
		}

		for _, test := range tests {
			routeConfigPtr := DomainTrie.Match(test.domain)
			if routeConfigPtr == nil {
				t.Fatalf("Domain not found in trie: %s", test.domain)
			}
			routeConfig := *routeConfigPtr
			assertEqual(t, routeConfig.RateLimit.Burst, test.expectedBurst, "Burst")
			assertEqual(t, routeConfig.RateLimit.Rate, test.expectedRate, "Rate")
			assertEqual(t, time.Duration(routeConfig.RateLimit.Cooldown), test.expectedCooldown, "Cooldown")
		}
	})

	t.Run("Validate Rewrite Rules", func(t *testing.T) {
		tests := []struct {
			domain          string
			path            string
			expectedRewrite rewriter.RewriteRule
		}{
			{"gitsense.dyastin.tech", "/api/v1", rewriter.RewriteRule{Type: "regex", Value: "^/api/v1/(.*)$", ReplaceVal: "/$1"}},
			{"gitsense.dyastin.tech", "/", rewriter.RewriteRule{Type: "", Value: "", ReplaceVal: ""}},
			{"filespace.dyastin.tech", "/api/v2", rewriter.RewriteRule{Type: "", Value: "", ReplaceVal: ""}},
			{"filespace.dyastin.tech", "/", rewriter.RewriteRule{Type: "", Value: "", ReplaceVal: ""}},
			{"omnisense.dyastin.tech", "/", rewriter.RewriteRule{Type: "", Value: "", ReplaceVal: ""}},
			{"filmpin.dyastin.tech", "/socket.io", rewriter.RewriteRule{Type: "", Value: "", ReplaceVal: ""}},
			{"filmpin.dyastin.tech", "/api", rewriter.RewriteRule{Type: "", Value: "", ReplaceVal: ""}},
			{"filmpin.dyastin.tech", "/", rewriter.RewriteRule{Type: "", Value: "", ReplaceVal: ""}},
			{"metrics.dyastin.tech", "/", rewriter.RewriteRule{Type: "", Value: "", ReplaceVal: ""}},
			{"dyastin.tech", "/", rewriter.RewriteRule{Type: "", Value: "", ReplaceVal: ""}},
		}

		for _, test := range tests {
			routeConfigPtr := DomainTrie.Match(test.domain)
			if routeConfigPtr == nil {
				t.Fatalf("Domain not found in trie: %s", test.domain)
			}
			routeConfig := *routeConfigPtr
			rewriteConfig := routeConfig.Routes[test.path].RewriteRule
			assertEqual(t, rewriteConfig.Type, test.expectedRewrite.Type, "Rewrite Type")
			assertEqual(t, rewriteConfig.Value, test.expectedRewrite.Value, "Rewrite Value")
			assertEqual(t, rewriteConfig.ReplaceVal, test.expectedRewrite.ReplaceVal, "Rewrite ReplaceVal")
		}
	})

}

func assertEqual[T comparable](t *testing.T, actual, expected T, fieldName string) {
	if actual != expected {
		t.Errorf("Expected %s to be %v, but got %v", fieldName, expected, actual)
	}
}

func TestDomainTrieRemove(t *testing.T) {
	trie := common.NewDomainTrie()

	trie.Insert("example.com", &common.Config{})
	trie.Insert("*.example.com", &common.Config{})

	trie.Remove("*.example.com")
	if config := trie.Match("sub.example.com"); config != nil {
		t.Error("'*.example.com' was not removed correctly")
	}

	if config := trie.Match("example.com"); config == nil {
		t.Error("'example.com' should still exist")
	}

}

func TestSetEnabled(t *testing.T) {
	DomainTrie = common.NewDomainTrie()

	testDomain := "example.com"
	DomainTrie.Insert(testDomain, &common.Config{Enabled: false})

	modified := DomainTrie.SetEnabled(testDomain, true)
	if !modified {
		t.Errorf("Expected SetEnabled to return true, got false")
	}

	config := DomainTrie.Match(testDomain)
	if config == nil || !config.Enabled {
		t.Errorf("Expected domain %s to be enabled, but got %v", testDomain, config)
	}

	modifiedAgain := DomainTrie.SetEnabled(testDomain, true)
	if modifiedAgain {
		t.Errorf("Expected SetEnabled to return false when setting same value")
	}

	configAfterNoChange := DomainTrie.Match(testDomain)
	if configAfterNoChange == nil || !configAfterNoChange.Enabled {
		t.Errorf("Expected domain %s to remain enabled, but got %v", testDomain, configAfterNoChange)
	}

	modifiedNonExistent := DomainTrie.SetEnabled("nonexistent.com", true)
	if modifiedNonExistent {
		t.Errorf("Expected SetEnabled to return false for a non-existent domain")
	}

	modifiedDisabled := DomainTrie.SetEnabled(testDomain, false)
	if !modifiedDisabled {
		t.Errorf("Expected SetEnabled to return true when disabling a domain")
	}

	configDisabled := DomainTrie.Match(testDomain)
	if configDisabled == nil || configDisabled.Enabled {
		t.Errorf("Expected domain %s to be disabled, but got %v", testDomain, configDisabled)
	}
}
