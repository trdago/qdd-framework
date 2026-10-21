package audit

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// walkProject recorre el proyecto como filepath.WalkDir, pero salta lo que no
// pertenece al código versionado del proyecto:
//   - repos anidados y worktrees (subcarpetas con su propio .git, p. ej.
//     .claude/worktrees/agent-*), que tienen su propio ciclo de auditoría;
//   - carpetas que git ignora (.gitignore), que no se despliegan ni se commitean.
//
// Sin esto el gate bloqueaba commits por violaciones en copias temporales de
// agentes, y la gente terminaba saltándose el gate.
func walkProject(root string, fn fs.WalkDirFunc) error {
	ignored := gitIgnoredDirs(root)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && path != root && isExcludedDir(path, ignored) {
			return filepath.SkipDir
		}
		return fn(path, d, err)
	})
}

func isExcludedDir(path string, ignored map[string]bool) bool {
	if ignored[filepath.Clean(path)] {
		return true
	}
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

var ignoredDirsCache sync.Map

// gitIgnoredDirs devuelve las carpetas ignoradas por git (rutas absolutas).
// Si root no es un repo git, o git no está disponible, devuelve un mapa vacío.
func gitIgnoredDirs(root string) map[string]bool {
	if cached, ok := ignoredDirsCache.Load(root); ok {
		return cached.(map[string]bool)
	}

	dirs := map[string]bool{}
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard", "--directory")
	cmd.Dir = root
	out, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.HasSuffix(line, "/") {
				continue
			}
			dirs[filepath.Clean(filepath.Join(root, line))] = true
		}
	}

	ignoredDirsCache.Store(root, dirs)
	return dirs
}
