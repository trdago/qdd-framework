package estado

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-292: activar una tarea cerrada decía «la tarea "a" está cerrado»: el estado es masculino («cerrado», del frente)
// y la tarea, femenina. Lo lee la persona en el panel y la IA en el error de qdd_estado.

func TestActividad_UnaTareaCerradaODescartadaLoDiceEnFemenino_FND292(t *testing.T) {
	for estado, dice := range map[string]string{EstadoCerrado: "está cerrada", EstadoDescartado: "está descartada"} {
		root := proyectoConTareas(t, "a")
		ruta := filepath.Join(root, Ruta)
		b, _ := os.ReadFile(ruta)
		if err := os.WriteFile(ruta, []byte(strings.Replace(string(b), "estado: abierto", "estado: "+estado, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
		err := Activar(root, "a", "", ahora)
		if err == nil || !strings.Contains(err.Error(), `la tarea "a" `+dice) {
			t.Fatalf("%s: %v", estado, err)
		}
	}
}
