package config

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RouteConfig map[string]string

type DomainConfig []string

type RateLimitConfig struct {
	Burst int        `yaml:"burst"`
	Rate  rate.Limit `yaml:"rate"`
}

type Client struct {
	Limiter     *rate.Limiter
	LastRequest time.Time
}

type CoolDownConfig struct {
	MU              sync.Mutex
	DefaultWaitTime time.Duration
	Client          map[string]time.Time
}
