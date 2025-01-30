package config

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	Domains         []string
	DomainTrie      *DomainTrieConfig
	ClientMngr      = sync.Map{}
	GlobalRateLimit RateLimitConfig
	Misc            MiscConfig
)

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

	// Assign the configuration at the final node, exact math
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

func Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()

	DomainTrie = NewDomainTrie()

	configData := YAML{}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&configData); err != nil {
		return fmt.Errorf("could not decode YAML: %v", err)
	}

	if !isValidEmail(configData.Misc.Email) {
		return fmt.Errorf("invalid email: %s", configData.Misc.Email)
	}

	Misc = configData.Misc
	if Misc.MetricsPort == "" {
		Misc.MetricsPort = "7070"
	}
	if Misc.ConfigAPIPort == "" {
		Misc.ConfigAPIPort = "6060"
	}

	GlobalRateLimit = configData.RateLimit

	for domain, cfg := range configData.Domains {
		if !isValidDomain(domain) {
			return fmt.Errorf("invalid domain: %s", domain)
		}
		if strings.Contains(domain, "*") && strings.Index(domain, "*") != 0 {
			return fmt.Errorf("wildcard must be at the end of the domain: %s", domain)
		}

		Domains = append(Domains, domain)

		//This slice is used to access the Routes sequentially based on the number of path segments
		sortedRoutes := make([]string, 0, len(cfg.Routes))
		for route := range cfg.Routes {
			if !isValidPath(route) {
				log.Printf("Invalid path: %s", route)
				return fmt.Errorf("invalid path: %s", route)
			}

			sortedRoutes = append(sortedRoutes, route)
		}

		// Sort the routes by the number of "/" in each route
		sort.Slice(sortedRoutes, func(i, j int) bool {
			countI := strings.Count(sortedRoutes[i], "/")
			countJ := strings.Count(sortedRoutes[j], "/")

			return countI < countJ
		})

		cfg.SortedRoutes = sortedRoutes

		sortedConfig := make(RouteConfig)
		for _, route := range sortedRoutes {
			sortedConfig[route] = cfg.Routes[route]
		}

		cfg.RateLimit.DefaultCooldown = time.Second

		DomainTrie.Insert(domain, &cfg)
	}
	return nil
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

func ParseToYAML() {
	config := YAML{
		Domains:   DomainTrie.GetAll(),
		Misc:      Misc,
		RateLimit: GlobalRateLimit,
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatalf("error marshalling YAML: %v", err)
	}

	err = os.WriteFile("mrps.yaml", data, 0644)
	if err != nil {
		log.Fatalf("error writing to file: %v", err)
	}
}

func reverseSlice(slice []string) []string {
	reversed := make([]string, len(slice))
	for i, v := range slice {
		reversed[len(slice)-1-i] = v
	}
	return reversed
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
