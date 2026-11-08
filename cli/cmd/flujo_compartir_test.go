package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 1: qdd flujo compartir escribe el archivo de solo lectura, por defecto en ./tmp del proyecto.

func TestFlujo_CompartirEscribeElArchivoEnTmp(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir()) // el baúl de prueba está vacío: nunca se lee el llavero real
	root := proyectoConFlujo(t)
	salida := correrFlujo(t, flujoCompartirCmd, map[string]string{"para": "Ana Pérez"}, "pago")
	archivo := filepath.Join(root, "tmp", "flujo-pago-ana-perez.html")
	datos, err := os.ReadFile(archivo)
	if err != nil {
		t.Fatalf("el archivo queda en ./tmp: %v (%s)", err, salida)
	}
	if !strings.Contains(salida, archivo) || !strings.Contains(salida, "solo lectura") {
		t.Fatalf("dice dónde quedó y que es de solo lectura: %q", salida)
	}
	if !strings.Contains(string(datos), "Cómo se cobra") || strings.Contains(string(datos), `"rc"`) {
		t.Fatal("lleva el flujo, pero no la tarea del tablero")
	}
	otro := filepath.Join(t.TempDir(), "x.html")
	correrFlujo(t, flujoCompartirCmd, map[string]string{"salida": otro}, "pago")
	if _, err := os.Stat(otro); err != nil {
		t.Fatalf("--salida elige el archivo: %v", err)
	}
}
