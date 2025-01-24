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

var Domains []string
var DomainTrie = NewDomainTrie()
var GlobalRateLimit RateLimitConfig
var Clients = make(map[string]map[string]*Client)
var Cooldowns = CoolDownConfig{
	MU:              &sync.Mutex{},
	DefaultWaitTime: 1 * time.Minute,
	DomainMutex:     make(map[string]*sync.Mutex),
	Client:          make(map[string]map[string]time.Time),
}
var Misc MiscConfig

func (t *DomainTrieConfig) Insert(domain string, config *Config) {
	parts := strings.Split(domain, ".")
	node := t.Root

	// Traverse the trie in reverse
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]

		// Handle wildcard nodes
		if part == "*" {
			if _, exists := node.Children["*"]; !exists {
				fmt.Println("Wildcard")
				t.mu.Lock()
				node.Children["*"] = &TrieNode{
					Children:   make(map[string]*TrieNode),
					IsWildcard: true,
					//Wild cards have access to the config
					Config: config,
				}
				t.mu.Unlock()
			}
			node = node.Children["*"]
		} else {
			if _, exists := node.Children[part]; !exists {
				node.Children[part] = &TrieNode{
					//Any non-exact match node does not have access to the config
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

func (t *DomainTrieConfig) Match(domain string) **Config {
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

	return &node.Config
}

func Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()

	configData := struct {
		Routes    RoutesConfig    `yaml:"routes"`
		Misc      MiscConfig      `yaml:"misc"`
		RateLimit RateLimitConfig `yaml:"rate_limit"`
	}{}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&configData); err != nil {
		return fmt.Errorf("could not decode YAML: %v", err)
	}

	if !isValidEmail(configData.Misc.Email) {
		return fmt.Errorf("invalid email: %s", configData.Misc.Email)
	}

	Misc = configData.Misc
	GlobalRateLimit = configData.RateLimit

	GlobalRateLimit.Cooldown *= time.Millisecond

	for domain, cfg := range configData.Routes {
		if !isValidDomain(domain) {
			return fmt.Errorf("invalid domain: %s", domain)
		}

		Domains = append(Domains, domain)

		//This slice is used to access the Routes based on path depth
		sortedRoutes := make([]string, 0, len(cfg.Routes))
		for route := range cfg.Routes {
			if !isValidPath(route) {
				return fmt.Errorf("invalid path: %s", route)
			}
			sortedRoutes = append(sortedRoutes, route)
		}

		//Sort the routes by the number of path segments in descending order
		sort.Slice(sortedRoutes, func(i, j int) bool {
			iParts := strings.Split(sortedRoutes[i], "/")
			jParts := strings.Split(sortedRoutes[j], "/")

			if len(iParts) != len(jParts) {
				return len(iParts) > len(jParts)
			}
			return sortedRoutes[i] != "/" && sortedRoutes[j] == "/"
		})

		cfg.SortedRoutes = sortedRoutes

		sortedConfig := make(RouteConfig)
		for _, route := range sortedRoutes {
			sortedConfig[route] = cfg.Routes[route]
		}

		cfg.Routes = sortedConfig

		cfg.RateLimit.Cooldown *= time.Millisecond
		cfg.RateLimit.DefaultCooldown = Cooldowns.DefaultWaitTime

		Cooldowns.DomainMutex[domain] = &sync.Mutex{}

		DomainTrie.Insert(domain, &cfg)
	}

	return nil
}

func isValidDomain(domain string) bool {
	return regexp.MustCompile(`^([a-zA-Z0-9\*]+(-[a-zA-Z0-9\*]+)*\.)+[a-zA-Z0-9]{2,}$`).MatchString(domain)
}

func isValidEmail(email string) bool {
	return regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`).MatchString(email)
}

func isValidPath(path string) bool {
	return regexp.MustCompile(`^\/([a-zA-Z0-9\-._~]+(?:\/[a-zA-Z0-9\-._~]+)*)?\/?$`).MatchString(path)
}
