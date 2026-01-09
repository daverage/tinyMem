package tokens

import (
	"strings"
	"unicode"
)

// CountTokens estimates the number of tokens in a text string
// This is a simple approximation based on character patterns common in English
// For more accurate token counting, integrate with specific tokenizer libraries
func CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Simple heuristic: split on whitespace and punctuation boundaries
	// This approximates tokenization for common cases
	count := 0
	inToken := false

	for _, char := range text {
		isWordChar := unicode.IsLetter(char) || unicode.IsNumber(char) || char == '\''

		if isWordChar && !inToken {
			inToken = true
			count++
		} else if !isWordChar && inToken {
			inToken = false
		}
	}

	// Additional tokens for punctuation sequences and special characters
	// This is a rough approximation - real tokenizers are more sophisticated
	punctuationTokens := countPunctuationTokens(text)

	// Adjust based on typical tokenization ratios
	estimatedTokens := count + punctuationTokens

	// Apply a factor to approximate real tokenizers (this is a rough estimate)
	// Real tokenizers often produce ~0.75x to 0.9x of this estimate depending on text
	return estimatedTokens
}

// countPunctuationTokens counts punctuation sequences as additional tokens
func countPunctuationTokens(text string) int {
	count := 0
	inPunctSeq := false

	for _, char := range text {
		isPunct := unicode.IsPunct(char) && char != '\'' // Exclude apostrophe from punctuation counting

		if isPunct && !inPunctSeq {
			inPunctSeq = true
			count++
		} else if !isPunct && inPunctSeq {
			inPunctSeq = false
		}
	}

	return count
}

// CountMessagesTokens estimates tokens for a slice of messages
func CountMessagesTokens(messages []interface{}) int {
	total := 0
	for _, msg := range messages {
		// Type assertion to handle the Message struct from the llm package
		if message, ok := msg.(interface{ GetContent() interface{} }); ok {
			// This is a placeholder - we'll handle this in the actual implementation
			// For now, we'll use a string-based approach
			total += CountTokens(getMessageContentString(msg))
			// Add tokens for role identifiers (we'll handle this differently in actual implementation)
		}
	}
	return total
}

// getMessageContentString extracts content string from a message interface
func getMessageContentString(msg interface{}) string {
	// This is a simplified approach - in the actual implementation we'll need to handle
	// the specific Message struct from the llm package
	switch v := msg.(type) {
	case map[string]interface{}:
		if content, ok := v["content"]; ok {
			if contentStr, ok := content.(string); ok {
				return contentStr
			}
		}
	case string:
		return v
	}
	return ""
}