package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// RFC 0018: qdd consola proyectos --json entrega las filas ya calculadas y nunca falla.

func TestConsolaProyectos_EntregaJSONSinEscribirDondeCorre(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	root := filepath.Join(t.TempDir(), "mi-proyecto")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	ruta, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	contenido := `{"formato":1,"proyectos":[{"ruta":` + string(ruta) + `,"clave":"c","visto":"2026-11-06T10:00:00Z"}]}`
	if err := os.WriteFile(filepath.Join(home, "proyectos.json"), []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	t.Chdir(cwd)
	var salida bytes.Buffer
	consolaProyectosCmd.SetOut(&salida)
	defer consolaProyectosCmd.SetOut(nil)

	runConsolaProyectos(consolaProyectosCmd, nil)

	var datos consola.DatosProyectos
	if err := json.Unmarshal(salida.Bytes(), &datos); err != nil {
		t.Fatalf("debe ser JSON: %v: %q", err, salida.String())
	}
	if len(datos.Proyectos) != 1 || datos.Proyectos[0].Nombre != "mi-proyecto" {
		t.Errorf("la fila del registro: %+v", datos)
	}
	entradas, err := os.ReadDir(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(entradas) != 0 {
		t.Errorf("no debe escribir en la carpeta donde corre: %v", entradas)
	}
}

func TestConsolaProyectos_EstaRegistradoConJSON(t *testing.T) {
	encontrado, _, err := rootCmd.Find([]string{"consola", "proyectos"})
	if err != nil || encontrado != consolaProyectosCmd {
		t.Fatalf("qdd consola proyectos debe existir: %v", err)
	}
	if encontrado.Flags().Lookup("json") == nil {
		t.Error("debe aceptar --json, como qdd consola panel")
	}
}
