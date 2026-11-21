package estado

import (
	"strings"
	"testing"
	"time"
)

// Lo que requiere aprobación la da la persona con Aprobar (qdd estado aprobar, desde su pedido o su terminal), nunca un
// set. Una aprobación vale para el ✅ que aprueba: un ✅ con otra evidencia (otra versión) necesita la suya.
// contrato: estado-por-tareas#c53e6daf
// contrato: estado-por-tareas#8e8e07df
func TestAprobar_ValeParaElOkQueApruebaYNoParaElSiguiente(t *testing.T) {
	root := t.TempDir()
	ahora := time.Now()
	if err := Init(root, []string{"Código", "Producción"}, ahora); err != nil {
		t.Fatal(err)
	}
	espera := Cambio{Frente: "deploy", Nombre: "Deploy", Etapa: "Producción", Estado: "espera", Bloqueo: "aprobar el despliegue", DeQuien: "persona", RequiereAprobacion: "persona"}
	if err := Set(root, espera, ahora); err != nil {
		t.Fatal(err)
	}
	ok := Cambio{Frente: "deploy", Etapa: "Producción", Estado: "ok", Evidencia: "v1.0.0"}
	if err := Set(root, ok, ahora); err == nil || !strings.Contains(err.Error(), "EPF-C06") {
		t.Fatalf("sin aprobación no pasa a ✅: %v", err)
	}
	if err := Aprobar(root, "deploy", "Producción", ahora); err != nil {
		t.Fatal(err)
	}
	if err := Set(root, ok, ahora); err != nil {
		t.Fatalf("aprobada, pasa: %v", err)
	}
	ok.Evidencia = "v1.1.0"
	if err := Set(root, ok, ahora); err == nil || !strings.Contains(err.Error(), "EPF-C06") {
		t.Fatalf("la aprobación de v1.0.0 no vale para v1.1.0: %v", err)
	}
	if err := Aprobar(root, "deploy", "Código", ahora); err == nil {
		t.Fatal("una celda que no espera aprobación no se aprueba")
	}
}
