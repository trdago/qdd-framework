package audit

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// RunModuleHeartCheck valida las fichas de módulo (.qdd/project/modules).
//   - HEART-01-INVALID-RECORD: siempre; una ficha rota o incompleta engaña a
//     cualquier IA que la lea.
//   - HEART-02-STALE-RECORD y HEART-03-ZONE-WITHOUT-RECORD: solo con la
//     política module_heart activa (adopción gradual por proyecto).
func RunModuleHeartCheck(cwd string, enforce bool) []Violation {
	mods, loadErrs := modules.Load(cwd)
	var violations []Violation
	for _, err := range loadErrs {
		violations = append(violations, heartViolation("HEART-01-INVALID-RECORD", "", fmt.Sprintf("Ficha ilegible: %v", err)))
	}
	for _, m := range mods {
		rel := relPath(cwd, m.File)
		if problems := m.Validate(cwd); len(problems) > 0 {
			violations = append(violations, heartViolation("HEART-01-INVALID-RECORD", rel, "Ficha incompleta: "+strings.Join(problems, "; ")))
			continue
		}
		if !enforce {
			continue
		}
		if drift := modules.Drift(cwd, m); m.Estado == modules.EstadoAprobado && drift > 0 {
			violations = append(violations, heartViolation("HEART-02-STALE-RECORD", rel,
				fmt.Sprintf("El código del módulo '%s' cambió en %d commit(s) desde su última revisión. Revisa la ficha y apruébala de nuevo (qdd modules approve %s).", m.ID, drift, m.ID)))
		}
	}
	if !enforce {
		return violations
	}
	for _, zone := range modules.UncoveredZones(cwd, mods) {
		violations = append(violations, heartViolation("HEART-03-ZONE-WITHOUT-RECORD", zone,
			"Zona de código sin ficha de módulo: falta registrar su objetivo operativo y tecnológico."))
	}
	return violations
}

func heartViolation(rule, file, desc string) Violation {
	return Violation{Category: "MODULE-HEART", RuleID: rule, Description: desc, File: file}
}

func relPath(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return rel
}
