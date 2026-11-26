package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// FND-271: 'qdd estado watch' en una terminal angosta (un panel lateral de tmux) dibujaba líneas más anchas que ella:
// sin estado.yaml legible, el título con el nombre de la carpeta y el aviso iban sin cortar, y bajo 20 columnas el
// tablero se dibujaba a 20.

// contrato: estado#327eb03e
func TestEstadoWatch_NingunaLineaMasAnchaQueLaTerminal_FND271(t *testing.T) {
	padre := t.TempDir()
	sinEstado := filepath.Join(padre, "un-proyecto-con-un-nombre-largo-de-verdad")
	conEstado := filepath.Join(padre, "otro-proyecto-con-un-nombre-largo-también")
	for _, dir := range []string{sinEstado, conEstado} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ahora := time.Now()
	if err := estado.Init(conEstado, []string{"Código", "Pruebas", "Producción", "Validación real"}, ahora); err != nil {
		t.Fatal(err)
	}
	cambios := []estado.Cambio{
		{Frente: "larga", Nombre: "Una tarea con un nombre larguísimo 🚀 que no cabe en ninguna terminal angosta", Etapa: "Código", Estado: "⏳", Bloqueo: "¿Sigo con la migración de los datos de producción a la base nueva?", DeQuien: "la persona"},
		{Frente: "rota", Nombre: "Otra", Etapa: "Pruebas", Estado: "❌", Falla: "tres pruebas fallan en Windows por el separador de rutas"},
	}
	for _, c := range cambios {
		if err := estado.Set(conEstado, c, ahora); err != nil {
			t.Fatal(err)
		}
	}
	for _, root := range []string{sinEstado, conEstado} {
		for ancho := 8; ancho <= 80; ancho++ {
			for _, l := range lineasDelWatch(root, ancho, 40) {
				if n := estado.AnchoVisible(l); n > ancho {
					t.Fatalf("%s a %d columnas: una línea de %d: %q", filepath.Base(root), ancho, n, l)
				}
			}
		}
	}
}
