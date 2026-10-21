package modules

import (
	"os/exec"
	"path"
	"sort"
	"strings"
)

// Carpetas "contenedoras": agrupan módulos pero no son un módulo en sí
// (internal/validador es un módulo; internal no).
var containerDirs = map[string]bool{
	"internal": true, "pkg": true, "cmd": true, "src": true, "apps": true, "packages": true,
	"services": true, "lib": true, "app": true, "modules": true, "frontend": true, "backend": true,
	"mobile": true, "functions": true, "lambdas": true, "features": true,
}

var sourceExts = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true, ".vue": true,
	".svelte": true, ".py": true, ".dart": true, ".java": true, ".kt": true, ".rs": true, ".rb": true,
	".php": true, ".cs": true, ".swift": true, ".c": true, ".cpp": true, ".h": true, ".ino": true,
}

var nonModuleDirs = map[string]bool{
	"docs": true, "test": true, "tests": true, "e2e": true, "testdata": true, "mocks": true,
	"__tests__": true, "fixtures": true, "scripts": true, "node_modules": true, "vendor": true,
	"dist": true, "build": true, "examples": true, "core_assets": true,
}

// UncoveredZones lista las zonas de código versionado que ninguna ficha cubre.
// Una zona es la primera carpeta "real" bajo las carpetas contenedoras
// (p. ej. internal/validador, frontend/src/views).
func UncoveredZones(root string, mods []Module) []string {
	zones := codeZones(root)
	var out []string
	for _, z := range zones {
		if zoneCovered(z, mods) {
			continue
		}
		out = append(out, z)
	}
	return out
}

func codeZones(root string) []string {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	set := map[string]bool{}
	for _, f := range strings.Split(string(output), "\n") {
		if zone, ok := zoneOf(f); ok {
			set[zone] = true
		}
	}
	zones := make([]string, 0, len(set))
	for z := range set {
		zones = append(zones, z)
	}
	sort.Strings(zones)
	return collapseNested(zones)
}

func zoneOf(file string) (string, bool) {
	if !sourceExts[path.Ext(file)] || isTestFile(file) {
		return "", false
	}
	parts := strings.Split(file, "/")
	dirs := parts[:len(parts)-1]
	if len(dirs) == 0 {
		return "", false
	}
	for _, d := range dirs {
		if strings.HasPrefix(d, ".") || nonModuleDirs[d] {
			return "", false
		}
	}
	// Como máximo 2 niveles: más abajo suelen ser capas técnicas
	// (components, stores, handlers), no módulos de negocio.
	for i, d := range dirs {
		if !containerDirs[d] || i == 1 {
			return strings.Join(dirs[:i+1], "/"), true
		}
	}
	return strings.Join(dirs, "/"), true
}

// collapseNested quita las zonas contenidas en otra zona de la lista
// (frontend/src queda dentro de frontend si este también tiene código).
func collapseNested(sorted []string) []string {
	var out []string
	for _, z := range sorted {
		if len(out) > 0 && strings.HasPrefix(z, out[len(out)-1]+"/") {
			continue
		}
		out = append(out, z)
	}
	return out
}

func isTestFile(file string) bool {
	base := path.Base(file)
	return strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") || strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.dart")
}

// Una zona está cubierta si alguna ficha la incluye, o si alguna ficha cubre
// una parte de ella (fichas más finas que la zona).
func zoneCovered(zone string, mods []Module) bool {
	for _, m := range mods {
		for _, r := range m.Rutas {
			ruta := strings.TrimSuffix(path.Clean(strings.ReplaceAll(r, "\\", "/")), "/")
			if zone == ruta || strings.HasPrefix(zone, ruta+"/") || strings.HasPrefix(ruta, zone+"/") {
				return true
			}
		}
	}
	return false
}
