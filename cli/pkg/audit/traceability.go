package audit

import (
	"os/exec"
	"regexp"
	"strings"
)

// RunTraceabilityCheck evalúa que la convención de commits siga el estándar.
func RunTraceabilityCheck(cwd string) []Violation {
	var violations []Violation

	cmd := exec.Command("git", "log", "-1", "--pretty=%B")
	cmd.Dir = cwd
	out, err := cmd.Output()
	
	if err == nil {
		message := strings.TrimSpace(string(out))
		// Validación simple de Conventional Commits
		matched, _ := regexp.MatchString(`^(feat|fix|docs|style|refactor|perf|test|chore|build|ci|revert)(\(.+\))?:\s.*`, message)
		if !matched && message != "Initial commit" && !strings.HasPrefix(message, "Merge") {
			violations = append(violations, Violation{
				Category:    "TRACEABILITY",
				RuleID:      "TRC-01-CONVENTIONAL-COMMITS",
				Description: "El último commit no sigue el estándar 'Conventional Commits'. Mensaje: " + message,
			})
		}
	}

	return violations
}
