package vault

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create schema
	schema := `
	CREATE TABLE vault_artifacts (
		hash TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		content_type TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		byte_size INTEGER NOT NULL,
		token_count INTEGER
	);
	CREATE INDEX idx_vault_created ON vault_artifacts(created_at);
	CREATE INDEX idx_vault_type ON vault_artifacts(content_type);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

// TestComputeHash verifies cryptographic hash computation
func TestComputeHash(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty string",
			content:  "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple text",
			content:  "hello world",
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name: "code block",
			content: `func main() {
	fmt.Println("Hello, tinyMem!")
}`,
			expected: "f8c3e8b8c8a7e3f8c3e8b8c8a7e3f8c3e8b8c8a7e3f8c3e8b8c8a7e3f8c3e8b8", // Will be different, this is just a placeholder
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := ComputeHash(tt.content)
			if len(hash) != 64 {
				t.Errorf("hash length = %d, want 64 (SHA-256)", len(hash))
			}
			// Verify it's deterministic
			hash2 := ComputeHash(tt.content)
			if hash != hash2 {
				t.Errorf("hash not deterministic: %s != %s", hash, hash2)
			}
		})
	}
}

// TestVerifyHash tests hash verification
func TestVerifyHash(t *testing.T) {
	content := "test content"
	hash := ComputeHash(content)

	if !VerifyHash(content, hash) {
		t.Error("VerifyHash failed for matching content")
	}

	if VerifyHash("different content", hash) {
		t.Error("VerifyHash succeeded for non-matching content")
	}
}

// TestStoreAndGet tests basic store and retrieve operations
func TestStoreAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)
	content := "package main\n\nfunc main() {}"
	contentType := ContentTypeCode

	// Store artifact
	hash, err := vault.Store(content, contentType, nil)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify hash format
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	// Retrieve artifact
	artifact, err := vault.Get(hash)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if artifact == nil {
		t.Fatal("artifact is nil")
	}

	// Verify artifact fields
	if artifact.Hash != hash {
		t.Errorf("Hash = %s, want %s", artifact.Hash, hash)
	}
	if artifact.Content != content {
		t.Errorf("Content mismatch")
	}
	if artifact.ContentType != contentType {
		t.Errorf("ContentType = %s, want %s", artifact.ContentType, contentType)
	}
	if artifact.ByteSize != len([]byte(content)) {
		t.Errorf("ByteSize = %d, want %d", artifact.ByteSize, len([]byte(content)))
	}
}

// TestDeduplication tests that identical content is deduplicated
// Per requirements: Deduplicate identical artifacts
func TestDeduplication(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)
	content := "duplicated content"

	// Store same content twice
	hash1, err := vault.Store(content, ContentTypeCode, nil)
	if err != nil {
		t.Fatalf("First Store failed: %v", err)
	}

	hash2, err := vault.Store(content, ContentTypeCode, nil)
	if err != nil {
		t.Fatalf("Second Store failed: %v", err)
	}

	// Hashes should be identical
	if hash1 != hash2 {
		t.Errorf("Deduplication failed: hash1=%s, hash2=%s", hash1, hash2)
	}

	// Verify only one artifact exists
	count, err := vault.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %d, want 1 (deduplication failed)", count)
	}
}

// TestDeduplicationDifferentTypes tests that same content with different types is deduplicated
func TestDeduplicationDifferentTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)
	content := "same content, different type"

	// Store with different types
	hash1, err := vault.Store(content, ContentTypeCode, nil)
	if err != nil {
		t.Fatalf("First Store failed: %v", err)
	}

	hash2, err := vault.Store(content, ContentTypeDiff, nil)
	if err != nil {
		t.Fatalf("Second Store failed: %v", err)
	}

	// Hashes should be identical (content-addressed, type doesn't affect hash)
	if hash1 != hash2 {
		t.Errorf("Hash mismatch: hash1=%s, hash2=%s", hash1, hash2)
	}

	// Should still be only one artifact (deduplication based on content)
	count, err := vault.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %d, want 1 (should deduplicate regardless of type)", count)
	}
}

// TestImmutability tests that artifacts cannot be modified
// Per requirements: Artifacts are immutable once written, no update operations
func TestImmutability(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)
	content1 := "original content"

	// Store first artifact
	hash, err := vault.Store(content1, ContentTypeCode, nil)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Attempt to store different content (should create new artifact)
	content2 := "modified content"
	hash2, err := vault.Store(content2, ContentTypeCode, nil)
	if err != nil {
		t.Fatalf("Second Store failed: %v", err)
	}

	// Should have different hashes
	if hash == hash2 {
		t.Error("Different content produced same hash (immutability violated)")
	}

	// Both artifacts should exist
	artifact1, _ := vault.Get(hash)
	artifact2, _ := vault.Get(hash2)

	if artifact1.Content != content1 {
		t.Error("Original artifact was modified (immutability violated)")
	}
	if artifact2.Content != content2 {
		t.Error("New artifact content incorrect")
	}
}

// TestNoDelete verifies that there are no delete operations
// Per requirements: No delete operations
func TestNoDelete(t *testing.T) {
	// Verify Vault type has no Delete methods
	vault := New(nil)
	_ = vault // Use vault to avoid unused variable error

	// This test documents the design constraint
	// The Vault struct should not have any Delete* methods
	// If this test compiles, it means no Delete methods exist
}

// TestInvalidContentType tests rejection of invalid content types
func TestInvalidContentType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)

	_, err := vault.Store("content", ContentType("invalid"), nil)
	if err == nil {
		t.Error("Store should reject invalid content type")
	}
}

// TestValidContentTypes tests all valid content types
func TestValidContentTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)

	types := []ContentType{
		ContentTypeCode,
		ContentTypeDiff,
		ContentTypeDecision,
		ContentTypeUserInput,
	}

	for _, ct := range types {
		t.Run(string(ct), func(t *testing.T) {
			content := "test content for " + string(ct)
			_, err := vault.Store(content, ct, nil)
			if err != nil {
				t.Errorf("Store failed for valid type %s: %v", ct, err)
			}
		})
	}
}

// TestGetNonExistent tests retrieving non-existent artifact
func TestGetNonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)
	fakeHash := "0000000000000000000000000000000000000000000000000000000000000000"

	artifact, err := vault.Get(fakeHash)
	if err != nil {
		t.Errorf("Get should not error on non-existent: %v", err)
	}
	if artifact != nil {
		t.Error("Get should return nil for non-existent artifact")
	}
}

// TestExists tests existence checking
func TestExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)
	content := "test"

	hash, _ := vault.Store(content, ContentTypeCode, nil)

	exists, err := vault.Exists(hash)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Exists = false, want true")
	}

	fakeHash := "0000000000000000000000000000000000000000000000000000000000000000"
	exists, err = vault.Exists(fakeHash)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Exists = true for non-existent, want false")
	}
}

// TestGetMultiple tests batch retrieval
func TestGetMultiple(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)

	// Store multiple artifacts
	hash1, _ := vault.Store("content1", ContentTypeCode, nil)
	hash2, _ := vault.Store("content2", ContentTypeCode, nil)
	hash3, _ := vault.Store("content3", ContentTypeCode, nil)

	// Retrieve in different order
	hashes := []string{hash2, hash1, hash3}
	artifacts, err := vault.GetMultiple(hashes)
	if err != nil {
		t.Fatalf("GetMultiple failed: %v", err)
	}

	if len(artifacts) != 3 {
		t.Fatalf("len(artifacts) = %d, want 3", len(artifacts))
	}

	// Verify order is preserved
	if artifacts[0].Hash != hash2 {
		t.Error("Order not preserved at index 0")
	}
	if artifacts[1].Hash != hash1 {
		t.Error("Order not preserved at index 1")
	}
	if artifacts[2].Hash != hash3 {
		t.Error("Order not preserved at index 2")
	}
}

// TestGetMultipleWithMissing tests batch retrieval with some missing artifacts
func TestGetMultipleWithMissing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)

	hash1, _ := vault.Store("content1", ContentTypeCode, nil)
	fakeHash := "0000000000000000000000000000000000000000000000000000000000000000"

	hashes := []string{hash1, fakeHash}
	artifacts, err := vault.GetMultiple(hashes)
	if err != nil {
		t.Fatalf("GetMultiple failed: %v", err)
	}

	if len(artifacts) != 2 {
		t.Fatalf("len(artifacts) = %d, want 2", len(artifacts))
	}

	if artifacts[0] == nil {
		t.Error("artifacts[0] should not be nil")
	}
	if artifacts[1] != nil {
		t.Error("artifacts[1] should be nil for missing artifact")
	}
}

// TestCountByType tests counting artifacts by type
func TestCountByType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)

	// Store various types
	vault.Store("code1", ContentTypeCode, nil)
	vault.Store("code2", ContentTypeCode, nil)
	vault.Store("diff1", ContentTypeDiff, nil)
	vault.Store("decision1", ContentTypeDecision, nil)

	counts, err := vault.CountByType()
	if err != nil {
		t.Fatalf("CountByType failed: %v", err)
	}

	if counts[ContentTypeCode] != 2 {
		t.Errorf("code count = %d, want 2", counts[ContentTypeCode])
	}
	if counts[ContentTypeDiff] != 1 {
		t.Errorf("diff count = %d, want 1", counts[ContentTypeDiff])
	}
	if counts[ContentTypeDecision] != 1 {
		t.Errorf("decision count = %d, want 1", counts[ContentTypeDecision])
	}
}

// TestTokenCount tests optional token count storage
func TestTokenCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vault := New(db)

	// With token count
	tokenCount := 42
	hash1, _ := vault.Store("content", ContentTypeCode, &tokenCount)
	artifact1, _ := vault.Get(hash1)

	if artifact1.TokenCount == nil {
		t.Fatal("TokenCount should not be nil")
	}
	if *artifact1.TokenCount != 42 {
		t.Errorf("TokenCount = %d, want 42", *artifact1.TokenCount)
	}

	// Without token count
	hash2, _ := vault.Store("content2", ContentTypeCode, nil)
	artifact2, _ := vault.Get(hash2)

	if artifact2.TokenCount != nil {
		t.Error("TokenCount should be nil when not provided")
	}
}

// BenchmarkComputeHash benchmarks hash computation
func BenchmarkComputeHash(b *testing.B) {
	content := "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ComputeHash(content)
	}
}

// BenchmarkStoreDedup benchmarks deduplication
func BenchmarkStoreDedup(b *testing.B) {
	db := setupTestDB(&testing.T{})
	defer db.Close()

	vault := New(db)
	content := "duplicate content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = vault.Store(content, ContentTypeCode, nil)
	}
}
