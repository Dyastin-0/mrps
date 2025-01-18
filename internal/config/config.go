package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var Routes RouteConfig
var Domains DomainConfig

func Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()

	configData := struct {
		Routes  RouteConfig  `yaml:"routes"`
		Domains DomainConfig `yaml:"domains"`
	}{}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&configData); err != nil {
		return fmt.Errorf("could not decode YAML: %v", err)
	}

	Routes = configData.Routes
	Domains = configData.Domains

	return nil
}
