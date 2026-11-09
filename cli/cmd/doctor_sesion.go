package cmd

import (
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'

// nombreSesionLate es el nombre del check de la sesión que late en doctor.
const nombreSesionLate = "Sesión de Claude Code que late (baúl, teléfono)"

// checkSesionLate revisa, desde una sesión de Claude Code, que el panel la anote como viva en el proyecto: el baúl le
// pide permiso solo a una sesión que late (FND-191). Fuera de una sesión no hay qué revisar.
func checkSesionLate(projectPath string, now time.Time) *CheckItem {
	sesion := estado.SesionActual()
	if sesion == "" {
		return nil
	}
	if consola.SesionLate(projectPath, sesion, now) {
		return &CheckItem{Name: nombreSesionLate, Success: true, Error: "la sesión " + sesion + " late en este proyecto"}
	}
	if arriba := carpetaDondeLate(projectPath, sesion, now); arriba != "" {
		return &CheckItem{Name: nombreSesionLate, Error: "la sesión " + sesion + " late en " + arriba + ", no en este proyecto: Claude Code se abrió en la carpeta de arriba. Abre Claude Code en esta carpeta para usar el baúl aquí"}
	}
	return &CheckItem{Name: nombreSesionLate, Error: "la sesión " + sesion + " no late en este proyecto: el baúl no puede pedir permiso ni llegan las instrucciones del teléfono. Escribe /reload-plugins y abre el panel; si sigue igual, actualiza qdd (FND-191 lo corrigió en 2.10.0-rc.55)"}
}

// carpetaDondeLate busca la sesión en las carpetas de arriba que también son proyectos de QDD.
func carpetaDondeLate(projectPath, sesion string, now time.Time) string {
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return ""
	}
	for dir := filepath.Dir(abs); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".qdd")); err != nil {
			continue
		}
		if consola.SesionLate(dir, sesion, now) {
			return dir
		}
	}
	return ""
}
