package entity

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

//go:embed symbols.json
var symbolsFS embed.FS

// Pattern represents a regex pattern for symbol extraction
type Pattern struct {
	Name         string `json:"name"`
	Regex        string `json:"regex"`
	CaptureGroup int    `json:"capture_group"`
	Confidence   string `json:"confidence"` // "CONFIRMED" or "INFERRED"
}

// LanguagePatterns holds patterns for a specific language
type LanguagePatterns struct {
	Patterns []Pattern `json:"patterns"`
}

// SymbolsConfig represents the symbols.json configuration
type SymbolsConfig struct {
	Comment   string                      `json:"comment"`
	Version   string                      `json:"version"`
	Languages map[string]LanguagePatterns `json:"languages"`
}

// regexCache caches compiled regex patterns
var regexCache = make(map[string]*regexp.Regexp)

// symbolsConfig is loaded once at startup
var symbolsConfig *SymbolsConfig

// LoadSymbolsConfig loads the symbols.json configuration
// Per spec: Load at startup, no defaults in code
func LoadSymbolsConfig() error {
	data, err := symbolsFS.ReadFile("symbols.json")
	if err != nil {
		return fmt.Errorf("failed to read symbols.json: %w", err)
	}

	var config SymbolsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse symbols.json: %w", err)
	}

	symbolsConfig = &config
	return nil
}

// GetSymbolsConfig returns the loaded symbols configuration
func GetSymbolsConfig() *SymbolsConfig {
	return symbolsConfig
}

// resolveViaRegexPatterns attempts entity resolution using loaded regex patterns
// Per spec section 4.1: Exact unique match → CONFIRMED, Partial/ambiguous → INFERRED
func (r *Resolver) resolveViaRegexPatterns(artifactHash, content, language string) *Resolution {
	if symbolsConfig == nil {
		// Symbols config not loaded - return unresolved
		return &Resolution{
			ArtifactHash: artifactHash,
			EntityKey:    nil,
			Confidence:   ConfidenceUnresolved,
			Method:       MethodUnresolved,
			Filepath:     nil,
			Symbols:      []string{},
			ASTNodeCount: nil,
			CreatedAt:    time.Now(),
		}
	}

	// Get patterns for this language
	langPatterns, ok := symbolsConfig.Languages[language]
	if !ok {
		// No patterns for this language
		return &Resolution{
			ArtifactHash: artifactHash,
			EntityKey:    nil,
			Confidence:   ConfidenceUnresolved,
			Method:       MethodUnresolved,
			Filepath:     nil,
			Symbols:      []string{},
			ASTNodeCount: nil,
			CreatedAt:    time.Now(),
		}
	}

	// Collect all symbol matches across all patterns
	symbolMatches := make(map[string]Confidence)

	for _, pattern := range langPatterns.Patterns {
		// Compile or get cached regex
		re, err := getOrCompileRegex(pattern.Regex)
		if err != nil {
			continue // Skip invalid patterns
		}

		// Find all matches
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > pattern.CaptureGroup {
				symbol := match[pattern.CaptureGroup]
				if symbol != "" {
					// Use the highest confidence for this symbol
					confidence := parseConfidence(pattern.Confidence)
					if _, ok := symbolMatches[symbol]; ok {
						// Already have this symbol, upgrade to CONFIRMED if applicable
						if confidence == ConfidenceConfirmed {
							symbolMatches[symbol] = confidence
						}
					} else {
						symbolMatches[symbol] = confidence
					}
				}
			}
		}
	}

	// Convert to symbol list
	var symbols []string
	for sym := range symbolMatches {
		symbols = append(symbols, sym)
	}

	if len(symbols) == 0 {
		// No matches found
		return &Resolution{
			ArtifactHash: artifactHash,
			EntityKey:    nil,
			Confidence:   ConfidenceUnresolved,
			Method:       MethodRegex,
			Filepath:     nil,
			Symbols:      []string{},
			ASTNodeCount: nil,
			CreatedAt:    time.Now(),
		}
	}

	// Determine overall confidence
	// Per spec: Exact unique match → CONFIRMED, Partial/ambiguous → INFERRED
	var overallConfidence Confidence
	var entityKey *string

	if len(symbols) == 1 {
		// Single unique symbol - use the confidence from pattern
		overallConfidence = symbolMatches[symbols[0]]
		// Create entity key if we have a single symbol
		key := MakeEntityKey("unknown", symbols[0])
		entityKey = &key
	} else {
		// Multiple symbols - ambiguous, downgrade to INFERRED
		overallConfidence = ConfidenceInferred
		entityKey = nil
	}

	return &Resolution{
		ArtifactHash: artifactHash,
		EntityKey:    entityKey,
		Confidence:   overallConfidence,
		Method:       MethodRegex,
		Filepath:     nil,
		Symbols:      symbols,
		ASTNodeCount: nil,
		CreatedAt:    time.Now(),
	}
}

// getOrCompileRegex gets a compiled regex from cache or compiles it
func getOrCompileRegex(pattern string) (*regexp.Regexp, error) {
	if re, ok := regexCache[pattern]; ok {
		return re, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	regexCache[pattern] = re
	return re, nil
}

// parseConfidence converts string confidence to Confidence type
func parseConfidence(s string) Confidence {
	switch s {
	case "CONFIRMED":
		return ConfidenceConfirmed
	case "INFERRED":
		return ConfidenceInferred
	default:
		return ConfidenceUnresolved
	}
}
