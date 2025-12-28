package state

import (
	"database/sql"
	"testing"

	"github.com/andrzejmarczewski/tslp/internal/entity"
	"github.com/andrzejmarczewski/tslp/internal/ledger"
	"github.com/andrzejmarczewski/tslp/internal/vault"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestVault(t *testing.T) (*vault.Vault, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create vault table
	_, err = db.Exec(`
		CREATE TABLE vault_artifacts (
			hash TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			content_type TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			byte_size INTEGER NOT NULL,
			token_count INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create vault table: %v", err)
	}

	v := vault.New(db)
	cleanup := func() { db.Close() }
	return v, cleanup
}

func TestCheckStructuralParity_NewEntity(t *testing.T) {
	v, cleanup := setupTestVault(t)
	defer cleanup()

	// No current state - parity should pass
	resolution := &entity.Resolution{
		Symbols:      []string{"NewFunction"},
		Confidence:   "CONFIRMED",
		ASTNodeCount: intPtr(10),
	}

	result, err := CheckStructuralParity(nil, resolution, v)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.OK {
		t.Errorf("Expected parity OK for new entity, got: %s", result.Reason)
	}
}

func TestCheckStructuralParity_InferredEntity(t *testing.T) {
	v, cleanup := setupTestVault(t)
	defer cleanup()

	// Store current artifact
	currentHash, _ := v.Store("old content", vault.ContentTypeCode, nil)

	currentState := &EntityState{
		EntityKey:    "file.go::Foo",
		ArtifactHash: currentHash,
		Confidence:   "CONFIRMED",
		State:        ledger.StateAuthoritative,
		Metadata: map[string]interface{}{
			"symbols": []string{"Foo", "Bar"},
		},
	}

	// INFERRED resolution - parity should pass without checks
	resolution := &entity.Resolution{
		Symbols:    []string{"Foo"}, // Missing Bar
		Confidence: "INFERRED",
	}

	result, err := CheckStructuralParity(currentState, resolution, v)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.OK {
		t.Errorf("Expected parity OK for INFERRED entity, got: %s", result.Reason)
	}
}

func TestCheckStructuralParity_AllSymbolsPresent(t *testing.T) {
	v, cleanup := setupTestVault(t)
	defer cleanup()

	// Store current artifact
	currentHash, _ := v.Store("old content", vault.ContentTypeCode, nil)

	currentState := &EntityState{
		EntityKey:    "file.go::Foo",
		ArtifactHash: currentHash,
		Confidence:   "CONFIRMED",
		State:        ledger.StateAuthoritative,
		Metadata: map[string]interface{}{
			"symbols": []string{"Foo", "Bar"},
		},
	}

	// New resolution with all symbols (superset is OK)
	resolution := &entity.Resolution{
		Symbols:      []string{"Foo", "Bar", "Baz"},
		Confidence:   "CONFIRMED",
		ASTNodeCount: intPtr(20),
	}

	result, err := CheckStructuralParity(currentState, resolution, v)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.OK {
		t.Errorf("Expected parity OK when all symbols present, got: %s", result.Reason)
	}
}

func TestCheckStructuralParity_MissingSymbols(t *testing.T) {
	v, cleanup := setupTestVault(t)
	defer cleanup()

	// Store current artifact
	currentHash, _ := v.Store("old content", vault.ContentTypeCode, nil)

	currentState := &EntityState{
		EntityKey:    "file.go::Foo",
		ArtifactHash: currentHash,
		Confidence:   "CONFIRMED",
		State:        ledger.StateAuthoritative,
		Metadata: map[string]interface{}{
			"symbols": []string{"Foo", "Bar", "Baz"},
		},
	}

	// New resolution missing "Bar"
	resolution := &entity.Resolution{
		Symbols:      []string{"Foo", "Baz"},
		Confidence:   "CONFIRMED",
		ASTNodeCount: intPtr(20),
	}

	result, err := CheckStructuralParity(currentState, resolution, v)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.OK {
		t.Error("Expected parity failure for missing symbols")
	}

	if len(result.MissingSymbols) != 1 {
		t.Errorf("Expected 1 missing symbol, got %d", len(result.MissingSymbols))
	}

	if result.MissingSymbols[0] != "Bar" {
		t.Errorf("Expected missing symbol 'Bar', got '%s'", result.MissingSymbols[0])
	}
}

func TestCheckStructuralParity_NoSymbols(t *testing.T) {
	v, cleanup := setupTestVault(t)
	defer cleanup()

	// Store current artifact
	currentHash, _ := v.Store("old content", vault.ContentTypeCode, nil)

	currentState := &EntityState{
		EntityKey:    "file.go::Foo",
		ArtifactHash: currentHash,
		Confidence:   "CONFIRMED",
		State:        ledger.StateAuthoritative,
		Metadata: map[string]interface{}{
			"symbols": []string{"Foo"},
		},
	}

	// New resolution with no symbols - collapse
	resolution := &entity.Resolution{
		Symbols:      []string{},
		Confidence:   "CONFIRMED",
		ASTNodeCount: nil,
	}

	result, err := CheckStructuralParity(currentState, resolution, v)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.OK {
		t.Error("Expected parity failure for structure collapse")
	}

	if !result.CollapsedStructure {
		t.Error("Expected CollapsedStructure flag to be true")
	}
}

func TestCheckStructuralParity_ASTNodeCountCollapse(t *testing.T) {
	v, cleanup := setupTestVault(t)
	defer cleanup()

	// Store current artifact
	currentHash, _ := v.Store("old content", vault.ContentTypeCode, nil)

	currentState := &EntityState{
		EntityKey:    "file.go::Foo",
		ArtifactHash: currentHash,
		Confidence:   "CONFIRMED",
		State:        ledger.StateAuthoritative,
		Metadata: map[string]interface{}{
			"symbols":         []string{"Foo"},
			"ast_node_count":  float64(100), // JSON unmarshals numbers as float64
		},
	}

	// New resolution with drastically reduced node count (40 < 50% of 100)
	resolution := &entity.Resolution{
		Symbols:      []string{"Foo"},
		Confidence:   "CONFIRMED",
		ASTNodeCount: intPtr(40),
	}

	result, err := CheckStructuralParity(currentState, resolution, v)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.OK {
		t.Error("Expected parity failure for AST node count collapse")
	}

	if !result.CollapsedStructure {
		t.Error("Expected CollapsedStructure flag to be true")
	}
}

func TestCheckStructuralParity_ASTNodeCountOK(t *testing.T) {
	v, cleanup := setupTestVault(t)
	defer cleanup()

	// Store current artifact
	currentHash, _ := v.Store("old content", vault.ContentTypeCode, nil)

	currentState := &EntityState{
		EntityKey:    "file.go::Foo",
		ArtifactHash: currentHash,
		Confidence:   "CONFIRMED",
		State:        ledger.StateAuthoritative,
		Metadata: map[string]interface{}{
			"symbols":        []string{"Foo"},
			"ast_node_count": float64(100),
		},
	}

	// New resolution with acceptable node count (60 >= 50% of 100)
	resolution := &entity.Resolution{
		Symbols:      []string{"Foo"},
		Confidence:   "CONFIRMED",
		ASTNodeCount: intPtr(60),
	}

	result, err := CheckStructuralParity(currentState, resolution, v)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.OK {
		t.Errorf("Expected parity OK for acceptable node count, got: %s", result.Reason)
	}
}

func intPtr(i int) *int {
	return &i
}
