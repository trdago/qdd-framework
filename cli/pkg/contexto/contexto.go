// Package contexto arma la lectura del proyecto que comparten los comandos de
// QDD (RFC 0005 §3.3): qué unidades tiene el repositorio y con qué stack.
// Antes cada comando buscaba por su cuenta, solo en la raíz y en la primera
// capa de subcarpetas, y las unidades que no veía quedaban fuera sin aviso.
package contexto

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Stacks que QDD reconoce por su manifiesto.
const (
	Go     = "go"
	Node   = "node"
	Python = "python"
	Dart   = "dart"
	Rust   = "rust"
	PHP    = "php"
	Ruby   = "ruby"
	JVM    = "jvm"
	DotNet = "dotnet"
)

var manifiestos = map[string]string{
	"go.mod":           Go,
	"package.json":     Node,
	"pyproject.toml":   Python,
	"setup.py":         Python,
	"requirements.txt": Python,
	"Pipfile":          Python,
	"pubspec.yaml":     Dart,
	"Cargo.toml":       Rust,
	"composer.json":    PHP,
	"Gemfile":          Ruby,
	"pom.xml":          JVM,
	"build.gradle":     JVM,
	"build.gradle.kts": JVM,
}

var extensionesDeManifiesto = map[string]string{
	".csproj": DotNet,
	".fsproj": DotNet,
	".sln":    DotNet,
}

// Carpetas de dependencias o de salida que nunca son una unidad del proyecto,
// aunque git no las ignore.
var carpetasAjenas = map[string]bool{
	"node_modules": true, "vendor": true, ".git": true, ".qdd": true,
	"dist": true, "build": true, "target": true, ".venv": true, "venv": true,
}

// Unidad es una parte del repositorio con su propio manifiesto.
type Unidad struct {
	Ruta   string   // relativa a la raíz, "." para la raíz
	Stacks []string // vacío si no se reconoce ningún manifiesto
}

// Contexto es la lectura del proyecto.
type Contexto struct {
	Raiz     string
	Unidades []Unidad
}

// Leer recorre todo el repositorio y devuelve sus unidades ordenadas por ruta.
// Si no encuentra ningún manifiesto, la raíz es una unidad sin stack.
func Leer(root string) Contexto {
	stacks := map[string]map[string]bool{}
	Recorrer(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && path != root && carpetasAjenas[d.Name()] {
			return filepath.SkipDir
		}
		stack, ok := stackDeManifiesto(d)
		if !ok {
			return nil
		}
		rel := relativa(root, filepath.Dir(path))
		if stacks[rel] == nil {
			stacks[rel] = map[string]bool{}
		}
		stacks[rel][stack] = true
		return nil
	})
	return Contexto{Raiz: root, Unidades: unidades(stacks)}
}

func stackDeManifiesto(d fs.DirEntry) (string, bool) {
	if d.IsDir() {
		return "", false
	}
	if s, ok := manifiestos[d.Name()]; ok {
		return s, true
	}
	s, ok := extensionesDeManifiesto[filepath.Ext(d.Name())]
	return s, ok
}

func unidades(stacks map[string]map[string]bool) []Unidad {
	if len(stacks) == 0 {
		return []Unidad{{Ruta: "."}}
	}
	var out []Unidad
	for ruta, set := range stacks {
		u := Unidad{Ruta: ruta}
		for s := range set {
			u.Stacks = append(u.Stacks, s)
		}
		sort.Strings(u.Stacks)
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ruta < out[j].Ruta })
	return out
}

func relativa(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "."
	}
	return filepath.ToSlash(rel)
}

// UnidadDe devuelve la unidad más específica que contiene el archivo (ruta
// relativa a la raíz). Un archivo fuera de toda unidad pertenece a la raíz.
func (c Contexto) UnidadDe(archivo string) Unidad {
	archivo = filepath.ToSlash(archivo)
	mejor := Unidad{Ruta: "."}
	for _, u := range c.Unidades {
		if contiene(u.Ruta, archivo) && len(u.Ruta) >= len(mejor.Ruta) {
			mejor = u
		}
	}
	return mejor
}

func contiene(ruta, archivo string) bool {
	return ruta == "." || strings.HasPrefix(archivo, ruta+"/")
}

// Recorrer recorre el proyecto como filepath.WalkDir, pero salta lo que no
// pertenece al código versionado del proyecto:
//   - repos anidados y worktrees (subcarpetas con su propio .git, p. ej.
//     .claude/worktrees/agent-*), que tienen su propio ciclo de auditoría;
//   - carpetas que git ignora (.gitignore), que no se despliegan ni se commitean.
//
// Sin esto el gate bloqueaba commits por violaciones en copias temporales de
// agentes, y la gente terminaba saltándose el gate (FND-038).
func Recorrer(root string, fn fs.WalkDirFunc) error {
	ignored := gitIgnoredDirs(root)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && path != root && isExcludedDir(path, ignored) {
			return filepath.SkipDir
		}
		return fn(path, d, err)
	})
}

func isExcludedDir(path string, ignored map[string]bool) bool {
	if ignored[filepath.Clean(path)] {
		return true
	}
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

var ignoredDirsCache sync.Map

// gitIgnoredDirs devuelve las carpetas ignoradas por git (rutas absolutas).
// Si root no es un repo git, o git no está disponible, devuelve un mapa vacío.
func gitIgnoredDirs(root string) map[string]bool {
	if cached, ok := ignoredDirsCache.Load(root); ok {
		return cached.(map[string]bool)
	}

	dirs := map[string]bool{}
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard", "--directory")
	cmd.Dir = root
	out, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.HasSuffix(line, "/") {
				continue
			}
			dirs[filepath.Clean(filepath.Join(root, line))] = true
		}
	}

	ignoredDirsCache.Store(root, dirs)
	return dirs
}
