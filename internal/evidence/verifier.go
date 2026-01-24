package evidence

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Verifier defines the interface for evidence verification
type Verifier interface {
	Verify(content string) (bool, error)
	GetType() string
}

// FileExistsVerifier checks if a file exists
type FileExistsVerifier struct{}

func (v *FileExistsVerifier) GetType() string {
	return "file_exists"
}

func (v *FileExistsVerifier) Verify(filePath string) (bool, error) {
	// Verify the path is safe (no traversal outside project)
	if strings.Contains(filePath, "../") || strings.HasPrefix(filePath, "/../") {
		return false, fmt.Errorf("unsafe path: %s", filePath)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false, err
	}

	// Check if file exists
	_, err = os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GrepHitVerifier checks if a pattern exists in a file
type GrepHitVerifier struct{}

func (v *GrepHitVerifier) GetType() string {
	return "grep_hit"
}

func (v *GrepHitVerifier) Verify(patternAndFile string) (bool, error) {
	// Format: "pattern::filename" or "pattern::filename::options"
	parts := strings.SplitN(patternAndFile, "::", 3)
	if len(parts) < 2 {
		return false, fmt.Errorf("invalid format, expected 'pattern::filename'")
	}

	pattern := parts[0]
	filename := parts[1]

	// Verify the filename is safe
	if strings.Contains(filename, "../") || strings.HasPrefix(filename, "/../") {
		return false, fmt.Errorf("unsafe path: %s", filename)
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return false, err
	}

	// Read the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}

	// Check if pattern exists in content
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return re.Match(content), nil
}

// CmdExit0Verifier executes a command and checks if it exits with code 0
type CmdExit0Verifier struct{}

func (v *CmdExit0Verifier) GetType() string {
	return "cmd_exit0"
}

func (v *CmdExit0Verifier) Verify(command string) (bool, error) {
	// For security, only allow certain commands
	// This is a simplified version - in production, you'd want more robust validation
	if strings.Contains(command, "|") || strings.Contains(command, "&") || strings.Contains(command, ";") {
		return false, fmt.Errorf("command contains unsafe characters")
	}

	cmd := exec.Command("/bin/sh", "-c", command)
	err := cmd.Run()

	// If there's no error, the command exited with 0
	return err == nil, nil
}

// TestPassVerifier checks if a test passes (placeholder implementation)
type TestPassVerifier struct{}

func (v *TestPassVerifier) GetType() string {
	return "test_pass"
}

func (v *TestPassVerifier) Verify(testIdentifier string) (bool, error) {
	// This is a placeholder - in a real implementation, this would run specific tests
	// For now, we'll just return false to indicate it's not implemented
	return false, fmt.Errorf("test_pass verifier not fully implemented: %s", testIdentifier)
}

// VerifyEvidence verifies evidence using the appropriate verifier
func VerifyEvidence(evidenceType, content string) (bool, error) {
	var verifier Verifier

	switch evidenceType {
	case "file_exists":
		verifier = &FileExistsVerifier{}
	case "grep_hit":
		verifier = &GrepHitVerifier{}
	case "cmd_exit0":
		verifier = &CmdExit0Verifier{}
	case "test_pass":
		verifier = &TestPassVerifier{}
	default:
		return false, fmt.Errorf("unknown evidence type: %s", evidenceType)
	}

	return verifier.Verify(content)
}