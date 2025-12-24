package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrzejmarczewski/tslp/internal/ledger"
	"github.com/andrzejmarczewski/tslp/internal/llm"
	"github.com/andrzejmarczewski/tslp/internal/logging"
	"github.com/andrzejmarczewski/tslp/internal/vault"
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
	llmClient *llm.Client
	vault     *vault.Vault
	ledger    *ledger.Ledger
	logger    *logging.Logger
}

// NewAuditor creates a new shadow auditor
func NewAuditor(llmClient *llm.Client, v *vault.Vault, l *ledger.Ledger, logger *logging.Logger) *Auditor {
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
	messages := []llm.Message{
		{
			Role:    "system",
			Content: "You are a code auditor. Analyze the following code artifact and return a JSON response with fields: entity (string, the primary entity name), status (one of: completed, partial, discussion).",
		},
		{
			Role:    "user",
			Content: artifact.Content,
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
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &auditResp); err != nil {
		// If JSON parsing fails, record as discussion
		auditResp = AuditResponse{
			Entity: "unknown",
			Status: StatusDiscussion,
		}
		a.logger.Debug("Failed to parse audit response as JSON, marking as discussion")
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
