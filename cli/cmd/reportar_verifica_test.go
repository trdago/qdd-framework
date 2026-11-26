package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/incidentes"
	"github.com/qdd-framework/qdd/pkg/mejoras"
	"github.com/qdd-framework/qdd/pkg/reportes"
)

// Rule: Prohibido usar 'else'
// MEJ-087 (REP-503): al enviar un bug, quien reporta queda con un ⏳ suyo para verificarlo cuando salga el arreglo, con
// el comando que lo cierra. La verificación que arma qdd reportar es la que lee la cola de QDD.

func TestReportar_AlEnviarUnBugQuienReportaQuedaConSuEspera_MEJ087(t *testing.T) {
	root, _ := entornoReporte(t)
	if err := estado.Init(root, []string{"Código", "Validación real"}, ahoraReporte); err != nil {
		t.Fatal(err)
	}
	r, err := reportes.Nuevo(reportes.Datos{Titulo: "Se cae el panel", Detalle: "al abrir", Verifico: "abro el panel con 3 proyectos", Version: "v", SO: "s"}, incidentes.Contexto{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	anotarVerificacion(root, "REP-7", r, ahoraReporte)
	e, err := estado.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var celda estado.Celda
	for _, f := range e.Frentes {
		if strings.Contains(f.Nombre, "REP-7") {
			celda = f.Celdas["Validación real"]
		}
	}
	if celda.Estado != estado.Espera || !strings.Contains(celda.Bloqueo, "abro el panel con 3 proyectos") || !strings.Contains(celda.Bloqueo, "qdd reportar --verifique REP-7 --evidencia") {
		t.Fatalf("un ⏳ de quien reporta en su última etapa, con cómo lo verifica y cómo lo cierra: %+v", celda)
	}
}

func TestReportar_SinTableroNoFallaElEnvio_MEJ087(t *testing.T) {
	root, r := entornoReporte(t)
	anotarVerificacion(root, "REP-7", r, ahoraReporte)
	if _, err := os.Stat(filepath.Join(root, estado.Ruta)); err == nil {
		t.Fatal("sin tablero no se crea uno")
	}
}

func TestReportar_LaVerificacionQueArmaEsLaQueLeeLaCola_MEJ087(t *testing.T) {
	for _, sigue := range []bool{false, true} {
		r, err := reportes.Nuevo(reportes.Datos{VerificaDe: "REP-500", Evidencia: "x", SigueFallando: sigue, Version: "2.10.0-rc.96", SO: "s"}, incidentes.Contexto{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		rep, s, version, ok := mejoras.VerificacionDe(r.Titulo)
		if !ok || rep != "REP-500" || s != sigue || version != "2.10.0-rc.96" {
			t.Fatalf("%q: %q %v %q %v", r.Titulo, rep, s, version, ok)
		}
	}
}
