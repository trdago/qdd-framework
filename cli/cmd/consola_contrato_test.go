package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/listas"
)

// Rule: Prohibido usar 'else'
// MEJ-045 y MEJ-050 (FND-189): en un proyecto recién creado y sin repositorio de blueprints, 'qdd consola blueprints
// --json' entregó "entradas": null y el panel quedó en negro. Contrato: lo que el panel lee de qdd, en un proyecto
// vacío y en uno con datos, nunca trae una lista ni un mapa null.

func TestContrato_ElPanelNuncaRecibeUnaListaNull_FND189(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	vacio := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vacio, ".qdd", "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	conDatos := t.TempDir()
	if err := estado.Init(conDatos, []string{"Código"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	for nombre, root := range map[string]string{"vacío": vacio, "con estado": conDatos} {
		revisarContrato(t, nombre+" · consola blueprints", reflect.TypeFor[consola.DatosBlueprints](), func() []byte {
			var out bytes.Buffer
			consolaBlueprints(root, true, &out)
			return out.Bytes()
		})
		revisarContrato(t, nombre+" · consola panel", reflect.TypeFor[consola.DatosConsola](), func() []byte {
			b, _ := lecturaPanel(root, []string{"flujo"})()
			return b
		})
		revisarContrato(t, nombre+" · flujo lista", reflect.TypeFor[[]flujo.Resumen](), func() []byte {
			b, _ := listas.Marshal(flujo.Listar(root, estadoDeTarea(root)))
			return b
		})
	}
	revisarContrato(t, "consola proyectos", reflect.TypeFor[consola.DatosProyectos](), func() []byte {
		b, _ := listas.Marshal(consola.Proyectos(time.Now()))
		return b
	})
}

func revisarContrato(t *testing.T, que string, tipo reflect.Type, leer func() []byte) {
	t.Helper()
	datos := leer()
	if len(bytes.TrimSpace(datos)) == 0 {
		t.Fatalf("%s: no entregó nada", que)
	}
	if nulas := listas.Nulas(datos, tipo); len(nulas) > 0 {
		t.Fatalf("%s: listas null en %v\n%s", que, nulas, datos)
	}
}
