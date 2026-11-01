package audit

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/herramientas"
)

// RunHerramientasCheck valida las definiciones de .qdd/herramientas (HER-01-DEFINICION, RFC 0011): una que no se lee,
// que no es válida o que trae un secreto no pasa el gate. No corre nada ni exige el ejecutable: en el CI puede no
// estar compilado (eso lo diagnostica qdd doctor).
func RunHerramientasCheck(cwd string) []Violation {
	hs, errs := herramientas.Cargar(cwd)
	var violations []Violation
	for _, err := range errs {
		violations = append(violations, violacionDeHerramienta(strings.SplitN(err.Error(), ":", 2)[0], err.Error()))
	}
	for _, h := range hs {
		if p := herramientas.Validar(h); len(p) > 0 {
			archivo := filepath.ToSlash(herramientas.Archivo(h.ID))
			violations = append(violations, violacionDeHerramienta(archivo, fmt.Sprintf("%s: %s", archivo, strings.Join(p, "; "))))
		}
	}
	return violations
}

func violacionDeHerramienta(archivo, descripcion string) Violation {
	return Violation{Category: "HERRAMIENTAS", RuleID: "HER-01-DEFINICION", Description: descripcion + ". Corrígela y guárdala con qdd herramienta guardar.", File: archivo}
}
