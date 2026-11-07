package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'

// Pedido de qdd-movile (2026-11-07): 6 de 8 proyectos registrados tenían las reglas de las IAs atrasadas y había que
// correr qdd doctor --fix en cada uno.
func TestDoctorTodos_CorreEnCadaProyectoYResume(t *testing.T) {
	sano, roto := t.TempDir(), t.TempDir()
	ido := filepath.Join(t.TempDir(), "ya-no-esta")
	proyectos := []registro.Proyecto{{Ruta: sano}, {Ruta: roto}, {Ruta: ido}}
	var corridos []string
	correr := func(ruta string) (string, int) {
		corridos = append(corridos, ruta)
		if ruta == roto {
			return "[x]\n  ✅ algo\n  ❌ Reglas para las IAs al día (AGENTS.md y otros) (Bloque de una versión anterior)\n", 1
		}
		return "  ✅ todo\n", 0
	}
	var salida bytes.Buffer
	codigo := doctorTodos(proyectos, correr, &salida)
	if codigo != 1 {
		t.Fatalf("un proyecto con anomalías hace fallar el total: %d", codigo)
	}
	if len(corridos) != 2 {
		t.Fatalf("una carpeta que ya no existe no se corre: %v", corridos)
	}
	texto := salida.String()
	for _, debe := range []string{"✅ " + filepath.Base(sano), "❌ " + filepath.Base(roto), "Reglas para las IAs al día", "⊘ ya-no-esta", "no existe"} {
		if !strings.Contains(texto, debe) {
			t.Fatalf("debe decir %q:\n%s", debe, texto)
		}
	}
	if strings.Contains(texto, "✅ algo") {
		t.Fatalf("de un proyecto con anomalías solo dice lo que falla:\n%s", texto)
	}
}

func TestDoctorTodos_SinAnomaliasSaleConCero(t *testing.T) {
	var salida bytes.Buffer
	if codigo := doctorTodos([]registro.Proyecto{{Ruta: t.TempDir()}}, func(string) (string, int) { return "", 0 }, &salida); codigo != 0 {
		t.Fatalf("todo sano: %d\n%s", codigo, salida.String())
	}
	if codigo := doctorTodos(nil, func(string) (string, int) { return "", 0 }, &salida); codigo != 0 || !strings.Contains(salida.String(), "no hay proyectos registrados") {
		t.Fatalf("sin registro lo dice: %d\n%s", codigo, salida.String())
	}
}

func TestDoctorTodos_UnArchivoEnVezDeCarpetaNoSeCorre(t *testing.T) {
	archivo := filepath.Join(t.TempDir(), "archivo")
	if err := os.WriteFile(archivo, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	corrio := false
	doctorTodos([]registro.Proyecto{{Ruta: archivo}}, func(string) (string, int) { corrio = true; return "", 0 }, &salida)
	if corrio {
		t.Fatal("solo se corre en carpetas")
	}
}
