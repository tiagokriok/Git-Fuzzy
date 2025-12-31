package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_FindsRepositories(t *testing.T) {
	tmpDir := t.TempDir()

	repos := []string{
		filepath.Join(tmpDir, "repo1"),
		filepath.Join(tmpDir, "repo2"),
		filepath.Join(tmpDir, "dir1", "repo3"),
	}

	for _, repoPath := range repos {
		os.MkdirAll(repoPath, 0755)
		gitPath := filepath.Join(repoPath, ".git")
		os.MkdirAll(gitPath, 0755)
	}

	found, err := Scan([]string{tmpDir})

	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if len(found) != 3 {
		t.Errorf("expected 3 repos, got %d", len(found))
	}
}

func TestScan_IgnoresDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	repoPath := filepath.Join(tmpDir, "valid-repo")
	os.MkdirAll(repoPath, 0755)
	os.MkdirAll(filepath.Join(repoPath, ".git"), 0755)

	ignoredPath := filepath.Join(tmpDir, "node_modules", "package", "repo")
	os.MkdirAll(ignoredPath, 0755)
	os.MkdirAll(filepath.Join(ignoredPath, ".git"), 0755)

	found, err := Scan([]string{tmpDir})

	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if len(found) != 1 {
		t.Errorf("expected 1 repo, got %d", len(found))
		for _, r := range found {
			t.Logf("  Found: %s", r.Name)
		}
	}

	if found[0].Name != "valid-repo" {
		t.Errorf("expected 'valid-repo', got '%s'", found[0].Name)
	}
}

func TestScan_DeduplicatesRepositories(t *testing.T) {
	tmpDir := t.TempDir()

	repoPath := filepath.Join(tmpDir, "repo")
	os.MkdirAll(repoPath, 0755)
	os.MkdirAll(filepath.Join(repoPath, ".git"), 0755)

	found, err := Scan([]string{tmpDir, tmpDir})

	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if len(found) != 1 {
		t.Errorf("expected 1 repo (deduplicated), got %d", len(found))
	}
}

func TestScan_InvalidPath(t *testing.T) {
	found, err := Scan([]string{"/nonexistent/path"})

	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if len(found) != 0 {
		t.Errorf("expected 0 repos, got %d", len(found))
	}
}

func TestScan_EmptySearchPaths(t *testing.T) {
	found, err := Scan([]string{})

	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if len(found) != 0 {
		t.Errorf("expected 0 repos, got %d", len(found))
	}
}
