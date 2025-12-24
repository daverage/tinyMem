package hydration

import (
	"fmt"
	"strings"

	"github.com/andrzejmarczewski/tslp/internal/state"
	"github.com/andrzejmarczewski/tslp/internal/vault"
)

// Engine handles JIT (Just-In-Time) hydration of state
// Per spec section 8: small models cannot dereference pointers, so we materialize truth
type Engine struct {
	vault *vault.Vault
	state *state.Manager
}

// New creates a new hydration engine
func New(v *vault.Vault, s *state.Manager) *Engine {
	return &Engine{
		vault: v,
		state: s,
	}
}

// HydrationBlock represents a single injected state block
type HydrationBlock struct {
	EntityKey    string
	ArtifactHash string
	Content      string
	Filepath     string
	Symbol       string
	Method       string
}

// Hydrate retrieves all AUTHORITATIVE entities and formats them for injection
// Per spec section 8.1: scan state map, retrieve artifacts, inject using strict template
func (h *Engine) Hydrate() (string, error) {
	// Get all authoritative entities
	entities, err := h.state.GetAuthoritative()
	if err != nil {
		return "", fmt.Errorf("failed to get authoritative entities: %w", err)
	}

	if len(entities) == 0 {
		return "", nil // No hydration needed
	}

	// Retrieve artifacts and build hydration blocks
	var blocks []HydrationBlock
	for _, entity := range entities {
		artifact, err := h.vault.Get(entity.ArtifactHash)
		if err != nil {
			return "", fmt.Errorf("failed to get artifact %s: %w", entity.ArtifactHash, err)
		}
		if artifact == nil {
			return "", fmt.Errorf("artifact %s not found in vault", entity.ArtifactHash)
		}

		// Extract resolution method from metadata if available
		method := "unknown"
		if entity.Metadata != nil {
			if m, ok := entity.Metadata["resolution_method"].(string); ok {
				method = m
			}
		}

		blocks = append(blocks, HydrationBlock{
			EntityKey:    entity.EntityKey,
			ArtifactHash: entity.ArtifactHash,
			Content:      artifact.Content,
			Filepath:     entity.Filepath,
			Symbol:       entity.Symbol,
			Method:       method,
		})
	}

	// Format using strict template
	return h.formatHydration(blocks), nil
}

// formatHydration applies the strict injection template
// Per spec section 8.2: fixed template format
func (h *Engine) formatHydration(blocks []HydrationBlock) string {
	var sb strings.Builder

	for _, block := range blocks {
		sb.WriteString("[CURRENT STATE: AUTHORITATIVE]\n")
		sb.WriteString(fmt.Sprintf("Entity: %s\n", block.EntityKey))
		sb.WriteString(fmt.Sprintf("Artifact: %s\n", block.ArtifactHash))

		// Determine source description based on method
		source := "Confirmed via AST"
		if block.Method == "regex" {
			source = "Confirmed via regex"
		} else if block.Method == "correlation" {
			source = "Inferred via correlation"
		}
		sb.WriteString(fmt.Sprintf("Source: %s\n", source))
		sb.WriteString("\n")
		sb.WriteString(block.Content)
		sb.WriteString("\n")
		sb.WriteString("[END CURRENT STATE]\n\n")
	}

	return sb.String()
}

// GenerateSafetyNotice creates a STATE NOTICE for non-CONFIRMED artifacts
// Per spec section 8.3: mandatory when previous artifact was INFERRED or UNRESOLVED
func GenerateSafetyNotice() string {
	return `[STATE NOTICE]
The previous output could not be structurally linked to a known entity.
It has NOT modified the State Map.
[END NOTICE]

`
}

// HydrateForEntities retrieves specific entities and formats them for injection
// Used when only specific entities are relevant to the current prompt
func (h *Engine) HydrateForEntities(entityKeys []string) (string, error) {
	if len(entityKeys) == 0 {
		return "", nil
	}

	var blocks []HydrationBlock
	for _, key := range entityKeys {
		entity, err := h.state.Get(key)
		if err != nil {
			return "", fmt.Errorf("failed to get entity %s: %w", key, err)
		}
		if entity == nil {
			continue // Entity not found, skip
		}

		// Only hydrate AUTHORITATIVE entities
		if !entity.IsAuthoritative() {
			continue
		}

		artifact, err := h.vault.Get(entity.ArtifactHash)
		if err != nil {
			return "", fmt.Errorf("failed to get artifact %s: %w", entity.ArtifactHash, err)
		}
		if artifact == nil {
			return "", fmt.Errorf("artifact %s not found in vault", entity.ArtifactHash)
		}

		method := "unknown"
		if entity.Metadata != nil {
			if m, ok := entity.Metadata["resolution_method"].(string); ok {
				method = m
			}
		}

		blocks = append(blocks, HydrationBlock{
			EntityKey:    entity.EntityKey,
			ArtifactHash: entity.ArtifactHash,
			Content:      artifact.Content,
			Filepath:     entity.Filepath,
			Symbol:       entity.Symbol,
			Method:       method,
		})
	}

	if len(blocks) == 0 {
		return "", nil
	}

	return h.formatHydration(blocks), nil
}

// HydrateByFilepath retrieves all entities for specific filepaths
// Useful when user mentions specific files in their prompt
func (h *Engine) HydrateByFilepath(filepaths []string) (string, error) {
	if len(filepaths) == 0 {
		return "", nil
	}

	var blocks []HydrationBlock
	for _, filepath := range filepaths {
		entities, err := h.state.GetByFilepath(filepath)
		if err != nil {
			return "", fmt.Errorf("failed to get entities for filepath %s: %w", filepath, err)
		}

		for _, entity := range entities {
			// Only hydrate AUTHORITATIVE entities
			if !entity.IsAuthoritative() {
				continue
			}

			artifact, err := h.vault.Get(entity.ArtifactHash)
			if err != nil {
				return "", fmt.Errorf("failed to get artifact %s: %w", entity.ArtifactHash, err)
			}
			if artifact == nil {
				return "", fmt.Errorf("artifact %s not found in vault", entity.ArtifactHash)
			}

			method := "unknown"
			if entity.Metadata != nil {
				if m, ok := entity.Metadata["resolution_method"].(string); ok {
					method = m
				}
			}

			blocks = append(blocks, HydrationBlock{
				EntityKey:    entity.EntityKey,
				ArtifactHash: entity.ArtifactHash,
				Content:      artifact.Content,
				Filepath:     entity.Filepath,
				Symbol:       entity.Symbol,
				Method:       method,
			})
		}
	}

	if len(blocks) == 0 {
		return "", nil
	}

	return h.formatHydration(blocks), nil
}
