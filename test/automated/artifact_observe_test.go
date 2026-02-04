package automated

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daverage/tinymem/internal/server"
)

// ---------------------------------------------------------------------------
// artifact_read tests
// ---------------------------------------------------------------------------

// TestArtifactReadReturnsContent verifies the round-trip: write via
// WriteArtifact, read back via ReadArtifact, content and hash match.
func TestArtifactReadReturnsContent(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	body := "<!DOCTYPE html><html><body>hello</body></html>\n"
	writeRes, err := server.WriteArtifact(workspace, "page.html", body, true, "")
	if err != nil {
		t.Fatalf("WriteArtifact: %v", err)
	}

	readRes, err := server.ReadArtifact(workspace, "page.html")
	if err != nil {
		t.Fatalf("ReadArtifact: %v", err)
	}

	if readRes.Content != body {
		t.Errorf("Content mismatch: got %q, want %q", readRes.Content, body)
	}
	if readRes.SHA256 != writeRes.SHA256 {
		t.Errorf("SHA256 mismatch: read=%q write=%q", readRes.SHA256, writeRes.SHA256)
	}
	if readRes.Size != int64(len(body)) {
		t.Errorf("Size: got %d, want %d", readRes.Size, len(body))
	}
	if readRes.Path != "page.html" {
		t.Errorf("Path: got %q, want %q", readRes.Path, "page.html")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactRead_ReturnsContent",
		Passed:   true,
	})
}

// TestArtifactReadNonExistentFails verifies that reading a file that does not
// exist on disk returns an explicit error, not a zero-value result.
func TestArtifactReadNonExistentFails(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	_, err := server.ReadArtifact(workspace, "ghost.txt")
	if err == nil {
		t.Fatal("ReadArtifact on non-existent file did not return an error")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactRead_NonExistentFails",
		Passed:   true,
	})
}

// TestArtifactReadTinyTasksMdBlocked is the key negative test for Phase 1
// invariant preservation.  TaskManager is the sole authority over
// tinyTasks.md; artifact_read must structurally reject it — regardless of
// what the agent prompt says.
func TestArtifactReadTinyTasksMdBlocked(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	// Seed the file on disk directly so it physically exists.
	if err := os.WriteFile(filepath.Join(workspace, "tinyTasks.md"), []byte("# Tasks\n"), 0644); err != nil {
		t.Fatalf("seed tinyTasks.md: %v", err)
	}

	_, err := server.ReadArtifact(workspace, "tinyTasks.md")
	if err == nil {
		t.Fatal("ReadArtifact allowed reading tinyTasks.md — PHASE 1 INVARIANT VIOLATION")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactRead_TinyTasksMdBlocked",
		Passed:   true,
	})
}

// TestArtifactReadPathTraversalBlocked reconfirms that the same containment
// validation used by artifact_create also guards artifact_read — no parallel
// validation path exists.
func TestArtifactReadPathTraversalBlocked(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	for _, p := range []string{"../etc/passwd", "../../shadow", "sub/../../out"} {
		_, err := server.ReadArtifact(workspace, p)
		if err == nil {
			t.Errorf("ReadArtifact did not reject traversal path: %s", p)
		}
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactRead_PathTraversalBlocked",
		Passed:   true,
	})
}

// TestArtifactReadDirectoryRejected verifies that passing a directory path
// returns an explicit error rather than silently succeeding or panicking.
func TestArtifactReadDirectoryRejected(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	if err := os.MkdirAll(filepath.Join(workspace, "subdir"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, err := server.ReadArtifact(workspace, "subdir")
	if err == nil {
		t.Fatal("ReadArtifact did not reject directory path")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactRead_DirectoryRejected",
		Passed:   true,
	})
}

// TestArtifactReadOversizedFileRejected seeds a file that exceeds the 1 MiB
// ceiling and verifies the read is refused before any data is returned.
func TestArtifactReadOversizedFileRejected(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	// Write 1 MiB + 1 byte directly (bypasses artifact_create intentionally —
	// we are testing the read-side size guard, not the write path).
	big := make([]byte, (1<<20)+1)
	if err := os.WriteFile(filepath.Join(workspace, "fat.bin"), big, 0644); err != nil {
		t.Fatalf("seed fat.bin: %v", err)
	}

	_, err := server.ReadArtifact(workspace, "fat.bin")
	if err == nil {
		t.Fatal("ReadArtifact did not reject oversized file")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactRead_OversizedFileRejected",
		Passed:   true,
	})
}

// TestArtifactReadNestedTinyTasksAllowed verifies that a file *named*
// tinyTasks.md inside a subdirectory is NOT blocked — only the root-level
// one is TaskManager-owned.
func TestArtifactReadNestedTinyTasksAllowed(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	nested := filepath.Join(workspace, "sub", "tinyTasks.md")
	if err := os.MkdirAll(filepath.Dir(nested), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(nested, []byte("nested content"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, err := server.ReadArtifact(workspace, "sub/tinyTasks.md")
	if err != nil {
		t.Fatalf("ReadArtifact rejected nested tinyTasks.md: %v", err)
	}
	if res.Content != "nested content" {
		t.Errorf("Content: got %q, want %q", res.Content, "nested content")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactRead_NestedTinyTasksAllowed",
		Passed:   true,
	})
}

// ---------------------------------------------------------------------------
// artifact_list tests
// ---------------------------------------------------------------------------

// TestArtifactListReturnsFiles creates a small tree and verifies all visible
// files are returned with correct relative paths.
func TestArtifactListReturnsFiles(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	// Create a small tree via WriteArtifact (the only governed write path).
	paths := []string{"a.txt", "sub/b.txt", "sub/deep/c.txt"}
	for _, p := range paths {
		if _, err := server.WriteArtifact(workspace, p, "x", true, ""); err != nil {
			t.Fatalf("WriteArtifact(%s): %v", p, err)
		}
	}

	entries, err := server.ListArtifacts(workspace, "")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}

	got := make(map[string]bool)
	for _, e := range entries {
		got[e.Path] = true
	}
	for _, p := range paths {
		if !got[p] {
			t.Errorf("Expected %q in listing but not found", p)
		}
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactList_ReturnsFiles",
		Passed:   true,
	})
}

// TestArtifactListEmptyWorkspaceReturnsNil verifies that an empty workspace
// yields a nil (or empty) slice, not an error.
func TestArtifactListEmptyWorkspaceReturnsNil(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	entries, err := server.ListArtifacts(workspace, "")
	if err != nil {
		t.Fatalf("ListArtifacts on empty workspace: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactList_EmptyWorkspace",
		Passed:   true,
	})
}

// TestArtifactListGlobBaseNameFilter verifies that a pattern without a path
// separator matches against the base filename in every subdirectory.
func TestArtifactListGlobBaseNameFilter(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	server.WriteArtifact(workspace, "index.html", "h", true, "")
	server.WriteArtifact(workspace, "style.css", "c", true, "")
	server.WriteArtifact(workspace, "sub/about.html", "h2", true, "")
	server.WriteArtifact(workspace, "sub/logo.png", "p", true, "")

	entries, err := server.ListArtifacts(workspace, "*.html")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}

	got := make(map[string]bool)
	for _, e := range entries {
		got[e.Path] = true
	}

	if !got["index.html"] || !got["sub/about.html"] {
		t.Errorf("*.html filter missed expected files; got %v", got)
	}
	if got["style.css"] || got["sub/logo.png"] {
		t.Errorf("*.html filter included non-HTML files; got %v", got)
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactList_GlobBaseNameFilter",
		Passed:   true,
	})
}

// TestArtifactListGlobPathFilter verifies that a pattern containing a "/"
// is matched against the full relative path.
func TestArtifactListGlobPathFilter(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	server.WriteArtifact(workspace, "root.txt", "r", true, "")
	server.WriteArtifact(workspace, "src/app.go", "g", true, "")
	server.WriteArtifact(workspace, "src/main.go", "m", true, "")
	server.WriteArtifact(workspace, "lib/util.go", "u", true, "")

	entries, err := server.ListArtifacts(workspace, "src/*")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}

	got := make(map[string]bool)
	for _, e := range entries {
		got[e.Path] = true
	}

	if !got["src/app.go"] || !got["src/main.go"] {
		t.Errorf("src/* filter missed src files; got %v", got)
	}
	if got["root.txt"] || got["lib/util.go"] {
		t.Errorf("src/* filter included files outside src/; got %v", got)
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactList_GlobPathFilter",
		Passed:   true,
	})
}

// TestArtifactListTinyTasksMdExcluded is a Phase 1 invariant negative test:
// even if tinyTasks.md physically exists, it must never appear in a listing.
func TestArtifactListTinyTasksMdExcluded(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	// Seed tinyTasks.md and a normal file.
	os.WriteFile(filepath.Join(workspace, "tinyTasks.md"), []byte("# Tasks\n"), 0644)
	server.WriteArtifact(workspace, "visible.txt", "yes", true, "")

	entries, err := server.ListArtifacts(workspace, "")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}

	for _, e := range entries {
		if e.Path == "tinyTasks.md" {
			t.Fatal("tinyTasks.md appeared in artifact_list — PHASE 1 INVARIANT VIOLATION")
		}
	}

	// Confirm the normal file IS present.
	found := false
	for _, e := range entries {
		if e.Path == "visible.txt" {
			found = true
		}
	}
	if !found {
		t.Error("visible.txt should appear in listing")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactList_TinyTasksMdExcluded",
		Passed:   true,
	})
}

// TestArtifactListExcludesInternalDirs verifies that .git/ and .tinyMem/
// subtrees are pruned from the listing entirely.
func TestArtifactListExcludesInternalDirs(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	// Seed internal directories with files.
	for _, dir := range []string{".git", ".tinyMem", "node_modules"} {
		p := filepath.Join(workspace, dir)
		os.MkdirAll(p, 0755)
		os.WriteFile(filepath.Join(p, "internal.txt"), []byte("secret"), 0644)
	}
	// Seed one visible file.
	server.WriteArtifact(workspace, "public.txt", "pub", true, "")

	entries, err := server.ListArtifacts(workspace, "")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}

	for _, e := range entries {
		for _, excluded := range []string{".git", ".tinyMem", "node_modules"} {
			if len(e.Path) > len(excluded) && e.Path[:len(excluded)+1] == excluded+"/" {
				t.Errorf("Internal dir %s/ leaked into listing: %s", excluded, e.Path)
			}
		}
	}

	found := false
	for _, e := range entries {
		if e.Path == "public.txt" {
			found = true
		}
	}
	if !found {
		t.Error("public.txt should appear in listing")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactList_ExcludesInternalDirs",
		Passed:   true,
	})
}

// TestArtifactListNestedTinyTasksVisible verifies that a tinyTasks.md file
// inside a subdirectory IS visible — only the root-level one is excluded.
func TestArtifactListNestedTinyTasksVisible(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	os.MkdirAll(filepath.Join(workspace, "docs"), 0755)
	os.WriteFile(filepath.Join(workspace, "docs", "tinyTasks.md"), []byte("nested"), 0644)

	entries, err := server.ListArtifacts(workspace, "")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}

	found := false
	for _, e := range entries {
		if e.Path == "docs/tinyTasks.md" {
			found = true
		}
	}
	if !found {
		t.Error("docs/tinyTasks.md should be visible in listing — only root-level tinyTasks.md is excluded")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactList_NestedTinyTasksVisible",
		Passed:   true,
	})
}

// TestArtifactListSizeField verifies that the Size field in each entry
// matches the actual file size on disk.
func TestArtifactListSizeField(t *testing.T) {
	workspace, cleanup := setupArtifactTestEnv(t)
	defer cleanup()

	content := "exactly 42 bytes of text content here!!\n" // len == 41, let's just check it matches
	server.WriteArtifact(workspace, "sized.txt", content, true, "")

	entries, err := server.ListArtifacts(workspace, "sized.txt")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	if entries[0].Size != int64(len(content)) {
		t.Errorf("Size: got %d, want %d", entries[0].Size, len(content))
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "ArtifactList_SizeField",
		Passed:   true,
	})
}
