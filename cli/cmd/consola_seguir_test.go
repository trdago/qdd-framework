package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

func TestLatidoSeguir_EscribeUnaLineaVacia(t *testing.T) {
	var salida bytes.Buffer
	if err := latidoSeguir(func() bool { return false }, &salida)(); err != nil {
		t.Fatal(err)
	}
	if salida.String() != "\n" {
		t.Fatalf("salida = %q", salida.String())
	}
}

func TestLatidoSeguir_ConBinarioNuevoTerminaSinEscribir(t *testing.T) {
	var salida bytes.Buffer
	err := latidoSeguir(func() bool { return true }, &salida)()
	if !errors.Is(err, errBinarioNuevo) {
		t.Fatalf("err = %v", err)
	}
	if salida.Len() != 0 {
		t.Fatalf("salida = %q", salida.String())
	}
}

// El panel recibe del flujo lo mismo que de 'consola panel --json': sin estado.yaml, el motivo de siempre.
func TestLecturaPanel_EntregaElJSONDelPanel(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := lecturaPanel(root, nil)()
	if err != nil {
		t.Fatal(err)
	}
	var d consola.DatosConsola
	if err := json.Unmarshal(out, &d); err != nil {
		t.Fatal(err)
	}
	if d.Formato != 1 || d.Motivo == "" || d.Estado != nil {
		t.Fatalf("datos = %+v", d)
	}
}
