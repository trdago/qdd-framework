package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'

// FND-147: en un proyecto recién creado el panel decía «créalo con 'qdd estado init'», y ese comando fallaba pidiendo
// --etapas. El aviso tiene que dar un comando que funcione tal cual está escrito.
func TestEstadoInit_ElComandoDelAvisoFuncionaEnUnProyectoNuevo_FND147(t *testing.T) {
	aviso := estado.ErrSinArchivo.Error()
	m := regexp.MustCompile(`qdd estado init --etapas "([^"]+)"`).FindStringSubmatch(aviso)
	if m == nil {
		t.Fatalf("el aviso no da un comando que funcione: %s", aviso)
	}
	root := t.TempDir()
	if err := estado.Init(root, etapasDeLaOpcion(m[1]), time.Now()); err != nil {
		t.Fatal(err)
	}
	datos, err := os.ReadFile(filepath.Join(root, estado.Ruta))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(datos), "Pruebas") {
		t.Fatalf("no quedaron las etapas del aviso: %s", datos)
	}
}

func TestEstadoInit_LasEtapasSeLeenSinEspaciosNiVacias_FND147(t *testing.T) {
	got := strings.Join(etapasDeLaOpcion(" Código, Pruebas ,,PR"), "|")
	if got != "Código|Pruebas|PR" {
		t.Fatalf("etapas: %s", got)
	}
}
