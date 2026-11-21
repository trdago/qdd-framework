package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Las pruebas de este paquete corren con un HOME y un TopePadres temporales: ninguna llega a los AGENTS.md o CLAUDE.md
// de las carpetas de la persona (~/AGENTS.md, ~/Documents/proyectos/AGENTS.md), aunque olvide fijar el suyo.

// homeDeLaPersona es el HOME con que se lanzó go test, antes de que TestMain lo cambie.
var homeDeLaPersona string

func TestMain(m *testing.M) {
	homeDeLaPersona, _ = os.UserHomeDir()
	caja, err := os.MkdirTemp("", "qdd-integration-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "no se pudo crear el HOME de prueba:", err)
		os.Exit(1)
	}
	os.Setenv("HOME", caja)
	os.Setenv("USERPROFILE", caja)
	TopePadres = func() string { return caja }
	codigo := m.Run()
	os.RemoveAll(caja)
	os.Exit(codigo)
}

// fueraDe dice si ruta no es tope ni cuelga de él.
func fueraDe(ruta, tope string) bool {
	rel, err := filepath.Rel(tope, ruta)
	return err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// contrato: integracion#92b2bc8e
func TestPruebas_NoAlcanzanLosAGENTSDeLaPersona(t *testing.T) {
	tope := TopePadres()
	if tope == "" || tope == homeDeLaPersona {
		t.Fatalf("durante las pruebas el tope de las carpetas padre no es el HOME de la persona: %q", tope)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if !fueraDe(cwd, tope) {
		t.Fatalf("el repositorio (%s) cuelga del tope de prueba %s: sus padres reales entrarían", cwd, tope)
	}
	if copias := BloquesEnPadres(cwd); len(copias) != 0 {
		t.Fatalf("desde el repositorio no se alcanza ningún AGENTS.md de afuera: %v", copias)
	}
	if home, _ := os.UserHomeDir(); home == homeDeLaPersona {
		t.Fatalf("el HOME de las pruebas es el de la persona: %s", home)
	}
}
