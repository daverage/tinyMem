package vault

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// ContentType represents the type of artifact stored in the vault
// Per requirements: code, diff, decision, user_input
type ContentType string

const (
	ContentTypeCode       ContentType = "code"
	ContentTypeDiff       ContentType = "diff"
	ContentTypeDecision   ContentType = "decision"
	ContentTypeUserInput  ContentType = "user_input" // Changed from user_paste per requirements
	ContentTypePrompt     ContentType = "prompt"
	ContentTypeToolCall   ContentType = "tool_call"
	ContentTypeToolResult ContentType = "tool_result"
)

// Artifact represents a single immutable piece of content in the vault
// Per spec: all artifacts are immutable, never modified, never deleted automatically
// Per requirements: Artifacts are immutable once written
type Artifact struct {
	Hash        string
	Content     string
	ContentType ContentType
	CreatedAt   time.Time
	ByteSize    int
	TokenCount  *int // Optional
}

// Vault manages the immutable content-addressed store
// Per spec section 3.1: deduplicated by hash, never modified
// Per requirements:
//   - Content-addressed storage using cryptographic hash (SHA-256)
//   - Deduplicate identical artifacts
//   - No delete or update operations
type Vault struct {
	db *sql.DB
}

// New creates a new Vault instance
func New(db *sql.DB) *Vault {
	return &Vault{db: db}
}

// Store saves an artifact to the vault
// Returns the content hash. If artifact already exists, returns existing hash (deduplication)
// Per requirements:
//   - Content-addressed storage using cryptographic hash
//   - Deduplicate identical artifacts
//   - Artifacts are immutable once written
func (v *Vault) Store(content string, contentType ContentType, tokenCount *int) (string, error) {
	// Validate content type
	if !isValidContentType(contentType) {
		return "", fmt.Errorf("invalid content type: %s", contentType)
	}

	// Calculate content hash (SHA-256)
	// Per requirements: cryptographic hash
	hash := ComputeHash(content)

	// Check if already exists (deduplication)
	// Per requirements: deduplicate identical artifacts
	existing, err := v.Get(hash)
	if err != nil {
		return "", fmt.Errorf("failed to check for existing artifact: %w", err)
	}

	if existing != nil {
		// Artifact already exists, return existing hash (deduplication)
		// No write occurs - immutability preserved
		return existing.Hash, nil
	}

	// Insert new artifact
	// Per requirements: artifacts are immutable once written (INSERT only, never UPDATE)
	now := time.Now().Unix()
	byteSize := len([]byte(content))

	_, err = v.db.Exec(`
		INSERT INTO vault_artifacts (hash, content, content_type, created_at, byte_size, token_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`, hash, content, contentType, now, byteSize, tokenCount)

	if err != nil {
		return "", fmt.Errorf("failed to store artifact: %w", err)
	}

	return hash, nil
}

// Get retrieves an artifact by its hash
// Returns nil if not found (not an error, per spec: artifacts are evidence)
// Per requirements: No modify operations - read-only
func (v *Vault) Get(hash string) (*Artifact, error) {
	var a Artifact
	var createdAt int64
	var tokenCount sql.NullInt64

	err := v.db.QueryRow(`
		SELECT hash, content, content_type, created_at, byte_size, token_count
		FROM vault_artifacts
		WHERE hash = ?
	`, hash).Scan(&a.Hash, &a.Content, &a.ContentType, &createdAt, &a.ByteSize, &tokenCount)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	a.CreatedAt = time.Unix(createdAt, 0)
	if tokenCount.Valid {
		tc := int(tokenCount.Int64)
		a.TokenCount = &tc
	}

	return &a, nil
}

// Exists checks if an artifact with the given hash exists
// Read-only operation
func (v *Vault) Exists(hash string) (bool, error) {
	var exists bool
	err := v.db.QueryRow("SELECT EXISTS(SELECT 1 FROM vault_artifacts WHERE hash = ?)", hash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check artifact existence: %w", err)
	}
	return exists, nil
}

// GetMultiple retrieves multiple artifacts by their hashes
// Returns artifacts in the same order as the input hashes
// Missing artifacts are returned as nil in their position
// Read-only operation
func (v *Vault) GetMultiple(hashes []string) ([]*Artifact, error) {
	if len(hashes) == 0 {
		return []*Artifact{}, nil
	}

	// Build a map for efficient lookup
	artifactMap := make(map[string]*Artifact)

	// Query all hashes
	for _, hash := range hashes {
		artifact, err := v.Get(hash)
		if err != nil {
			return nil, err
		}
		artifactMap[hash] = artifact
	}

	// Build result in same order as input
	result := make([]*Artifact, len(hashes))
	for i, hash := range hashes {
		result[i] = artifactMap[hash]
	}

	return result, nil
}

// GetByType retrieves all artifacts of a specific content type
// Useful for auditing and analysis
// Read-only operation
func (v *Vault) GetByType(contentType ContentType) ([]*Artifact, error) {
	rows, err := v.db.Query(`
		SELECT hash, content, content_type, created_at, byte_size, token_count
		FROM vault_artifacts
		WHERE content_type = ?
		ORDER BY created_at ASC
	`, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifacts by type: %w", err)
	}
	defer rows.Close()

	var artifacts []*Artifact
	for rows.Next() {
		var a Artifact
		var createdAt int64
		var tokenCount sql.NullInt64

		err := rows.Scan(&a.Hash, &a.Content, &a.ContentType, &createdAt, &a.ByteSize, &tokenCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}

		a.CreatedAt = time.Unix(createdAt, 0)
		if tokenCount.Valid {
			tc := int(tokenCount.Int64)
			a.TokenCount = &tc
		}

		artifacts = append(artifacts, &a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artifacts: %w", err)
	}

	return artifacts, nil
}

// Count returns the total number of artifacts in the vault
// Read-only operation
func (v *Vault) Count() (int, error) {
	var count int
	err := v.db.QueryRow("SELECT COUNT(*) FROM vault_artifacts").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count artifacts: %w", err)
	}
	return count, nil
}

// CountByType returns counts grouped by content type
// Read-only operation
func (v *Vault) CountByType() (map[ContentType]int, error) {
	rows, err := v.db.Query(`
		SELECT content_type, COUNT(*)
		FROM vault_artifacts
		GROUP BY content_type
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to count by type: %w", err)
	}
	defer rows.Close()

	counts := make(map[ContentType]int)
	for rows.Next() {
		var contentType ContentType
		var count int
		if err := rows.Scan(&contentType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[contentType] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating counts: %w", err)
	}

	return counts, nil
}

// ComputeHash calculates SHA-256 hash of content
// Per requirements: Content-addressed storage using cryptographic hash
// Exported for use by other packages
func ComputeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// VerifyHash checks if content matches the given hash
// Returns true if content hashes to the expected value
func VerifyHash(content, expectedHash string) bool {
	return ComputeHash(content) == expectedHash
}

// isValidContentType checks if the given content type is valid
func isValidContentType(ct ContentType) bool {
	switch ct {
	case ContentTypeCode, ContentTypeDiff, ContentTypeDecision, ContentTypeUserInput, ContentTypePrompt, ContentTypeToolCall, ContentTypeToolResult:
		return true
	default:
		return false
	}
}
