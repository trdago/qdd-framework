package cmd

import (
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
)

// RFC 0005 §3.4: no se informa un 100 sin su cobertura, y un principio que no
// se pudo revisar se dice, con el motivo, en vez de aprobar en silencio.
func TestFormatAuditResult_MuestraLaCobertura(t *testing.T) {
	inyeccion := audit.Principio{ID: "P-NO-INJECTION", Categoria: "seguridad"}
	flujo := audit.Principio{ID: "P-READABLE-FLOW", Categoria: "auditabilidad"}
	commits := audit.Principio{ID: "P-TRACEABLE-COMMITS", Categoria: "auditabilidad"}
	ia := audit.Principio{ID: "P-AI-POLICY", Categoria: "madurez"}
	r := audit.Resultado{Cobertura: audit.Cobertura{
		Principios: []audit.CoberturaPrincipio{
			{Principio: inyeccion, Estado: audit.Parcial, Evaluadas: []string{"cli (go)"},
				NoEvaluadas: []audit.UnidadNoEvaluada{{Ruta: "cli/ui", Motivo: "sin detector para node"}}},
			{Principio: flujo, Estado: audit.Evaluado, Evaluadas: []string{"cli (go)"}},
			{Principio: commits, Estado: audit.NoEvaluado,
				NoEvaluadas: []audit.UnidadNoEvaluada{{Ruta: ".", Motivo: "no es un repositorio git"}}},
			{Principio: ia, Estado: audit.Desactivado},
		},
		Aplicables: 3, Evaluados: 1, Parciales: 1, NoEvaluados: 1, Desactivados: 1,
	}}

	out := formatAuditResult(r)
	for _, want := range []string{
		"Puntaje QDD: 100/100 (cobertura: 1 de 3 principios evaluados en todo el proyecto; 1 parcial, 1 no evaluado, 1 desactivado)",
		"~ P-NO-INJECTION",
		"seguridad",
		"no evaluado en cli/ui: sin detector para node",
		"? P-TRACEABLE-COMMITS",
		"no es un repositorio git",
		"- P-AI-POLICY",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "P-READABLE-FLOW") {
		t.Errorf("un principio evaluado en todo el proyecto no necesita listarse:\n%s", out)
	}
}

// Un principio de seguridad sin evaluar va antes que uno de madurez.
func TestFormatAuditResult_SeguridadPrimero(t *testing.T) {
	r := audit.Resultado{Cobertura: audit.Cobertura{
		Principios: []audit.CoberturaPrincipio{
			{Principio: audit.Principio{ID: "P-B", Categoria: "madurez"}, Estado: audit.NoEvaluado, NoEvaluadas: []audit.UnidadNoEvaluada{{Ruta: ".", Motivo: "x"}}},
			{Principio: audit.Principio{ID: "P-A", Categoria: "seguridad"}, Estado: audit.NoEvaluado, NoEvaluadas: []audit.UnidadNoEvaluada{{Ruta: ".", Motivo: "x"}}},
		},
		Aplicables: 2, NoEvaluados: 2,
	}}
	out := formatAuditResult(r)
	if strings.Index(out, "P-A") > strings.Index(out, "P-B") {
		t.Errorf("seguridad debería ir antes que madurez:\n%s", out)
	}
}

// Muchas unidades con el mismo motivo se agrupan, para que la salida se pueda
// leer en un monorepo.
func TestFormatAuditResult_AgrupaPorMotivo(t *testing.T) {
	var faltan []audit.UnidadNoEvaluada
	for _, r := range []string{"a", "b", "c", "d", "e"} {
		faltan = append(faltan, audit.UnidadNoEvaluada{Ruta: r, Motivo: "sin detector para node"})
	}
	r := audit.Resultado{Cobertura: audit.Cobertura{
		Principios: []audit.CoberturaPrincipio{
			{Principio: audit.Principio{ID: "P-X", Categoria: "seguridad"}, Estado: audit.NoEvaluado, NoEvaluadas: faltan},
			{Principio: audit.Principio{ID: "P-Y", Categoria: "madurez", Politicas: []string{"enterprise"}}, Estado: audit.Desactivado},
		},
		Aplicables: 1, NoEvaluados: 1, Desactivados: 1,
	}}
	out := formatAuditResult(r)
	for _, want := range []string{"no evaluado en 5 unidades (a, b, c y 2 más): sin detector para node", "desactivado (enterprise: false)"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
}
