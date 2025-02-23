package config

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"golang.org/x/time/rate"
)

func TestLoadComplexConfig(t *testing.T) {
	testYAML := `
domains:
  dyastin.tech:
    enabled: true
    routes:
      /:
        dests:
        - url: http://localhost:4002
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
  filespace.dyastin.tech:
    enabled: true
    routes:
      /:
        dests:
        - url: http://localhost:5005
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
      /api/v2:
        dests:
        - url: http://localhost:3004
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
  filmpin.dyastin.tech:
    enabled: false
    routes:
      /:
        dests:
        - url: http://localhost:5002
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
      /api:
        dests:
        - url: http://localhost:5001
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
      /socket.io:
        dests:
        - url: http://localhost:5001
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
    rate_limit:
      burst: 100
      rate: 50
      cooldown: 60000
  gitsense.dyastin.tech:
    enabled: true
    routes:
      /:
        dests:
        - url: http://localhost:4001
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
      /api/v1:
        dests:
        - url: http://localhost:4000
        rewrite:
          type: regex
          value: ^/api/v1/(.*)$
          replace_val: /$1
        balancer: ""
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
  metrics.dyastin.tech:
    enabled: false
    routes:
      /:
        dests:
        - url: http://localhost:3000
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
    rate_limit:
      burst: 100
      rate: 50
      cooldown: 60000
  mrps.dyastin.tech:
    enabled: true
    routes:
      /:
        dests:
        - url: http://localhost:5050
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
      /api:
        dests:
        - url: http://localhost:6060
        rewrite:
          type: regex
          value: ^/api/(.*)$
          replace_val: /$1
        balancer: ""
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
  omnisense.dyastin.tech:
    enabled: true
    routes:
      /:
        dests:
        - url: http://localhost:4004
        rewrite:
          type: ""
          value: ""
          replace_val: ""
        balancer: ""
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
  sandbox.dyastin.tech:
    enabled: true
    routes:
      /free-wall:
        dests:
        - url: http://localhost:9001
        rewrite:
          type: regex
          value: ^/free-wall/(.*)$
          replace_val: /$1
        balancer: ""
      /free-wall/api:
        dests:
        - url: http://localhost:5000
        rewrite:
          type: regex
          value: ^/free-wall/api/(.*)$
          replace_val: /$1
        balancer: ""
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
http:
  routes:
    /:
      dests:
      - url: http://localhost:9001
misc:
  email: mail@dyastin.tech
  enable_metrics: true
  metrics_port: "7070"
  enable_api: true
  api_port: "6060"
  allowed_origins:
  - https://mrps.dyastin.tech
  - http://localhost:5173
  domain: .mrps.dyastin.tech
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

	if err := Load(context.Background(), tmpFile.Name()); err != nil {
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
			dest := routeConfig.Routes[test.path].Dests[0].URL
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
			rewriteRule := routeConfig.Routes[test.path].RewriteRule
			assertEqual(t, rewriteRule, test.expectedRewrite, "Rewrite Rules")
		}
	})
}

func assertEqual(t *testing.T, got, want interface{}, name string) {
	t.Helper()
	if got != want {
		t.Errorf("%s mismatch: got %v, want %v", name, got, want)
	}
}
