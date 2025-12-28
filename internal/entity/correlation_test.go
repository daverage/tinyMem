package entity

import (
	"testing"
)

// MockStateMap implements StateMapProvider for testing
type MockStateMap struct {
	entities []StateEntity
}

func (m *MockStateMap) GetAllEntities() ([]StateEntity, error) {
	return m.entities, nil
}

func TestResolveViaCorrelation_SingleMatch(t *testing.T) {
	resolver := &Resolver{}

	mockState := &MockStateMap{
		entities: []StateEntity{
			{
				EntityKey: "file.go::ProcessData",
				Symbols:   []string{"ProcessData", "validateInput", "transformOutput"},
			},
		},
	}

	// Code that references the symbols
	code := `func ProcessData(input string) string {
		if !validateInput(input) {
			return ""
		}
		return transformOutput(input)
	}`

	resolution := resolver.resolveViaCorrelation("test-hash", code, mockState)

	if resolution.Confidence != ConfidenceInferred {
		t.Errorf("Expected INFERRED confidence from correlation, got %s", resolution.Confidence)
	}

	if resolution.Method != MethodCorrelation {
		t.Errorf("Expected MethodCorrelation, got %s", resolution.Method)
	}

	if resolution.EntityKey == nil {
		t.Fatal("EntityKey should not be nil for single match")
	}

	if *resolution.EntityKey != "file.go::ProcessData" {
		t.Errorf("Expected entity key 'file.go::ProcessData', got '%s'", *resolution.EntityKey)
	}
}

func TestResolveViaCorrelation_NoMatch(t *testing.T) {
	resolver := &Resolver{}

	mockState := &MockStateMap{
		entities: []StateEntity{
			{
				EntityKey: "file.go::ProcessData",
				Symbols:   []string{"ProcessData"},
			},
		},
	}

	// Code with no matching symbols
	code := `func CompletelyDifferent() {
		println("nothing matches")
	}`

	resolution := resolver.resolveViaCorrelation("test-hash", code, mockState)

	if resolution.Confidence != ConfidenceUnresolved {
		t.Errorf("Expected UNRESOLVED when no match, got %s", resolution.Confidence)
	}

	if resolution.EntityKey != nil {
		t.Error("EntityKey should be nil when no match")
	}
}

func TestResolveViaCorrelation_AmbiguousMatch(t *testing.T) {
	resolver := &Resolver{}

	mockState := &MockStateMap{
		entities: []StateEntity{
			{
				EntityKey: "file1.go::Helper",
				Symbols:   []string{"Helper"},
			},
			{
				EntityKey: "file2.go::Utility",
				Symbols:   []string{"Utility"},
			},
		},
	}

	// Code that matches both equally
	code := `func DoSomething() {
		Helper()
		Utility()
	}`

	resolution := resolver.resolveViaCorrelation("test-hash", code, mockState)

	// Ambiguous match should return UNRESOLVED
	if resolution.Confidence != ConfidenceUnresolved {
		t.Errorf("Expected UNRESOLVED for ambiguous match, got %s", resolution.Confidence)
	}

	if resolution.EntityKey != nil {
		t.Error("EntityKey should be nil for ambiguous match")
	}
}

func TestResolveViaCorrelation_LowScore(t *testing.T) {
	resolver := &Resolver{}

	mockState := &MockStateMap{
		entities: []StateEntity{
			{
				EntityKey: "file.go::ComplexFunction",
				Symbols:   []string{"foo", "bar", "baz", "qux", "quux"},
			},
		},
	}

	// Code that only matches one symbol out of five (20% < 50% threshold)
	code := `func Something() {
		foo()
	}`

	resolution := resolver.resolveViaCorrelation("test-hash", code, mockState)

	if resolution.Confidence != ConfidenceUnresolved {
		t.Errorf("Expected UNRESOLVED for low score, got %s", resolution.Confidence)
	}
}

func TestResolveViaCorrelation_NilStateMap(t *testing.T) {
	resolver := &Resolver{}

	resolution := resolver.resolveViaCorrelation("test-hash", "some code", nil)

	if resolution.Confidence != ConfidenceUnresolved {
		t.Errorf("Expected UNRESOLVED with nil state map, got %s", resolution.Confidence)
	}
}

func TestExtractTokens(t *testing.T) {
	code := `func ProcessData(input string) {
		validateInput(input)
		return transformOutput(input)
	}`

	tokens := extractTokens(code)

	expectedTokens := []string{"func", "ProcessData", "input", "string", "validateInput", "transformOutput"}

	for _, expected := range expectedTokens {
		if !tokens[expected] {
			t.Errorf("Expected token '%s' not found", expected)
		}
	}

	// Should not include very short tokens
	if tokens["in"] {
		t.Error("Should not include very short tokens like 'in'")
	}
}

func TestCalculateOverlap(t *testing.T) {
	tokens := map[string]bool{
		"ProcessData":     true,
		"validateInput":   true,
		"transformOutput": true,
		"input":           true,
	}

	// All symbols match
	symbols := []string{"ProcessData", "validateInput", "transformOutput"}
	score := calculateOverlap(tokens, symbols)
	if score != 1.0 {
		t.Errorf("Expected score 1.0 for all matches, got %f", score)
	}

	// Partial match (2 out of 3)
	symbols = []string{"ProcessData", "validateInput", "notPresent"}
	score = calculateOverlap(tokens, symbols)
	expected := 2.0 / 3.0
	if score != expected {
		t.Errorf("Expected score %f for 2/3 matches, got %f", expected, score)
	}

	// No matches
	symbols = []string{"foo", "bar", "baz"}
	score = calculateOverlap(tokens, symbols)
	if score != 0.0 {
		t.Errorf("Expected score 0.0 for no matches, got %f", score)
	}
}

func TestCorrelationCannotConfirm(t *testing.T) {
	// Per spec section 11: Correlation can never return CONFIRMED confidence
	resolver := &Resolver{}

	mockState := &MockStateMap{
		entities: []StateEntity{
			{
				EntityKey: "file.go::Perfect",
				Symbols:   []string{"Perfect"},
			},
		},
	}

	// Even with perfect match, should still be INFERRED
	code := `func Perfect() {}`

	resolution := resolver.resolveViaCorrelation("test-hash", code, mockState)

	if resolution.Confidence == ConfidenceConfirmed {
		t.Error("Correlation must never return CONFIRMED confidence")
	}

	// Should be INFERRED or UNRESOLVED, never CONFIRMED
	if resolution.Confidence != ConfidenceInferred && resolution.Confidence != ConfidenceUnresolved {
		t.Errorf("Expected INFERRED or UNRESOLVED, got %s", resolution.Confidence)
	}
}
