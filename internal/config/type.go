package config

import "golang.org/x/time/rate"

type RouteConfig map[string]string

type DomainConfig []string

type RateLimitConfig struct {
	Burst int        `yaml:"burst"`
	Rate  rate.Limit `yaml:"rate"`
}
