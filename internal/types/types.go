package types

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"golang.org/x/time/rate"
)

type DomainsConfig map[string]Config

type ClientLimiter struct {
	Limiter  *rate.Limiter
	LastReq  time.Time
	Cooldown time.Time
}

const (
	HTTPProtocol  = "http"
	HTTPSProtocol = "https"
)

type Config struct {
	Enabled      bool            `yaml:"enabled"`
	Routes       RouteConfig     `yaml:"routes,omitempty"`
	SortedRoutes []string        `yaml:"-"`
	RateLimit    RateLimitConfig `yaml:"rate_limit,omitempty"`
	Protocol     string          `yaml:"protocol,omitempty"`
}

type RouteConfig map[string]PathConfig

type PathConfig struct {
	Dests        []Dest               `json:"Dests,omitempty" yaml:"dests,omitempty"`
	RewriteRule  rewriter.RewriteRule `yaml:"rewrite,omitempty"`
	BalancerType string               `yaml:"balancer,omitempty"`
	Balancer     interface {
		GetDests() []*common.Dest
		Serve(w http.ResponseWriter, r *http.Request, retries int) bool
		First() *common.Dest
		StopHealthChecks()
	} `yaml:"-"`
}

type Dest struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight,omitempty"`
}

type RateLimitConfig struct {
	Burst           int           `yaml:"burst,omitempty"`
	Rate            rate.Limit    `yaml:"rate,omitempty"`
	Cooldown        int64         `yaml:"cooldown,omitempty"`
	DefaultCooldown time.Duration `yaml:"-"`
}

type MiscConfig struct {
	Email          string   `yaml:"email,omitempty"`
	Secure         bool     `yaml:"secure"`
	MetricsEnabled bool     `yaml:"enable_metrics"`
	MetricsPort    string   `yaml:"metrics_port,omitempty"`
	APIEnabled     bool     `yaml:"enable_api,omitempty"`
	ConfigAPIPort  string   `yaml:"api_port,omitempty"`
	AllowedOrigins []string `yaml:"allowed_origins,omitempty"`
	Domain         string   `yaml:"domain,omitempty"`
	IP             string   `yaml:"ip,omitempty"`
	AllowHTTP      bool     `yaml:"allow_http"`
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
	Domains   DomainsConfig   `yaml:"domains,omitempty"`
	Misc      MiscConfig      `yaml:"misc,omitempty"`
	RateLimit RateLimitConfig `yaml:"rate_limit,omitempty"`
	HTTP      Config          `yaml:"http,omitempty"`
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

	// Assign the configuration at the final node, exact match
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

func (t *DomainTrieConfig) GetHealth() map[string]map[string]bool {
	healthStatus := make(map[string]map[string]bool)

	var traverse func(node *TrieNode, path []string)
	traverse = func(node *TrieNode, path []string) {
		if node.Config != nil && node.Config.Routes != nil {
			domain := strings.Join(reverseSlice(path), ".")
			healthStatus[domain] = make(map[string]bool)

			for _, routeConfig := range node.Config.Routes {
				if routeConfig.Balancer == nil {
					continue
				}

				dests := routeConfig.Balancer.GetDests()
				for _, dest := range dests {
					healthStatus[domain][dest.URL] = dest.Alive
				}
			}
		}

		for part, child := range node.Children {
			traverse(child, append(path, part))
		}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	traverse(t.Root, []string{})
	return healthStatus
}

func (t *DomainTrieConfig) StopHealthChecks() {
	t.mu.Lock()
	defer t.mu.Unlock()

	var traverse func(node *TrieNode, path []string)
	traverse = func(node *TrieNode, path []string) {
		if node.Config != nil {
			for _, config := range node.Config.Routes {
				config.Balancer.StopHealthChecks()
			}
		}
		for part, child := range node.Children {
			traverse(child, append(path, part))
		}
	}

	traverse(t.Root, []string{})
}
