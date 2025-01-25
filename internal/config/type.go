package config

import (
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"golang.org/x/time/rate"
)

type DomainsConfig map[string]Config

type Config struct {
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

type CoolDownConfig struct {
	MU              *sync.Mutex
	DomainMutex     map[string]*sync.Mutex
	Client          map[string]map[string]time.Time
	DefaultWaitTime time.Duration
}

type MiscConfig struct {
	Email       string `yaml:"email"`
	MetricsPort string `yaml:"metrics_port"`
}

type Client struct {
	Limiter     *rate.Limiter
	LastRequest time.Time
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
