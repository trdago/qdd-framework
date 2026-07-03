package audit

import (
	"os"
	"path/filepath"
)

// RunTwelveFactorCheck verifica la adherencia a la metodología 12-Factor App
func RunTwelveFactorCheck(cwd string) []Violation {
	var violations []Violation

	gitPath := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		violations = append(violations, Violation{
			Category:    "12-FACTOR",
			RuleID:      "12F-01-CODEBASE",
			Description: "No se detectó un repositorio Git. Todo proyecto debe estar bajo control de versiones.",
		})
	}

	return violations
}
