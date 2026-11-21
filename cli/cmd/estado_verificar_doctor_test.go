package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'

// Un ⏳ con una verificación de tipo desconocido rompe el grupo de verificaciones, y doctor dice qué decidir (corregirla
// con --verifica tipo:arg o quitarla): no la reescribe solo. Sin verificaciones, el grupo no aparece.
// contrato: doctor#41288493
func TestDoctorVerificaciones_UnaVerificacionInvalidaDiceQueDecidir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("QDD_HOME", t.TempDir())
	if err := estado.Init(root, []string{"Código", "Publicación"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if g := buildVerificacionesChecks(root); len(g.Items) != 0 {
		t.Fatalf("sin verificaciones no hay nada que revisar: %+v", g.Items)
	}
	e, err := estado.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Frentes = append(e.Frentes, estado.Frente{ID: "rel", Nombre: "Release", Estado: estado.EstadoAbierto, Celdas: map[string]estado.Celda{
		"Código":      {Estado: estado.SinEmpezar},
		"Publicación": {Estado: estado.Espera, Bloqueo: "que npm la tenga", DeQuien: "CI", Verifica: "ftp:algo"},
	}})
	if err := estado.Save(root, e); err != nil {
		t.Fatal(err)
	}
	antes, _ := os.ReadFile(filepath.Join(root, estado.Ruta))
	g := buildVerificacionesChecks(root)
	var tipos *CheckItem
	for _, it := range g.Items {
		if it.Name == itemVerificacionesTipo {
			tipos = it
		}
	}
	if tipos == nil || tipos.Success || !strings.Contains(tipos.Error, "--verifica tipo:arg") || !strings.Contains(tipos.Error, "rel/Publicación") {
		t.Fatalf("dice cuál está mal y qué decidir: %+v", tipos)
	}
	if despues, _ := os.ReadFile(filepath.Join(root, estado.Ruta)); string(despues) != string(antes) {
		t.Fatal("revisar no reescribe el tablero")
	}
}
