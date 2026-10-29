package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'

func proyectoConBorrador(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	m := modules.Module{
		ID: "pagos", Nombre: "Pagos", Rutas: []string{"src"},
		ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "p", CriterioDeExito: "c"},
		ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "a"},
		Pruebas:             modules.Pruebas{Comandos: []string{"go test ./..."}},
	}
	if _, problemas := modules.Save(root, m); len(problemas) > 0 {
		t.Fatal(problemas)
	}
	return root
}

func TestConsolaAccion_RechazaSinConfirmarYIdDesconocido_CCC_C11(t *testing.T) {
	root := proyectoConBorrador(t)
	for _, caso := range []struct {
		id         string
		confirmado bool
		dice       string
	}{
		{"aprobar:pagos", false, "--confirmado"},
		{"aprobar:otra", true, "no está entre las que ofrece"},
		{"sh -c id", true, "no está entre las que ofrece"},
	} {
		var salida, errores bytes.Buffer
		if codigo := consolaAccion(root, caso.id, caso.confirmado, &salida, &errores); codigo != 1 {
			t.Fatalf("%s: código %d, se esperaba 1", caso.id, codigo)
		}
		if !strings.Contains(errores.String(), caso.dice) || salida.Len() > 0 {
			t.Fatalf("%s: errores %q, salida %q", caso.id, errores.String(), salida.String())
		}
		if m, _ := modules.Find(root, "pagos"); m.Estado != modules.EstadoBorrador {
			t.Fatalf("%s: la ficha no debió cambiar", caso.id)
		}
	}
}

func TestConsolaAccion_AprobarConfirmado_CCC_C10(t *testing.T) {
	root := proyectoConBorrador(t)
	var salida, errores bytes.Buffer
	if codigo := consolaAccion(root, "aprobar:pagos", true, &salida, &errores); codigo != 0 {
		t.Fatalf("código %d: %s %s", codigo, salida.String(), errores.String())
	}
	if m, _ := modules.Find(root, "pagos"); m.Estado != modules.EstadoAprobado {
		t.Fatalf("la ficha debió quedar aprobada: %+v", m)
	}
}
