package estado

import (
	"fmt"
	"testing"
)

// Rule: Prohibido usar 'else'
// Lo que el lector automático acepta como evidencia de una Validación real: el despliegue mismo (una versión o un
// commit) no lo es.

// checkConCelda revisa una tarea sana (desplegada y esperando un uso real) con una de sus celdas cambiada.
func checkConCelda(t *testing.T, etapa, celda string) []Violacion {
	t.Helper()
	celdas := map[string]string{
		"Código":          `{estado: "✅", evidencia: "commit e29393e"}`,
		"Pruebas":         `{estado: "✅", evidencia: "12/12 pruebas"}`,
		"Producción":      `{estado: "✅", evidencia: "v1.2.0"}`,
		"Validación real": `{estado: "⏳", bloqueo: "ver un uso real", de_quien: "la persona"}`,
	}
	celdas[etapa] = celda
	s := "formato: 1\netapas: [Código, Pruebas, Producción, Validación real]\nfrentes:\n  - id: t1\n    nombre: Tarea\n" +
		"    estado: abierto\n    celdas:\n"
	for _, nombre := range []string{"Código", "Pruebas", "Producción", "Validación real"} {
		s += fmt.Sprintf("      %s: %s\n", nombre, celdas[nombre])
	}
	return Check(mustParse(t, s), nil)
}

// contrato: estado-por-tareas#6ecea039
func TestCheck_EPFC04_ValidacionRealConLaEvidenciaDeUnDespliegue(t *testing.T) {
	despliegues := []string{"v1.2.0", "v1.2.1", "commit e29393e", "tag v1.2.1", "1.2.1-rc.2"}
	for _, ev := range despliegues {
		vs := checkConCelda(t, "Validación real", fmt.Sprintf(`{estado: "✅", evidencia: %q}`, ev))
		if !tieneViolacion(vs, "EPF-C04") {
			t.Errorf("Validación real ✅ con %q (un despliegue) no es validado: %s", ev, ids(vs))
		}
	}
	usos := []string{"uso real del 2026-10-02", "Ana cobró 3 pedidos con v1.2.0 (corrida 4082)"}
	for _, ev := range usos {
		vs := checkConCelda(t, "Validación real", fmt.Sprintf(`{estado: "✅", evidencia: %q}`, ev))
		if len(vs) > 0 {
			t.Errorf("Validación real ✅ con un uso real (%q) es válida: %s", ev, ids(vs))
		}
	}
}
