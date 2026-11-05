package contexto

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func escribir(t *testing.T, root, rel, contenido string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(contenido), 0644); err != nil {
		t.Fatal(err)
	}
}

// RFC 0005 §3.3: QDD revisaba la raíz y, solo si ahí no había nada, la
// primera capa de subcarpetas. Con un frontend en la raíz, backend/ se
// ignoraba. El Contexto recorre todo el repositorio.
func TestLeer_EncuentraTodasLasUnidades(t *testing.T) {
	root := t.TempDir()
	escribir(t, root, "package.json", `{"name":"web"}`)
	escribir(t, root, "backend/go.mod", "module backend\n")
	escribir(t, root, "services/pagos/pyproject.toml", "[project]\n")
	escribir(t, root, "services/pagos/requirements.txt", "pytest\n")
	escribir(t, root, "apps/movil/pubspec.yaml", "name: movil\n")
	escribir(t, root, "tools/cli/Cargo.toml", "[package]\n")
	escribir(t, root, "node_modules/lib/package.json", `{}`)
	escribir(t, root, "backend/vendor/x/go.mod", "module x\n")

	got := Leer(root).Unidades
	want := []Unidad{
		{Ruta: ".", Stacks: []string{Node}},
		{Ruta: "apps/movil", Stacks: []string{Dart}},
		{Ruta: "backend", Stacks: []string{Go}},
		{Ruta: "services/pagos", Stacks: []string{Python}},
		{Ruta: "tools/cli", Stacks: []string{Rust}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unidades =\n%+v\nwant\n%+v", got, want)
	}
}

func TestLeer_UnaUnidadConVariosStacks(t *testing.T) {
	root := t.TempDir()
	escribir(t, root, "go.mod", "module x\n")
	escribir(t, root, "package.json", `{}`)
	escribir(t, root, "app.csproj", "<Project/>")

	got := Leer(root).Unidades
	want := []Unidad{{Ruta: ".", Stacks: []string{DotNet, Go, Node}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unidades = %+v; want %+v", got, want)
	}
}

// Un repositorio sin manifiestos reconocidos tiene una sola unidad sin stack:
// sus principios de código quedan "no evaluados", no aprobados en silencio.
func TestLeer_SinManifiestos(t *testing.T) {
	root := t.TempDir()
	escribir(t, root, "main.c", "int main(){}\n")

	got := Leer(root).Unidades
	want := []Unidad{{Ruta: "."}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unidades = %+v; want %+v", got, want)
	}
}

func TestLeer_SaltaLoQueGitIgnoraYLosReposAnidados(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no disponible")
	}
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	escribir(t, root, ".gitignore", "build/\n")
	escribir(t, root, "go.mod", "module x\n")
	escribir(t, root, "build/package.json", `{}`)
	escribir(t, root, ".claude/worktrees/agent-1/.git", "gitdir: /otro\n")
	escribir(t, root, ".claude/worktrees/agent-1/go.mod", "module copia\n")

	got := Leer(root).Unidades
	want := []Unidad{{Ruta: ".", Stacks: []string{Go}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unidades = %+v; want %+v", got, want)
	}
}

func TestUnidadDe(t *testing.T) {
	c := Contexto{Unidades: []Unidad{{Ruta: "."}, {Ruta: "cli"}, {Ruta: "cli/ui"}}}
	casos := map[string]string{
		"main.go":            ".",
		"cli/cmd/root.go":    "cli",
		"cli/ui/src/App.vue": "cli/ui",
		"client/x.go":        ".",
	}
	for archivo, want := range casos {
		if got := c.UnidadDe(archivo).Ruta; got != want {
			t.Errorf("UnidadDe(%q) = %q; want %q", archivo, got, want)
		}
	}
}
