package audit

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Regresión: el gate escaneaba worktrees de agentes (.claude/worktrees/agent-*,
// ignorados por git) y bloqueaba commits por código que no es del proyecto.
func TestWalkProject_SkipsNestedReposAndGitIgnoredDirs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no disponible")
	}
	root := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", "build_output/\n")
	write("main.go", "package main\n")
	write("build_output/gen.go", "package gen\n")
	write(".claude/worktrees/agent-1/.git", "gitdir: /elsewhere\n")
	write(".claude/worktrees/agent-1/copy.go", "package copy\n")

	var seen []string
	err := walkProject(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".go" {
			rel, _ := filepath.Rel(root, path)
			seen = append(seen, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0] != "main.go" {
		t.Errorf("walkProject vio %v; solo debía ver [main.go]", seen)
	}
}
