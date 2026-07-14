package audit

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	imgRegex   = regexp.MustCompile(`<img\s+[^>]*>`)
	altRegex   = regexp.MustCompile(`\balt\s*=`)
	clickRegex = regexp.MustCompile(`<(div|span)\s+[^>]*(@click|v-on:click)[^>]*>`)
	roleRegex  = regexp.MustCompile(`\brole\s*=`)
	tabRegex   = regexp.MustCompile(`\btabindex\s*=`)
)

// RunWCAGCheck valida reglas estáticas de Accesibilidad WCAG 2.2
func RunWCAGCheck(cwd string) []Violation {
	var violations []Violation
	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if shouldIgnoreFrontendPath(path, d, err) {
			return nil
		}
		scanFileWCAG(path, &violations)
		return nil
	})
	return violations
}

func shouldIgnoreFrontendPath(path string, d os.DirEntry, err error) bool {
	if err != nil || d.IsDir() {
		return true
	}
	if isIgnoredDir(path) {
		return true
	}
	return !isValidFrontendExt(path)
}

func isIgnoredDir(path string) bool {
	if strings.Contains(path, "node_modules") {
		return true
	}
	if strings.Contains(path, ".git") {
		return true
	}
	return strings.Contains(path, "dist")
}

func isValidFrontendExt(path string) bool {
	ext := filepath.Ext(path)
	if ext == ".vue" || ext == ".html" {
		return true
	}
	return ext == ".js" || ext == ".ts"
}

func scanFileWCAG(path string, violations *[]Violation) {
	if filepath.Ext(path) != ".vue" && filepath.Ext(path) != ".html" {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 1
	for scanner.Scan() {
		checkLineWCAG(scanner.Text(), path, lineNum, violations)
		lineNum++
	}
}

func checkLineWCAG(line, path string, lineNum int, violations *[]Violation) {
	checkImgAlt(line, path, lineNum, violations)
	checkClickSemantic(line, path, lineNum, violations)
}

func checkImgAlt(line, path string, lineNum int, violations *[]Violation) {
	matches := imgRegex.FindAllString(line, -1)
	for _, match := range matches {
		if !altRegex.MatchString(match) {
			*violations = append(*violations, Violation{
				Category:    "FRONTEND",
				RuleID:      "CERT-012-WCAG-2-2",
				Description: "WCAG: Elemento <img> sin atributo 'alt' detectado.",
				File:        path,
				Line:        lineNum,
			})
		}
	}
}

func checkClickSemantic(line, path string, lineNum int, violations *[]Violation) {
	matches := clickRegex.FindAllString(line, -1)
	for _, match := range matches {
		if !roleRegex.MatchString(match) && !tabRegex.MatchString(match) {
			*violations = append(*violations, Violation{
				Category:    "FRONTEND",
				RuleID:      "CERT-012-WCAG-2-2",
				Description: "WCAG: Elemento interactivo no semántico sin 'role' o 'tabindex'.",
				File:        path,
				Line:        lineNum,
			})
		}
	}
}
