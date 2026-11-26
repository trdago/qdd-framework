package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// FND-191: el baúl pide permiso a una sesión que late, y nada decía que la sesión de la IA no latía. Doctor lo revisa
// desde la misma sesión (CLAUDE_CODE_SESSION_ID).

func TestCheckSesionLate_SinSesionDeClaudeCodeNoSeRevisa(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	if item := checkSesionLate(t.TempDir(), time.Now()); item != nil {
		t.Fatalf("fuera de una sesión no hay qué revisar: %+v", item)
	}
}

func TestCheckSesionLate_LaSesionQueLateEstaBien(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv("CLAUDE_CODE_SESSION_ID", "s1")
	root := t.TempDir()
	now := time.Now()
	if err := consola.LatirSesion(root, "s1", now); err != nil {
		t.Fatal(err)
	}
	if item := checkSesionLate(root, now); item == nil || !item.Success {
		t.Fatalf("la sesión late: %+v", item)
	}
}

func TestCheckSesionLate_LaQueNoLateFallaYDiceQueHacer_FND191(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv("CLAUDE_CODE_SESSION_ID", "s1")
	item := checkSesionLate(t.TempDir(), time.Now())
	if item == nil || item.Success || !strings.Contains(item.Error, "baúl") || !strings.Contains(item.Error, "/reload-plugins") {
		t.Fatalf("falla, dice para qué importa y qué hacer: %+v", item)
	}
}

// El reporte de MEJ-056: el proyecto es una subcarpeta de otro que también tiene .qdd.
func TestCheckSesionLate_DiceSiLateEnLaCarpetaDeArriba_FND191(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv("CLAUDE_CODE_SESSION_ID", "s1")
	padre := t.TempDir()
	hijo := filepath.Join(padre, "app")
	for _, d := range []string{filepath.Join(padre, ".qdd"), filepath.Join(hijo, ".qdd")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	if err := consola.LatirSesion(padre, "s1", now); err != nil {
		t.Fatal(err)
	}
	item := checkSesionLate(hijo, now)
	if item == nil || item.Success || !strings.Contains(item.Error, padre) {
		t.Fatalf("dice que la sesión late en la carpeta de arriba: %+v", item)
	}
}
