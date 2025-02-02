package common

import (
	"strings"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"golang.org/x/time/rate"
)

type DomainsConfig map[string]Config

type ClientLimiter struct {
	Limiter  *rate.Limiter
	LastReq  time.Time
	Cooldown time.Time
}

type Config struct {
	Enabled      bool            `yaml:"enabled"`
	Routes       RouteConfig     `yaml:"routes"`
	SortedRoutes []string        `yaml:"-"`
	RateLimit    RateLimitConfig `yaml:"rate_limit"`
}

type RouteConfig map[string]PathConfig

type PathConfig struct {
	Dest        string               `yaml:"dest"`
	RewriteRule rewriter.RewriteRule `yaml:"rewrite"`
}

type RateLimitConfig struct {
	Burst           int           `yaml:"burst"`
	Rate            rate.Limit    `yaml:"rate"`
	Cooldown        int64         `yaml:"cooldown"`
	DefaultCooldown time.Duration `yaml:"-"`
}

type MiscConfig struct {
	Email          string   `yaml:"email"`
	MetricsPort    string   `yaml:"metrics_port"`
	ConfigAPIPort  string   `yaml:"config_api_port"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	Domain         string   `yaml:"domain"`
}

type TrieNode struct {
	Children   map[string]*TrieNode
	Config     *Config
	IsWildcard bool
}

type DomainTrieConfig struct {
	Root *TrieNode
	mu   sync.RWMutex
}

func NewDomainTrie() *DomainTrieConfig {
	return &DomainTrieConfig{
		Root: &TrieNode{
			Children: make(map[string]*TrieNode),
		},
	}
}

type YAML struct {
	Domains   DomainsConfig   `yaml:"domains"`
	Misc      MiscConfig      `yaml:"misc"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

func (t *DomainTrieConfig) Insert(domain string, config *Config) {
	parts := strings.Split(domain, ".")
	node := t.Root

	// Traverse the trie in reverse
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]

		// Handle wildcard nodes
		if part == "*" {
			if _, exists := node.Children["*"]; !exists {
				t.mu.Lock()
				node.Children["*"] = &TrieNode{
					Children:   make(map[string]*TrieNode),
					IsWildcard: true,
				}
				t.mu.Unlock()
			}
			node = node.Children["*"]
		} else {
			if _, exists := node.Children[part]; !exists {
				node.Children[part] = &TrieNode{
					Children: make(map[string]*TrieNode),
				}
			}
			node = node.Children[part]
		}
	}

	// Assign the configuration at the final node, exact math
	t.mu.Lock()
	defer t.mu.Unlock()
	node.Config = config
}

func (t *DomainTrieConfig) Match(domain string) *Config {
	parts := strings.Split(domain, ".")
	node := t.Root

	// Traverse the trie in reverse
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]

		// Exact match
		if childNode, exists := node.Children[part]; exists {
			node = childNode
			continue
		}

		// Wildcard match
		if wildcardNode, exists := node.Children["*"]; exists {
			node = wildcardNode
			continue
		}

		return nil
	}

	return node.Config
}

func (t *DomainTrieConfig) Remove(domain string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	parts := strings.Split(domain, ".")
	return t.remove(t.Root, parts, len(parts)-1)
}

func (t *DomainTrieConfig) remove(node *TrieNode, parts []string, idx int) bool {
	if idx < 0 {
		if node.Config == nil {
			return false
		}
		node.Config = nil

		return len(node.Children) == 0
	}

	part := parts[idx]
	childNode, exists := node.Children[part]

	if !exists {
		return false
	}

	shouldDeleteChild := t.remove(childNode, parts, idx-1)

	if shouldDeleteChild {
		delete(node.Children, part)
	}

	return len(node.Children) == 0 && node.Config == nil
}

func (t *DomainTrieConfig) GetAll() DomainsConfig {
	result := DomainsConfig{}
	var traverse func(node *TrieNode, path []string)

	traverse = func(node *TrieNode, path []string) {
		if node.Config != nil {
			key := strings.Join(reverseSlice(path), ".")
			result[key] = *node.Config
		}
		for part, child := range node.Children {
			traverse(child, append(path, part))
		}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	traverse(t.Root, []string{})
	return result
}

func reverseSlice(slice []string) []string {
	reversed := make([]string, len(slice))
	for i, v := range slice {
		reversed[len(slice)-1-i] = v
	}
	return reversed
}

func (t *DomainTrieConfig) SetEnabled(domain string, enabled bool) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	modified := false

	config := t.Match(domain)
	if config != nil {
		modified = config.Enabled != enabled
		config.Enabled = enabled
	}

	return modified
}
