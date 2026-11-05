package consola

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'

func TestBinarioPorVersion(t *testing.T) {
	si := []string{`C:\Users\ana\AppData\Local\qdd\bin\2.10.0-rc.3\qdd.exe`, "/tmp/qdd/bin/2.10.0/qdd.exe"}
	for _, ruta := range si {
		if !BinarioPorVersion(ruta) {
			t.Errorf("%s es un binario por versión", ruta)
		}
	}
	no := []string{`C:\npm\node_modules\qdd-framework\bin\qdd.exe`, "/usr/lib/node_modules/qdd-framework/bin/qdd", "/home/ana/qdd/bin/rama/qdd.exe"}
	for _, ruta := range no {
		if BinarioPorVersion(ruta) {
			t.Errorf("%s no es un binario por versión", ruta)
		}
	}
}

func TestRutaVigilada(t *testing.T) {
	if got := RutaVigilada(binNPM); got != binNPM {
		t.Fatalf("en una instalación normal se vigila el binario: %s", got)
	}
	bin := filepath.Join("base", "qdd", "bin", "2.10.0", "qdd.exe")
	if got, quiere := RutaVigilada(bin), filepath.Join("base", "qdd", "bin", "actual.txt"); got != quiere {
		t.Fatalf("RutaVigilada = %s; quiere %s", got, quiere)
	}
}

// El qdd.exe de una versión no cambia al actualizar: el flujo tiene que enterarse por el puntero.
func TestVigiaBinario_PorVersionMiraElPuntero(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "qdd", "bin")
	bin := filepath.Join(dir, "2.10.0", "qdd.exe")
	if err := os.MkdirAll(filepath.Dir(bin), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("exe"), 0o700); err != nil {
		t.Fatal(err)
	}
	puntero := filepath.Join(dir, "actual.txt")
	if err := os.WriteFile(puntero, []byte("2.10.0"), 0o600); err != nil {
		t.Fatal(err)
	}
	cambio := VigiaBinario(bin)
	if cambio() {
		t.Fatal("sin instalar nada, no hay cambio")
	}
	if err := os.WriteFile(puntero, []byte("2.10.0-rc.9"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !cambio() {
		t.Fatal("otra versión instalada termina el flujo, aunque su propio qdd.exe siga igual")
	}
}
