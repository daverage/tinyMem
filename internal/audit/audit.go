package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrzejmarczewski/tinyMem/internal/ledger"
	"github.com/andrzejmarczewski/tinyMem/internal/llm"
	"github.com/andrzejmarczewski/tinyMem/internal/logging"
	"github.com/andrzejmarczewski/tinyMem/internal/vault"
)

// AuditStatus represents the result of an audit
type AuditStatus string

const (
	StatusCompleted  AuditStatus = "completed"
	StatusPartial    AuditStatus = "partial"
	StatusDiscussion AuditStatus = "discussion"
)

// AuditResponse represents the JSON response from the audit LLM call
type AuditResponse struct {
	Entity string      `json:"entity"`
	Status AuditStatus `json:"status"`
}

// Auditor performs shadow audits on artifacts
// Per spec section 10: non-blocking, metadata only, affects durability only
type Auditor struct {
	llmClient interface {
		Chat(ctx context.Context, messages []llm.Message) (*llm.ChatResponse, error)
		GetModel() string
	}
	vault  *vault.Vault
	ledger *ledger.Ledger
	logger *logging.Logger
}

// NewAuditor creates a new shadow auditor
func NewAuditor(llmClient interface {
	Chat(ctx context.Context, messages []llm.Message) (*llm.ChatResponse, error)
	GetModel() string
}, v *vault.Vault, l *ledger.Ledger, logger *logging.Logger) *Auditor {
	return &Auditor{
		llmClient: llmClient,
		vault:     v,
		ledger:    l,
		logger:    logger,
	}
}

// AuditAsync performs an asynchronous shadow audit
// Per spec: user receives stream immediately, audit happens in background
func (a *Auditor) AuditAsync(episodeID, artifactHash string) {
	go func() {
		if err := a.performAudit(episodeID, artifactHash); err != nil {
			a.logger.Error("Shadow audit failed: episode=%s artifact=%s error=%v", episodeID, artifactHash, err)
		}
	}()
}

// performAudit executes the actual audit
func (a *Auditor) performAudit(episodeID, artifactHash string) error {
	a.logger.AuditStarted(episodeID, artifactHash)

	// Retrieve artifact from vault
	artifact, err := a.vault.Get(artifactHash)
	if err != nil {
		return fmt.Errorf("failed to get artifact: %w", err)
	}
	if artifact == nil {
		return fmt.Errorf("artifact not found: %s", artifactHash)
	}

	// Construct audit prompt
	// Per spec: 1-turn JSON audit
	// Strict prompt to force JSON output from small models
	messages := []llm.Message{
		{
			Role:    "system",
			Content: llm.Content{Value: "You are a code auditor. You MUST respond with ONLY valid JSON. No other text is allowed.\n\nAnalyze the code artifact and return this exact JSON format:\n{\"entity\": \"<primary entity name>\", \"status\": \"<completed|partial|discussion>\"}\n\nRules:\n- entity: the main function/class/type name in the code\n- status: completed (full implementation), partial (incomplete), discussion (no code/planning)\n- Return ONLY the JSON object, nothing else"},
		},
		{
			Role:    "user",
			Content: llm.Content{Value: fmt.Sprintf("Analyze this code and return JSON only:\n\n%s", artifact.Content)},
		},
	}

	// Call LLM with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := a.llmClient.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return fmt.Errorf("no choices in response")
	}

	// Parse JSON response
	var auditResp AuditResponse
	raw := response.Choices[0].Message.Content.GetString()
	if err := json.Unmarshal([]byte(raw), &auditResp); err != nil {
		truncated := truncateAuditLog(raw)
		if extracted, ok := extractJSONObject(raw); ok {
			if err2 := json.Unmarshal([]byte(extracted), &auditResp); err2 == nil {
				a.logger.Debug("Extracted JSON from audit response (truncated): %s", truncateAuditLog(extracted))
			} else {
				auditResp = AuditResponse{
					Entity: "unknown",
					Status: StatusDiscussion,
				}
				a.logger.Debug("Failed to parse extracted JSON (raw=%s, extracted=%s): %v", truncated, truncateAuditLog(extracted), err2)
			}
		} else {
			auditResp = AuditResponse{
				Entity: "unknown",
				Status: StatusDiscussion,
			}
			a.logger.Debug("Failed to parse audit response as JSON (raw=%s): %v", truncated, err)
		}
	}

	// Store audit result in ledger
	auditResponseJSON, _ := json.Marshal(auditResp)
	entityKey := &auditResp.Entity
	if auditResp.Entity == "" || auditResp.Entity == "unknown" {
		entityKey = nil
	}

	if err := a.ledger.RecordAudit(episodeID, artifactHash, entityKey, string(auditResp.Status), string(auditResponseJSON)); err != nil {
		return fmt.Errorf("failed to record audit: %w", err)
	}

	a.logger.AuditCompleted(episodeID, artifactHash, string(auditResp.Status))

	return nil
}

// GetAuditResults retrieves audit results for an episode
func (a *Auditor) GetAuditResults(episodeID string) ([]*ledger.AuditResult, error) {
	return a.ledger.GetAuditResults(episodeID)
}

const maxAuditResponseLogBytes = 512

func truncateAuditLog(content string) string {
	if len(content) <= maxAuditResponseLogBytes {
		return content
	}

	return content[:maxAuditResponseLogBytes] + "...(truncated)"
}

func extractJSONObject(content string) (string, bool) {
	start := -1
	depth := 0
	for idx, ch := range content {
		switch ch {
		case '{':
			if depth == 0 {
				start = idx
			}
			depth++
		case '}':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					return content[start : idx+1], true
				}
			}
		}
	}
	return "", false
}
