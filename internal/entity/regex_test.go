package entity

import (
	"testing"
)

func init() {
	// Load symbols config for tests
	LoadSymbolsConfig()
}

func TestLoadSymbolsConfig(t *testing.T) {
	err := LoadSymbolsConfig()
	if err != nil {
		t.Fatalf("LoadSymbolsConfig failed: %v", err)
	}

	config := GetSymbolsConfig()
	if config == nil {
		t.Fatal("Config is nil after loading")
	}

	// Check that Go patterns are present
	goPatterns, ok := config.Languages["go"]
	if !ok {
		t.Fatal("Go language patterns not found")
	}

	if len(goPatterns.Patterns) == 0 {
		t.Error("No patterns found for Go")
	}
}

func TestResolveViaRegexPatterns_GoFunction(t *testing.T) {
	code := `package main

func HelloWorld() {
	println("Hello, World!")
}
`

	resolver := &Resolver{}
	resolution := resolver.resolveViaRegexPatterns("test-hash", code, "go")

	if resolution == nil {
		t.Fatal("Resolution is nil")
	}

	if len(resolution.Symbols) != 1 {
		t.Errorf("Expected 1 symbol, got %d", len(resolution.Symbols))
	}

	if resolution.Symbols[0] != "HelloWorld" {
		t.Errorf("Expected symbol 'HelloWorld', got '%s'", resolution.Symbols[0])
	}

	// Single unique match should be CONFIRMED
	if resolution.Confidence != ConfidenceConfirmed {
		t.Errorf("Expected CONFIRMED confidence, got %s", resolution.Confidence)
	}

	if resolution.Method != MethodRegex {
		t.Errorf("Expected MethodRegex, got %s", resolution.Method)
	}

	if resolution.EntityKey == nil {
		t.Error("EntityKey should not be nil for single symbol")
	}
}

func TestResolveViaRegexPatterns_MultipleFunctions(t *testing.T) {
	code := `package main

func Foo() {}
func Bar() {}
func Baz() {}
`

	resolver := &Resolver{}
	resolution := resolver.resolveViaRegexPatterns("test-hash", code, "go")

	if len(resolution.Symbols) != 3 {
		t.Errorf("Expected 3 symbols, got %d", len(resolution.Symbols))
	}

	// Multiple symbols should be INFERRED (ambiguous)
	if resolution.Confidence != ConfidenceInferred {
		t.Errorf("Expected INFERRED confidence for multiple symbols, got %s", resolution.Confidence)
	}

	// EntityKey should be nil for ambiguous match
	if resolution.EntityKey != nil {
		t.Error("EntityKey should be nil for ambiguous match")
	}
}

func TestResolveViaRegexPatterns_GoType(t *testing.T) {
	code := `package main

type User struct {
	Name string
}
`

	resolver := &Resolver{}
	resolution := resolver.resolveViaRegexPatterns("test-hash", code, "go")

	if len(resolution.Symbols) != 1 {
		t.Errorf("Expected 1 symbol, got %d", len(resolution.Symbols))
	}

	if resolution.Symbols[0] != "User" {
		t.Errorf("Expected symbol 'User', got '%s'", resolution.Symbols[0])
	}

	if resolution.Confidence != ConfidenceConfirmed {
		t.Errorf("Expected CONFIRMED confidence, got %s", resolution.Confidence)
	}
}

func TestResolveViaRegexPatterns_UnsupportedLanguage(t *testing.T) {
	code := `SELECT * FROM users;`

	resolver := &Resolver{}
	resolution := resolver.resolveViaRegexPatterns("test-hash", code, "sql")

	if resolution.Confidence != ConfidenceUnresolved {
		t.Errorf("Expected UNRESOLVED for unsupported language, got %s", resolution.Confidence)
	}

	if len(resolution.Symbols) != 0 {
		t.Errorf("Expected 0 symbols, got %d", len(resolution.Symbols))
	}
}

func TestResolveViaRegexPatterns_NoMatches(t *testing.T) {
	code := `package main

// Just a comment
`

	resolver := &Resolver{}
	resolution := resolver.resolveViaRegexPatterns("test-hash", code, "go")

	if resolution.Confidence != ConfidenceUnresolved {
		t.Errorf("Expected UNRESOLVED when no matches, got %s", resolution.Confidence)
	}

	if len(resolution.Symbols) != 0 {
		t.Errorf("Expected 0 symbols, got %d", len(resolution.Symbols))
	}
}

func TestResolveViaRegexPatterns_PythonFunction(t *testing.T) {
	code := `def hello_world():
    print("Hello, World!")
`

	resolver := &Resolver{}
	resolution := resolver.resolveViaRegexPatterns("test-hash", code, "python")

	if len(resolution.Symbols) != 1 {
		t.Errorf("Expected 1 symbol, got %d", len(resolution.Symbols))
	}

	if resolution.Symbols[0] != "hello_world" {
		t.Errorf("Expected symbol 'hello_world', got '%s'", resolution.Symbols[0])
	}

	if resolution.Confidence != ConfidenceConfirmed {
		t.Errorf("Expected CONFIRMED confidence, got %s", resolution.Confidence)
	}
}

func TestResolveViaRegexPatterns_JavaScriptFunction(t *testing.T) {
	code := `function doSomething() {
    console.log("test");
}
`

	resolver := &Resolver{}
	resolution := resolver.resolveViaRegexPatterns("test-hash", code, "javascript")

	if len(resolution.Symbols) != 1 {
		t.Errorf("Expected 1 symbol, got %d", len(resolution.Symbols))
	}

	if resolution.Symbols[0] != "doSomething" {
		t.Errorf("Expected symbol 'doSomething', got '%s'", resolution.Symbols[0])
	}
}
