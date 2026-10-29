package audit

import (
	"os"
	"path/filepath"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// RunEstadoCheck aplica 'qdd estado check' (EPF-C01…C07, RFC 0006) al archivo de estado por frente del proyecto,
// contra su versión en git HEAD para detectar frentes que desaparecen. Sin el archivo no reporta nada.
func RunEstadoCheck(cwd string) []Violation {
	ruta := filepath.Join(cwd, estado.Ruta)
	if _, err := os.Stat(ruta); err != nil {
		return nil
	}
	e, err := estado.Load(cwd)
	if err != nil {
		return []Violation{{Category: "ESTADO", RuleID: "EPF-01-CHECK", Description: err.Error(), File: estado.Ruta}}
	}
	var violations []Violation
	for _, v := range estado.Check(e, estado.Anterior(cwd)) {
		violations = append(violations, Violation{Category: "ESTADO", RuleID: "EPF-01-CHECK", Description: v.String(), File: estado.Ruta})
	}
	return violations
}
