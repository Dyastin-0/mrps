package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var Routes RouteConfig
var Domains DomainConfig
var RateLimit RateLimitConfig

func Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()

	configData := struct {
		Routes    RouteConfig     `yaml:"routes"`
		Domains   DomainConfig    `yaml:"domains"`
		RateLimit RateLimitConfig `yaml:"rate_limit"`
	}{}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&configData); err != nil {
		return fmt.Errorf("could not decode YAML: %v", err)
	}

	Routes = configData.Routes
	Domains = configData.Domains
	RateLimit = configData.RateLimit

	return nil
}
