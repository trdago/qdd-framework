package audit

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckDatabasePerformance audits the codebase for dangerous database query patterns
// such as executing SELECT COUNT(*) without filters on large tables.
func CheckDatabasePerformance(cwd string) []Violation {
	var violations []Violation

	// Regex to detect "SELECT COUNT(*)" or similar patterns without a WHERE clause
	countRegex := regexp.MustCompile(`(?i)SELECT\s+COUNT\s*\(\s*\*\s*\)\s+FROM\s+[a-zA-Z0-9_]+(\s*;|\s*$)`)

	err := filepath.WalkDir(cwd, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip vendor, .git, and .qdd directories
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == ".git" || d.Name() == ".qdd" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}

		// Only check Go files and SQL files
		ext := filepath.Ext(path)
		if !d.IsDir() && (ext == ".go" || ext == ".sql") {
			content, readErr := fs.ReadFile(osFS{}, path)
			if readErr != nil {
				return nil
			}

			strContent := string(content)
			lines := strings.Split(strContent, "\n")

			for i, line := range lines {
				if countRegex.MatchString(line) {
					// We found a SELECT COUNT(*) without a WHERE clause
					violations = append(violations, Violation{
						Category:    "Database",
						RuleID:      "DB-PERF-01",
						Description: "Uso detectado de SELECT COUNT(*) sin filtros. Reemplazar por pg_class.",
						File:        path,
						Line:        i + 1,
					})
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %v: %v\n", cwd, err)
	}

	return violations
}

// osFS is a simple implementation of fs.FS to read local files
type osFS struct{}

func (osFS) Open(name string) (fs.File, error) {
	panic("not implemented") // Only using ReadFile from os package
}
