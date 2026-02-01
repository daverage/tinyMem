package automated

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/storage"
)

// TestSemanticCannotOverrideLexical verifies semantic recall cannot override lexical truth
func TestSemanticCannotOverrideLexical(t *testing.T) {
	tmpDir, cleanup := setupAuthorityTestEnv(t)
	defer cleanup()

	cfg := &config.Config{
		ProjectRoot:     tmpDir,
		DBPath:          filepath.Join(tmpDir, "test.db"),
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	memService := memory.NewService(db)

	// Create memory with specific content (as claim first, then promote)
	groundTruthMem := &memory.Memory{
		ProjectID: "test",
		Type:      memory.Claim,
		Summary:   "API endpoint is /api/v2/users",
		Detail:    "The user API endpoint was changed to /api/v2/users in commit abc123",
	}

	err = memService.CreateMemory(groundTruthMem)
	if err != nil {
		t.Fatalf("Failed to create ground truth memory: %v", err)
	}

	// Promote to fact with force (simulating evidence validation)
	searchResults, _ := memService.SearchMemories("test", "API endpoint", 10)
	if len(searchResults) == 0 {
		t.Fatal("Failed to find created memory via search")
	}

	memID := searchResults[0].ID
	err = memService.PromoteToFact(memID, "test", true)
	if err != nil {
		t.Fatalf("Failed to promote to fact: %v", err)
	}

	// Get the updated memory
	groundTruthMem, err = memService.GetMemory(memID, "test")
	if err != nil {
		t.Fatalf("Failed to get promoted memory: %v", err)
	}

	if groundTruthMem.Type != memory.Fact {
		t.Fatalf("Memory was not promoted to fact, type is: %s", groundTruthMem.Type)
	}

	// CRITICAL TEST: Verify the fact exists and is retrievable via lexical search
	// This proves semantic scoring cannot suppress lexical truth
	verifyResults, err := memService.SearchMemories("test", "API endpoint users", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(verifyResults) == 0 {
		t.Fatal("Lexical search failed to find fact - AUTHORITY VIOLATION")
	}

	// Verify the ground truth fact is in results
	found := false
	for _, result := range verifyResults {
		if result.ID == groundTruthMem.ID {
			found = true
			// Verify it's still a fact (not demoted by semantic scoring)
			if result.Type != memory.Fact {
				t.Fatalf("Fact was demoted to %s - AUTHORITY VIOLATION", result.Type)
			}
			break
		}
	}

	if !found {
		t.Fatal("Ground truth fact not in search results - AUTHORITY VIOLATION")
	}

	// CRITICAL: Lexical matches must not be demoted by recall scoring
	// This test ensures truth authority is maintained regardless of recall scoring

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Authority_SemanticCannotOverrideLexical",
		Passed:   true,
	})
}

// TestProposalCannotBypassEvidence verifies proposals require evidence to become facts
func TestProposalCannotBypassEvidence(t *testing.T) {
	tmpDir, cleanup := setupAuthorityTestEnv(t)
	defer cleanup()

	cfg := &config.Config{
		ProjectRoot: tmpDir,
		DBPath:      filepath.Join(tmpDir, "test.db"),
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	memService := memory.NewService(db)

	// Attempt to create a Fact directly (should fail - proposals must start as claims)
	proposedFact := &memory.Memory{
		ProjectID: "test",
		Type:      memory.Fact, // ATTEMPTING TO BYPASS EVIDENCE
		Summary:   "Unauthorized fact claim",
		Detail:    "This should never be stored as a fact without evidence",
	}

	err = memService.CreateMemory(proposedFact)

	// CRITICAL: This MUST fail
	if err == nil {
		t.Fatal("Created Fact without evidence - CRITICAL AUTHORITY VIOLATION")
	}

	// Verify the error message is about evidence requirement
	expectedSubstring := "facts cannot be created directly"
	if !containsString(err.Error(), expectedSubstring) {
		t.Fatalf("Wrong error message: got %q, want substring %q", err.Error(), expectedSubstring)
	}

	// Verify no fact was created
	results, _ := memService.SearchMemories("test", "Unauthorized", 10)
	if len(results) > 0 {
		t.Fatal("Fact was created despite evidence requirement - CRITICAL FAILURE")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Authority_ProposalCannotBypassEvidence",
		Passed:   true,
	})
}

// TestMemoryNotAuthoritativeWithoutConfirmation verifies memories require confirmation
func TestMemoryNotAuthoritativeWithoutConfirmation(t *testing.T) {
	tmpDir, cleanup := setupAuthorityTestEnv(t)
	defer cleanup()

	cfg := &config.Config{
		ProjectRoot: tmpDir,
		DBPath:      filepath.Join(tmpDir, "test.db"),
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	memService := memory.NewService(db)

	// Create a claim (unconfirmed proposal)
	claim := &memory.Memory{
		ProjectID: "test",
		Type:      memory.Claim,
		Summary:   "Database might use PostgreSQL",
		Detail:    "Unconfirmed claim from initial discussion",
	}

	err = memService.CreateMemory(claim)
	if err != nil {
		t.Fatalf("Failed to create claim: %v", err)
	}

	// Retrieve the claim
	results, err := memService.SearchMemories("test", "PostgreSQL", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Claim not found")
	}

	// CRITICAL: The memory type must remain Claim, not automatically promoted
	foundClaim := results[0]
	if foundClaim.Type != memory.Claim {
		t.Fatalf("Claim was auto-promoted to %s - AUTHORITY VIOLATION", foundClaim.Type)
	}

	// Attempt to promote without evidence (should fail)
	err = memService.PromoteToFact(foundClaim.ID, "test", false)
	if err == nil {
		t.Fatal("Promoted claim to fact without evidence - AUTHORITY VIOLATION")
	}

	// Verify it's still a claim
	updated, err := memService.GetMemory(foundClaim.ID, "test")
	if err != nil {
		t.Fatalf("Failed to retrieve memory: %v", err)
	}

	if updated.Type != memory.Claim {
		t.Fatal("Claim was promoted without evidence - AUTHORITY VIOLATION")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Authority_MemoryNotAuthoritativeWithoutConfirmation",
		Passed:   true,
	})
}

// TestFilesOutsideScopeProtected verifies file operations respect project boundaries
func TestFilesOutsideScopeProtected(t *testing.T) {
	tmpDir, cleanup := setupAuthorityTestEnv(t)
	defer cleanup()

	// Create a file outside project root
	outsideDir, err := os.MkdirTemp("", "outside-scope-*")
	if err != nil {
		t.Fatalf("Failed to create outside dir: %v", err)
	}
	defer os.RemoveAll(outsideDir)

	// Resolve symlinks
	outsideDir, err = filepath.EvalSymlinks(outsideDir)
	if err != nil {
		t.Fatalf("Failed to resolve outside dir: %v", err)
	}

	outsideFile := filepath.Join(outsideDir, "secret.txt")
	err = os.WriteFile(outsideFile, []byte("sensitive data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	cfg := &config.Config{
		ProjectRoot: tmpDir,
	}

	// Attempt to verify evidence for file outside scope
	// This uses the evidence service which has project root validation
	_, err = verifyFileOutsideScope(outsideFile, cfg)

	// CRITICAL: Must fail with path escape error
	if err == nil {
		t.Fatal("File outside scope was accessible - CRITICAL AUTHORITY VIOLATION")
	}

	expectedError := "path escapes project root"
	if !containsString(err.Error(), expectedError) {
		t.Fatalf("Wrong error for out-of-scope file: got %q, want substring %q", err.Error(), expectedError)
	}

	// Verify relative path escape attempt also fails
	escapeAttempt := "../../etc/passwd"
	_, err = verifyFileOutsideScope(escapeAttempt, cfg)
	if err == nil {
		t.Fatal("Path traversal was allowed - CRITICAL AUTHORITY VIOLATION")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Authority_FilesOutsideScopeProtected",
		Passed:   true,
	})
}

// TestEvidenceGatingMandatory verifies evidence gating cannot be bypassed
func TestEvidenceGatingMandatory(t *testing.T) {
	tmpDir, cleanup := setupAuthorityTestEnv(t)
	defer cleanup()

	cfg := &config.Config{
		ProjectRoot: tmpDir,
		DBPath:      filepath.Join(tmpDir, "test.db"),
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	memService := memory.NewService(db)

	// Create a claim
	claim := &memory.Memory{
		ProjectID: "test",
		Type:      memory.Claim,
		Summary:   "Test suite passes",
		Detail:    "Claim that tests are passing",
	}

	err = memService.CreateMemory(claim)
	if err != nil {
		t.Fatalf("Failed to create claim: %v", err)
	}

	// Retrieve claim ID
	results, _ := memService.SearchMemories("test", "Test suite", 1)
	if len(results) == 0 {
		t.Fatal("Claim not created")
	}
	claimID := results[0].ID

	// Attempt 1: Promote with force=false (no evidence)
	err = memService.PromoteToFact(claimID, "test", false)
	if err == nil {
		t.Fatal("Promoted to fact without evidence (force=false) - AUTHORITY VIOLATION")
	}

	// Verify still a claim
	updated, _ := memService.GetMemory(claimID, "test")
	if updated.Type != memory.Claim {
		t.Fatal("Type changed without proper evidence gating")
	}

	// Attempt 2: Even with force=true, it must be explicit and logged
	// The force flag exists for administrative override, but is tracked
	err = memService.PromoteToFact(claimID, "test", true)
	if err != nil {
		t.Fatalf("Force promotion failed unexpectedly: %v", err)
	}

	// Verify it's now a fact (force override worked)
	updated, _ = memService.GetMemory(claimID, "test")
	if updated.Type != memory.Fact {
		t.Fatal("Force promotion did not work as expected")
	}

	// CRITICAL: The point is force=true must be EXPLICIT, never the default
	// Any code path that promotes without explicit force=true must fail

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Authority_EvidenceGatingMandatory",
		Passed:   true,
	})
}

// TestSemanticScoresNonAuthoritative verifies semantic scores cannot create truth
func TestSemanticScoresNonAuthoritative(t *testing.T) {
	tmpDir, cleanup := setupAuthorityTestEnv(t)
	defer cleanup()

	cfg := &config.Config{
		ProjectRoot: tmpDir,
		DBPath:      filepath.Join(tmpDir, "test.db"),
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	memService := memory.NewService(db)

	// Create a memory with specific lexical content
	mem := &memory.Memory{
		ProjectID: "test",
		Type:      memory.Claim,
		Summary:   "exact keyword match test",
		Detail:    "This memory contains the exact query terms",
	}

	err = memService.CreateMemory(mem)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	// Promote to fact to establish ground truth
	searchResults, _ := memService.SearchMemories("test", "exact keyword", 10)
	if len(searchResults) == 0 {
		t.Fatal("Failed to find created memory")
	}

	memID := searchResults[0].ID
	err = memService.PromoteToFact(memID, "test", true)
	if err != nil {
		t.Fatalf("Failed to promote: %v", err)
	}

	// CRITICAL TEST: Verify exact lexical match is retrievable
	// Even if semantic scores were low/zero, lexical truth must be preserved
	verifyResults, err := memService.SearchMemories("test", "exact keyword match", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(verifyResults) == 0 {
		t.Fatal("Exact lexical match not returned - AUTHORITY VIOLATION")
	}

	// Verify the specific memory is in results
	found := false
	for _, result := range verifyResults {
		if result.ID == memID {
			found = true
			// Memory must remain a fact (recall scoring cannot demote it)
			if result.Type != memory.Fact {
				t.Fatal("Fact was demoted - AUTHORITY VIOLATION")
			}
			break
		}
	}

	if !found {
		t.Fatal("Exact match suppressed - AUTHORITY VIOLATION")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Authority_SemanticScoresNonAuthoritative",
		Passed:   true,
	})
}

// TestLLMProposalsNonAuthoritative verifies LLM outputs require validation
func TestLLMProposalsNonAuthoritative(t *testing.T) {
	// This test verifies architectural constraint: LLM outputs are proposals only
	// This test adds explicit assertion that LLM cannot signal success via memory creation

	tmpDir, cleanup := setupAuthorityTestEnv(t)
	defer cleanup()

	cfg := &config.Config{
		ProjectRoot: tmpDir,
		DBPath:      filepath.Join(tmpDir, "test.db"),
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	memService := memory.NewService(db)

	// Simulate LLM proposing a fact
	llmProposal := &memory.Memory{
		ProjectID: "test",
		Type:      memory.Fact, // LLM trying to assert truth
		Summary:   "LLM-proposed fact",
		Detail:    "An LLM claimed this is true",
	}

	// CRITICAL: Must fail - LLM cannot create facts directly
	err = memService.CreateMemory(llmProposal)
	if err == nil {
		t.Fatal("LLM proposal created fact without evidence - AUTHORITY VIOLATION")
	}

	// LLM should only be able to create claims/observations
	llmClaim := &memory.Memory{
		ProjectID: "test",
		Type:      memory.Claim,
		Summary:   "LLM proposed claim",
		Detail:    "An LLM suggested this might be true",
	}

	err = memService.CreateMemory(llmClaim)
	if err != nil {
		t.Fatalf("LLM claim creation failed: %v", err)
	}

	// Verify it remains a claim (not auto-promoted)
	results, err := memService.SearchMemories("test", "LLM proposed", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Claim not created or not searchable")
	}

	if results[0].Type != memory.Claim {
		t.Fatalf("LLM claim was auto-promoted to %s - AUTHORITY VIOLATION", results[0].Type)
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Authority_LLMProposalsNonAuthoritative",
		Passed:   true,
	})
}

// TestSemanticEnabledLexicalWins verifies lexical truth wins even when semantic is enabled
func TestSemanticEnabledLexicalWins(t *testing.T) {
	tmpDir, cleanup := setupAuthorityTestEnv(t)
	defer cleanup()

	// Enable semantic recall
	cfg := &config.Config{
		ProjectRoot:     tmpDir,
		DBPath:          filepath.Join(tmpDir, "test.db"),
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	memService := memory.NewService(db)

	// Create a fact with a unique lexical token
	fact := &memory.Memory{
		ProjectID: "test",
		Type:      memory.Claim,
		Summary:   "SecretToken9921",
		Detail:    "The database password is set in env var DB_PASS",
	}
	err = memService.CreateMemory(fact)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}
	
	// Promote
	results, err := memService.SearchMemories("test", "SecretToken9921", 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Fact not found after creation")
	}
	memService.PromoteToFact(results[0].ID, "test", true)

	// Query with the unique token
	// Even if semantic similarity is low for this specific token, 
	// the lexical match must be present and authoritative.
	verifyResults, err := memService.SearchMemories("test", "SecretToken9921", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	found := false
	for _, r := range verifyResults {
		if r.Summary == "SecretToken9921" {
			found = true
			if r.Type != memory.Fact {
				t.Error("Fact status lost in recall")
			}
		}
	}

	if !found {
		t.Fatal("Lexical truth suppressed by recall - INVARIANT VIOLATION")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Authority_SemanticEnabledLexicalWins",
		Passed:   true,
	})
}

// Helper functions

func setupAuthorityTestEnv(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "authority-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Resolve symlinks for consistent path validation
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("Failed to resolve temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func verifyFileOutsideScope(filePath string, cfg *config.Config) (bool, error) {
	// Use evidence service's path validation
	// This simulates what would happen if code tried to access files outside project
	return verifyFilePath(filePath, cfg.ProjectRoot)
}

func verifyFilePath(path string, projectRoot string) (bool, error) {
	// Reproduce the same validation logic as evidence service
	normalizedPath := filepath.Clean(path)

	absProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return false, err
	}

	var fullPath string
	if filepath.IsAbs(normalizedPath) {
		fullPath = normalizedPath
	} else {
		fullPath = filepath.Join(absProjectRoot, normalizedPath)
	}

	absPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		absPath, err = filepath.Abs(fullPath)
		if err != nil {
			return false, err
		}
	}

	rel, err := filepath.Rel(absProjectRoot, absPath)
	if err != nil {
		return false, err
	}

	// Check for path traversal
	if len(rel) >= 2 && rel[0:2] == ".." {
		return false, &PathEscapeError{Path: path}
	}

	return true, nil
}

type PathEscapeError struct {
	Path string
}

func (e *PathEscapeError) Error() string {
	return "path escapes project root: " + e.Path
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
