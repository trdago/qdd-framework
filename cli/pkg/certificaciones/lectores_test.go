package certificaciones_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/certificaciones"
	"github.com/qdd-framework/qdd/pkg/evolution"
)

// Cada sinónimo que Normalizar reconoce, más uno que no reconoce: todos los
// lectores tienen que clasificarlos igual que este paquete.
var estadosDeArchivo = []string{
	"certified", "PASSED", "aprobada", "failed", "rechazada",
	"pending", "Pending", "draft", "borrador", "pendiente", "in_progress", "in-progress",
	"quizás", "",
}

func proyectoConEstados(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	carpeta := filepath.Join(dir, ".qdd", "project", "certification")
	if err := os.MkdirAll(carpeta, 0755); err != nil {
		t.Fatal(err)
	}
	for i, s := range estadosDeArchivo {
		nombre := filepath.Join(carpeta, "CERT-"+string(rune('A'+i))+".yaml")
		contenido := "id: CERT-" + string(rune('A'+i)) + "\ntitle: t\nstatus: \"" + s + "\"\n"
		if err := os.WriteFile(nombre, []byte(contenido), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// FND-285: evolution decidía por su cuenta qué status es pendiente y solo
// reconocía «pending»; un «borrador» o «draft» no aparecía como pendiente.
// contrato: certificaciones#ec8154f1
func TestLectores_EvolutionCuentaPendientesConNormalizar(t *testing.T) {
	dir := proyectoConEstados(t)
	conteo := certificaciones.ContarEn(dir)
	if conteo.Pendientes == 0 {
		t.Fatal("la prueba necesita certificaciones pendientes")
	}
	reporte, err := evolution.Analyze(dir, evolution.Medicion{Score: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(reporte.PendingCerts) != conteo.Pendientes {
		t.Errorf("evolution ve %d pendientes %+v; certificaciones.Normalizar ve %d", len(reporte.PendingCerts), reporte.PendingCerts, conteo.Pendientes)
	}
}
