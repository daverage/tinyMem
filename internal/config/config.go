package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultProxyPort         = 8080
	DefaultExtractionBufferBytes = 16384  // 16KB default
)

// Config holds the application configuration
type Config struct {
	ProxyPort              int    `toml:"proxy_port"`
	SearchEnabled          bool   `toml:"search_enabled"`
	Streaming              bool   `toml:"streaming"`
	LogLevel               string `toml:"log_level"`
	DBPath                 string `toml:"db_path"`
	ConfigPath             string `toml:"config_path"`
	TinyMemDir             string `toml:"tiny_mem_dir"`
	ExtractionBufferBytes  int    `toml:"extraction_buffer_bytes"`
}

// LoadConfig loads configuration from file, environment variables, and defaults
func LoadConfig() (*Config, error) {
	projectRoot, err := FindProjectRoot()
	if err != nil {
		return nil, err
	}

	tinyMemDir := GetTinyMemDir(projectRoot)
	configPath := filepath.Join(tinyMemDir, "config.toml")

	// Ensure .tinyMem directories exist
	if err := EnsureTinyMemDirs(tinyMemDir); err != nil {
		return nil, err
	}

	// Create default config
	cfg := &Config{
		ProxyPort:              DefaultProxyPort,
		SearchEnabled:          true,
		Streaming:              true,
		LogLevel:               "info",
		DBPath:                 filepath.Join(tinyMemDir, "store.sqlite3"),
		ConfigPath:             configPath,
		TinyMemDir:             tinyMemDir,
		ExtractionBufferBytes:  DefaultExtractionBufferBytes,
	}

	// Try to load config from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		fileData, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		var fileConfig Config
		if err := toml.Unmarshal(fileData, &fileConfig); err != nil {
			return nil, err
		}

		// Override defaults with file config values
		if fileConfig.ProxyPort != 0 {
			cfg.ProxyPort = fileConfig.ProxyPort
		}
		if fileConfig.LogLevel != "" {
			cfg.LogLevel = fileConfig.LogLevel
		}
		if fileConfig.ExtractionBufferBytes != 0 {
			cfg.ExtractionBufferBytes = fileConfig.ExtractionBufferBytes
		}
		// Note: We don't override TinyMemDir, DBPath, etc. as they're derived from project root
	}

	// Apply environment variable overrides
	if port := os.Getenv("TINYMEM_PROXY_PORT"); port != "" {
		if _, err := fmt.Sscanf(port, "%d", &cfg.ProxyPort); err == nil {
			// Valid integer, use it
		}
	}

	if level := os.Getenv("TINYMEM_LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	if search := os.Getenv("TINYMEM_SEARCH_ENABLED"); search != "" {
		if search == "true" || search == "1" {
			cfg.SearchEnabled = true
		} else {
			cfg.SearchEnabled = false
		}
	}

	if streaming := os.Getenv("TINYMEM_STREAMING"); streaming != "" {
		if streaming == "true" || streaming == "1" {
			cfg.Streaming = true
		} else {
			cfg.Streaming = false
		}
	}

	if extractionBufSize := os.Getenv("TINYMEM_EXTRACTION_BUFFER_BYTES"); extractionBufSize != "" {
		if size, err := strconv.Atoi(extractionBufSize); err == nil {
			cfg.ExtractionBufferBytes = size
		}
	}

	return cfg, nil
}

	return nil
}

// GenerateProjectID creates a consistent project ID from the project root path.
// It uses the absolute path to ensure uniqueness across different working directories
// but normalizes it to be filesystem-agnostic for consistency (e.g., replaces backslashes with forward slashes).
func GenerateProjectID(projectRoot string) string {
	// Clean the path to remove redundant separators and symlinks
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		// If we can't get the absolute path, fall back to the provided root
		// This might lead to non-unique IDs in some edge cases, but it's better than failing
		return strings.ReplaceAll(filepath.Clean(projectRoot), "\\", "/")
	}
	// Normalize path separators for cross-OS consistency
	return strings.ReplaceAll(filepath.Clean(absPath), "\\", "/")
}

// Context key for storing config in context
type configContextKey struct{}

// WithConfig adds the config to the context
func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configContextKey{}, cfg)
}

// FromContext retrieves the config from the context
func FromContext(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(configContextKey{}).(*Config); ok {
		return cfg
	}
	return nil
}