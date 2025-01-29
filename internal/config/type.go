package config

import (
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
	Enabled      bool        `yaml:"enabled"`
	Routes       RouteConfig `yaml:"routes"`
	SortedRoutes []string
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
	Cooldown        time.Duration `yaml:"cooldown"`
	DefaultCooldown time.Duration
}

type MiscConfig struct {
	Email          string   `yaml:"email"`
	MetricsPort    string   `yaml:"metrics_port"`
	ConfigAPIPort  string   `yaml:"config_api_port"`
	AllowedOrigins []string `yaml:"allowed_origins"`
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
