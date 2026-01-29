package evidence

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/daverage/tinymem/internal/config"
)

// Verifier defines the interface for evidence verification
type Verifier interface {
	Verify(content string, cfg *config.Config) (bool, error)
	GetType() string
}

// FileExistsVerifier checks if a file exists
type FileExistsVerifier struct{}

func (v *FileExistsVerifier) GetType() string {
	return "file_exists"
}

// FileExistsVerifier is deterministic and safe (local filesystem only).
func (v *FileExistsVerifier) Verify(filePath string, cfg *config.Config) (bool, error) {
	absPath, err := resolveSafePath(filePath, cfg.ProjectRoot)
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

// GrepHitVerifier is deterministic and safe (local filesystem only).
func (v *GrepHitVerifier) Verify(patternAndFile string, cfg *config.Config) (bool, error) {
	// Format: "pattern::filename" or "pattern::filename::options"
	parts := strings.SplitN(patternAndFile, "::", 3)
	if len(parts) < 2 {
		return false, fmt.Errorf("invalid format, expected 'pattern::filename'")
	}

	pattern := parts[0]
	filename := parts[1]

	absPath, err := resolveSafePath(filename, cfg.ProjectRoot)
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

// CmdExit0Verifier is potentially unsafe or nondeterministic and is disabled by default.
func (v *CmdExit0Verifier) Verify(command string, cfg *config.Config) (bool, error) {
	return runWhitelistedCommand(command, cfg)
}

// TestPassVerifier checks if a test passes (placeholder implementation)
type TestPassVerifier struct{}

func (v *TestPassVerifier) GetType() string {
	return "test_pass"
}

// TestPassVerifier is potentially unsafe or nondeterministic and is disabled by default.
func (v *TestPassVerifier) Verify(testIdentifier string, cfg *config.Config) (bool, error) {
	return runWhitelistedCommand(testIdentifier, cfg)
}

func resolveSafePath(path string, baseDir string) (string, error) {
	if baseDir == "" {
		return "", fmt.Errorf("project root is not set")
	}

	// Normalize the input path
	normalizedPath := filepath.Clean(path)
	if normalizedPath == "." || normalizedPath == "" {
		return "", fmt.Errorf("invalid path: %s", path)
	}

	// Ensure the base directory is absolute
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base directory: %w", err)
	}

	var fullPath string
	if filepath.IsAbs(normalizedPath) {
		fullPath = normalizedPath
	} else {
		fullPath = filepath.Join(absBaseDir, normalizedPath)
	}

	// Get the absolute path to resolve any symbolic links
	absPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		// If EvalSymlinks fails, fall back to Abs
		absPath, err = filepath.Abs(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	// Check if the resolved path is within the base directory
	rel, err := filepath.Rel(absBaseDir, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	// Check for path traversal
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
	   strings.HasSuffix(rel, "..") ||
	   rel == ".." {
		return "", fmt.Errorf("path escapes project root: %s", path)
	}

	return absPath, nil
}

// VerifyEvidence verifies evidence using the appropriate verifier
func VerifyEvidence(evidenceType, content string, cfg *config.Config) (bool, error) {
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

	return verifier.Verify(content, cfg)
}

func runWhitelistedCommand(command string, cfg *config.Config) (bool, error) {
	if !cfg.EvidenceAllowCommand {
		return false, fmt.Errorf("command evidence is disabled by policy")
	}
	if strings.TrimSpace(command) == "" {
		return false, fmt.Errorf("evidence command is empty")
	}

	// Use proper shell command parsing to prevent injection
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false, fmt.Errorf("evidence command is empty (no fields)")
	}

	allowed := false
	cmdBase := filepath.Base(parts[0])
	for _, allowedCmd := range cfg.EvidenceAllowedCommands {
		if cmdBase == allowedCmd {
			allowed = true
			break
		}
	}
	if !allowed {
		return false, fmt.Errorf("command not in allowlist: %s", cmdBase)
	}

	// Additional validation: ensure no shell metacharacters in arguments
	for _, arg := range parts[1:] {
		if strings.ContainsAny(arg, "|&;><`$()[]{};") {
			return false, fmt.Errorf("argument contains unsafe characters: %s", arg)
		}
	}

	timeout := time.Duration(cfg.EvidenceCommandTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Use exec.Command directly with validated parts to prevent shell injection
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = cfg.ProjectRoot

	if err := cmd.Run(); err != nil {
		return false, err
	}
	return true, nil
}
