package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cwdActual(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(wd)
	return real
}

// FND-044: todo comando fallaba desde una subcarpeta ("No existe la carpeta
// .qdd/") porque el gatekeeper y los comandos usaban el directorio actual.
func TestEnterProjectRoot_DesdeSubcarpeta(t *testing.T) {
	raiz, _ := filepath.EvalSymlinks(t.TempDir())
	os.MkdirAll(filepath.Join(raiz, ".qdd"), 0755)
	os.MkdirAll(filepath.Join(raiz, ".git"), 0755)
	sub := filepath.Join(raiz, "internal", "usuarios")
	os.MkdirAll(sub, 0755)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(sub)

	enterProjectRoot("audit")

	if got := cwdActual(t); got != raiz {
		t.Errorf("cwd = %q, want la raíz %q", got, raiz)
	}
}

// init decide su propia raíz; y sin .qdd arriba no hay a dónde ir.
func TestEnterProjectRoot_NoMueveSinProyecto(t *testing.T) {
	raiz, _ := filepath.EvalSymlinks(t.TempDir())
	os.MkdirAll(filepath.Join(raiz, ".git"), 0755)
	sub := filepath.Join(raiz, "src")
	os.MkdirAll(sub, 0755)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(sub)

	enterProjectRoot("audit")
	if got := cwdActual(t); got != sub {
		t.Errorf("sin .qdd no debe moverse: cwd = %q", got)
	}

	os.MkdirAll(filepath.Join(raiz, ".qdd"), 0755)
	enterProjectRoot("init")
	if got := cwdActual(t); got != sub {
		t.Errorf("init no debe moverse: cwd = %q", got)
	}
}

// FND-046: `qdd run` lanzaba os.Args[0] ("qdd") con os.StartProcess, que no
// busca en el PATH: fallaba siempre que QDD se invocaba por su nombre.
func TestSelfExecutable_EsRutaAbsoluta(t *testing.T) {
	exe := selfExecutable()
	if !filepath.IsAbs(exe) {
		t.Fatalf("selfExecutable = %q, want una ruta absoluta", exe)
	}
	if _, err := os.Stat(exe); err != nil {
		t.Fatalf("selfExecutable apunta a un archivo inexistente: %v", err)
	}
}

// FND-045: la ayuda tenía la primera frase cortada y comandos sin descripción.
func TestHelp_TodosLosComandosDescritos(t *testing.T) {
	if strings.Contains(rootCmd.Long, "obligatorias.\ngarantizando") {
		t.Error("la descripción de qdd tiene la frase cortada")
	}
	for _, c := range rootCmd.Commands() {
		if !c.Hidden && strings.TrimSpace(c.Short) == "" {
			t.Errorf("el comando %q no tiene descripción en qdd --help", c.Name())
		}
	}
}
