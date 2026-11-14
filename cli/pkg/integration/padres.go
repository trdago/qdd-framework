package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-062: las IAs cargan también los AGENTS.md (y CLAUDE.md) de las carpetas padre, así que un bloque de QDD en
// ~/AGENTS.md o en la carpeta que agrupa los proyectos llega repetido a cada sesión. Vale el del proyecto; los de
// arriba sobran. Un proyecto QDD que contiene otros proyectos (la carpeta de proyectos) no lleva el bloque en sus
// archivos heredables, para que su propio sync no lo vuelva a escribir.

// archivosHeredables son los que una IA carga también desde las carpetas padre.
var archivosHeredables = []string{"AGENTS.md", "CLAUDE.md"}

// TopePadres es la carpeta más alta que se revisa (la personal). Las pruebas la cambian para no tocar la de la persona.
var TopePadres = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// Contenedor dice si la carpeta tiene un proyecto QDD directamente adentro (una carpeta de proyectos). Las carpetas
// ocultas (.claude/worktrees) son del propio proyecto, no proyectos aparte.
func Contenedor(ruta string) bool {
	hijos, err := os.ReadDir(ruta)
	if err != nil {
		return false
	}
	for _, h := range hijos {
		if !h.IsDir() || strings.HasPrefix(h.Name(), ".") {
			continue
		}
		if IsQDDProject(filepath.Join(ruta, h.Name())) {
			return true
		}
	}
	return false
}

// BloquesEnPadres son los archivos heredables de las carpetas padre del proyecto, hasta TopePadres, que traen el
// bloque de QDD: del más cercano al más lejano. Un padre que es un proyecto QDD cuenta solo si es una carpeta de
// proyectos (si no, el bloque es suyo); uno del que el proyecto cuelga por una carpeta oculta (un worktree en
// .claude/worktrees) no cuenta.
func BloquesEnPadres(projectPath string) []string {
	proyecto, err := filepath.Abs(projectPath)
	if err != nil {
		return nil
	}
	tope := TopePadres()
	if tope == "" || !dentroDe(proyecto, tope) {
		return nil
	}
	var copias []string
	for dir := filepath.Dir(proyecto); dentroDe(dir, tope); dir = filepath.Dir(dir) {
		copias = append(copias, copiasEn(dir, proyecto)...)
		if dir == tope || filepath.Dir(dir) == dir {
			break
		}
	}
	return copias
}

func copiasEn(dir, proyecto string) []string {
	rel, err := filepath.Rel(dir, proyecto)
	if err != nil || strings.HasPrefix(rel, ".") {
		return nil
	}
	if IsQDDProject(dir) && !Contenedor(dir) {
		return nil
	}
	var copias []string
	for _, nombre := range archivosHeredables {
		ruta := filepath.Join(dir, nombre)
		if tieneBloque(ruta) {
			copias = append(copias, ruta)
		}
	}
	return copias
}

// dentroDe dice si ruta es tope o cuelga de él.
func dentroDe(ruta, tope string) bool {
	rel, err := filepath.Rel(tope, ruta)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func tieneBloque(ruta string) bool {
	data, err := os.ReadFile(ruta)
	return err == nil && strings.Contains(string(data), markerBegin)
}

// QuitarBloquesDePadres saca el bloque de las copias de BloquesEnPadres y deja junto a cada una un respaldo con el
// archivo entero (<archivo>.qdd-respaldo-<fecha>). Lo demás del archivo queda; uno que era solo el bloque se borra.
func QuitarBloquesDePadres(projectPath string, ahora time.Time) ([]string, error) {
	var quitados []string
	for _, ruta := range BloquesEnPadres(projectPath) {
		if err := quitarBloque(ruta, ahora); err != nil {
			return quitados, err
		}
		quitados = append(quitados, ruta)
	}
	return quitados, nil
}

func quitarBloque(ruta string, ahora time.Time) error {
	data, err := os.ReadFile(ruta)
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), markerBegin) {
		return nil
	}
	respaldo := ruta + ".qdd-respaldo-" + ahora.UTC().Format("20060102T150405")
	if err := os.WriteFile(respaldo, data, 0o644); err != nil {
		return fmt.Errorf("no se pudo dejar el respaldo de %s: %w", ruta, err)
	}
	resto := strings.TrimSpace(removeOldBlock(string(data)))
	if resto == "" {
		return os.Remove(ruta)
	}
	return os.WriteFile(ruta, []byte(resto+"\n"), 0o644)
}
