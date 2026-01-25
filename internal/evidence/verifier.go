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

	"github.com/a-marczewski/tinymem/internal/config"
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
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("invalid path: %s", path)
	}
	if baseDir == "" {
		return "", fmt.Errorf("project root is not set")
	}

	if filepath.IsAbs(cleaned) {
		rel, err := filepath.Rel(baseDir, cleaned)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("path escapes project root: %s", path)
		}
		return cleaned, nil
	}

	if strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("unsafe relative path: %s", path)
	}

	absPath := filepath.Join(baseDir, cleaned)
	rel, err := filepath.Rel(baseDir, absPath)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
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
		return false, fmt.Errorf("command is empty")
	}
	if strings.ContainsAny(command, "|&;><`$") || strings.Contains(command, "\n") || strings.Contains(command, "\r") {
		return false, fmt.Errorf("command contains unsafe characters")
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false, fmt.Errorf("command is empty")
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

	timeout := time.Duration(cfg.EvidenceCommandTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = cfg.ProjectRoot
	if err := cmd.Run(); err != nil {
		return false, err
	}
	return true, nil
}
