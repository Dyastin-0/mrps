package config

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type EmailConfig string
type RoutesConfig map[string]Config
type RouteConfig map[string]string
type Config struct {
	Routes       RouteConfig `yaml:"routes"`
	SortedRoutes []string
	RateLimit    RateLimitConfig `yaml:"rate_limit"`
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

type Client struct {
	Limiter     *rate.Limiter
	LastRequest time.Time
}
