package blueprints

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Un repositorio de blueprints se clona sin prompts interactivos y con protocol.ext.allow=never. Las pruebas ponen una
// configuración de git (la del HOME, como la de la persona) que sí permite ext:: y no dan ningún GIT_ASKPASS: solo lo
// que gitRaw fija evita que un origen ext:: ejecute un comando o que git se quede esperando una clave en la terminal.

// gitDeLaPersona deja un HOME cuya configuración de git permite ext:: y sin ayudantes de contraseña.
func gitDeLaPersona(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("el origen ext:: de prueba usa sh")
	}
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[protocol \"ext\"]\n\tallow = always\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_ASKPASS", "")
	t.Setenv("SSH_ASKPASS", "")
	t.Setenv("GIT_TERMINAL_PROMPT", "")
	os.Unsetenv("GIT_TERMINAL_PROMPT")
}

// contrato: blueprints#274de549
func TestAdd_UnOrigenExtNoEjecutaNada(t *testing.T) {
	gitDeLaPersona(t)
	marca := filepath.Join(t.TempDir(), "marca")
	_, problemas := Add(AddInput{Root: t.TempDir(), Origen: "ext::sh -c touch% " + marca})
	if len(problemas) == 0 {
		t.Fatal("un origen ext:: no se importa")
	}
	if _, err := os.Stat(marca); err == nil {
		t.Fatal("un origen ext:: ejecutó un comando en esta máquina (protocol.ext.allow=never)")
	}
}

// contrato: blueprints#274de549
func TestAdd_UnRepositorioQuePideClaveFallaSinPreguntarEnLaTerminal(t *testing.T) {
	gitDeLaPersona(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="blueprints"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	listo := make(chan []string, 1)
	go func() {
		_, problemas := Add(AddInput{Root: t.TempDir(), Origen: srv.URL + "/org/qdd-blueprints.git"})
		listo <- problemas
	}()
	select {
	case problemas := <-listo:
		if len(problemas) == 0 || !strings.Contains(strings.Join(problemas, "\n"), "terminal prompts disabled") {
			t.Fatalf("sin credenciales git falla sin preguntar en la terminal (GIT_TERMINAL_PROMPT=0): %v", problemas)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("git se quedó esperando una clave")
	}
}
