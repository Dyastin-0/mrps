package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

var Email string
var Routes RoutesConfig
var Domains []string
var GlobalRateLimit RateLimitConfig
var Clients = make(map[string]map[string]*Client)
var Cooldowns = CoolDownConfig{
	MU:              &sync.Mutex{},
	DefaultWaitTime: 1 * time.Minute,
	DomainMutex:     make(map[string]*sync.Mutex),
	Client:          make(map[string]map[string]time.Time),
}

func Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()

	configData := struct {
		Routes    RoutesConfig    `yaml:"routes"`
		Email     string          `yaml:"email"`
		RateLimit RateLimitConfig `yaml:"rate_limit"`
	}{}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&configData); err != nil {
		return fmt.Errorf("could not decode YAML: %v", err)
	}

	Routes = configData.Routes
	Email = configData.Email
	GlobalRateLimit = configData.RateLimit

	GlobalRateLimit.Cooldown *= time.Millisecond
	for domain, cfg := range Routes {
		Domains = append(Domains, domain)
		cfg.RateLimit.Cooldown *= time.Millisecond
		cfg.RateLimit.DefaultCooldown *= Cooldowns.DefaultWaitTime
		Routes[domain] = cfg

		Cooldowns.DomainMutex[domain] = &sync.Mutex{}
	}

	return nil
}
