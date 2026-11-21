package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'

func TestSincronizar_UnaLineaPorProyectoYLoQueEncontro(t *testing.T) {
	proyectos := []remoto.ProyectoSincronizado{
		{Nombre: "sano", Consola: true, Inconsistencias: []remoto.Inconsistencia{}},
		{Nombre: "raro", Consola: true, Inconsistencias: []remoto.Inconsistencia{
			{Tipo: remoto.InconsistenciaEsperaTrasPedido, Tarea: "Barra", Etapa: "PR", Detalle: "¿Confirmas [a]?"},
			{Tipo: remoto.InconsistenciaFlujoInvalido, Detalle: "roto: no se lee"},
		}},
		{Nombre: "quieto", Inconsistencias: []remoto.Inconsistencia{}},
	}
	var salida bytes.Buffer
	imprimirSincronizacion(&salida, proyectos)
	texto := salida.String()
	for _, debe := range []string{"✅ sano", "⚠ raro: 2", "Barra · PR", "¿Confirmas [a]?", "roto: no se lee", "– quieto: sin consola abierta", "no se corrigió nada"} {
		if !strings.Contains(texto, debe) {
			t.Fatalf("debe decir %q:\n%s", debe, texto)
		}
	}
}
