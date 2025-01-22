package config

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
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
	return regexp.MustCompile(`^([a-zA-Z0-9]+(-[a-zA-Z0-9]+)*\.)+[a-zA-Z0-9]{2,}$`).MatchString(domain)
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

		//Sort the routes based on the length of the path, essentially by part depth
		//This is to ensure that the most specific routes are checked first
		//Since map iteration order is random
		sortedRoutes := make([]string, 0, len(cfg.Routes))
		for route := range cfg.Routes {
			if !isValidPath(route) {
				return fmt.Errorf("invalid path: %s", route)
			}
			sortedRoutes = append(sortedRoutes, route)
		}

		sort.Slice(sortedRoutes, func(i, j int) bool {
			return len(strings.Split(sortedRoutes[i], "/")) > len(strings.Split(sortedRoutes[j], "/"))
		})

		cfg.SortedRoutes = sortedRoutes

		sortedConfig := make(RouteConfig)
		for _, route := range sortedRoutes {
			sortedConfig[route] = cfg.Routes[route]
		}

		cfg.Routes = sortedConfig

		Domains = append(Domains, domain)
		cfg.RateLimit.Cooldown *= time.Millisecond
		cfg.RateLimit.DefaultCooldown *= Cooldowns.DefaultWaitTime

		Routes[domain] = cfg
		Cooldowns.DomainMutex[domain] = &sync.Mutex{}
	}

	return nil
}
