package entity

import (
	"testing"
)

func TestParseAST_GoFunction(t *testing.T) {
	code := `package main

func HelloWorld() {
	println("Hello, World!")
}

func Goodbye() {
	println("Goodbye!")
}
`

	result, err := ParseAST(code, "go")
	if err != nil {
		t.Fatalf("ParseAST failed: %v", err)
	}

	if result == nil {
		t.Fatal("ParseAST returned nil result")
	}

	if len(result.Symbols) != 2 {
		t.Errorf("Expected 2 symbols, got %d", len(result.Symbols))
	}

	// Verify symbols
	expectedSymbols := map[string]bool{
		"HelloWorld": false,
		"Goodbye":    false,
	}

	for _, sym := range result.Symbols {
		if _, exists := expectedSymbols[sym.Name]; exists {
			expectedSymbols[sym.Name] = true
			if sym.Kind != SymbolKindFunction {
				t.Errorf("Symbol %s has wrong kind: got %s, want %s", sym.Name, sym.Kind, SymbolKindFunction)
			}
		} else {
			t.Errorf("Unexpected symbol: %s", sym.Name)
		}
	}

	for name, found := range expectedSymbols {
		if !found {
			t.Errorf("Expected symbol not found: %s", name)
		}
	}

	if result.ASTNodeCount == 0 {
		t.Error("ASTNodeCount should be > 0")
	}

	if result.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", result.Language)
	}
}

func TestParseAST_GoStruct(t *testing.T) {
	code := `package main

type User struct {
	Name string
	Age  int
}

type Config struct {
	Host string
	Port int
}
`

	result, err := ParseAST(code, "go")
	if err != nil {
		t.Fatalf("ParseAST failed: %v", err)
	}

	if len(result.Symbols) != 2 {
		t.Errorf("Expected 2 symbols, got %d", len(result.Symbols))
	}

	for _, sym := range result.Symbols {
		if sym.Kind != SymbolKindStruct {
			t.Errorf("Symbol %s has wrong kind: got %s, want %s", sym.Name, sym.Kind, SymbolKindStruct)
		}
	}
}

func TestParseAST_GoInterface(t *testing.T) {
	code := `package main

type Reader interface {
	Read(p []byte) (n int, err error)
}
`

	result, err := ParseAST(code, "go")
	if err != nil {
		t.Fatalf("ParseAST failed: %v", err)
	}

	if len(result.Symbols) != 1 {
		t.Errorf("Expected 1 symbol, got %d", len(result.Symbols))
	}

	if result.Symbols[0].Kind != SymbolKindInterface {
		t.Errorf("Expected interface, got %s", result.Symbols[0].Kind)
	}

	if result.Symbols[0].Name != "Reader" {
		t.Errorf("Expected name 'Reader', got '%s'", result.Symbols[0].Name)
	}
}

func TestParseAST_GoMethod(t *testing.T) {
	code := `package main

type Counter struct {
	count int
}

func (c *Counter) Increment() {
	c.count++
}

func (c *Counter) Get() int {
	return c.count
}
`

	result, err := ParseAST(code, "go")
	if err != nil {
		t.Fatalf("ParseAST failed: %v", err)
	}

	// Should find: 1 struct + 2 methods = 3 symbols
	if len(result.Symbols) != 3 {
		t.Errorf("Expected 3 symbols, got %d", len(result.Symbols))
	}

	methodCount := 0
	structCount := 0
	for _, sym := range result.Symbols {
		if sym.Kind == SymbolKindMethod {
			methodCount++
		}
		if sym.Kind == SymbolKindStruct {
			structCount++
		}
	}

	if methodCount != 2 {
		t.Errorf("Expected 2 methods, got %d", methodCount)
	}

	if structCount != 1 {
		t.Errorf("Expected 1 struct, got %d", structCount)
	}
}

func TestParseAST_UnsupportedLanguage(t *testing.T) {
	code := `console.log("hello");`

	_, err := ParseAST(code, "javascript")
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

func TestParseAST_EmptyCode(t *testing.T) {
	code := `package main`

	result, err := ParseAST(code, "go")
	if err != nil {
		t.Fatalf("ParseAST failed: %v", err)
	}

	if len(result.Symbols) != 0 {
		t.Errorf("Expected 0 symbols for empty code, got %d", len(result.Symbols))
	}
}

func TestDetectLanguage_GoExtension(t *testing.T) {
	filepath := "main.go"
	lang := DetectLanguage("", &filepath)
	if lang != "go" {
		t.Errorf("Expected 'go', got '%s'", lang)
	}
}

func TestDetectLanguage_GoHeuristic(t *testing.T) {
	code := `package main

func main() {
	println("hello")
}
`
	lang := DetectLanguage(code, nil)
	if lang != "go" {
		t.Errorf("Expected 'go', got '%s'", lang)
	}
}

func TestDetectLanguage_Unknown(t *testing.T) {
	code := `SELECT * FROM users;`
	lang := DetectLanguage(code, nil)
	if lang != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", lang)
	}
}
