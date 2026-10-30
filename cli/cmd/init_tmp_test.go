package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/modules"
)

func gitInit(t *testing.T, dir string) {
	t.Helper()
	if err := exec.Command("git", "-C", dir, "init", "-q").Run(); err != nil {
		t.Skipf("git no disponible: %v", err)
	}
}

func gitStatus(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// FND-075: los scripts de un solo uso, logs y salidas quedaban sueltos en la
// raíz del proyecto. init crea ./tmp, invisible para git sin tocar el
// .gitignore del proyecto.
func TestEnsureTmpDir_CreaCarpetaIgnorada(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)

	if aviso := ensureTmpDir(dir); aviso != "" {
		t.Fatalf("aviso inesperado: %s", aviso)
	}
	escribir(t, filepath.Join(dir, tmpDir, "probar_api.py"), "print('hola')\n")

	if st := gitStatus(t, dir); strings.Contains(st, tmpDir) {
		t.Errorf("git no debe ver ./tmp ni su contenido:\n%s", st)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err == nil {
		t.Error("no debe crear ni tocar el .gitignore del proyecto")
	}

	ensureTmpDir(dir)
	if _, err := os.Stat(filepath.Join(dir, tmpDir, "probar_api.py")); err != nil {
		t.Error("repetir init no debe borrar lo que hay en ./tmp")
	}
}

// Si el proyecto ya usa tmp/ para archivos versionados, QDD no se lo apropia.
func TestEnsureTmpDir_RespetaTmpVersionado(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)
	escribir(t, filepath.Join(dir, "tmp", "fixture.json"), "{}")
	exec.Command("git", "-C", dir, "add", "tmp/fixture.json").Run()

	aviso := ensureTmpDir(dir)
	if !strings.Contains(aviso, "tmp/") {
		t.Errorf("debe avisar que tmp/ tiene archivos versionados, aviso = %q", aviso)
	}
	if _, err := os.Stat(filepath.Join(dir, "tmp", ".gitignore")); err == nil {
		t.Error("no debe ignorar un tmp/ que el proyecto versiona")
	}
}

func TestEnsureTmpDir_SinGit(t *testing.T) {
	dir := t.TempDir()
	if aviso := ensureTmpDir(dir); aviso != "" {
		t.Fatalf("aviso inesperado sin git: %s", aviso)
	}
	if _, err := os.Stat(filepath.Join(dir, tmpDir, ".gitignore")); err != nil {
		t.Error("sin git también debe quedar lista para cuando se inicie el repositorio")
	}
}

// FND-075: las IA deben saber dónde van los archivos temporales.
func TestReglasIA_MencionanTmp(t *testing.T) {
	if !strings.Contains(integration.QDDCommandProtocol, "./tmp") {
		t.Error("las reglas sincronizadas (CLAUDE.md, AGENTS.md…) deben indicar ./tmp")
	}
	if !strings.Contains(modules.MetodoDeTrabajo(), "./tmp") {
		t.Error("el método de trabajo de qdd_project_context debe indicar ./tmp")
	}
}

// FND-076: QDD versionaba archivos __pycache__ en .qdd/core y los copiaba a
// cada proyecto en init.
func TestUnpackCoreAssets_SinPycache(t *testing.T) {
	dir := t.TempDir()
	qddDir := filepath.Join(dir, ".qdd")
	if err := unpackCoreAssets(qddDir); err != nil {
		t.Fatal(err)
	}
	filepath.WalkDir(qddDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && (d.Name() == "__pycache__" || strings.HasSuffix(d.Name(), ".pyc")) {
			t.Errorf("init copió caché de Python al proyecto: %s", path)
		}
		return nil
	})
}

func TestLimpiarPycacheDelCore(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, ".qdd", "core", "plugins", "docs", "__pycache__")
	escribir(t, filepath.Join(cache, "engine.cpython-314.pyc"), "x")
	propio := filepath.Join(dir, "src", "__pycache__", "app.pyc")
	escribir(t, propio, "x")

	limpiarPycacheDelCore(filepath.Join(dir, ".qdd"))

	if _, err := os.Stat(cache); err == nil {
		t.Error("debe borrar el __pycache__ que QDD dejó en .qdd/core")
	}
	if _, err := os.Stat(propio); err != nil {
		t.Error("no debe tocar nada fuera de .qdd/core")
	}
}
