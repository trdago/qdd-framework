package audit

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/qdd-framework/qdd/pkg/findings"
)

// Rule: Prohibido usar 'else'
// MEJ-002: un finding sin estado no se cuenta como abierto ni como resuelto, y nadie se enteraba. El gate lo bloquea, sin
// inferirlo: lo decide quien escribe el finding. A un resuelto le pide su prueba solo desde desdePruebaEnFinding: los
// proyectos de esta máquina tenían unos 145 resueltos antiguos sin test_result, y exigirlo los dejaría a todos sin gate.

const desdePruebaEnFinding = "2026-11-06"

var reFechaFinding = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// RunFindingsCheck revisa los findings del proyecto. Sin carpeta no reporta nada.
func RunFindingsCheck(cwd string) []Violation {
	lista, err := findings.Leer(cwd)
	if err != nil {
		return nil
	}
	var violations []Violation
	for _, f := range lista {
		archivo := filepath.ToSlash(filepath.Join(".qdd", "project", "findings", f.Archivo))
		if f.Estado == findings.SinEstado {
			violations = append(violations, Violation{Category: "FINDINGS", RuleID: "FND-01-SIN-ESTADO", File: archivo,
				Description: fmt.Sprintf("%s no dice su estado: escribe status: open, resolved o wontfix.", f.Archivo)})
			continue
		}
		if faltaPrueba(f) {
			violations = append(violations, Violation{Category: "FINDINGS", RuleID: "FND-02-SIN-PRUEBA", File: archivo,
				Description: fmt.Sprintf("%s está resuelto y no dice qué lo prueba: escribe test_result con su prueba de regresión.", f.Archivo)})
		}
	}
	return violations
}

func faltaPrueba(f findings.Finding) bool {
	if f.Estado != findings.Resuelto || strings.TrimSpace(f.Prueba) != "" {
		return false
	}
	return reFechaFinding.MatchString(f.Fecha) && f.Fecha >= desdePruebaEnFinding
}
