package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// tmpDir es la carpeta para lo temporal y casuístico: scripts de un solo uso,
// pruebas manuales, descargas, logs y salidas (FND-075).
const tmpDir = "tmp"

// tmpGitignore hace que git ignore ./tmp entera, incluido este archivo, sin
// tocar el .gitignore del proyecto.
const tmpGitignore = `# Carpeta temporal de QDD: scripts de un solo uso, pruebas manuales, descargas,
# logs y salidas. Git la ignora entera; lo que valga la pena conservar se mueve
# a su lugar en el proyecto, con su prueba.
*
`

// ensureTmpDir deja lista ./tmp. Si el proyecto ya versiona archivos en tmp/,
// no se la apropia y devuelve un aviso para el usuario.
func ensureTmpDir(root string) string {
	dir := filepath.Join(root, tmpDir)
	marcador := filepath.Join(dir, ".gitignore")
	if fileExists(marcador) {
		return ""
	}
	if tieneArchivosVersionados(root, tmpDir) {
		return "tmp/ ya tiene archivos versionados en git, así que QDD no la usa como carpeta temporal. Usa otra para lo temporal y no la dejes en la raíz."
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "no se pudo crear ./tmp: " + err.Error()
	}
	if err := os.WriteFile(marcador, []byte(tmpGitignore), 0644); err != nil {
		return "no se pudo preparar ./tmp: " + err.Error()
	}
	return ""
}

// tieneArchivosVersionados indica si git sigue algún archivo bajo ruta. Sin git
// (o fuera de un repositorio) no hay nada versionado.
func tieneArchivosVersionados(root, ruta string) bool {
	out, err := exec.Command("git", "-C", root, "ls-files", "--", ruta).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// limpiarPycacheDelCore borra la caché de Python que versiones anteriores de
// QDD copiaron dentro de .qdd/core (FND-076). Solo toca .qdd/core: es de QDD.
func limpiarPycacheDelCore(qddDir string) {
	var caches []string
	filepath.WalkDir(filepath.Join(qddDir, "core"), func(path string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && d.Name() == "__pycache__" {
			caches = append(caches, path)
			return filepath.SkipDir
		}
		return nil
	})
	for _, c := range caches {
		_ = os.RemoveAll(c)
	}
}

// esCachePython reconoce archivos de caché de Python que no deben empaquetarse
// ni copiarse a los proyectos.
func esCachePython(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/__pycache__/") || strings.HasSuffix(path, ".pyc")
}
