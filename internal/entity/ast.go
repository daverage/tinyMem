package entity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// SymbolKind represents the type of symbol extracted from AST
type SymbolKind string

const (
	SymbolKindFunction  SymbolKind = "function"
	SymbolKindMethod    SymbolKind = "method"
	SymbolKindType      SymbolKind = "type"
	SymbolKindStruct    SymbolKind = "struct"
	SymbolKindInterface SymbolKind = "interface"
	SymbolKindConst     SymbolKind = "const"
	SymbolKindVar       SymbolKind = "var"
)

// Symbol represents a top-level symbol extracted from AST
type Symbol struct {
	Name  string
	Kind  SymbolKind
	Start uint32
	End   uint32
}

// ASTResult represents the result of AST parsing
type ASTResult struct {
	Symbols      []Symbol
	ASTNodeCount int
	Language     string
}

// ParseAST parses source code using Tree-sitter and extracts top-level symbols
// Per spec section 4.1: AST extraction is the primary resolution method
// Returns CONFIRMED confidence when successful
func ParseAST(content string, language string) (*ASTResult, error) {
	// Currently only Go is supported
	// Other languages can be added later per step requirements
	if language != "go" {
		return nil, fmt.Errorf("unsupported language: %s (only 'go' is currently supported)", language)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(content))
	if err != nil {
		return nil, fmt.Errorf("AST parsing failed: %w", err)
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	if rootNode == nil {
		return nil, fmt.Errorf("AST parsing failed: no root node")
	}

	// Extract top-level symbols
	symbols := extractTopLevelSymbols(rootNode, []byte(content))

	// Count total AST nodes for structural parity checks
	nodeCount := countNodes(rootNode)

	return &ASTResult{
		Symbols:      symbols,
		ASTNodeCount: nodeCount,
		Language:     language,
	}, nil
}

// extractTopLevelSymbols walks the AST and extracts top-level declarations
func extractTopLevelSymbols(node *sitter.Node, source []byte) []Symbol {
	var symbols []Symbol

	// Only process source_file children (top-level declarations)
	if node.Type() != "source_file" {
		return symbols
	}

	childCount := int(node.ChildCount())
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_declaration":
			// Extract function name (identifier child)
			funcChildCount := int(child.ChildCount())
			for j := 0; j < funcChildCount; j++ {
				funcChild := child.Child(j)
				if funcChild != nil && funcChild.Type() == "identifier" {
					name := funcChild.Content(source)
					symbols = append(symbols, Symbol{
						Name:  name,
						Kind:  SymbolKindFunction,
						Start: child.StartByte(),
						End:   child.EndByte(),
					})
					break
				}
			}

		case "method_declaration":
			// Extract method name (field_identifier child)
			methodChildCount := int(child.ChildCount())
			for j := 0; j < methodChildCount; j++ {
				methodChild := child.Child(j)
				if methodChild != nil && methodChild.Type() == "field_identifier" {
					name := methodChild.Content(source)
					symbols = append(symbols, Symbol{
						Name:  name,
						Kind:  SymbolKindMethod,
						Start: child.StartByte(),
						End:   child.EndByte(),
					})
					break
				}
			}

		case "type_declaration":
			// Extract type specifications by walking children
			typeChildCount := int(child.ChildCount())
			for j := 0; j < typeChildCount; j++ {
				typeChild := child.Child(j)
				if typeChild != nil && typeChild.Type() == "type_spec" {
					extractTypeSpec(typeChild, source, &symbols, child.StartByte(), child.EndByte())
				}
			}

		case "const_declaration":
			// Extract const names
			extractVarOrConstNames(child, source, SymbolKindConst, &symbols)

		case "var_declaration":
			// Extract var names
			extractVarOrConstNames(child, source, SymbolKindVar, &symbols)
		}
	}

	return symbols
}

// extractTypeSpec extracts a type specification (struct, interface, or type alias)
func extractTypeSpec(specNode *sitter.Node, source []byte, symbols *[]Symbol, start, end uint32) {
	// type_spec structure: type_identifier followed by type definition
	var name string
	var kind SymbolKind = SymbolKindType

	childCount := int(specNode.ChildCount())
	for i := 0; i < childCount; i++ {
		child := specNode.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "type_identifier":
			// This is the name
			name = child.Content(source)
		case "struct_type":
			kind = SymbolKindStruct
		case "interface_type":
			kind = SymbolKindInterface
		}
	}

	if name != "" {
		*symbols = append(*symbols, Symbol{
			Name:  name,
			Kind:  kind,
			Start: start,
			End:   end,
		})
	}
}

// extractVarOrConstNames extracts variable or constant names from declarations
func extractVarOrConstNames(node *sitter.Node, source []byte, kind SymbolKind, symbols *[]Symbol) {
	// Const and var declarations can have multiple specs
	childCount := int(node.ChildCount())
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "const_spec" || child.Type() == "var_spec" {
			// Extract name(s) from spec
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(source)
				*symbols = append(*symbols, Symbol{
					Name:  name,
					Kind:  kind,
					Start: node.StartByte(),
					End:   node.EndByte(),
				})
			}
		}
	}
}

// countNodes recursively counts all nodes in the AST
// Used for structural parity checks per spec section 7
func countNodes(node *sitter.Node) int {
	if node == nil {
		return 0
	}

	count := 1 // Count this node
	childCount := int(node.ChildCount())
	for i := 0; i < childCount; i++ {
		count += countNodes(node.Child(i))
	}

	return count
}

// DetectLanguage attempts to detect the programming language from content
// Currently uses simple heuristics - can be enhanced later
func DetectLanguage(content string, filepath *string) string {
	// If filepath is provided, use extension
	if filepath != nil {
		switch strings.ToLower(filepathExt(*filepath)) {
		case ".go":
			return "go"
		case ".js", ".jsx":
			return "javascript"
		case ".ts", ".tsx":
			return "typescript"
		case ".html", ".htm":
			return "html"
		case ".json":
			return "json"
		case ".py":
			return "python"
		case ".rs":
			return "rust"
		}
	}

	trimmed := strings.TrimSpace(content)

	// Simple heuristic for Go
	if strings.Contains(content, "package ") && strings.Contains(content, "func ") {
		return "go"
	}

	if strings.Contains(content, "<!DOCTYPE html") || strings.Contains(content, "<html") {
		return "html"
	}

	if isValidJSON(trimmed) {
		return "json"
	}

	if strings.Contains(content, "interface ") || (strings.Contains(content, "type ") && strings.Contains(content, ":")) {
		return "typescript"
	}

	if strings.Contains(content, "function ") || strings.Contains(content, "=>") || strings.Contains(content, "const ") || strings.Contains(content, "class ") {
		return "javascript"
	}

	return "unknown"
}

func filepathExt(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return ""
	}
	return path[idx:]
}

func isValidJSON(content string) bool {
	if content == "" {
		return false
	}
	return json.Valid([]byte(content))
}
