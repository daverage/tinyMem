# tinyMem Artifact Vault

## Overview

The Vault is tinyMem's **immutable, content-addressed storage layer** for all artifacts. It implements a write-once, read-many pattern where every piece of content is stored exactly once and never modified.

Per requirements:
- **Content-addressed storage** using SHA-256 cryptographic hash
- **Automatic deduplication** of identical content
- **Strict immutability** - no update or delete operations
- **Four artifact types** - code, diff, decision, user_input
- **No content modification** - artifacts are evidence, not memory
- **No semantic inference** - pure storage, no interpretation

## Core Principles

**"Artifacts are evidence, not memory."**

1. **Immutable** - Once written, never modified
2. **Content-Addressed** - Hash is the identifier
3. **Deduplicated** - Identical content stored once
4. **Type-Agnostic** - Content type doesn't affect hash
5. **No Delete** - Artifacts never removed (disk is cheap)
6. **No Interpretation** - Storage only, no meaning inferred

## Architecture

```
┌─────────────────────────────────────────────┐
│  User Code                                  │
│  ├─ Store("func main() {}", Code, nil)     │
│  └─ Returns: SHA-256 hash                   │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  Vault.Store()                              │
│  ├─ Compute SHA-256 hash                    │
│  ├─ Check if hash exists (deduplication)    │
│  ├─ If exists: return existing hash         │
│  └─ If new: INSERT artifact                 │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  vault_artifacts Table                      │
│  ├─ hash (PRIMARY KEY)                      │
│  ├─ content (immutable)                     │
│  ├─ content_type                            │
│  ├─ created_at                              │
│  ├─ byte_size                               │
│  └─ token_count (optional)                  │
└─────────────────────────────────────────────┘
```

## Content Types

### Defined Types

```go
const (
    ContentTypeCode      ContentType = "code"       // Source code
    ContentTypeDiff      ContentType = "diff"       // Code diffs
    ContentTypeDecision  ContentType = "decision"   // LLM decisions
    ContentTypeUserInput ContentType = "user_input" // User-pasted content
)
```

### Type Semantics

- **code** - LLM-generated source code
- **diff** - Changes/patches to existing code
- **decision** - Metadata about LLM reasoning
- **user_input** - Content explicitly provided by user

**Important:** Content type does NOT affect the hash. Same content with different types deduplicates to one artifact.

## API Reference

### Store

```go
func (v *Vault) Store(content string, contentType ContentType, tokenCount *int) (string, error)
```

Stores an artifact and returns its SHA-256 hash.

**Deduplication:** If content already exists, returns existing hash without writing.

**Parameters:**
- `content` - The content to store (never modified)
- `contentType` - One of the four valid types
- `tokenCount` - Optional token count estimate (can be nil)

**Returns:**
- `hash` - SHA-256 hex string (64 characters)
- `error` - If storage fails or invalid content type

**Example:**
```go
hash, err := vault.Store("package main", ContentTypeCode, nil)
// hash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
```

### Get

```go
func (v *Vault) Get(hash string) (*Artifact, error)
```

Retrieves an artifact by its hash.

**Returns:**
- `*Artifact` - The artifact, or nil if not found
- `error` - Only on database errors (not found is not an error)

**Example:**
```go
artifact, err := vault.Get(hash)
if artifact == nil {
    // Not found
}
```

### Exists

```go
func (v *Vault) Exists(hash string) (bool, error)
```

Checks if an artifact exists without retrieving content.

**Example:**
```go
exists, err := vault.Exists(hash)
if exists {
    // Artifact present
}
```

### GetMultiple

```go
func (v *Vault) GetMultiple(hashes []string) ([]*Artifact, error)
```

Retrieves multiple artifacts in a single call.

**Returns:** Artifacts in same order as input hashes. Missing artifacts are nil.

**Example:**
```go
artifacts, err := vault.GetMultiple([]string{hash1, hash2, hash3})
// artifacts[0] != nil, artifacts[1] == nil (missing), artifacts[2] != nil
```

### Count

```go
func (v *Vault) Count() (int, error)
```

Returns total number of artifacts in the vault.

### CountByType

```go
func (v *Vault) CountByType() (map[ContentType]int, error)
```

Returns counts grouped by content type.

**Example:**
```go
counts, err := vault.CountByType()
// counts[ContentTypeCode] = 42
// counts[ContentTypeDiff] = 10
```

## Hash Functions

### ComputeHash

```go
func ComputeHash(content string) string
```

Computes SHA-256 hash of content. Deterministic and cryptographically secure.

**Example:**
```go
hash := vault.ComputeHash("hello world")
// hash = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
```

### VerifyHash

```go
func VerifyHash(content, expectedHash string) bool
```

Verifies content matches expected hash.

**Example:**
```go
valid := vault.VerifyHash("hello world", expectedHash)
```

## Deduplication

### How It Works

1. Content is hashed using SHA-256
2. Vault checks if hash exists in database
3. If exists: returns existing hash (no write)
4. If new: inserts artifact and returns hash

### Deduplication Guarantees

✓ **Byte-for-byte identical content** is stored once  
✓ **Same content, different type** deduplicates (hash is content-only)  
✓ **Case-sensitive** - "Hello" and "hello" are different  
✓ **Whitespace-sensitive** - "a b" and "a  b" are different  

### Example

```go
// First store
hash1, _ := vault.Store("func main() {}", ContentTypeCode, nil)

// Second store - identical content, same type
hash2, _ := vault.Store("func main() {}", ContentTypeCode, nil)

// Third store - identical content, DIFFERENT type
hash3, _ := vault.Store("func main() {}", ContentTypeDiff, nil)

// All three return the same hash
assert(hash1 == hash2 == hash3)

// Only ONE artifact exists in database
count, _ := vault.Count() // count = 1
```

## Immutability

### No Update Operations

The Vault has **zero methods** that modify existing artifacts:

✗ `Update(hash, newContent)` - Does not exist  
✗ `Modify(hash, changes)` - Does not exist  
✗ `Delete(hash)` - Does not exist  

### Write-Once Pattern

```go
// Store artifact
hash, _ := vault.Store("original content", ContentTypeCode, nil)

// To "update" content, store new artifact
newHash, _ := vault.Store("modified content", ContentTypeCode, nil)

// Both artifacts exist independently
original, _ := vault.Get(hash)     // Still accessible
modified, _ := vault.Get(newHash)  // New artifact
```

### Immutability Verification

**Unit Test:** `TestImmutability`
```go
// Stores two different contents
// Verifies both artifacts exist unchanged
// Confirms no modification occurred
```

## Performance

### Hash Computation

```
BenchmarkComputeHash-8   9,886,486 ops   118.9 ns/op
```

SHA-256 is extremely fast. Hash computation is not a bottleneck.

### Deduplication Check

```
BenchmarkStoreDedup-8    272,275 ops     4,317 ns/op
```

Deduplication check includes:
- Hash computation (~119 ns)
- Database lookup (~4,200 ns)

Database lookup dominates. Still very fast for typical usage.

### Storage Growth

- **Linear growth** - One artifact per unique content
- **Deduplication ratio** depends on workload
- **No automatic cleanup** - vault grows unbounded
- **Disk is cheap** - design assumes ample storage

## Error Handling

### Invalid Content Type

```go
_, err := vault.Store("content", ContentType("invalid"), nil)
// err != nil: "invalid content type: invalid"
```

### Database Errors

```go
artifact, err := vault.Get(hash)
if err != nil {
    // Database error (connection lost, disk full, etc.)
}
```

### Not Found (Not an Error)

```go
artifact, err := vault.Get("nonexistent_hash")
// err == nil
// artifact == nil
```

Not found is a **valid state**, not an error condition.

## Testing

### Unit Tests

All tests use in-memory SQLite (`":memory:"`).

**Coverage:**
- ✓ Hash computation determinism
- ✓ Hash verification
- ✓ Store and retrieve
- ✓ Deduplication (same type)
- ✓ Deduplication (different types)
- ✓ Immutability guarantees
- ✓ No delete operations exist
- ✓ Invalid content type rejection
- ✓ All valid content types
- ✓ Non-existent artifact retrieval
- ✓ Existence checking
- ✓ Batch retrieval
- ✓ Batch retrieval with missing
- ✓ Count by type
- ✓ Token count storage

### Running Tests

```bash
# Run all tests
go test ./internal/vault/

# Verbose output
go test -v ./internal/vault/

# With coverage
go test -cover ./internal/vault/

# Benchmarks
go test -bench=. ./internal/vault/
```

## Usage Examples

### Basic Usage

```go
import "github.com/andrzejmarczewski/tinyMem/internal/vault"

// Create vault (requires database connection)
v := vault.New(db.Conn())

// Store code
hash, err := v.Store("package main", vault.ContentTypeCode, nil)
if err != nil {
    log.Fatal(err)
}

// Retrieve code
artifact, err := v.Get(hash)
if artifact != nil {
    fmt.Println(artifact.Content) // "package main"
}
```

### With Token Count

```go
tokenCount := 150
hash, _ := v.Store(content, vault.ContentTypeCode, &tokenCount)

artifact, _ := v.Get(hash)
if artifact.TokenCount != nil {
    fmt.Printf("Tokens: %d\n", *artifact.TokenCount)
}
```

### User Input Handling

```go
// User-pasted code always uses user_input type
userCode := getUserInput()
hash, _ := v.Store(userCode, vault.ContentTypeUserInput, nil)

// Per spec: user-pasted code takes precedence
// This is enforced by runtime, not vault
```

### Batch Operations

```go
// Store multiple
hashes := []string{}
for _, content := range contents {
    hash, _ := v.Store(content, vault.ContentTypeCode, nil)
    hashes = append(hashes, hash)
}

// Retrieve multiple
artifacts, _ := v.GetMultiple(hashes)
```

## Design Rationale

### Why SHA-256?

- **Cryptographically secure** - Collision resistance
- **Deterministic** - Same input = same output
- **Fast** - ~119 ns per hash on modern hardware
- **Standard** - Well-understood, widely supported

### Why No Delete?

**"Disk is cheap, RAM is bounded."**

- **Audit trail** - Vault is evidence log
- **State recovery** - Can rebuild state from vault
- **Simplicity** - No lifecycle management needed
- **Safety** - No accidental data loss

### Why Deduplicate?

- **Storage efficiency** - LLMs often regenerate similar code
- **Idempotency** - Re-running operations doesn't duplicate
- **Performance** - Reduces database size

### Why Content-Addressed?

- **Integrity** - Hash verifies content
- **Deduplication** - Natural key for duplicate detection
- **Reproducibility** - Same content always has same hash

## Troubleshooting

### Hash Mismatch

**Problem:** Content doesn't match expected hash

**Cause:** Content was modified

**Solution:**
```go
if !vault.VerifyHash(content, expectedHash) {
    // Content was corrupted or modified
}
```

### Deduplication Not Working

**Problem:** Same content stored multiple times

**Cause:** Whitespace or encoding differences

**Solution:** Normalize content before storing
```go
normalized := strings.TrimSpace(content)
hash, _ := vault.Store(normalized, ...)
```

### Database Growth

**Problem:** Vault growing too large

**Mitigation:**
- Monitor disk usage
- Plan for storage capacity
- Consider external archival (not implemented)

## Future Considerations

**Not Currently Implemented:**

- Compression (content stored verbatim)
- Encryption (content stored in plaintext)
- Archival/purge (no automatic cleanup)
- Sharding (single database only)

These are intentionally **not** included per spec requirements for simplicity and correctness.

## Philosophy

**"The Vault does not remember what was said. It preserves what was written."**

The Vault is pure storage with zero intelligence:

✗ No summarization  
✗ No interpretation  
✗ No semantic analysis  
✗ No content modification  
✗ No lifecycle management  

✓ Store bytes  
✓ Retrieve bytes  
✓ Verify integrity  
✓ Never modify  

This simplicity is intentional and critical for system correctness.
