package vault

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// ContentType represents the type of artifact stored in the vault
type ContentType string

const (
	ContentTypeCode      ContentType = "code"
	ContentTypeDiff      ContentType = "diff"
	ContentTypeDecision  ContentType = "decision"
	ContentTypeUserPaste ContentType = "user_paste"
)

// Artifact represents a single immutable piece of content in the vault
// Per spec: all artifacts are immutable, never modified, never deleted automatically
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
type Vault struct {
	db *sql.DB
}

// New creates a new Vault instance
func New(db *sql.DB) *Vault {
	return &Vault{db: db}
}

// Store saves an artifact to the vault
// Returns the content hash. If artifact already exists, returns existing hash (deduplication)
// Per spec: content-addressed storage, deduplicated by hash
func (v *Vault) Store(content string, contentType ContentType, tokenCount *int) (string, error) {
	// Calculate content hash (SHA-256)
	hash := computeHash(content)

	// Check if already exists (deduplication)
	var existingHash string
	err := v.db.QueryRow("SELECT hash FROM vault_artifacts WHERE hash = ?", hash).Scan(&existingHash)
	if err == nil {
		// Already exists, return existing hash
		return existingHash, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to check for existing artifact: %w", err)
	}

	// Insert new artifact
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

// computeHash calculates SHA-256 hash of content
func computeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// ComputeHash is exported for use by other packages
func ComputeHash(content string) string {
	return computeHash(content)
}
