package audit

import (
	"path/filepath"
	"strings"
	"testing"
)

// FND-047: un proyecto con pytest en requirements.txt y carpeta tests/ recibía
// QA-00-NO-TEST-COMMAND: solo se reconocía pytest.ini, pyproject.toml o setup.cfg.
func TestDetectTestCommands_PytestDesdeRequirements(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "requirements.txt"), "fastapi==0.110.0\npytest==8.0.0\n")
	writeFile(t, filepath.Join(dir, "tests", "test_main.py"), "def test_x():\n    assert True\n")

	got := labels(detectTestCommands(dir))
	if !strings.HasSuffix(got, "-m pytest") {
		t.Errorf("detectTestCommands = %q, want pytest", got)
	}
}

func TestDetectTestCommands_PytestPorArchivosDePrueba(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "requirements.txt"), "flask\n")
	writeFile(t, filepath.Join(dir, "tests", "test_api.py"), "")

	if got := labels(detectTestCommands(dir)); !strings.HasSuffix(got, "-m pytest") {
		t.Errorf("detectTestCommands = %q, want pytest", got)
	}
}

// Una carpeta tests/ sin nada de Python no es una suite de pytest.
func TestDetectTestCommands_TestsSinPythonNoEsPytest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "tests", "README.md"), "")
	if got := labels(detectTestCommands(dir)); got != "" {
		t.Errorf("detectTestCommands = %q, want nada", got)
	}
}

// FND-049: una app Flutter recién creada con test/ vacía corría `flutter test`,
// que falla con "does not appear to contain any test files": un QA-01 falso.
func TestDetectTestCommands_FlutterSinArchivosDePrueba(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pubspec.yaml"), "name: app\n")
	writeFile(t, filepath.Join(dir, "test", ".gitkeep"), "")
	if got := labels(detectTestCommands(dir)); got != "" {
		t.Errorf("detectTestCommands = %q, want nada (no hay *_test.dart)", got)
	}

	writeFile(t, filepath.Join(dir, "test", "widgets", "home_test.dart"), "")
	if got := labels(detectTestCommands(dir)); got != "flutter test" {
		t.Errorf("detectTestCommands = %q, want flutter test", got)
	}
}

// FND-049: QA-00 decía "no se detectó cómo ejecutar las pruebas" sin decir qué
// falta en el stack concreto.
func TestRunCoverageCheck_SinPruebasDiceQueFalta(t *testing.T) {
	casos := map[string]struct{ archivo, contenido, pista string }{
		"node":    {"package.json", `{"scripts":{"dev":"vite"}}`, `script "test"`},
		"flutter": {"pubspec.yaml", "name: app\n", "_test.dart"},
		"python":  {"requirements.txt", "flask\n", "tests/test_"},
	}
	for nombre, c := range casos {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, c.archivo), c.contenido)
		desc := descripcionDe(RunCoverageCheck(dir), "QA-00-NO-TEST-COMMAND")
		if !strings.Contains(desc, c.pista) {
			t.Errorf("%s: QA-00 = %q, want que mencione %q", nombre, desc, c.pista)
		}
	}
}

// FND-049: en un clon recién hecho las pruebas fallan por dependencias sin
// instalar; QA-01 debe decirlo en vez de solo "exit status 2".
func TestPistaDeFallo_DependenciasFaltantes(t *testing.T) {
	salidas := []string{
		"E   ModuleNotFoundError: No module named 'fastapi'",
		"Error: Cannot find module 'vitest'",
		"sh: 1: vite: not found",
		`exec: "flutter": executable file not found in $PATH`,
	}
	for _, s := range salidas {
		if !strings.Contains(pistaDeFallo(s), "dependencias") {
			t.Errorf("sin pista para %q", s)
		}
	}
	if pistaDeFallo("--- FAIL: TestSuma (0.00s)") != "" {
		t.Error("un fallo de prueba real no lleva pista de dependencias")
	}
}

func descripcionDe(violations []Violation, regla string) string {
	for _, v := range violations {
		if v.RuleID == regla {
			return v.Description
		}
	}
	return ""
}

// FND-077: al correr pytest, el gate dejaba __pycache__/ en las carpetas de
// código y .pytest_cache/ en la raíz del proyecto.
func TestResolveTestCommands_PytestNoEnsuciaElProyecto(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "requirements.txt"), "pytest\n")
	writeFile(t, filepath.Join(dir, "tests", "test_x.py"), "")
	writeFile(t, filepath.Join(dir, "tmp", ".gitignore"), "*\n")

	cmds := resolveTestCommands(dir)
	if len(cmds) != 1 {
		t.Fatalf("se esperaba pytest, se obtuvo %q", labels(cmds))
	}
	tc := cmds[0]
	if !tieneVariable(tc.Env, "PYTEST_ADDOPTS=", "-p no:cacheprovider") {
		t.Errorf("pytest no debe crear .pytest_cache: env = %v", tc.Env)
	}
	want := "PYTHONPYCACHEPREFIX=" + filepath.Join(dir, "tmp", "pycache")
	if !contiene(tc.Env, want) {
		t.Errorf("la caché de Python debe ir a ./tmp: env = %v, want %s", tc.Env, want)
	}
}

// Sin ./tmp administrada por QDD, la caché va a la carpeta temporal del sistema.
func TestResolveTestCommands_PytestSinTmp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "requirements.txt"), "pytest\n")
	writeFile(t, filepath.Join(dir, "tests", "test_x.py"), "")

	tc := resolveTestCommands(dir)[0]
	for _, e := range tc.Env {
		if strings.HasPrefix(e, "PYTHONPYCACHEPREFIX=") && strings.HasPrefix(e, "PYTHONPYCACHEPREFIX="+dir) {
			t.Errorf("sin ./tmp no debe escribir caché dentro del proyecto: %s", e)
		}
	}
	if len(tc.Env) == 0 {
		t.Error("debe definir PYTHONPYCACHEPREFIX igual")
	}
}

// Los comandos declarados (p. ej. "make test" que llama a pytest) también.
func TestResolveTestCommands_DeclaradosTambienSinCache(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "config.yaml"), "test_commands: [\"make test\"]\n")
	tc := resolveTestCommands(dir)[0]
	if !tieneVariable(tc.Env, "PYTHONPYCACHEPREFIX=", "") || !tieneVariable(tc.Env, "PYTEST_ADDOPTS=", "no:cacheprovider") {
		t.Errorf("los comandos declarados deben correr sin ensuciar: env = %v", tc.Env)
	}
}

func tieneVariable(env []string, prefijo, contenido string) bool {
	for _, e := range env {
		if strings.HasPrefix(e, prefijo) && strings.Contains(e, contenido) {
			return true
		}
	}
	return false
}

func contiene(lista []string, s string) bool {
	for _, x := range lista {
		if x == s {
			return true
		}
	}
	return false
}

// FND-047: en un monorepo (backend/ y frontend/) el gate no encontraba las
// pruebas porque solo miraba la raíz y cli/.
func TestDetectTestCommands_Monorepo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "backend", "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "frontend", "package.json"), `{"scripts":{"test":"vitest run"}}`)
	writeFile(t, filepath.Join(dir, "node_modules", "lib", "package.json"), `{"scripts":{"test":"jest"}}`)

	cmds := detectTestCommands(dir)
	got := labels(cmds)
	want := "go test ./... (backend) | npm run test (frontend)"
	if got != want {
		t.Errorf("detectTestCommands = %q, want %q", got, want)
	}
	if len(cmds) == 2 && cmds[0].Dir != filepath.Join(dir, "backend") {
		t.Errorf("go test debe correr dentro de backend/, corre en %q", cmds[0].Dir)
	}
}
