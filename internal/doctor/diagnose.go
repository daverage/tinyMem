package doctor

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/storage"
)

// Diagnostics holds diagnostic information
type Diagnostics struct {
	Checks []CheckResult `json:"checks"`
	Issues []string      `json:"issues"`
	Status string        `json:"status"`
}

// CheckResult represents the result of a single check
type CheckResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // "pass", "fail", "warn"
	Message  string `json:"message"`
	Severity string `json:"severity"` // "info", "warning", "error"
}

// ServerMode represents the mode in which tinyMem is running
type ServerMode string

const (
	ProxyMode ServerMode = "proxy"
	MCPMode   ServerMode = "mcp"
	StandaloneMode ServerMode = "standalone"
)

// Runner runs diagnostic checks
type Runner struct {
	config        *config.Config
	db            *storage.DB
	projectID     string
	memoryService *memory.Service
	serverMode    ServerMode
}

// NewRunner creates a new diagnostic runner
func NewRunner(cfg *config.Config, db *storage.DB, projectID string, memoryService *memory.Service) *Runner {
	return &Runner{
		config:        cfg,
		db:            db,
		projectID:     projectID,
		memoryService: memoryService,
		serverMode:    StandaloneMode, // Default to standalone mode
	}
}

// NewRunnerWithMode creates a new diagnostic runner with a specific server mode
func NewRunnerWithMode(cfg *config.Config, db *storage.DB, projectID string, memoryService *memory.Service, mode ServerMode) *Runner {
	return &Runner{
		config:        cfg,
		db:            db,
		projectID:     projectID,
		memoryService: memoryService,
		serverMode:    mode,
	}
}

// RunAll runs all diagnostic checks
func (d *Runner) RunAll() *Diagnostics {
	var results []CheckResult
	var issues []string

	// Run individual checks
	results = append(results, d.checkDatabaseConnectivity()...)
	results = append(results, d.checkFileSystemPermissions()...)
	results = append(results, d.checkConfiguration()...)
	results = append(results, d.checkStorageHealth()...)
	results = append(results, d.checkMemoryServiceHealth()...) // NEW CHECK
	results = append(results, d.checkExternalDependencies()...)
	results = append(results, d.checkFactEvidenceIntegrity()...)

	// Collect issues from failed checks
	for _, result := range results {
		if result.Status == "fail" {
			issues = append(issues, result.Message)
		}
	}

	status := "healthy"
	if len(issues) > 0 {
		status = "issues_found"
	}

	return &Diagnostics{
		Checks: results,
		Issues: issues,
		Status: status,
	}
}

func (d *Runner) checkExternalDependencies() []CheckResult {
	var results []CheckResult

	// Check if we're running in proxy mode by checking if the command is "proxy"
	// For now, we'll assume that if we're running doctor from the context of an MCP server,
	// we should skip the proxy-specific checks
	// This is a simplified approach - in a real implementation, we'd need to pass the server mode

	// Check LLM backend reachability (relevant for both proxy and MCP when using LLM features)
	// For memory-only operations, LLM backend is not required, so we'll downgrade this to a warning
	llmErr := checkReachable(d.config.LLMBaseURL)
	if llmErr != nil {
		results = append(results, CheckResult{
			Name:     "llm_backend_reachability",
			Status:   "fail",
			Message:  fmt.Sprintf("LLM backend unreachable: %v (not critical for memory-only operations)", llmErr),
			Severity: "warning", // Downgrade to warning since LLM isn't always required
		})
	} else {
		results = append(results, CheckResult{
			Name:     "llm_backend_reachability",
			Status:   "pass",
			Message:  "LLM backend reachable",
			Severity: "info",
		})
	}

	if d.config.SemanticEnabled {
		embedErr := checkReachable(d.config.EmbeddingBaseURL)
		if embedErr != nil {
			results = append(results, CheckResult{
				Name:     "embedding_backend_reachability",
				Status:   "fail",
				Message:  fmt.Sprintf("Embedding backend unreachable: %v", embedErr),
				Severity: "error",
			})
		} else {
			results = append(results, CheckResult{
				Name:     "embedding_backend_reachability",
				Status:   "pass",
				Message:  "Embedding backend reachable",
				Severity: "info",
			})
		}
	} else {
		results = append(results, CheckResult{
			Name:     "embedding_backend_reachability",
			Status:   "pass",
			Message:  "Embedding checks skipped (semantic disabled)",
			Severity: "info",
		})
	}

	// Check proxy readiness based on server mode
	// For MCP mode, this check is not critical since MCP doesn't operate as an HTTP proxy
	if d.serverMode == MCPMode {
		// For MCP mode, only log if proxy is not available but don't treat as a critical error
		if err := checkProxyListening(d.config.ProxyPort); err != nil {
			results = append(results, CheckResult{
				Name:     "proxy_readiness",
				Status:   "fail", // Still mark as fail but with warning severity for MCP mode
				Message:  fmt.Sprintf("Proxy not listening on port %d: %v (not critical in MCP mode)", d.config.ProxyPort, err),
				Severity: "warning", // Downgrade to warning for MCP mode
			})
		} else {
			results = append(results, CheckResult{
				Name:     "proxy_readiness",
				Status:   "pass",
				Message:  fmt.Sprintf("Proxy listening on port %d", d.config.ProxyPort),
				Severity: "info",
			})
		}
	} else {
		// For proxy mode, check if we're actually supposed to be running in proxy mode
		// For standalone mode, the proxy not running might be expected
		if d.serverMode == ProxyMode {
			// In proxy mode, the proxy should be running
			if err := checkProxyListening(d.config.ProxyPort); err != nil {
				results = append(results, CheckResult{
					Name:     "proxy_readiness",
					Status:   "fail",
					Message:  fmt.Sprintf("Proxy not listening on port %d: %v", d.config.ProxyPort, err),
					Severity: "error", // Keep as error for proxy mode
				})
			} else {
				results = append(results, CheckResult{
					Name:     "proxy_readiness",
					Status:   "pass",
					Message:  fmt.Sprintf("Proxy listening on port %d", d.config.ProxyPort),
					Severity: "info",
				})
			}
		} else {
			// In standalone mode, proxy not running is expected
			if err := checkProxyListening(d.config.ProxyPort); err != nil {
				results = append(results, CheckResult{
					Name:     "proxy_readiness",
					Status:   "pass", // Consider this a pass in standalone mode
					Message:  fmt.Sprintf("Proxy not listening on port %d (expected in standalone mode)", d.config.ProxyPort),
					Severity: "info",
				})
			} else {
				results = append(results, CheckResult{
					Name:     "proxy_readiness",
					Status:   "pass",
					Message:  fmt.Sprintf("Proxy listening on port %d", d.config.ProxyPort),
					Severity: "info",
				})
			}
		}
	}

	return results
}

func checkReachable(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("base URL is empty")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	host := parsed.Host
	if host == "" {
		return fmt.Errorf("invalid base URL: %s", baseURL)
	}

	if !strings.Contains(host, ":") {
		if parsed.Scheme == "https" {
			host = net.JoinHostPort(host, "443")
		} else {
			host = net.JoinHostPort(host, "80")
		}
	}

	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return err
	}
	return conn.Close()
}

func checkProxyListening(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port %s", strconv.Itoa(port))
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return err
	}
	return conn.Close()
}

// checkMemoryServiceHealth checks the health of the memory service
func (d *Runner) checkMemoryServiceHealth() []CheckResult {
	var results []CheckResult

	if d.memoryService == nil {
		results = append(results, CheckResult{
			Name:     "memory_service_initialization",
			Status:   "fail",
			Message:  "Memory service not initialized in doctor runner",
			Severity: "error",
		})
		return results
	}

	// Try to get a few memories from the current project
	memories, err := d.memoryService.GetAllMemories(d.projectID)
	if err != nil {
		results = append(results, CheckResult{
			Name:     "memory_service_access",
			Status:   "fail",
			Message:  fmt.Sprintf("Failed to access memories for project '%s': %v", d.projectID, err),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "memory_service_access",
			Status:   "pass",
			Message:  fmt.Sprintf("Successfully accessed %d memories for project '%s'", len(memories), d.projectID),
			Severity: "info",
		})
	}

	// Try a simple search
	_, err = d.memoryService.SearchMemories(d.projectID, "test", 1)
	if err != nil {
		results = append(results, CheckResult{
			Name:     "memory_service_search",
			Status:   "fail",
			Message:  fmt.Sprintf("Failed to perform memory search for project '%s': %v", d.projectID, err),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "memory_service_search",
			Status:   "pass",
			Message:  fmt.Sprintf("Successfully performed memory search for project '%s'", d.projectID),
			Severity: "info",
		})
	}

	return results
}

// checkDatabaseConnectivity checks database connectivity and basic operations
func (d *Runner) checkDatabaseConnectivity() []CheckResult {
	var results []CheckResult

	// Check if we can ping the database
	if err := d.db.GetConnection().Ping(); err != nil {
		results = append(results, CheckResult{
			Name:     "database_connectivity",
			Status:   "fail",
			Message:  fmt.Sprintf("Cannot connect to database: %v", err),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "database_connectivity",
			Status:   "pass",
			Message:  "Database connection successful",
			Severity: "info",
		})
	}

	// Check if we can perform a basic query
	if _, err := d.db.GetConnection().Exec("SELECT 1"); err != nil {
		results = append(results, CheckResult{
			Name:     "database_query",
			Status:   "fail",
			Message:  fmt.Sprintf("Cannot execute basic query: %v", err),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "database_query",
			Status:   "pass",
			Message:  "Basic database query successful",
			Severity: "info",
		})
	}

	return results
}

// checkFileSystemPermissions checks filesystem permissions for .tinyMem directory
func (d *Runner) checkFileSystemPermissions() []CheckResult {
	var results []CheckResult

	tinyMemDir := d.config.TinyMemDir

	// Check if .tinyMem directory exists
	if _, err := os.Stat(tinyMemDir); os.IsNotExist(err) {
		results = append(results, CheckResult{
			Name:     "tinyMem_directory_exists",
			Status:   "fail",
			Message:  fmt.Sprintf(".tinyMem directory does not exist: %s", tinyMemDir),
			Severity: "error",
		})
		return results // Early return since other checks will fail
	} else if err != nil {
		results = append(results, CheckResult{
			Name:     "tinyMem_directory_access",
			Status:   "fail",
			Message:  fmt.Sprintf("Cannot access .tinyMem directory: %v", err),
			Severity: "error",
		})
		return results
	}

	// Check read/write permissions
	if err := d.testDirectoryPermissions(tinyMemDir); err != nil {
		results = append(results, CheckResult{
			Name:     "tinyMem_directory_permissions",
			Status:   "fail",
			Message:  fmt.Sprintf("Insufficient permissions for .tinyMem directory: %v", err),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "tinyMem_directory_permissions",
			Status:   "pass",
			Message:  "Sufficient permissions for .tinyMem directory",
			Severity: "info",
		})
	}

	// Check subdirectories
	subdirs := []string{
		filepath.Join(tinyMemDir, "logs"),
		filepath.Join(tinyMemDir, "run"),
		filepath.Join(tinyMemDir, "store"),
	}

	for _, subdir := range subdirs {
		if _, err := os.Stat(subdir); os.IsNotExist(err) {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("%s_exists", filepath.Base(subdir)),
				Status:   "warn",
				Message:  fmt.Sprintf("Subdirectory does not exist: %s", subdir),
				Severity: "warning",
			})
		} else if err != nil {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("%s_access", filepath.Base(subdir)),
				Status:   "fail",
				Message:  fmt.Sprintf("Cannot access subdirectory: %v", err),
				Severity: "error",
			})
		} else {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("%s_access", filepath.Base(subdir)),
				Status:   "pass",
				Message:  fmt.Sprintf("Accessible subdirectory: %s", subdir),
				Severity: "info",
			})
		}
	}

	return results
}

// testDirectoryPermissions tests if we can read and write to a directory
func (d *Runner) testDirectoryPermissions(dir string) error {
	// Try to create a temporary file
	testFile := filepath.Join(dir, ".permission_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return err
	}

	// Clean up
	os.Remove(testFile)

	return nil
}

// checkConfiguration checks configuration validity
func (d *Runner) checkConfiguration() []CheckResult {
	var results []CheckResult

	// Validate config
	if err := d.config.Validate(); err != nil {
		results = append(results, CheckResult{
			Name:     "configuration_validation",
			Status:   "fail",
			Message:  fmt.Sprintf("Configuration validation failed: %v", err),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "configuration_validation",
			Status:   "pass",
			Message:  "Configuration is valid",
			Severity: "info",
		})
	}

	// Check if proxy port is available
	// This would require actually trying to bind to the port, which is complex
	// So we'll skip this check for now

	return results
}

// checkStorageHealth checks the health of the storage system
func (d *Runner) checkStorageHealth() []CheckResult {
	var results []CheckResult

	// Check if DB file exists and is accessible
	if _, err := os.Stat(d.config.DBPath); os.IsNotExist(err) {
		results = append(results, CheckResult{
			Name:     "database_file_exists",
			Status:   "fail",
			Message:  fmt.Sprintf("Database file does not exist: %s", d.config.DBPath),
			Severity: "error",
		})
	} else if err != nil {
		results = append(results, CheckResult{
			Name:     "database_file_access",
			Status:   "fail",
			Message:  fmt.Sprintf("Cannot access database file: %v", err),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "database_file_access",
			Status:   "pass",
			Message:  "Database file is accessible",
			Severity: "info",
		})
	}

	// Check database integrity
	if _, err := d.db.GetConnection().Exec("PRAGMA integrity_check"); err != nil {
		results = append(results, CheckResult{
			Name:     "database_integrity",
			Status:   "fail",
			Message:  fmt.Sprintf("Database integrity check failed: %v", err),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "database_integrity",
			Status:   "pass",
			Message:  "Database integrity check passed",
			Severity: "info",
		})
	}

	// Check for FTS5 extension availability (important for search functionality)
	if _, err := d.db.GetConnection().Exec("CREATE VIRTUAL TABLE IF NOT EXISTS fts5_test USING fts5(content); DROP TABLE fts5_test;"); err != nil {
		results = append(results, CheckResult{
			Name:     "fts5_extension_availability",
			Status:   "fail",
			Message:  fmt.Sprintf("FTS5 extension not available: %v", err),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "fts5_extension_availability",
			Status:   "pass",
			Message:  "FTS5 extension is available",
			Severity: "info",
		})
	}

	return results
}

func (d *Runner) checkFactEvidenceIntegrity() []CheckResult {
	var results []CheckResult

	var invalidFacts int
	err := d.db.GetConnection().QueryRow(`
		SELECT COUNT(*)
		FROM memories m
		WHERE m.type = 'fact'
		  AND NOT EXISTS (
			SELECT 1 FROM evidence e WHERE e.memory_id = m.id AND e.verified = 1
		  )
	`).Scan(&invalidFacts)
	if err != nil {
		results = append(results, CheckResult{
			Name:     "fact_evidence_integrity",
			Status:   "fail",
			Message:  fmt.Sprintf("Failed to validate fact evidence integrity: %v", err),
			Severity: "error",
		})
		return results
	}

	if invalidFacts > 0 {
		results = append(results, CheckResult{
			Name:     "fact_evidence_integrity",
			Status:   "fail",
			Message:  fmt.Sprintf("Found %d fact(s) without verified evidence", invalidFacts),
			Severity: "error",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "fact_evidence_integrity",
			Status:   "pass",
			Message:  "All facts have verified evidence",
			Severity: "info",
		})
	}

	return results
}

// PrintReport prints a formatted diagnostic report
func (d *Diagnostics) PrintReport() {
	fmt.Printf("=== tinyMem Diagnostic Report ===\n")
	fmt.Printf("Status: %s\n\n", d.Status)

	if len(d.Issues) > 0 {
		fmt.Printf("Issues Found:\n")
		for i, issue := range d.Issues {
			fmt.Printf("  %d. %s\n", i+1, issue)
		}
		fmt.Println()
	}

	fmt.Printf("Detailed Checks:\n")
	for _, check := range d.Checks {
		statusSymbol := "✓"
		if check.Status == "fail" {
			statusSymbol = "✗"
		} else if check.Status == "warn" {
			statusSymbol = "!"
		}

		fmt.Printf("  %s %s: %s\n", statusSymbol, check.Name, check.Message)
	}

	fmt.Println("\nRecommendations:")
	if len(d.Issues) == 0 {
		fmt.Println("  ✓ System is operating normally")
	} else {
		fmt.Println("  • Check the .tinyMem directory permissions")
		fmt.Println("  • Verify database file is not corrupted")
		fmt.Println("  • Ensure sufficient disk space is available")
		fmt.Println("  • Review configuration settings")
	}
}
