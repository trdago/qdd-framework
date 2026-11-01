package audit

import (
	"path/filepath"
	"strings"
	"testing"
)

// Una suite que necesita un programa que esta máquina no tiene (p. ej. la CLI de Claude en el CI) no se corre ni se
// da por aprobada: queda sin medir, sin bloquear. La que sí puede correr, corre y bloquea si falla.
func TestRunCoverageCheck_UnaSuiteSinSuHerramientaQuedaSinMedir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "config.yaml"), `test_commands:
  - "true"
  - comando: "exit 7"
    requiere: [qdd-programa-que-no-existe]
  - comando: "exit 4"
    requiere: [sh]
`)
	violations := RunCoverageCheck(dir)
	var sinMedir, fallas []Violation
	for _, v := range violations {
		switch v.RuleID {
		case "QA-03-SIN-MEDIR":
			sinMedir = append(sinMedir, v)
		case "QA-01-ALL-TESTS-PASS":
			fallas = append(fallas, v)
		}
	}
	if len(sinMedir) != 1 || !sinMedir[0].Recomendacion || !strings.Contains(sinMedir[0].Description, "qdd-programa-que-no-existe") || !strings.Contains(sinMedir[0].Description, "sin medir") {
		t.Fatalf("la suite sin su programa queda sin medir, sin bloquear, y dice qué falta: %+v", sinMedir)
	}
	if len(fallas) != 1 || !strings.Contains(fallas[0].Description, "exit 4") {
		t.Fatalf("la que tiene su programa corre y su falla bloquea; la que no corrió nunca es una falla ni una aprobación: %+v", fallas)
	}
	if got := labels(resolveTestCommands(dir)); got != "true | exit 7 | exit 4" {
		t.Fatalf("las suites se nombran por su comando: %q", got)
	}
}
