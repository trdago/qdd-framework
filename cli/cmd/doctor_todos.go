package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// Pedido de qdd-movile (2026-11-07): las reglas de las IAs se renuevan con qdd doctor --fix proyecto por proyecto
// (FND-126), y 6 de 8 proyectos registrados quedaron atrasados. --todos corre el mismo doctor en cada proyecto de
// <QDD_HOME>/proyectos.json, como si la persona lo corriera ahí, y resume una línea por proyecto. Nunca corre solo: lo
// pide la persona, porque --fix deja cambios en git en cada repositorio.

var doctorTodosFlag bool

func init() {
	doctorCmd.Flags().BoolVar(&doctorTodosFlag, "todos", false, "Corre doctor (y --fix si lo pides) en cada proyecto registrado en esta máquina")
}

// correrDoctorTodos lo llama doctor con --todos.
func correrDoctorTodos() {
	reg, err := registro.Leer()
	if err != nil {
		fmt.Println("[!] No se pudo leer el registro de proyectos: qdd doctor --fix lo repara.")
		os.Exit(1)
	}
	os.Exit(doctorTodos(reg.Proyectos, doctorEn, os.Stdout))
}

// doctorEn corre este mismo qdd doctor en la carpeta del proyecto.
func doctorEn(ruta string) (string, int) {
	yo, err := os.Executable()
	if err != nil {
		return "  ❌ No se pudo ubicar el binario de qdd: " + err.Error(), 1
	}
	args := []string{"doctor"}
	if autoFix {
		args = append(args, "--fix")
	}
	c := exec.Command(yo, args...)
	c.Dir = ruta
	salida, err := c.CombinedOutput()
	if err == nil {
		return string(salida), 0
	}
	if _, salio := err.(*exec.ExitError); !salio {
		return "  ❌ No se pudo correr qdd doctor: " + err.Error(), 1
	}
	return string(salida), 1
}

// doctorTodos corre el doctor en cada proyecto que existe y dice, de cada uno, si está sano o qué falla.
func doctorTodos(proyectos []registro.Proyecto, correr func(string) (string, int), out io.Writer) int {
	if len(proyectos) == 0 {
		fmt.Fprintln(out, "[i] En esta máquina no hay proyectos registrados: se registran al abrir Claude Code en ellos.")
		return 0
	}
	codigo := 0
	for _, p := range proyectos {
		nombre := filepath.Base(p.Ruta)
		if info, err := os.Stat(p.Ruta); err != nil || !info.IsDir() {
			fmt.Fprintf(out, "⊘ %s: la carpeta no existe (%s)\n", nombre, p.Ruta)
			continue
		}
		salida, c := correr(p.Ruta)
		if c == 0 {
			fmt.Fprintf(out, "✅ %s\n", nombre)
			continue
		}
		codigo = 1
		fmt.Fprintf(out, "❌ %s (%s)\n", nombre, p.Ruta)
		for _, linea := range strings.Split(salida, "\n") {
			if strings.HasPrefix(strings.TrimSpace(linea), "❌") {
				fmt.Fprintf(out, "    %s\n", strings.TrimSpace(linea))
			}
		}
	}
	if autoFix {
		fmt.Fprintln(out, "[i] Lo reparado queda como cambio en git en cada proyecto: commitéalo ahí.")
	}
	return codigo
}
