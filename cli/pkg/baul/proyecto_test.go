package baul

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// Criterio de éxito de la ficha baul: el permiso vale para esa sesión, ese proyecto y ese fin; el valor nunca va a un
// archivo de QDD.

func TestUsar_UnPermisoNoValeEnOtroProyectoConLaMismaSesion(t *testing.T) {
	a := entorno(t)
	guardarCloudflare(t)
	var salida bytes.Buffer
	Usar(opcionesUsar(a, &salida, "true"))
	permitirTodo(t, a)
	if codigo, err := Usar(opcionesUsar(a, &salida, "true")); err != nil || codigo != 0 {
		t.Fatalf("en el proyecto permitido corre: %d %v", codigo, err)
	}
	b := t.TempDir()
	if err := os.MkdirAll(filepath.Join(b, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := consola.LatirSesion(b, "s1", ahora); err != nil {
		t.Fatal(err)
	}
	marca := filepath.Join(b, "corrio")
	codigo, err := Usar(opcionesUsar(b, &salida, "touch", marca))
	if !errors.Is(err, ErrSinPermiso) || codigo != CodigoTodaviaNo {
		t.Fatalf("la misma sesión en otro proyecto no tiene el permiso: %d %v", codigo, err)
	}
	if _, err := os.Stat(marca); err == nil {
		t.Fatal("sin el permiso de ese proyecto no corre nada")
	}
	pedidos := consola.PedidosA(b, ahora)
	if len(pedidos) != 1 || !pedidos[0].DePersona {
		t.Fatalf("en el otro proyecto queda su propio pedido de la persona: %+v", pedidos)
	}
}

func TestUsar_ElValorNoQuedaEnNingunArchivoDeQDD(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("usa sh")
	}
	root := entorno(t)
	guardarCloudflare(t)
	var salida bytes.Buffer
	Usar(opcionesUsar(root, &salida, "true"))
	permitirTodo(t, root)
	if codigo, err := Usar(opcionesUsar(root, &salida, "sh", "-c", `echo "$CLOUDFLARE_API_TOKEN"`)); err != nil || codigo != 0 {
		t.Fatalf("con permiso corre: %d %v", codigo, err)
	}
	for _, carpeta := range []string{os.Getenv("QDD_HOME"), os.Getenv("HOME"), root} {
		filepath.WalkDir(carpeta, func(camino string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			datos, _ := os.ReadFile(camino)
			if bytes.Contains(datos, []byte(secreto)) {
				t.Errorf("el valor del token quedó en %s", camino)
			}
			return nil
		})
	}
}
