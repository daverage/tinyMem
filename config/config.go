package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/pelletier/go-toml/v2"
)

//go:embed config.schema.json
var schemaFS embed.FS

// Config represents the complete tinyMem configuration
// Per spec: minimal and boring, no tuning knobs, no feature flags
type Config struct {
	Database  DatabaseConfig  `toml:"database" json:"database"`
	Logging   LoggingConfig   `toml:"logging" json:"logging"`
	LLM       LLMConfig       `toml:"llm" json:"llm"`
	Proxy     ProxyConfig     `toml:"proxy" json:"proxy"`
	Hydration HydrationConfig `toml:"hydration" json:"hydration"`
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	DatabasePath string `toml:"database_path" json:"database_path"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	LogPath string `toml:"log_path" json:"log_path"`
	Debug   bool   `toml:"debug" json:"debug"`
}

// LLMConfig holds LLM provider settings
type LLMConfig struct {
	Provider string `toml:"llm_provider" json:"llm_provider"`
	Endpoint string `toml:"llm_endpoint" json:"llm_endpoint"`
	APIKey   string `toml:"llm_api_key" json:"llm_api_key"`
	Model    string `toml:"llm_model" json:"llm_model"`
}

// IsCLIProvider checks if the provider is a CLI-based provider
func (c *LLMConfig) IsCLIProvider() bool {
	// Known CLI providers
	cliProviders := []string{"claude", "gemini", "sgpt", "aichat"}
	for _, cli := range cliProviders {
		if c.Provider == cli {
			return true
		}
	}
	// Check if provider starts with "cli:"
	return len(c.Provider) > 4 && c.Provider[:4] == "cli:"
}

// ProxyConfig holds proxy server settings
type ProxyConfig struct {
	ListenAddress string `toml:"listen_address" json:"listen_address"`
}

// HydrationConfig holds hydration budget and retrieval settings
type HydrationConfig struct {
	// Budget settings
	MaxTokens   int `toml:"max_tokens" json:"max_tokens"`     // 0 = unlimited
	MaxEntities int `toml:"max_entities" json:"max_entities"` // 0 = unlimited

	// Structural anchors (always enabled for determinism)
	EnableFileMentionAnchors     bool `toml:"enable_file_mention_anchors" json:"enable_file_mention_anchors"`
	EnableSymbolMentionAnchors   bool `toml:"enable_symbol_mention_anchors" json:"enable_symbol_mention_anchors"`
	EnablePreviousHydrationAnchors bool `toml:"enable_previous_hydration_anchors" json:"enable_previous_hydration_anchors"`

	// Semantic ranking (optional)
	EnableSemanticRanking bool    `toml:"enable_semantic_ranking" json:"enable_semantic_ranking"`
	SemanticThreshold     float64 `toml:"semantic_threshold" json:"semantic_threshold"`           // Cosine similarity cutoff (e.g., 0.7)
	SemanticBudgetTokens  int     `toml:"semantic_budget_tokens" json:"semantic_budget_tokens"`   // Max tokens for semantic expansion
	SemanticBudgetEntities int     `toml:"semantic_budget_entities" json:"semantic_budget_entities"` // Max entities from semantic ranking

	// Embedding provider
	EmbeddingProvider string `toml:"embedding_provider" json:"embedding_provider"` // "openai", "local", "none"
	EmbeddingModel    string `toml:"embedding_model" json:"embedding_model"`
	EmbeddingCacheTTL int    `toml:"embedding_cache_ttl" json:"embedding_cache_ttl"` // Seconds
}

// Load reads and parses the configuration file
// Per spec: fail fast on missing or invalid config
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
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate against JSON schema
	if err := cfg.ValidateSchema(); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate checks that all required fields are present and valid
// Per spec: no defaults, fail fast
func (c *Config) Validate() error {
	// Database validation
	if c.Database.DatabasePath == "" {
		return fmt.Errorf("database.database_path is required")
	}

	// Logging validation
	if c.Logging.LogPath == "" {
		return fmt.Errorf("logging.log_path is required")
	}

	// LLM validation
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm.llm_provider is required")
	}

	// Check if it's a CLI provider
	isCLI := c.LLM.IsCLIProvider()

	if isCLI {
		// CLI providers don't need an endpoint
		// Endpoint can be empty or "cli" or "local"
		if c.LLM.Model == "" {
			return fmt.Errorf("llm.llm_model is required")
		}
	} else {
		// HTTP providers need an endpoint
		if c.LLM.Endpoint == "" {
			return fmt.Errorf("llm.llm_endpoint is required for HTTP providers")
		}
		if c.LLM.Model == "" {
			return fmt.Errorf("llm.llm_model is required")
		}

		// Validate endpoint format
		endpointPattern := regexp.MustCompile(`^https?://`)
		if !endpointPattern.MatchString(c.LLM.Endpoint) {
			return fmt.Errorf("llm.llm_endpoint must start with http:// or https://")
		}
	}

	// Proxy validation
	if c.Proxy.ListenAddress == "" {
		return fmt.Errorf("proxy.listen_address is required")
	}

	// Validate listen address format
	listenPattern := regexp.MustCompile(`^[0-9.]+:[0-9]+$`)
	if !listenPattern.MatchString(c.Proxy.ListenAddress) {
		return fmt.Errorf("proxy.listen_address must be in format 'host:port' (e.g., '127.0.0.1:8080')")
	}

	// Note: llm_api_key can be empty for local models (not validated)

	return nil
}

// ValidateSchema validates the configuration against the JSON schema
// Per spec: JSON schema validation on startup
func (c *Config) ValidateSchema() error {
	// Load schema
	schemaData, err := schemaFS.ReadFile("config.schema.json")
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Convert config to JSON for validation
	configJSON, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(configJSON, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate structure (basic validation - checks for required fields and no additional properties)
	if err := validateAgainstSchema(configMap, schema); err != nil {
		return err
	}

	return nil
}

// validateAgainstSchema performs basic schema validation
// Checks required fields and no additional properties per spec
func validateAgainstSchema(data map[string]interface{}, schema map[string]interface{}) error {
	// Check required properties
	required, ok := schema["required"].([]interface{})
	if ok {
		for _, req := range required {
			field := req.(string)
			if _, exists := data[field]; !exists {
				return fmt.Errorf("missing required field: %s", field)
			}
		}
	}

	// Check no additional properties
	additionalProps, ok := schema["additionalProperties"].(bool)
	if ok && !additionalProps {
		allowedProps := make(map[string]bool)
		if props, ok := schema["properties"].(map[string]interface{}); ok {
			for key := range props {
				allowedProps[key] = true
			}
		}

		for key := range data {
			if !allowedProps[key] {
				return fmt.Errorf("unknown field: %s (additionalProperties not allowed)", key)
			}
		}
	}

	// Validate nested objects
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for key, value := range data {
			if propSchema, ok := props[key].(map[string]interface{}); ok {
				if propSchema["type"] == "object" {
					if nestedData, ok := value.(map[string]interface{}); ok {
						if err := validateAgainstSchema(nestedData, propSchema); err != nil {
							return fmt.Errorf("%s.%w", key, err)
						}
					}
				}
			}
		}
	}

	return nil
}
