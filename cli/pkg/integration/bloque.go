package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Rule: Prohibido usar 'else'
// FND-126: un proyecto recibe las reglas nuevas al correr qdd doctor --fix tras actualizar qdd. La revisión dice qué
// bloque es más nuevo: un qdd más viejo no pisa lo que escribió uno más nuevo, igual que en las fichas (FND-099).

var lineaRevision = regexp.MustCompile(`(?m)^# QCP block revision: (\d+)\s*$`)

// archivosConBloque son los archivos en los que los adaptadores escriben el bloque.
var archivosConBloque = []string{
	"AGENTS.md",
	".agentrules",
	".antigravityrules",
	".clauderc",
	".cursorrules",
	filepath.Join(".agents", "workflows", "qdd.md"),
	filepath.Join(".claude", "commands", "qdd.md"),
	filepath.Join(".opencode", "qdd.md"),
}

// revisionDe es la revisión del bloque de un archivo; 0 si no la dice (de antes de FND-126).
func revisionDe(contenido string) int {
	m := lineaRevision.FindStringSubmatch(contenido)
	if m == nil {
		return 0
	}
	revision, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return revision
}

// BloquesViejos son los archivos del proyecto cuyo bloque es de una revisión anterior a la de este qdd, o que no debe
// llevarlo: el AGENTS.md de una carpeta de proyectos (MEJ-062). El sync los deja bien.
func BloquesViejos(projectPath string) []string {
	var viejos []string
	actual := revisionDe(QDDCommandProtocol)
	contenedor := Contenedor(projectPath)
	for _, rel := range archivosConBloque {
		data, err := os.ReadFile(filepath.Join(projectPath, rel))
		if err != nil || !strings.Contains(string(data), markerBegin) {
			continue
		}
		if revisionDe(string(data)) < actual || (contenedor && rel == "AGENTS.md") {
			viejos = append(viejos, filepath.ToSlash(rel))
		}
	}
	return viejos
}
