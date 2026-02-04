package automated

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daverage/tinymem/internal/server"
)

// TestArtifactCreateWritesFile verifies that WriteArtifact creates a new file
// with the expected content and returns a valid confirmation payload.
func TestArtifactCreateWritesFile(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	content := "<h1>Hello, World</h1>\n"
	result, err := server.WriteArtifact(workspace, "index.html", content, true, "")
	if err != nil {
		t.Fatalf("WriteArtifact failed: %v", err)
	}

	if !result.Created {
		t.Error("Expected created=true for a new file")
	}
	if result.BytesWritten != len(content) {
		t.Errorf("BytesWritten: got %d, want %d", result.BytesWritten, len(content))
	}
	if result.SHA256 == "" {
		t.Error("SHA256 must not be empty")
	}
	if result.Path != "index.html" {
		t.Errorf("Path: got %q, want %q", result.Path, "index.html")
	}

	on, err := os.ReadFile(filepath.Join(workspace, "index.html"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(on) != content {
		t.Errorf("File content: got %q, want %q", string(on), content)
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_CreateWritesFile",
		Passed:   true,
	})
}

// TestArtifactCreatePathTraversalRejected verifies that directory-traversal
// sequences (../) are blocked before any filesystem operation.
func TestArtifactCreatePathTraversalRejected(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	traversalPaths := []string{
		"../secret.txt",
		"../../etc/passwd",
		"sub/../../outside.txt",
		"a/b/../../../pwned",
	}

	for _, p := range traversalPaths {
		_, err := server.WriteArtifact(workspace, p, "payload", true, "")
		if err == nil {
			t.Errorf("Path traversal not rejected for: %s", p)
		}
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_PathTraversalRejected",
		Passed:   true,
	})
}

// TestArtifactCreateAbsolutePathRejected verifies that absolute paths are
// rejected regardless of whether they point inside the workspace.
func TestArtifactCreateAbsolutePathRejected(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	absInside := filepath.Join(workspace, "inside.txt")
	_, err := server.WriteArtifact(workspace, absInside, "data", true, "")
	if err == nil {
		t.Fatal("Absolute path (inside workspace) was not rejected")
	}

	_, err = server.WriteArtifact(workspace, "/etc/passwd", "data", true, "")
	if err == nil {
		t.Fatal("Absolute path /etc/passwd was not rejected")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_AbsolutePathRejected",
		Passed:   true,
	})
}

// TestArtifactCreateOverwriteFalseProtectsExisting verifies that setting
// overwrite=false causes the write to fail when the file already exists and
// leaves the original content intact.
func TestArtifactCreateOverwriteFalseProtectsExisting(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	// Seed the file.
	_, err := server.WriteArtifact(workspace, "existing.txt", "v1", true, "")
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Attempt overwrite with overwrite=false.
	_, err = server.WriteArtifact(workspace, "existing.txt", "v2", false, "")
	if err == nil {
		t.Fatal("overwrite=false did not reject write to existing file")
	}

	// Original content must be untouched.
	got, err := os.ReadFile(filepath.Join(workspace, "existing.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("File was modified despite overwrite=false: got %q", string(got))
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_OverwriteFalseProtectsExisting",
		Passed:   true,
	})
}

// TestArtifactCreateOverwriteTrueUpdatesFile verifies that an existing file
// is replaced when overwrite=true and the result reflects created=false.
func TestArtifactCreateOverwriteTrueUpdatesFile(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	_, err := server.WriteArtifact(workspace, "update.txt", "v1", true, "")
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	result, err := server.WriteArtifact(workspace, "update.txt", "v2", true, "")
	if err != nil {
		t.Fatalf("Overwrite failed: %v", err)
	}
	if result.Created {
		t.Error("Expected created=false for an overwritten file")
	}

	got, err := os.ReadFile(filepath.Join(workspace, "update.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != "v2" {
		t.Errorf("File not updated: got %q, want %q", string(got), "v2")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_OverwriteTrueUpdatesFile",
		Passed:   true,
	})
}

// TestArtifactCreateSubdirectories verifies that intermediate directories
// are created automatically when the path contains nested folders.
func TestArtifactCreateSubdirectories(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	_, err := server.WriteArtifact(workspace, "a/b/deep.txt", "nested", true, "")
	if err != nil {
		t.Fatalf("Subdirectory write failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(workspace, "a", "b", "deep.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != "nested" {
		t.Errorf("Nested file content: got %q, want %q", string(got), "nested")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_SubdirectoryCreation",
		Passed:   true,
	})
}

// TestArtifactCreateLinkTaskIDEchoed verifies that the optional link_task_id
// field is carried through to the result payload.
func TestArtifactCreateLinkTaskIDEchoed(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	result, err := server.WriteArtifact(workspace, "linked.txt", "body", true, "task-42")
	if err != nil {
		t.Fatalf("WriteArtifact failed: %v", err)
	}
	if result.LinkTaskID != "task-42" {
		t.Errorf("LinkTaskID: got %q, want %q", result.LinkTaskID, "task-42")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_LinkTaskIDEchoed",
		Passed:   true,
	})
}

// TestArtifactCreateEmptyPathRejected verifies that an empty or whitespace-only
// path is rejected before any filesystem operation.
func TestArtifactCreateEmptyPathRejected(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	for _, p := range []string{"", "   ", "\t"} {
		_, err := server.WriteArtifact(workspace, p, "data", true, "")
		if err == nil {
			t.Errorf("Empty/whitespace path %q was not rejected", p)
		}
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_EmptyPathRejected",
		Passed:   true,
	})
}

// TestArtifactCreateDotPathRejected verifies that "." (the workspace root
// itself) is rejected as a write target.
func TestArtifactCreateDotPathRejected(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	_, err := server.WriteArtifact(workspace, ".", "data", true, "")
	if err == nil {
		t.Fatal("Dot path '.' was not rejected")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_DotPathRejected",
		Passed:   true,
	})
}

// TestArtifactValidatePathParity verifies that ValidateArtifactPath and
// WriteArtifact enforce the same containment rules, ensuring MCP and proxy
// (both callers of WriteArtifact) behave identically.
func TestArtifactValidatePathParity(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	// Cases that must fail validation.
	rejectCases := []string{"../x", "/abs", "a/../../b", "."}
	for _, p := range rejectCases {
		_, err := server.ValidateArtifactPath(workspace, p)
		if err == nil {
			t.Errorf("ValidateArtifactPath accepted %q but should reject", p)
		}
	}

	// Cases that must pass validation.
	acceptCases := []string{"ok.txt", "sub/ok.txt", "a/b/c.html"}
	for _, p := range acceptCases {
		abs, err := server.ValidateArtifactPath(workspace, p)
		if err != nil {
			t.Errorf("ValidateArtifactPath rejected %q: %v", p, err)
		}
		expected := filepath.Join(workspace, filepath.Clean(p))
		if abs != expected {
			t.Errorf("ValidateArtifactPath(%q): got %q, want %q", p, abs, expected)
		}
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Artifact_ValidatePathParity",
		Passed:   true,
	})
}

func setupArtifactTestEnv(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "artifact-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("Failed to resolve temp dir symlinks: %v", err)
	}
	return tmpDir, func() { os.RemoveAll(tmpDir) }
}
