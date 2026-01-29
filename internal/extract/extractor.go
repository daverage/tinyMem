package extract

import (
	"fmt"
	"github.com/daverage/tinymem/internal/cove"
	"github.com/daverage/tinymem/internal/evidence"
	"github.com/daverage/tinymem/internal/memory"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Extractor handles automatic extraction of memories from text
type Extractor struct {
	evidenceService *evidence.Service
	coveVerifier    *cove.Verifier
}

// NewExtractor creates a new memory extractor
func NewExtractor(evidenceService *evidence.Service) *Extractor {
	return &Extractor{
		evidenceService: evidenceService,
		coveVerifier:    nil, // Can be set later via SetCoVeVerifier
	}
}

// SetCoVeVerifier sets the CoVe verifier for candidate filtering
func (e *Extractor) SetCoVeVerifier(verifier *cove.Verifier) {
	e.coveVerifier = verifier
}

// GetCoVeStats returns CoVe statistics if available
func (e *Extractor) GetCoVeStats() *cove.Stats {
	if e.coveVerifier == nil {
		return nil
	}
	stats := e.coveVerifier.GetStats()
	return &stats
}

// WantsFactMetadata is metadata indicating a memory wants to be promoted to fact
type WantsFactMetadata struct {
	WantsFact bool   `json:"wants_fact"`
	Original  string `json:"original_text"`
}

// ExtractMemories extracts potential memories from text
func (e *Extractor) ExtractMemories(text string, projectID string) ([]*memory.Memory, error) {
	var memories []*memory.Memory

	// Extract potential decisions
	decisions := e.extractDecisions(text)
	for _, decision := range decisions {
		memories = append(memories, &memory.Memory{
			ProjectID: projectID,
			Type:      memory.Decision,
			Summary:   decision,
			Detail:    fmt.Sprintf("Extracted from context: %s", truncateString(text, 200)),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	// Extract potential plans
	plans := e.extractPlans(text)
	for _, plan := range plans {
		memories = append(memories, &memory.Memory{
			ProjectID: projectID,
			Type:      memory.Plan,
			Summary:   plan,
			Detail:    fmt.Sprintf("Extracted from context: %s", truncateString(text, 200)),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	// Extract potential constraints
	constraints := e.extractConstraints(text)
	for _, constraint := range constraints {
		memories = append(memories, &memory.Memory{
			ProjectID: projectID,
			Type:      memory.Constraint,
			Summary:   constraint,
			Detail:    fmt.Sprintf("Extracted from context: %s", truncateString(text, 200)),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	// Extract potential observations
	observations := e.extractObservations(text)
	for _, observation := range observations {
		memories = append(memories, &memory.Memory{
			ProjectID: projectID,
			Type:      memory.Observation,
			Summary:   observation,
			Detail:    fmt.Sprintf("Extracted from context: %s", truncateString(text, 200)),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	// Extract potential claims - including fact-like statements as claims
	claims := e.extractClaims(text)
	for _, claim := range claims {
		// Check if this claim sounds like a fact (contains fact-indicating language)
		memoryType := memory.Claim
		detail := fmt.Sprintf("Extracted from context: %s", truncateString(text, 200))

		// If the claim contains language suggesting it's a fact, mark it specially
		if e.containsFactIndicators(claim) {
			// Still store as claim, but with metadata indicating it wants to be a fact
			detail = fmt.Sprintf("EXTRACTED AS CLAIM (wanted fact): %s", detail)
		}

		memories = append(memories, &memory.Memory{
			ProjectID: projectID,
			Type:      memoryType,
			Summary:   claim,
			Detail:    detail,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	return memories, nil
}

// containsFactIndicators checks if text contains language that suggests it's a fact
func (e *Extractor) containsFactIndicators(text string) bool {
	factIndicators := []string{
		"is ", "are ", "was ", "were ", "has ", "have ", "implies ", "means ",
		"now ", "currently ", "recently ", "already ", "always ", "never ",
		"implemented", "supports", "uses", "contains", "includes",
	}

	lowerText := strings.ToLower(text)
	for _, indicator := range factIndicators {
		if strings.Contains(lowerText, indicator) {
			return true
		}
	}
	return false
}

// extractDecisions extracts potential decisions from text
func (e *Extractor) extractDecisions(text string) []string {
	var decisions []string

	// Look for decision indicators
	decisionPatterns := []string{
		`(?i)(?:we|I|the team)\s+(?:have decided|has decided|decided|will|should|ought to)\s+(?:implement|do|take|follow|adopt|choose)\s+(?:the|a|an)?\s*([^.!?]*?)(?:\.|!|\?)`,
		`(?i)(?:decision|agreement):\s*(.*?)(?:\.|!|\?)`,
		`(?i)(?:the decision is|our decision|it was decided)\s+(?:that|to)\s+(.*?)(?:\.|!|\?)`,
		`(?i)(?:we agreed|it was agreed)\s+(?:that|on|to)\s+(.*?)(?:\.|!|\?)`,
	}

	for _, pattern := range decisionPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				decision := strings.TrimSpace(match[1])
				if len(decision) > 5 { // Avoid very short extractions
					decisions = append(decisions, decision)
				}
			}
		}
	}

	return decisions
}

// extractPlans extracts potential plans from text
func (e *Extractor) extractPlans(text string) []string {
	var plans []string

	// Look for plan indicators
	planPatterns := []string{
		`(?i)(?:plan|planning|will|going to|intend to|aim to|looking to)\s+(?:implement|do|create|develop|build|add|fix|resolve|address)\s+(?:the|a|an)?\s*([^.!?]*?)(?:\.|!|\?)`,
		`(?i)(?:next steps?|future work|upcoming|plan):\s*(.*?)(?:\.|!|\?)`,
		`(?i)(?:we will|we'll|I will|I'll)\s+(?:be|start|begin|continue)\s+(.*?)(?:\.|!|\?)`,
		`(?i)(?:in the future|later|soon|eventually)\s*,?\s+(?:we|I)\s+(?:will|plan to)\s+(.*?)(?:\.|!|\?)`,
	}

	for _, pattern := range planPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				plan := strings.TrimSpace(match[1])
				if len(plan) > 5 { // Avoid very short extractions
					plans = append(plans, plan)
				}
			}
		}
	}

	return plans
}

// extractConstraints extracts potential constraints from text
func (e *Extractor) extractConstraints(text string) []string {
	var constraints []string

	// Look for constraint indicators
	constraintPatterns := []string{
		`(?i)(?:constraint|limitation|restriction|requirement|must|need to|has to|required|obligated to)\s+(?:that|be|do|have|include|follow)\s+(?:the|a|an)?\s*([^.!?]*?)(?:\.|!|\?)`,
		`(?i)(?:we can't|we cannot|it's not possible|impossible|prohibited|forbidden|not allowed)\s+(?:to|because)\s+(.*?)(?:\.|!|\?)`,
		`(?i)(?:due to|because of|owing to|on account of)\s+(?:the|a|an)?\s*([^.!?]*constraint[^.!?]*|[^.!?]*limitation[^.!?]*|[^.!?]*requirement[^.!?]*)(?:\.|!|\?)`,
		`(?i)(?:must|need|require|necessary|essential)\s+(?:to|that)\s+(.*?)(?:\.|!|\?)`,
	}

	for _, pattern := range constraintPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				constraint := strings.TrimSpace(match[1])
				if len(constraint) > 5 { // Avoid very short extractions
					constraints = append(constraints, constraint)
				}
			}
		}
	}

	return constraints
}

// extractObservations extracts potential observations from text
func (e *Extractor) extractObservations(text string) []string {
	var observations []string

	// Look for observation indicators
	observationPatterns := []string{
		`(?i)(?:observed|noticed|found|discovered|seen|detected|identified)\s+(?:that|the|a|an)?\s*([^.!?]*?)(?:\.|!|\?)`,
		`(?i)(?:it appears|appears to be|seems to be|looks like|seems|appears)\s+(?:that|to be)?\s+(.*?)(?:\.|!|\?)`,
		`(?i)(?:currently|right now|at present|today)\s+(?:the|a|an)?\s*([^.!?]*?)(?:\.|!|\?)`,
		`(?i)(?:testing showed|results indicate|analysis revealed|data shows)\s+(?:that)?\s+(.*?)(?:\.|!|\?)`,
	}

	for _, pattern := range observationPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				observation := strings.TrimSpace(match[1])
				if len(observation) > 5 { // Avoid very short extractions
					observations = append(observations, observation)
				}
			}
		}
	}

	return observations
}

// extractClaims extracts potential claims from text
func (e *Extractor) extractClaims(text string) []string {
	var claims []string

	// Look for claim indicators
	claimPatterns := []string{
		`(?i)(?:claim|assertion|believe|think|assume|presume|suggest|indicate)\s+(?:that)?\s+(.*?)(?:\.|!|\?)`,
		`(?i)(?:it is said|people say|some say|research suggests|studies show)\s+(?:that)?\s+(.*?)(?:\.|!|\?)`,
		`(?i)(?:apparently|reportedly|allegedly)\s+(?:the|a|an)?\s*([^.!?]*?)(?:\.|!|\?)`,
		`(?i)(?:according to|based on|from)\s+(?:the|a|an)?\s*([^.!?]*?)(?:\.|!|\?)`,
	}

	for _, pattern := range claimPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				claim := strings.TrimSpace(match[1])
				if len(claim) > 5 { // Avoid very short extractions
					claims = append(claims, claim)
				}
			}
		}
	}

	return claims
}

// ValidateAndStoreMemories validates and stores extracted memories
func (e *Extractor) ValidateAndStoreMemories(memories []*memory.Memory, memoryService *memory.Service) error {
	for _, mem := range memories {
		// Apply validation rules
		if !mem.Type.IsValid() {
			continue // Skip invalid memory types
		}

		// Facts require evidence - default to non-fact types
		if mem.Type == memory.Fact {
			// For now, don't store facts without evidence
			// In a real system, we might queue these for evidence verification
			continue
		}

		// Store the memory
		if err := memoryService.CreateMemory(mem); err != nil {
			return fmt.Errorf("failed to store memory: %w", err)
		}
	}

	return nil
}

// ExtractAndStoreFromStreamingResponse processes a streaming response and extracts memories
func (e *Extractor) ExtractAndStoreFromStreamingResponse(responseText string, memoryService *memory.Service, projectID string) error {
	// Extract memories from the response text
	memories, err := e.ExtractMemories(responseText, projectID)
	if err != nil {
		return fmt.Errorf("failed to extract memories: %w", err)
	}

	// Validate and store the extracted memories
	return e.ValidateAndStoreMemories(memories, memoryService)
}

// ExtractAndQueueForVerification extracts memories and queues fact candidates for verification
func (e *Extractor) ExtractAndQueueForVerification(responseText string, memoryService *memory.Service, evidenceService *evidence.Service, projectID string) error {
	// Extract memories from the response text
	memories, err := e.ExtractMemories(responseText, projectID)
	if err != nil {
		return fmt.Errorf("failed to extract memories: %w", err)
	}

	// COVE INTEGRATION POINT 1: Filter candidates before storage
	// This is a probabilistic filter that reduces hallucinated claims
	// IMPORTANT: CoVe can NEVER create facts or bypass evidence requirements
	if e.coveVerifier != nil {
		// Convert memories to CoVe candidates
		candidates := memoriesToCandidates(memories)

		// Apply CoVe verification (bounded, fail-safe)
		verifiedCandidates, err := e.coveVerifier.VerifyCandidates(candidates)
		if err != nil {
			// CoVe errors are non-fatal - continue with unfiltered memories
		} else {
			// Filter memories based on CoVe results
			memories = filterMemoriesByCoVe(memories, verifiedCandidates)
		}
	}

	// Validate and store the extracted memories
	if err := e.ValidateAndStoreMemories(memories, memoryService); err != nil {
		return fmt.Errorf("failed to store memories: %w", err)
	}

	// After storing, check if any of the stored memories should be promoted to facts
	// This would typically happen for claims that have supporting evidence
	// NOTE: CoVe is NOT involved in fact promotion - only evidence verification matters
	for _, mem := range memories {
		// Check if this is a claim that might be eligible for fact promotion
		if mem.Type == memory.Claim {
			// Check if this claim has been validated with evidence
			isValidated, err := evidenceService.IsMemoryValidated(mem)
			if err != nil {
				// Log error but continue processing other memories
				continue
			}

			if isValidated {
				// Promote the claim to a fact if it has valid evidence
				err = memoryService.PromoteToFact(mem.ID, projectID, true)
				if err != nil {
					// Log error but continue processing other memories
					continue
				}
			}
		}
	}

	return nil
}

// Helper function to truncate strings
func truncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen] + "..."
}

// memoriesToCandidates converts memory.Memory objects to cove.CandidateMemory for verification
func memoriesToCandidates(memories []*memory.Memory) []cove.CandidateMemory {
	candidates := make([]cove.CandidateMemory, 0, len(memories))

	for i, mem := range memories {
		// Generate temporary ID for tracking (we'll use index since memories don't have IDs yet)
		candidates = append(candidates, cove.CandidateMemory{
			ID:      strconv.Itoa(i),
			Type:    string(mem.Type),
			Summary: mem.Summary,
			Detail:  mem.Detail,
		})
	}

	return candidates
}

// filterMemoriesByCoVe filters memories based on CoVe verification results
func filterMemoriesByCoVe(memories []*memory.Memory, verified []cove.CandidateMemory) []*memory.Memory {
	// Build a set of accepted IDs
	acceptedIDs := make(map[string]bool)
	for _, candidate := range verified {
		acceptedIDs[candidate.ID] = true
	}

	// Filter memories based on accepted IDs
	filtered := make([]*memory.Memory, 0, len(verified))
	for i, mem := range memories {
		if acceptedIDs[strconv.Itoa(i)] {
			filtered = append(filtered, mem)
		}
	}

	return filtered
}
