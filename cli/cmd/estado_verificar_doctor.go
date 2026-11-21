package cmd

import (
	"strings"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Esperas que se responden solas: doctor dice cuáles no se pueden verificar. VRF-01: la última revisión falló (sin red,
// sin gh, sin remote); se avisa y la espera sigue abierta. VRF-02: una verificación que este qdd no entiende (editada a
// mano en estado.yaml); se corrige con 'qdd estado set … --verifica' o se quita. Ninguna se repara sola: no se adivina.

const (
	grupoVerificaciones    = "14. Esperas que se responden solas (qdd estado verificar)"
	itemVerificacionesCorr = "Las verificaciones de los ⏳ responden (VRF-01)"
	itemVerificacionesTipo = "Cada verificación es de un tipo conocido (VRF-02)"
)

func buildVerificacionesChecks(projectPath string) *CheckGroup {
	g := &CheckGroup{Name: grupoVerificaciones}
	r := estado.RevisarVerificaciones(projectPath)
	if !r.Hay {
		return g
	}
	corren := &CheckItem{Name: itemVerificacionesCorr, Success: len(r.Fallan) == 0}
	if !corren.Success {
		corren.Error = "su última revisión falló (la espera sigue abierta): " + strings.Join(r.Fallan, "; ")
	}
	tipos := &CheckItem{Name: itemVerificacionesTipo, Success: len(r.Invalidas) == 0}
	if !tipos.Success {
		tipos.Error = "corrígelas con 'qdd estado set … --verifica tipo:arg' o quítalas: " + strings.Join(r.Invalidas, ", ")
	}
	g.Items = append(g.Items, corren, tipos)
	return g
}
