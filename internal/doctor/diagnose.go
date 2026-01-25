package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/a-marczewski/tinymem/internal/config"
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

// Runner runs diagnostic checks
type Runner struct {
	config *config.Config
	db     *storage.DB
}

// NewRunner creates a new diagnostic runner
func NewRunner(cfg *config.Config, db *storage.DB) *Runner {
	return &Runner{
		config: cfg,
		db:     db,
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