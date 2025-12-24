package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config represents the complete TSLP configuration
// Per spec: minimal and boring, no tuning knobs
type Config struct {
	Database DatabaseConfig `toml:"database"`
	Logging  LoggingConfig  `toml:"logging"`
	LLM      LLMConfig      `toml:"llm"`
	Proxy    ProxyConfig    `toml:"proxy"`
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	DatabasePath string `toml:"database_path"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	LogPath string `toml:"log_path"`
	Debug   bool   `toml:"debug"`
}

// LLMConfig holds LLM provider settings
type LLMConfig struct {
	Provider string `toml:"llm_provider"`
	Endpoint string `toml:"llm_endpoint"`
	APIKey   string `toml:"llm_api_key"`
	Model    string `toml:"llm_model"`
}

// ProxyConfig holds proxy server settings
type ProxyConfig struct {
	ListenAddress string `toml:"listen_address"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks that all required fields are present
func (c *Config) Validate() error {
	if c.Database.DatabasePath == "" {
		return fmt.Errorf("database_path is required")
	}
	if c.Logging.LogPath == "" {
		return fmt.Errorf("log_path is required")
	}
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm_provider is required")
	}
	if c.LLM.Endpoint == "" {
		return fmt.Errorf("llm_endpoint is required")
	}
	if c.LLM.Model == "" {
		return fmt.Errorf("llm_model is required")
	}
	if c.Proxy.ListenAddress == "" {
		return fmt.Errorf("listen_address is required")
	}

	// Note: llm_api_key can be empty for local models

	return nil
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		Database: DatabaseConfig{
			DatabasePath: "./runtime/tslp.db",
		},
		Logging: LoggingConfig{
			LogPath: "./runtime/tslp.log",
			Debug:   false,
		},
		LLM: LLMConfig{
			Provider: "openai",
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "",
			Model:    "gpt-4",
		},
		Proxy: ProxyConfig{
			ListenAddress: "127.0.0.1:8080",
		},
	}
}
