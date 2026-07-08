package audit

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckDatabasePerformance audits the codebase for dangerous database query patterns
// such as executing SELECT COUNT(*) without filters on large tables.
func CheckDatabasePerformance(cwd string) []Violation {
	var violations []Violation
	countRegex := regexp.MustCompile(`(?i)SELECT\s+COUNT\s*\(\s*\*\s*\)\s+FROM\s+[a-zA-Z0-9_]+(\s*;|\s*$)`)

	err := filepath.WalkDir(cwd, func(path string, d fs.DirEntry, err error) error {
		return processDBPath(path, d, err, countRegex, &violations)
	})

	if err != nil {
		fmt.Printf("Error walking the path %v: %v\n", cwd, err)
	}

	return violations
}

func processDBPath(path string, d fs.DirEntry, err error, countRegex *regexp.Regexp, violations *[]Violation) error {
	if isIgnoredDBPath(path, d, err) {
		return skipIgnoredDBDir(d)
	}

	if isCheckableDBFile(path, d) {
		checkDBFileContent(path, countRegex, violations)
	}
	return nil
}

func skipIgnoredDBDir(d fs.DirEntry) error {
	if d != nil && d.IsDir() {
		return filepath.SkipDir
	}
	return nil
}

func isIgnoredDBPath(path string, d fs.DirEntry, err error) bool {
	if err != nil {
		return true
	}
	return d.IsDir() && isIgnoredDBDirName(d.Name())
}

func isIgnoredDBDirName(name string) bool {
	return name == "vendor" || name == ".git" || name == ".qdd" || name == "node_modules"
}

func isCheckableDBFile(path string, d fs.DirEntry) bool {
	if d.IsDir() {
		return false
	}
	ext := filepath.Ext(path)
	return ext == ".go" || ext == ".sql"
}

func checkDBFileContent(path string, countRegex *regexp.Regexp, violations *[]Violation) {
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return
	}

	strContent := string(content)
	lines := strings.Split(strContent, "\n")

	for i, line := range lines {
		if countRegex.MatchString(line) {
			*violations = append(*violations, Violation{
				Category:    "Database",
				RuleID:      "DB-PERF-01",
				Description: "Uso detectado de SELECT COUNT(*) sin filtros. Reemplazar por pg_class.",
				File:        path,
				Line:        i + 1,
			})
		}
	}
}
