package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// detectores reconocen un stack a partir de sus archivos de proyecto. El orden
// importa: el primero que fija la arquitectura gana (Node la fija siempre).
var detectores = []func(dir string, meta *ProjectMetadata){
	detectNodeJS,
	detectGo,
	detectJava,
	detectPython,
	detectDart,
	detectRust,
	detectPHP,
	detectRuby,
	detectDotNet,
}

// carpetasSinCodigoPropio no se miran al buscar subproyectos.
var carpetasSinCodigoPropio = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	"target": true, "venv": true, "__pycache__": true, "docs": true,
}

func detectarEn(dir string) ProjectMetadata {
	meta := ProjectMetadata{Name: filepath.Base(dir), Languages: []string{}}
	for _, detectar := range detectores {
		detectar(dir, &meta)
	}
	return meta
}

// subproyecto es un stack encontrado en una subcarpeta (monorepo).
type subproyecto struct {
	ruta string
	meta ProjectMetadata
}

// detectarSubproyectos busca stacks en subcarpetas de primer y segundo nivel
// (backend/, frontend/, apps/web, services/api...).
func detectarSubproyectos(dir string, nivel int) []subproyecto {
	var encontrados []subproyecto
	for _, sub := range subcarpetas(dir) {
		meta := detectarEn(filepath.Join(dir, sub))
		if len(meta.Languages) > 0 {
			encontrados = append(encontrados, subproyecto{ruta: sub, meta: meta})
			continue
		}
		if nivel > 1 {
			continue
		}
		for _, hijo := range detectarSubproyectos(filepath.Join(dir, sub), nivel+1) {
			hijo.ruta = sub + "/" + hijo.ruta
			encontrados = append(encontrados, hijo)
		}
	}
	return encontrados
}

func subcarpetas(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var subs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !carpetasSinCodigoPropio[e.Name()] {
			subs = append(subs, e.Name())
		}
	}
	return subs
}

func metadataDeMonorepo(dir string, subs []subproyecto) ProjectMetadata {
	meta := ProjectMetadata{Name: filepath.Base(dir), Languages: []string{}}
	vistos := map[string]bool{}
	var partes []string
	for _, s := range subs {
		for _, l := range s.meta.Languages {
			if !vistos[l] {
				vistos[l] = true
				meta.Languages = append(meta.Languages, l)
			}
		}
		partes = append(partes, s.ruta+" ("+s.meta.Architecture+")")
	}
	sort.Strings(meta.Languages)
	meta.Architecture = subs[0].meta.Architecture
	if len(subs) > 1 {
		meta.Architecture = "Monorepo: " + strings.Join(partes, ", ")
	}
	return meta
}

// aplicarDeclaracion deja que --lenguaje y --arquitectura manden sobre la
// detección: es la forma de iniciar un proyecto que aún no tiene código.
func aplicarDeclaracion(meta ProjectMetadata, lenguajes []string, arquitectura string) ProjectMetadata {
	var declarados []string
	for _, l := range lenguajes {
		if l = strings.TrimSpace(l); l != "" {
			declarados = append(declarados, l)
		}
	}
	if len(declarados) > 0 {
		meta.Languages = declarados
	}
	if a := strings.TrimSpace(arquitectura); a != "" {
		meta.Architecture = a
	}
	return meta
}

func setArchitecture(meta *ProjectMetadata, arquitectura string) {
	if meta.Architecture == "" {
		meta.Architecture = arquitectura
	}
}

// leerManifiestos devuelve, en minúsculas, el contenido de los archivos que
// existan en dir (para buscar frameworks por nombre).
func leerManifiestos(dir string, nombres ...string) (string, bool) {
	var b strings.Builder
	encontrado := false
	for _, n := range nombres {
		data, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			continue
		}
		encontrado = true
		b.WriteString(strings.ToLower(string(data)))
		b.WriteString("\n")
	}
	return b.String(), encontrado
}

func contieneAlguno(texto string, palabras ...string) bool {
	for _, p := range palabras {
		if strings.Contains(texto, p) {
			return true
		}
	}
	return false
}

func detectPython(dir string, meta *ProjectMetadata) {
	texto, ok := leerManifiestos(dir, "requirements.txt", "requirements-dev.txt", "pyproject.toml", "setup.py", "Pipfile")
	if !ok {
		return
	}
	meta.Languages = append(meta.Languages, "Python")
	if contieneAlguno(texto, "fastapi", "flask", "django") {
		setArchitecture(meta, "Backend REST API (Python)")
	}
	setArchitecture(meta, "Python Application")
}

func detectDart(dir string, meta *ProjectMetadata) {
	texto, ok := leerManifiestos(dir, "pubspec.yaml")
	if !ok {
		return
	}
	meta.Languages = append(meta.Languages, "Dart")
	if strings.Contains(texto, "sdk: flutter") {
		setArchitecture(meta, "Mobile App (Flutter)")
	}
	setArchitecture(meta, "Dart Application")
}

func detectRust(dir string, meta *ProjectMetadata) {
	texto, ok := leerManifiestos(dir, "Cargo.toml")
	if !ok {
		return
	}
	meta.Languages = append(meta.Languages, "Rust")
	if contieneAlguno(texto, "axum", "actix-web", "rocket", "warp") {
		setArchitecture(meta, "Backend REST API (Rust)")
	}
	setArchitecture(meta, "Rust Application")
}

func detectPHP(dir string, meta *ProjectMetadata) {
	texto, ok := leerManifiestos(dir, "composer.json")
	if !ok {
		return
	}
	meta.Languages = append(meta.Languages, "PHP")
	if strings.Contains(texto, "laravel/framework") {
		setArchitecture(meta, "Backend Web (Laravel)")
	}
	setArchitecture(meta, "PHP Application")
}

func detectRuby(dir string, meta *ProjectMetadata) {
	texto, ok := leerManifiestos(dir, "Gemfile")
	if !ok {
		return
	}
	meta.Languages = append(meta.Languages, "Ruby")
	if contieneAlguno(texto, "'rails'", "\"rails\"") {
		setArchitecture(meta, "Backend Web (Rails)")
	}
	setArchitecture(meta, "Ruby Application")
}

func detectDotNet(dir string, meta *ProjectMetadata) {
	proyectos, _ := filepath.Glob(filepath.Join(dir, "*.csproj"))
	soluciones, _ := filepath.Glob(filepath.Join(dir, "*.sln"))
	if len(proyectos)+len(soluciones) == 0 {
		return
	}
	meta.Languages = append(meta.Languages, "C#")
	setArchitecture(meta, ".NET Application")
}
