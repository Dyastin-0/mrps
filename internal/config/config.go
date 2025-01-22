package config

import (
	"fmt"
	"os"
	"regexp"
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

func isValidDomain(domain string) bool {
	return regexp.MustCompile(`^([a-zA-Z0-9]+(-[a-zA-Z0-9]+)*\.)+[a-zA-Z]{2,}$`).MatchString(domain)
}

func isValidEmail(email string) bool {
	return regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`).MatchString(email)
}

func isValidPath(path string) bool {
	return regexp.MustCompile(`^\/([a-zA-Z0-9\-._~]+(?:\/[a-zA-Z0-9\-._~]+)*)?\/?$`).MatchString(path)
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

	if !isValidEmail(configData.Email) {
		return fmt.Errorf("invalid email: %s", configData.Email)
	}

	Routes = configData.Routes
	Email = configData.Email
	GlobalRateLimit = configData.RateLimit

	GlobalRateLimit.Cooldown *= time.Millisecond
	for domain, cfg := range Routes {
		if !isValidDomain(domain) {
			return fmt.Errorf("invalid domain: %s", domain)
		}

		for route := range cfg.Routes {
			if !isValidPath(route) {
				return fmt.Errorf("invalid path: %s", route)
			}
		}

		Domains = append(Domains, domain)
		cfg.RateLimit.Cooldown *= time.Millisecond
		cfg.RateLimit.DefaultCooldown *= Cooldowns.DefaultWaitTime
		cfg.MU = &sync.Mutex{}

		Routes[domain] = cfg

		Cooldowns.DomainMutex[domain] = &sync.Mutex{}
	}

	return nil
}
