package config

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/loadbalancer"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/internal/watcher"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

var (
	Domains         []string
	HTTP            types.HTTP
	DomainTrie      *types.DomainTrieConfig
	ClientMngr      = sync.Map{}
	GlobalRateLimit types.RateLimitConfig
	Misc            types.MiscConfig
	StartTime       time.Time
)

func Load(ctx context.Context, filename string) error {
	// Reset all global variables
	Domains = nil
	HTTP = types.HTTP{}
	DomainTrie = types.NewDomainTrie()
	GlobalRateLimit = types.RateLimitConfig{}
	Misc = types.MiscConfig{}
	StartTime = time.Now()

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()

	DomainTrie = types.NewDomainTrie()

	configData := types.YAML{}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&configData); err != nil {
		return fmt.Errorf("could not decode YAML: %v", err)
	}

	if configData.Misc.Email != "" && !regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`).MatchString(configData.Misc.Email) {
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
		if !regexp.MustCompile(`^([a-zA-Z0-9\*]+(-[a-zA-Z0-9\*]+)*\.)+[a-zA-Z0-9]{2,}$`).MatchString(domain) {
			return fmt.Errorf("invalid domain: %s", domain)
		}
		if strings.Contains(domain, "*") && strings.Index(domain, "*") != 0 {
			return fmt.Errorf("wildcard must be at the end of the domain: %s", domain)
		}

		Domains = append(Domains, domain)

		//This slice is used to access the Routes sequentially based on the number of path segments
		sortedRoutes := make([]string, 0, len(cfg.Routes))
		for path, config := range cfg.Routes {
			if !regexp.MustCompile(`^\/([a-zA-Z0-9\-._~]+(?:\/[a-zA-Z0-9\-._~]+)*)?\/?$`).MatchString(path) {
				return fmt.Errorf("invalid path: %s", path)
			}

			config.Balancer, err = loadbalancer.New(ctx, config.Dests, config.RewriteRule, config.BalancerType, path, domain)
			if err != nil {
				log.Fatal().Err(err).Msg("config")
			}
			cfg.Routes[path] = config
			sortedRoutes = append(sortedRoutes, path)
		}

		// Sort the routes by the number of "/" and then by string length
		sort.Slice(sortedRoutes, func(i, j int) bool {
			countI := strings.Count(sortedRoutes[i], "/")
			countJ := strings.Count(sortedRoutes[j], "/")

			// First sort by the number of "/"
			if countI != countJ {
				return countI > countJ
			}

			// If they have the same number of "/", sort by string length
			return len(sortedRoutes[i]) > len(sortedRoutes[j])
		})

		cfg.SortedRoutes = sortedRoutes

		sortedConfig := make(types.RouteConfig)
		for _, route := range sortedRoutes {
			sortedConfig[route] = cfg.Routes[route]
		}

		cfg.RateLimit.DefaultCooldown = time.Second

		DomainTrie.Insert(domain, &cfg)
	}

	if configData.HTTP.Routes != nil {
		HTTP = configData.HTTP
		sortedRoutes := make([]string, 0, len(configData.HTTP.Routes))

		for path, config := range configData.HTTP.Routes {
			if !regexp.MustCompile(`^\/([a-zA-Z0-9\-._~]+(?:\/[a-zA-Z0-9\-._~]+)*)?\/?$`).MatchString(path) {
				return fmt.Errorf("invalid path: %s", path)
			}

			config.Balancer, err = loadbalancer.New(ctx, config.Dests, config.RewriteRule, config.BalancerType, path, "http")
			if err != nil {
				log.Fatal().Err(err).Msg("config")
			}
			configData.HTTP.Routes[path] = config
			sortedRoutes = append(sortedRoutes, path)
		}

		sort.Slice(sortedRoutes, func(i, j int) bool {
			countI := strings.Count(sortedRoutes[i], "/")
			countJ := strings.Count(sortedRoutes[j], "/")

			if countI != countJ {
				return countI > countJ
			}

			return len(sortedRoutes[i]) > len(sortedRoutes[j])
		})

		HTTP.SortedRoutes = sortedRoutes
	}

	return nil
}

func ParseToYAML() {
	config := types.YAML{
		Domains:   DomainTrie.GetAll(),
		Misc:      Misc,
		RateLimit: GlobalRateLimit,
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatal().Err(err).Msg("config")
	}

	err = os.WriteFile("mrps.yaml", data, 0644)
	if err != nil {
		log.Fatal().Err(err).Msg("config")
	}
}

func Watch(ctx context.Context, path string) {
	watcher.Watch(ctx, path, func() {
		if err := Load(ctx, path); err != nil {
			log.Error().Err(fmt.Errorf("failed to reload")).Msg("config")
		} else {
			log.Info().Str("status", "reloaded").Str("path", path).Msg("config")
		}
	})
}
