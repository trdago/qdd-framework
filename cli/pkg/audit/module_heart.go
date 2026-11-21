package audit

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// RunModuleHeartCheck valida las fichas de módulo (.qdd/project/modules).
//   - HEART-01-INVALID-RECORD: siempre; una ficha rota o incompleta engaña a
//     cualquier IA que la lea.
//   - HEART-04-INVARIANTE-SIN-PRUEBA: siempre, como recomendación; un invariante de una ficha aprobada que ninguna
//     prueba cita («contrato: <ficha>#<id>») es una promesa que nada demuestra.
//   - HEART-02-STALE-RECORD y HEART-03-ZONE-WITHOUT-RECORD: solo con la
//     política module_heart activa (adopción gradual por proyecto).
func RunModuleHeartCheck(cwd string, enforce bool) []Violation {
	mods, loadErrs := modules.Load(cwd)
	var violations []Violation
	citas := sync.OnceValue(func() map[string][]string { return modules.CitasDePruebas(cwd) })
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
	violations = append(violations, invariantesSinPrueba(mods, citas())...)
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

// invariantesSinPrueba recomienda, una vez por proyecto, citar desde una prueba cada invariante de una ficha aprobada que
// nadie demuestra: con las fichas que más tienen primero. El detalle por invariante está en qdd modules context.
func invariantesSinPrueba(mods []modules.Module, citas map[string][]string) []Violation {
	type porFicha struct {
		id       string
		sin, son int
	}
	var fichas []porFicha
	total, sinPrueba := 0, 0
	for _, m := range mods {
		if m.Estado != modules.EstadoAprobado {
			continue
		}
		contrato := modules.Contrato(m, citas)
		sin := len(modules.SinPrueba(contrato))
		total += len(contrato)
		sinPrueba += sin
		if sin > 0 {
			fichas = append(fichas, porFicha{m.ID, sin, len(contrato)})
		}
	}
	if sinPrueba == 0 {
		return nil
	}
	sort.SliceStable(fichas, func(i, j int) bool { return fichas[i].sin > fichas[j].sin })
	detalle := make([]string, 0, maxFichasEnAviso)
	for _, f := range fichas[:min(len(fichas), maxFichasEnAviso)] {
		detalle = append(detalle, fmt.Sprintf("%s %d/%d", f.id, f.sin, f.son))
	}
	v := heartViolation("HEART-04-INVARIANTE-SIN-PRUEBA", modules.Dir, fmt.Sprintf(
		"%d de %d invariantes de las fichas aprobadas sin una prueba que los cite (%s): una prueba que demuestre uno lleva «contrato: <ficha>#<id>» en un comentario; qdd modules context muestra los ids.",
		sinPrueba, total, strings.Join(detalle, ", ")))
	v.Recomendacion = true
	return []Violation{v}
}

// maxFichasEnAviso es cuántas fichas nombra el aviso.
const maxFichasEnAviso = 5
