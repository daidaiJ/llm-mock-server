package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for the mock server.
type Config struct {
	Providers []ProviderConfig `yaml:"providers"`
}

// ProviderConfig represents a single provider's configuration.
type ProviderConfig struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`
}

// DefaultProviders returns the default set of enabled providers.
func DefaultProviders() []string {
	return []string{"deepseek", "openai", "anthropic"}
}

// DefaultConfig returns a Config with the default providers enabled.
func DefaultConfig() *Config {
	defaults := DefaultProviders()
	providers := []ProviderConfig{}
	for _, name := range defaults {
		providers = append(providers, ProviderConfig{Name: name, Enabled: true})
	}
	return &Config{Providers: providers}
}

// LoadConfig reads and parses a YAML config file.
// If the file does not exist, it returns the default config.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	return cfg, nil
}

// EnabledProviders returns the list of enabled provider names from the config.
func (c *Config) EnabledProviders() []string {
	var enabled []string
	for _, p := range c.Providers {
		if p.Enabled {
			enabled = append(enabled, p.Name)
		}
	}
	return enabled
}

// IsProviderEnabled checks if a specific provider is enabled in the config.
func (c *Config) IsProviderEnabled(name string) bool {
	for _, p := range c.Providers {
		if p.Name == name {
			return p.Enabled
		}
	}
	return false
}
