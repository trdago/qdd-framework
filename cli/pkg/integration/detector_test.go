package integration

import (
	"os"
	"path/filepath"
	"testing"
)

func mustMkdirTemp(t *testing.T, pattern string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to create dir %q: %v", path, err)
	}
}

func mustEvalSymlinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("failed to resolve %q: %v", path, err)
	}
	return resolved
}

func TestIsQDDProject(t *testing.T) {
	tempDir := mustMkdirTemp(t, "qdd-test-*")

	// Should return false for empty directory
	if IsQDDProject(tempDir) {
		t.Errorf("expected IsQDDProject to return false for empty dir")
	}

	mustMkdir(t, filepath.Join(tempDir, ".qdd"))

	// Should return true now
	if !IsQDDProject(tempDir) {
		t.Errorf("expected IsQDDProject to return true after creating .qdd dir")
	}
}

// FND: qdd sync/init/doctor used to resolve paths from os.Getwd() with no
// upward search, so running them from a subdirectory of an already-initialized
// project wrote duplicate .cursorrules/.claude/etc into that subdirectory
// instead of the real project root. FindProjectRoot fixes this by walking up
// to the nearest .git.
func TestFindProjectRoot_WalksUpToGitRoot(t *testing.T) {
	repoRoot := mustMkdirTemp(t, "qdd-root-*")
	mustMkdir(t, filepath.Join(repoRoot, ".git"))

	nestedDir := filepath.Join(repoRoot, "cli", "cmd")
	mustMkdir(t, nestedDir)

	got := FindProjectRoot(nestedDir)
	want := mustEvalSymlinks(t, repoRoot)
	gotResolved := mustEvalSymlinks(t, got)
	if gotResolved != want {
		t.Errorf("FindProjectRoot(%q) = %q, want %q", nestedDir, gotResolved, want)
	}
}

// A directory that already has its own .qdd (e.g. an isolated test sandbox,
// or a legitimate independently-governed package in a monorepo) must win
// over an outer .git root — FindProjectRoot must not hijack an intentionally
// scoped nested project.
func TestFindProjectRoot_PrefersNearestExistingQDDOverOuterGit(t *testing.T) {
	repoRoot := mustMkdirTemp(t, "qdd-outer-*")
	mustMkdir(t, filepath.Join(repoRoot, ".git"))

	nestedDir := filepath.Join(repoRoot, "cli", "cmd")
	mustMkdir(t, nestedDir)
	mustMkdir(t, filepath.Join(nestedDir, ".qdd"))

	got := FindProjectRoot(nestedDir)
	want := mustEvalSymlinks(t, nestedDir)
	gotResolved := mustEvalSymlinks(t, got)
	if gotResolved != want {
		t.Errorf("FindProjectRoot(%q) = %q, want %q (nearest .qdd, not outer .git)", nestedDir, gotResolved, want)
	}
}

func TestFindProjectRoot_FallsBackWhenNoGitFound(t *testing.T) {
	standaloneDir := mustMkdirTemp(t, "qdd-standalone-*")

	got := FindProjectRoot(standaloneDir)
	if got != standaloneDir {
		t.Errorf("FindProjectRoot(%q) = %q, want %q (fallback to startPath)", standaloneDir, got, standaloneDir)
	}
}

// Regression: a stray .qdd in a workspace folder ABOVE a repo (e.g.
// ~/proyectos/.qdd created by an accidental init) must not capture the
// commands of every sibling repo that has no .qdd of its own.
func TestFindProjectRoot_DoesNotCrossGitRootToOuterQDD(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".qdd"), 0755); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(workspace, "sibling-repo")
	nested := filepath.Join(repo, "internal", "pkg")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	got := FindProjectRoot(nested)
	if got != repo {
		t.Errorf("FindProjectRoot(%q) = %q, want repo root %q (outer workspace .qdd must be ignored)", nested, got, repo)
	}
}

// Worktrees have a .git FILE instead of a directory; they are repo roots too.
func TestFindProjectRoot_TreatsGitFileAsRepoRoot(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".qdd"), 0755); err != nil {
		t.Fatal(err)
	}
	worktree := filepath.Join(workspace, "wt")
	if err := os.MkdirAll(worktree, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: /elsewhere\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if got := FindProjectRoot(worktree); got != worktree {
		t.Errorf("FindProjectRoot(%q) = %q, want %q", worktree, got, worktree)
	}
}
