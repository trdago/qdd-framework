//go:build !windows

package consola

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// abridorFalso reemplaza el abridor del sistema por un script que anota la carpeta recibida y sale con codigo.
func abridorFalso(t *testing.T, codigo int) string {
	t.Helper()
	dir := t.TempDir()
	anotado := filepath.Join(dir, "abierta.txt")
	script := filepath.Join(dir, "abridor-falso")
	cuerpo := "#!/bin/sh\nprintf '%s' \"$1\" > '" + anotado + "'\nexit " + string(rune('0'+codigo)) + "\n"
	if err := os.WriteFile(script, []byte(cuerpo), 0755); err != nil {
		t.Fatal(err)
	}
	previo := buscarAbridor
	buscarAbridor = func() (Abridor, bool) { return Abridor{Programa: script}, true }
	t.Cleanup(func() { buscarAbridor = previo })
	return anotado
}

func TestAcciones_AbrirCarpetaSeOfreceSinConfirmar_CCC_C21(t *testing.T) {
	abridorFalso(t, 0)
	root := proyectoConFichas(t, nil, nil)
	got := ids(Acciones(root, false))
	if strings.Join(got, ",") != "auditar,abrir-carpeta" {
		t.Fatalf("con abridor se ofrece abrir la carpeta después de auditar: %v", got)
	}
	a, _ := BuscarAccion(Acciones(root, false), "abrir-carpeta")
	if a.Tipo != TipoAbrir || a.Confirmar || a.Detalle != "abridor-falso "+root {
		t.Fatalf("abrir la carpeta no escribe: sin confirmación y con el comando equivalente: %+v", a)
	}
	if err := Autorizar(a, false); err != nil {
		t.Fatalf("abrir la carpeta no pide --confirmado: %v", err)
	}
}

func TestAcciones_SinAbridorNoSeOfreceAbrir(t *testing.T) {
	root := proyectoConFichas(t, nil, nil)
	if _, err := BuscarAccion(Acciones(root, false), "abrir-carpeta"); err == nil {
		t.Fatal("sin xdg-open/open/explorer no se ofrece abrir la carpeta")
	}
}

func TestEjecutar_AbrirCarpetaPasaLaRaizAlAbridor_CCC_C21(t *testing.T) {
	anotado := abridorFalso(t, 0)
	root := proyectoConFichas(t, nil, nil)
	a, err := BuscarAccion(Acciones(root, false), "abrir-carpeta")
	if err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd-que-no-se-usa", &salida); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, salida.String())
	}
	var abierta []byte
	for i := 0; i < 50 && len(abierta) == 0; i++ {
		abierta, _ = os.ReadFile(anotado)
		time.Sleep(20 * time.Millisecond)
	}
	if string(abierta) != root {
		t.Fatalf("el abridor debió recibir la raíz %q; recibió %q", root, abierta)
	}
	if !strings.Contains(salida.String(), "Carpeta abierta") {
		t.Fatalf("salida: %q", salida.String())
	}
}

func TestEjecutar_AbridorQueFallaSeInforma(t *testing.T) {
	abridorFalso(t, 4)
	root := proyectoConFichas(t, nil, nil)
	a, _ := BuscarAccion(Acciones(root, false), "abrir-carpeta")
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd-que-no-se-usa", &salida); codigo != 1 || !strings.Contains(salida.String(), "aplicación predeterminada para carpetas") {
		t.Fatalf("un abridor que falla rápido se informa: código %d, %q", codigo, salida.String())
	}
}

func TestPanel_TraeLaCarpetaDelProyecto(t *testing.T) {
	root := proyectoConFichas(t, nil, nil)
	out, _ := json.Marshal(Panel(root, time.Now()))
	want, _ := json.Marshal(root)
	if !strings.Contains(string(out), `"carpeta":`+string(want)) {
		t.Fatalf("el panel recibe la raíz para el enlace: %s", out)
	}
}
