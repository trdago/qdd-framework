package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
)

// Rule: Prohibido usar 'else'

// contrato: reporte-y-release#c553bffc
// FND-311: el reporte decía CERTIFIED en gobernanza de IA aunque nada la hubiera evaluado.
func TestReport_LaGobernanzaDeIASoloDiceLoQueSeEvaluo_FND311(t *testing.T) {
	conIA := t.TempDir()
	mustWrite(t, filepath.Join(conIA, "a.go"), "package a\n\n// usa openai\n")
	sinIA := t.TempDir()
	apagada := audit.DefaultPolicies()
	apagada.AIGovernance = false
	falla := []audit.Violation{{Category: "ISO/IEC 42001 (AIMS)"}}

	casos := []struct {
		nombre      string
		dir         string
		politicas   audit.QDDPolicies
		violaciones []audit.Violation
		quiere      string
	}{
		{"política apagada", conIA, apagada, nil, "SIN_EVALUAR"},
		{"sin IA", sinIA, audit.DefaultPolicies(), nil, "NO_APLICA"},
		{"con IA y sin hallazgos", conIA, audit.DefaultPolicies(), nil, "SIN_HALLAZGOS"},
		{"con hallazgos", conIA, audit.DefaultPolicies(), falla, "VIOLATIONS_DETECTED"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if got := estadoGobernanzaIA(c.dir, c.politicas, c.violaciones); got != c.quiere {
				t.Errorf("estado = %s, se esperaba %s", got, c.quiere)
			}
		})
	}
}

func TestReport_NoPresentaLasHeuristicasComoNormas_FND311(t *testing.T) {
	r := ComprehensiveReport{AIGovernanceStatus: "SIN_HALLAZGOS"}
	for _, formato := range []string{"text", "md"} {
		salida := formatReportOutput(r, formato)
		if !strings.Contains(salida, "no acredita") {
			t.Errorf("%s: no dice que las comprobaciones no acreditan una norma:\n%s", formato, salida)
		}
	}
}
