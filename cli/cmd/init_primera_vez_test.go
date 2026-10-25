package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
	"gopkg.in/yaml.v3"
)

func escribir(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// FND-042: init solo reconocía Node, Go y Java; en Python, Flutter, monorepos y
// el resto de stacks dejaba languages: [] y el gatekeeper bloqueaba todo.
func TestDetectProjectMetadata_Stacks(t *testing.T) {
	casos := []struct {
		nombre, archivo, contenido string
		lenguaje, arquitectura     string
	}{
		{"fastapi", "requirements.txt", "fastapi==0.110.0\npytest==8.0.0\n", "Python", "Backend REST API (Python)"},
		{"python", "pyproject.toml", "[project]\nname = \"x\"\n", "Python", "Python Application"},
		{"flutter", "pubspec.yaml", "name: app\ndependencies:\n  flutter:\n    sdk: flutter\n", "Dart", "Mobile App (Flutter)"},
		{"dart", "pubspec.yaml", "name: cli\n", "Dart", "Dart Application"},
		{"rust", "Cargo.toml", "[dependencies]\naxum = \"0.7\"\n", "Rust", "Backend REST API (Rust)"},
		{"php", "composer.json", "{\"require\":{\"laravel/framework\":\"^11\"}}", "PHP", "Backend Web (Laravel)"},
		{"ruby", "Gemfile", "gem 'rails'\n", "Ruby", "Backend Web (Rails)"},
		{"dotnet", "Api.csproj", "<Project/>", "C#", ".NET Application"},
		{"kotlin", "build.gradle.kts", "plugins {}", "Java", "Java Application"},
	}
	for _, c := range casos {
		dir := t.TempDir()
		escribir(t, filepath.Join(dir, c.archivo), c.contenido)
		meta := detectProjectMetadata(dir)
		if !reflect.DeepEqual(meta.Languages, []string{c.lenguaje}) {
			t.Errorf("%s: Languages = %v, want [%s]", c.nombre, meta.Languages, c.lenguaje)
		}
		if meta.Architecture != c.arquitectura {
			t.Errorf("%s: Architecture = %q, want %q", c.nombre, meta.Architecture, c.arquitectura)
		}
	}
}

func TestDetectProjectMetadata_Monorepo(t *testing.T) {
	dir := t.TempDir()
	escribir(t, filepath.Join(dir, "backend", "go.mod"), "module x/api\n")
	escribir(t, filepath.Join(dir, "frontend", "package.json"), `{"name":"web","dependencies":{"vue":"^3"}}`)
	escribir(t, filepath.Join(dir, "node_modules", "lib", "package.json"), `{}`)
	escribir(t, filepath.Join(dir, "apps", "movil", "pubspec.yaml"), "name: m\ndependencies:\n  flutter:\n    sdk: flutter\n")

	meta := detectProjectMetadata(dir)

	if !reflect.DeepEqual(meta.Languages, []string{"Dart", "Go", "Node"}) {
		t.Errorf("Languages = %v, want [Dart Go Node]", meta.Languages)
	}
	want := "Monorepo: apps/movil (Mobile App (Flutter)), backend (Go Application), frontend (Frontend Web App)"
	if meta.Architecture != want {
		t.Errorf("Architecture = %q\nwant          %q", meta.Architecture, want)
	}
	if meta.Name != filepath.Base(dir) {
		t.Errorf("Name = %q, want el nombre de la carpeta", meta.Name)
	}
}

// Un proyecto vacío no se adivina: queda sin lenguaje y se declara con flags.
func TestDetectProjectMetadata_Vacio(t *testing.T) {
	dir := t.TempDir()
	meta := detectProjectMetadata(dir)
	if len(meta.Languages) != 0 || meta.Architecture != "" {
		t.Errorf("un proyecto vacío no debe inventar stack: %+v", meta)
	}
}

func leerConfig(t *testing.T, dir string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".qdd", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

// Si el proyecto se inicializó vacío y después llegó el código (o se pasan
// --lenguaje/--arquitectura), volver a correr init completa la configuración
// sin pisar el resto de config.yaml.
func TestCreateConfigFile_CompletaLenguajeVacio(t *testing.T) {
	dir := t.TempDir()
	qddDir := filepath.Join(dir, ".qdd")
	escribir(t, filepath.Join(qddDir, "config.yaml"), "# mi comentario\nproject: demo\narchitecture: \"\"\nlanguages: []\ntest_commands: [\"make test\"]\n")

	if err := createConfigFile(qddDir, ProjectMetadata{Name: "otro", Architecture: "Go Application", Languages: []string{"Go"}}); err != nil {
		t.Fatal(err)
	}

	cfg := leerConfig(t, dir)
	if cfg["architecture"] != "Go Application" || !reflect.DeepEqual(cfg["languages"], []interface{}{"Go"}) {
		t.Errorf("no completó lenguaje/arquitectura: %v", cfg)
	}
	if cfg["project"] != "demo" || cfg["test_commands"] == nil {
		t.Errorf("pisó la configuración existente: %v", cfg)
	}
	data, _ := os.ReadFile(filepath.Join(qddDir, "config.yaml"))
	if !strings.Contains(string(data), "# mi comentario") {
		t.Error("perdió los comentarios del usuario")
	}
}

func TestCreateConfigFile_NoPisaLenguajeDeclarado(t *testing.T) {
	dir := t.TempDir()
	qddDir := filepath.Join(dir, ".qdd")
	escribir(t, filepath.Join(qddDir, "config.yaml"), "project: demo\narchitecture: Hexagonal\nlanguages: [Go]\n")

	if err := createConfigFile(qddDir, ProjectMetadata{Architecture: "Frontend Web App", Languages: []string{"Node"}}); err != nil {
		t.Fatal(err)
	}

	cfg := leerConfig(t, dir)
	if cfg["architecture"] != "Hexagonal" || !reflect.DeepEqual(cfg["languages"], []interface{}{"Go"}) {
		t.Errorf("la detección no debe pisar lo que el usuario declaró: %v", cfg)
	}
}

func TestAplicarDeclaracion_FlagsMandan(t *testing.T) {
	meta := aplicarDeclaracion(ProjectMetadata{Languages: []string{"Node"}, Architecture: "Frontend Web App"}, []string{"Go", " Node "}, "API REST")
	if !reflect.DeepEqual(meta.Languages, []string{"Go", "Node"}) || meta.Architecture != "API REST" {
		t.Errorf("las flags deben mandar sobre la detección: %+v", meta)
	}
}

// FND-040: un proyecto nuevo queda en adopción gradual; uno existente (con
// config.yaml previo) conserva su modo.
func TestEnsureAdopcion(t *testing.T) {
	nuevo := t.TempDir()
	ensureAdopcion(nuevo, true)
	if p := audit.LoadPolicies(nuevo); p.Adopcion != audit.AdopcionGradual || !p.OWASP || !p.ZeroElse {
		t.Errorf("proyecto nuevo: políticas = %+v, want gradual con el resto por defecto", p)
	}

	existente := t.TempDir()
	ensureAdopcion(existente, false)
	if _, err := os.Stat(filepath.Join(existente, ".qdd", "policies.yaml")); err == nil {
		t.Error("no debe crear policies.yaml en un proyecto que ya existía")
	}

	declarado := t.TempDir()
	escribir(t, filepath.Join(declarado, ".qdd", "policies.yaml"), "owasp: true\nadopcion: estricta\n")
	ensureAdopcion(declarado, true)
	if p := audit.LoadPolicies(declarado); p.Adopcion != "estricta" {
		t.Errorf("no debe pisar policies.yaml: %+v", p)
	}
}

// FND-043: el primer init mostraba "Reparando contratos JSON del Dashboard"
// porque la estructura no creaba cognitive_history.json.
func TestCreateQDDStructure_CreaHistorialCognitivo(t *testing.T) {
	dir := t.TempDir()
	if err := createQDDStructure(dir, ProjectMetadata{Name: "x", Architecture: "Go Application", Languages: []string{"Go"}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".qdd", "project", "metrics", "cognitive_history.json"))
	if err != nil || strings.TrimSpace(string(data)) != "[]" {
		t.Errorf("cognitive_history.json = %q, %v", data, err)
	}
}
