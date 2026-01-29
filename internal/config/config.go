package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultProxyPort               = 8080
	DefaultExtractionBufferBytes   = 16384 // 16KB default
	DefaultLLMBaseURL              = "http://localhost:11434/v1"
	DefaultEmbeddingModel          = "nomic-embed-text"
	DefaultHybridWeight            = 0.5
	DefaultCoVeConfidenceThreshold = 0.6
	DefaultCoVeMaxCandidates       = 20
	DefaultCoVeTimeoutSeconds      = 30
	DefaultCoVeModel               = "" // Empty = use default LLM
)

// Config holds the application configuration
type Config struct {
	ProxyPort                     int
	LLMBaseURL                    string
	LLMAPIKey                     string
	LLMModel                      string
	EmbeddingBaseURL              string
	EmbeddingModel                string
	SemanticEnabled               bool
	HybridWeight                  float64
	EvidenceAllowCommand          bool
	EvidenceAllowedCommands       []string
	EvidenceCommandTimeoutSeconds int
	SearchEnabled                 bool
	Streaming                     bool
	LogLevel                      string
	LogFile                       string
	DBPath                        string
	ConfigPath                    string
	TinyMemDir                    string
	ProjectRoot                   string
	ExtractionBufferBytes         int
	RecallMaxItems                int
	RecallMaxTokens               int
	// Metrics configuration
	MetricsEnabled bool
	// CoVe (Chain-of-Verification) configuration
	CoVeEnabled             bool
	CoVeConfidenceThreshold float64
	CoVeMaxCandidates       int
	CoVeTimeoutSeconds      int
	CoVeModel               string
	CoVeRecallFilterEnabled bool
}

type fileConfig struct {
	Proxy struct {
		Port    int    `toml:"port"`
		BaseURL string `toml:"base_url"`
	} `toml:"proxy"`
	Recall struct {
		MaxItems        int     `toml:"max_items"`
		MaxTokens       int     `toml:"max_tokens"`
		SemanticEnabled bool    `toml:"semantic_enabled"`
		HybridWeight    float64 `toml:"hybrid_weight"`
	} `toml:"recall"`
	Memory struct {
		AutoExtract         bool `toml:"auto_extract"`
		RequireConfirmation bool `toml:"require_confirmation"`
	} `toml:"memory"`
	Logging struct {
		Level string `toml:"level"`
		File  string `toml:"file"`
	} `toml:"logging"`
	Evidence struct {
		AllowCommand          bool     `toml:"allow_command"`
		AllowedCommands       []string `toml:"allowed_commands"`
		CommandTimeoutSeconds int      `toml:"command_timeout_seconds"`
	} `toml:"evidence"`
	LLM struct {
		BaseURL string `toml:"base_url"`
		APIKey  string `toml:"api_key"`
		Model   string `toml:"model"`
	} `toml:"llm"`
	Embedding struct {
		BaseURL string `toml:"base_url"`
		Model   string `toml:"model"`
	} `toml:"embedding"`
	Metrics struct {
		Enabled bool `toml:"enabled"`
	} `toml:"metrics"`
	CoVe struct {
		Enabled             bool    `toml:"enabled"`
		ConfidenceThreshold float64 `toml:"confidence_threshold"`
		MaxCandidates       int     `toml:"max_candidates"`
		TimeoutSeconds      int     `toml:"timeout_seconds"`
		Model               string  `toml:"model"`
		RecallFilterEnabled bool    `toml:"recall_filter_enabled"`
	} `toml:"cove"`
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
		ProxyPort:                     DefaultProxyPort,
		LLMBaseURL:                    DefaultLLMBaseURL,
		EmbeddingBaseURL:              "",
		EmbeddingModel:                DefaultEmbeddingModel,
		SemanticEnabled:               false,
		HybridWeight:                  DefaultHybridWeight,
		EvidenceAllowCommand:          false,
		EvidenceAllowedCommands:       nil,
		EvidenceCommandTimeoutSeconds: 20,
		SearchEnabled:                 true,
		Streaming:                     true,
		LogLevel:                      "info",
		LogFile:                       filepath.Join(tinyMemDir, "logs", "tinymem.log"),
		DBPath:                        filepath.Join(tinyMemDir, "store.sqlite3"),
		ConfigPath:                    configPath,
		TinyMemDir:                    tinyMemDir,
		ProjectRoot:                   projectRoot,
		ExtractionBufferBytes:         DefaultExtractionBufferBytes,
		RecallMaxItems:                10,
		RecallMaxTokens:               2000,
		MetricsEnabled:                false,
		// CoVe defaults (enabled by default)
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: DefaultCoVeConfidenceThreshold,
		CoVeMaxCandidates:       DefaultCoVeMaxCandidates,
		CoVeTimeoutSeconds:      DefaultCoVeTimeoutSeconds,
		CoVeModel:               DefaultCoVeModel,
		CoVeRecallFilterEnabled: true,
	}

	embeddingBaseURLSet := false
	// Try to load config from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		fileData, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		var parsed fileConfig
		if err := toml.Unmarshal(fileData, &parsed); err != nil {
			return nil, err
		}
		var raw map[string]interface{}
		if err := toml.Unmarshal(fileData, &raw); err != nil {
			return nil, err
		}
		_, coveSectionPresent := raw["cove"]

		if parsed.Proxy.Port != 0 {
			cfg.ProxyPort = parsed.Proxy.Port
		}
		if parsed.Proxy.BaseURL != "" {
			cfg.LLMBaseURL = parsed.Proxy.BaseURL
		}
		if parsed.Recall.MaxItems != 0 {
			cfg.RecallMaxItems = parsed.Recall.MaxItems
		}
		if parsed.Recall.MaxTokens != 0 {
			cfg.RecallMaxTokens = parsed.Recall.MaxTokens
		}
		if parsed.Recall.HybridWeight != 0 {
			cfg.HybridWeight = parsed.Recall.HybridWeight
		}
		cfg.SemanticEnabled = parsed.Recall.SemanticEnabled
		if parsed.Logging.Level != "" {
			cfg.LogLevel = parsed.Logging.Level
		}
		if parsed.Logging.File != "" {
			cfg.LogFile = parsed.Logging.File
		}
		cfg.EvidenceAllowCommand = parsed.Evidence.AllowCommand
		if len(parsed.Evidence.AllowedCommands) > 0 {
			cfg.EvidenceAllowedCommands = parsed.Evidence.AllowedCommands
		}
		if parsed.Evidence.CommandTimeoutSeconds > 0 {
			cfg.EvidenceCommandTimeoutSeconds = parsed.Evidence.CommandTimeoutSeconds
		}
		if parsed.LLM.BaseURL != "" {
			cfg.LLMBaseURL = parsed.LLM.BaseURL
		}
		if parsed.LLM.APIKey != "" {
			cfg.LLMAPIKey = parsed.LLM.APIKey
		}
		if parsed.LLM.Model != "" {
			cfg.LLMModel = parsed.LLM.Model
		}
		if parsed.Embedding.BaseURL != "" {
			cfg.EmbeddingBaseURL = parsed.Embedding.BaseURL
			embeddingBaseURLSet = true
		}
		if parsed.Embedding.Model != "" {
			cfg.EmbeddingModel = parsed.Embedding.Model
		}
		// Metrics configuration
		cfg.MetricsEnabled = parsed.Metrics.Enabled
		// CoVe configuration (only override defaults if [cove] is present)
		if coveSectionPresent {
			cfg.CoVeEnabled = parsed.CoVe.Enabled
			if parsed.CoVe.ConfidenceThreshold > 0 {
				cfg.CoVeConfidenceThreshold = parsed.CoVe.ConfidenceThreshold
			}
			if parsed.CoVe.MaxCandidates > 0 {
				cfg.CoVeMaxCandidates = parsed.CoVe.MaxCandidates
			}
			if parsed.CoVe.TimeoutSeconds > 0 {
				cfg.CoVeTimeoutSeconds = parsed.CoVe.TimeoutSeconds
			}
			if parsed.CoVe.Model != "" {
				cfg.CoVeModel = parsed.CoVe.Model
			}
			cfg.CoVeRecallFilterEnabled = parsed.CoVe.RecallFilterEnabled
		}
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
	if logFile := os.Getenv("TINYMEM_LOG_FILE"); logFile != "" {
		cfg.LogFile = logFile
	}

	if baseURL := os.Getenv("TINYMEM_LLM_BASE_URL"); baseURL != "" {
		cfg.LLMBaseURL = baseURL
	}
	if apiKey := os.Getenv("TINYMEM_LLM_API_KEY"); apiKey != "" {
		cfg.LLMAPIKey = apiKey
	}
	if llmModel := os.Getenv("TINYMEM_LLM_MODEL"); llmModel != "" {
		cfg.LLMModel = llmModel
	}
	if embedBaseURL := os.Getenv("TINYMEM_EMBEDDING_BASE_URL"); embedBaseURL != "" {
		cfg.EmbeddingBaseURL = embedBaseURL
		embeddingBaseURLSet = true
	}
	if embedModel := os.Getenv("TINYMEM_EMBEDDING_MODEL"); embedModel != "" {
		cfg.EmbeddingModel = embedModel
	}
	if allowCmd := os.Getenv("TINYMEM_EVIDENCE_ALLOW_COMMAND"); allowCmd != "" {
		cfg.EvidenceAllowCommand = allowCmd == "true" || allowCmd == "1"
	}
	if allowed := os.Getenv("TINYMEM_EVIDENCE_ALLOWED_COMMANDS"); allowed != "" {
		parts := strings.Split(allowed, ",")
		var cmds []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				cmds = append(cmds, part)
			}
		}
		if len(cmds) > 0 {
			cfg.EvidenceAllowedCommands = cmds
		}
	}
	if timeoutStr := os.Getenv("TINYMEM_EVIDENCE_COMMAND_TIMEOUT_SECONDS"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			cfg.EvidenceCommandTimeoutSeconds = timeout
		}
	}
	if semantic := os.Getenv("TINYMEM_SEMANTIC_ENABLED"); semantic != "" {
		cfg.SemanticEnabled = semantic == "true" || semantic == "1"
	}
	if hybridWeight := os.Getenv("TINYMEM_HYBRID_WEIGHT"); hybridWeight != "" {
		if weight, err := strconv.ParseFloat(hybridWeight, 64); err == nil {
			cfg.HybridWeight = weight
		}
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
	if maxItems := os.Getenv("TINYMEM_RECALL_MAX_ITEMS"); maxItems != "" {
		if size, err := strconv.Atoi(maxItems); err == nil {
			cfg.RecallMaxItems = size
		}
	}
	if maxTokens := os.Getenv("TINYMEM_RECALL_MAX_TOKENS"); maxTokens != "" {
		if size, err := strconv.Atoi(maxTokens); err == nil {
			cfg.RecallMaxTokens = size
		}
	}

	// Metrics environment variable overrides
	if metricsEnabled := os.Getenv("TINYMEM_METRICS_ENABLED"); metricsEnabled != "" {
		cfg.MetricsEnabled = metricsEnabled == "true" || metricsEnabled == "1"
	}

	// CoVe environment variable overrides
	if coveEnabled := os.Getenv("TINYMEM_COVE_ENABLED"); coveEnabled != "" {
		cfg.CoVeEnabled = coveEnabled == "true" || coveEnabled == "1"
	}
	if coveThreshold := os.Getenv("TINYMEM_COVE_CONFIDENCE_THRESHOLD"); coveThreshold != "" {
		if threshold, err := strconv.ParseFloat(coveThreshold, 64); err == nil {
			cfg.CoVeConfidenceThreshold = threshold
		}
	}
	if coveMaxCandidates := os.Getenv("TINYMEM_COVE_MAX_CANDIDATES"); coveMaxCandidates != "" {
		if max, err := strconv.Atoi(coveMaxCandidates); err == nil {
			cfg.CoVeMaxCandidates = max
		}
	}
	if coveTimeout := os.Getenv("TINYMEM_COVE_TIMEOUT_SECONDS"); coveTimeout != "" {
		if timeout, err := strconv.Atoi(coveTimeout); err == nil {
			cfg.CoVeTimeoutSeconds = timeout
		}
	}
	if coveModel := os.Getenv("TINYMEM_COVE_MODEL"); coveModel != "" {
		cfg.CoVeModel = coveModel
	}
	if coveRecallFilter := os.Getenv("TINYMEM_COVE_RECALL_FILTER_ENABLED"); coveRecallFilter != "" {
		cfg.CoVeRecallFilterEnabled = coveRecallFilter == "true" || coveRecallFilter == "1"
	}

	cfg.LLMBaseURL = normalizeBaseURL(cfg.LLMBaseURL)
	if !embeddingBaseURLSet {
		cfg.EmbeddingBaseURL = cfg.LLMBaseURL
	}
	cfg.EmbeddingBaseURL = normalizeBaseURL(cfg.EmbeddingBaseURL)

	return cfg, nil
}

func normalizeBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return baseURL
	}
	return strings.TrimRight(baseURL, "/")
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

// Validate verifies the configuration is usable.
func (c *Config) Validate() error {
	if c.ProxyPort <= 0 || c.ProxyPort > 65535 {
		return fmt.Errorf("proxy port out of range: %d", c.ProxyPort)
	}
	if strings.TrimSpace(c.LLMBaseURL) == "" {
		return fmt.Errorf("LLM base URL is empty")
	}
	if c.RecallMaxItems < 0 {
		return fmt.Errorf("recall max items cannot be negative")
	}
	if c.RecallMaxTokens < 0 {
		return fmt.Errorf("recall max tokens cannot be negative")
	}
	if c.ExtractionBufferBytes <= 0 {
		return fmt.Errorf("extraction buffer bytes must be positive")
	}
	if c.HybridWeight < 0 || c.HybridWeight > 1 {
		return fmt.Errorf("hybrid weight must be between 0 and 1")
	}
	if c.EvidenceCommandTimeoutSeconds <= 0 {
		return fmt.Errorf("evidence command timeout must be positive")
	}
	if c.EvidenceAllowCommand && len(c.EvidenceAllowedCommands) == 0 {
		return fmt.Errorf("evidence commands enabled but no allowed commands configured")
	}
	// CoVe validation
	if c.CoVeConfidenceThreshold < 0 || c.CoVeConfidenceThreshold > 1 {
		return fmt.Errorf("CoVe confidence threshold must be between 0 and 1")
	}
	if c.CoVeMaxCandidates < 0 {
		return fmt.Errorf("CoVe max candidates cannot be negative")
	}
	if c.CoVeTimeoutSeconds <= 0 {
		return fmt.Errorf("CoVe timeout must be positive")
	}
	return nil
}
