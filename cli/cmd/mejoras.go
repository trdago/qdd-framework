package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/mejoras"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Cola de mejoras (pkg/mejoras, pedido de la persona, 2026-11-03): las sesiones reportan, el ciclo determinista lo
// pasa a .qdd/project/cola_de_mejoras/ y al objetivo mejora-continua del tablero, 2 tareas abiertas como mucho. El mod
// qdd-tareas llama 'ciclo --si-toca' cada minuto (toca cada 10) y el botón del panel lo fuerza.

var (
	mejorasFuentes   = mejoras.FuentesReales
	mejorasSiToca    bool
	mejorasJSON      bool
	mejorasTitulo    string
	mejorasDetalle   string
	mejorasAutor     string
	mejorasProyecto  string
	mejorasEtiquetas []string
)

var mejorasCmd = &cobra.Command{
	Use:   "mejoras",
	Short: "La cola de mejoras del proyecto (reportar | ciclo | lista)",
	Long: `La cola_de_mejoras recoge lo que reportan las sesiones de Claude Code y los issues de GitHub (de quien tiene
acceso al repositorio, o con la etiqueta «mejora»), y mantiene en el tablero el objetivo mejora-continua con una tarea
por mejora, 2 abiertas como mucho. Es determinista: solo lo que ya es tarea pasa a la IA.

  qdd mejoras reportar --titulo … [--detalle …] [--autor …] [--proyecto <ruta|nombre>] [--etiqueta bug]
  qdd mejoras ciclo [--si-toca] [--json]     recibe, sigue las tareas y entrega (--si-toca: solo cada 10 min)
  qdd mejoras lista [--json]                 la cola y en qué está cada mejora`,
}

var mejorasReportarCmd = &cobra.Command{
	Use: "reportar --titulo <texto>", Short: "Deja un reporte en la cola de mejoras de un proyecto", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(mejorasReportar(estadoRaiz(), mejorasProyecto, mejoras.Reporte{Titulo: mejorasTitulo, Detalle: mejorasDetalle, Autor: mejorasAutor, Etiquetas: mejorasEtiquetas}, os.Stdout, os.Stderr))
	},
}

var mejorasCicloCmd = &cobra.Command{
	Use: "ciclo", Short: "Corre el ciclo de la cola de mejoras: recibir, seguir las tareas y entregar", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(mejorasCiclo(estadoRaiz(), mejorasSiToca, mejorasJSON, time.Now(), os.Stdout, os.Stderr))
	},
}

var mejorasListaCmd = &cobra.Command{
	Use: "lista", Short: "La cola de mejoras y en qué está cada una", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(mejorasLista(estadoRaiz(), mejorasJSON, os.Stdout, os.Stderr))
	},
}

func init() {
	f := mejorasReportarCmd.Flags()
	f.StringVar(&mejorasTitulo, "titulo", "", "qué hay que mejorar, en una línea (es lo que se lee en el tablero)")
	f.StringVar(&mejorasDetalle, "detalle", "", "qué pasó, cómo reproducirlo y qué se espera")
	f.StringVar(&mejorasAutor, "autor", "", "quién lo reporta (p. ej. la sesión de otro proyecto)")
	f.StringVar(&mejorasProyecto, "proyecto", "", "el proyecto que la recibe: su ruta o su nombre en el registro (por defecto, este)")
	f.StringSliceVar(&mejorasEtiquetas, "etiqueta", nil, "bug, seguridad…: las urgentes van primero")
	mejorasCicloCmd.Flags().BoolVar(&mejorasSiToca, "si-toca", false, "solo si pasaron 10 min desde el último ciclo")
	mejorasCicloCmd.Flags().BoolVar(&mejorasJSON, "json", false, "salida JSON")
	mejorasListaCmd.Flags().BoolVar(&mejorasJSON, "json", false, "salida JSON")
	mejorasCmd.AddCommand(mejorasReportarCmd, mejorasCicloCmd, mejorasListaCmd)
	rootCmd.AddCommand(mejorasCmd)
}

func mejorasReportar(aqui, proyecto string, r mejoras.Reporte, salida, errores io.Writer) int {
	root, err := proyectoQueRecibe(aqui, proyecto)
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	if err := mejoras.Reportar(root, r, time.Now()); err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintf(salida, "[+] Reporte en la cola de mejoras de %s: lo recibe su próximo ciclo (cada 10 min, o «Ciclo de mejoras» en su panel).\n", filepath.Base(root))
	return 0
}

// proyectoQueRecibe: sin --proyecto, este; con una ruta, esa; si no, el proyecto del registro con ese nombre (la
// carpeta) o esa clave.
func proyectoQueRecibe(aqui, proyecto string) (string, error) {
	proyecto = strings.TrimSpace(proyecto)
	if proyecto == "" {
		return aqui, nil
	}
	if info, err := os.Stat(proyecto); err == nil && info.IsDir() {
		return filepath.Abs(proyecto)
	}
	reg, err := registro.Leer()
	if err != nil {
		return "", err
	}
	var nombres []string
	for _, p := range reg.Proyectos {
		if filepath.Base(p.Ruta) == proyecto || p.Clave == proyecto {
			return p.Ruta, nil
		}
		nombres = append(nombres, filepath.Base(p.Ruta))
	}
	return "", fmt.Errorf("no hay un proyecto «%s» en esta máquina; los registrados: %s", proyecto, strings.Join(nombres, ", "))
}

func mejorasCiclo(root string, siToca, comoJSON bool, now time.Time, salida, errores io.Writer) int {
	if siToca && !mejoras.TocaCiclo(root, now) {
		return 0
	}
	r, err := mejoras.Ciclo(root, mejorasFuentes(), now)
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	if comoJSON {
		json.NewEncoder(salida).Encode(r)
		return 0
	}
	fmt.Fprintf(salida, "[+] Ciclo de mejoras: %d nuevas, %d a tareas, %d hechas, %d en cola.\n", len(r.Nuevas), len(r.Tareas), len(r.Hechas), r.EnCola)
	for _, a := range r.Avisos {
		fmt.Fprintf(salida, "[i] %s\n", a)
	}
	return 0
}

func mejorasLista(root string, comoJSON bool, salida, errores io.Writer) int {
	cola, err := mejoras.Lista(root)
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	if comoJSON {
		if cola == nil {
			cola = []mejoras.Mejora{}
		}
		json.NewEncoder(salida).Encode(cola)
		return 0
	}
	if len(cola) == 0 {
		fmt.Fprintln(salida, "[i] La cola de mejoras está vacía.")
		return 0
	}
	for _, m := range cola {
		fmt.Fprintf(salida, "%s  %-10s %s  (%s)\n", m.ID, m.Estado, m.Titulo, m.Ref)
	}
	return 0
}
