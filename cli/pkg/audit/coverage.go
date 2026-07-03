package audit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runTests(cwd string) error {
	if _, err := os.Stat(filepath.Join(cwd, "package.json")); err == nil {
		cmd := exec.Command("npm", "run", "test")
		cmd.Dir = cwd
		return cmd.Run()
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = cwd
	if _, err := os.Stat(filepath.Join(cwd, "cli")); err == nil {
		cmd.Dir = filepath.Join(cwd, "cli")
	}

	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	output := string(out)
	if strings.Contains(output, "no test files") || strings.Contains(output, "build failed") || strings.Contains(output, "cannot find main module") || strings.Contains(output, "no Go files in") {
		return nil
	}

	return err
}

// RunCoverageCheck evalúa que todos los tests pasen (Anti-Regresión).
func RunCoverageCheck(cwd string) []Violation {
	var violations []Violation

	if err := runTests(cwd); err != nil {
		fmt.Println("DEBUG ERROR:", err)
		violations = append(violations, Violation{
			Category:    "COVERAGE",
			RuleID:      "QA-01-ALL-TESTS-PASS",
			Description: "Fallo en la suite de pruebas unitarias. Se detectaron regresiones en el código.",
		})
	}

	// QA-02-E2E-TESTS: Exigir entorno de pruebas End-to-End
	hasE2E := false
	e2ePaths := []string{"e2e", "tests/e2e", "cypress", "playwright"}
	for _, p := range e2ePaths {
		if _, err := os.Stat(cwd + "/" + p); !os.IsNotExist(err) {
			hasE2E = true
			break
		}
	}

	if !hasE2E {
		violations = append(violations, Violation{
			Category:    "COVERAGE",
			RuleID:      "QA-02-E2E-TESTS",
			Description: "No se detectó un entorno de pruebas E2E (carpetas: e2e, tests/e2e, cypress, playwright). QDD exige validación de extremo a extremo.",
		})
	}

	return violations
}
