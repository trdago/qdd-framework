package audit

import (
	"os"
	"os/exec"
	"strings"
)

// RunCoverageCheck evalúa que todos los tests pasen (Anti-Regresión).
func RunCoverageCheck(cwd string) []Violation {
	var violations []Violation

	// Para este MVP validamos simplemente que los tests corran exitosamente
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = cwd + "/cli"
	out, err := cmd.CombinedOutput()
	
	if err != nil {
		output := string(out)
		if strings.Contains(output, "no test files") || strings.Contains(output, "build failed") {
			// Ignoramos si simplemente no hay tests o no compila (el build ya lo detecta)
			goto skipTestFail
		}
		
		violations = append(violations, Violation{
			Category:    "COVERAGE",
			RuleID:      "QA-01-ALL-TESTS-PASS",
			Description: "Fallo en la suite de pruebas unitarias. Se detectaron regresiones en el código.",
		})
	skipTestFail:
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
