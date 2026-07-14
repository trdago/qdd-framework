package audit

import (
	"os"
	"path/filepath"
	"strings"
)

// RunISO9241Check valida reglas estáticas de ISO 9241 (Diseño Centrado en el Usuario)
func RunISO9241Check(cwd string) []Violation {
	var violations []Violation
	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if shouldIgnoreFrontendPath(path, d, err) {
			return nil
		}
		scanFileISO(path, &violations)
		return nil
	})
	return violations
}

func scanFileISO(path string, violations *[]Violation) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	text := string(content)
	
	if hasAsyncWithoutLoading(text) {
		*violations = append(*violations, Violation{
			Category:    "FRONTEND",
			RuleID:      "CERT-011-ISO-9241",
			Description: "ISO 9241: Operación asíncrona detectada sin estado de 'loading' evidente.",
			File:        path,
		})
	}
}

func hasAsyncWithoutLoading(text string) bool {
	if !hasAsyncCall(text) {
		return false
	}
	return !hasLoadingIndicator(strings.ToLower(text))
}

func hasAsyncCall(text string) bool {
	if strings.Contains(text, "fetch(") {
		return true
	}
	return strings.Contains(text, "axios.")
}

func hasLoadingIndicator(low string) bool {
	if strings.Contains(low, "loading") || strings.Contains(low, "cargando") {
		return true
	}
	return strings.Contains(low, "spinner") || strings.Contains(low, "pending")
}
