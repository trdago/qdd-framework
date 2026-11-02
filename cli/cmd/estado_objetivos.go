package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// 'qdd estado objetivo' y 'qdd estado tarea' (blueprint estado-por-tareas, objetivos y dependencias): para qué se
// hacen las tareas, cuál es el foco y de qué depende cada una. Escriben validado, como 'set'.

var (
	objetivoCambio     estado.CambioObjetivo
	objetivoEvidencia  string
	objetivoMotivo     string
	objetivoDescartado bool
	objetivoReabrir    bool
	tareaCambio        estado.CambioTarea
	tareaDependeDe     string
	nuevaCambio        estado.CambioNueva
	nuevaDependeDe     string
)

// estadoNuevaCmd (FND-109): crear una tarea es explícito y qdd le da el id desde el nombre; nombrar una que no existe
// en set, cerrar o tarea es un error que sugiere la más parecida.
var estadoNuevaCmd = &cobra.Command{
	Use:   "nueva <nombre>",
	Short: "Crea una tarea abierta y le da su id desde el nombre (lo imprime: úsalo en set, cerrar y tarea)",
	Long: `  qdd estado nueva "Quitar claves de Gemini" --objetivo gemini     → quitar-claves-gemini
  qdd estado nueva "Panel en Antigravity" --depende-de ag-rfc

El id son las primeras palabras del nombre, en minúsculas, sin tildes ni palabras vacías (de, la, en…); si ya existe,
lleva -2, -3… Si el nombre se parece mucho al de una tarea abierta, no la crea y la nombra: usa esa, o --aunque-exista.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := nuevaCambio
		c.Nombre, c.DependeDe = args[0], listaDeIDs(nuevaDependeDe)
		id, err := estado.Nueva(estadoRaiz(), c, time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[+] tarea %s creada en %s: %s\n", id, estado.Ruta, c.Nombre)
	},
}

var estadoObjetivoCmd = &cobra.Command{
	Use:   "objetivo <id>",
	Short: "Crea o cambia un objetivo (nombre, criterio, foco) o lo cierra: cumplido con --evidencia, descartado con --motivo",
	Long: `Un objetivo dice para qué se hacen sus tareas y cuándo se cumple (su criterio de éxito).

  qdd estado objetivo release --nombre "Publicar 2.8.0" --criterio "5 h de uso medido" --foco
  qdd estado objetivo release --evidencia "corrida uso-5h"     se cumple: terminar sus tareas no basta
  qdd estado objetivo release --descartado --motivo "…"        se descarta; nunca desaparece en silencio
  qdd estado objetivo release --reabrir
  qdd estado objetivo tutor --ciclo agentico                   sus tareas siguen el ciclo «agentico»

A lo más un objetivo abierto es el foco: marcar uno desmarca al anterior. Con --ciclo, sus tareas toman las etapas de
ese ciclo (declarado con 'qdd estado ciclo'); --ciclo "" las vuelve al del proyecto.`,
	Args: cobra.ExactArgs(1),
	Run:  runEstadoObjetivo,
}

var estadoTareaCmd = &cobra.Command{
	Use:   "tarea <id>",
	Short: "Cambia el objetivo de una tarea o de qué depende (la crea solo con --nombre, con sus etapas en «·»)",
	Long: `  qdd estado tarea panel --objetivo release --depende-de receta,modelo
  qdd estado tarea panel --objetivo ""          la deja sin objetivo
  qdd estado tarea panel --depende-de ""        sin dependencias

Una tarea no termina antes que sus dependencias: su última etapa no pasa a ✅ mientras una no termine.`,
	Args: cobra.ExactArgs(1),
	Run:  runEstadoTarea,
}

func init() {
	f := estadoObjetivoCmd.Flags()
	f.StringVar(&objetivoCambio.Nombre, "nombre", "", "nombre del objetivo")
	f.StringVar(&objetivoCambio.Criterio, "criterio", "", "criterio de éxito (obligatorio al crearlo)")
	f.BoolVar(&objetivoCambio.Foco, "foco", false, "lo marca como el foco actual (desmarca al anterior)")
	f.StringVar(&objetivoEvidencia, "evidencia", "", "evidencia de que se cumple el criterio: lo cierra cumplido")
	f.BoolVar(&objetivoDescartado, "descartado", false, "lo descarta (con --motivo)")
	f.StringVar(&objetivoMotivo, "motivo", "", "por qué se descarta")
	f.BoolVar(&objetivoReabrir, "reabrir", false, "vuelve a abrir un objetivo cumplido o descartado")
	f.StringVar(&objetivoCambio.Ciclo, "ciclo", "", "el ciclo que siguen sus tareas (\"\" el del proyecto)")
	t := estadoTareaCmd.Flags()
	t.StringVar(&tareaCambio.Nombre, "nombre", "", "nombre de la tarea")
	t.StringVar(&tareaCambio.Objetivo, "objetivo", "", "id del objetivo al que sirve (\"\" la deja sin objetivo)")
	t.StringVar(&tareaDependeDe, "depende-de", "", "ids de las tareas de las que depende, separados por coma (\"\" ninguna)")
	t.BoolVar(&tareaCambio.AunqueExista, "aunque-exista", false, "con --nombre: la crea aunque se parezca a una tarea abierta")
	n := estadoNuevaCmd.Flags()
	n.StringVar(&nuevaCambio.Objetivo, "objetivo", "", "id del objetivo al que sirve")
	n.StringVar(&nuevaDependeDe, "depende-de", "", "ids de las tareas de las que depende, separados por coma")
	n.BoolVar(&nuevaCambio.AunqueExista, "aunque-exista", false, "la crea aunque se parezca a una tarea abierta")
	estadoCmd.AddCommand(estadoObjetivoCmd, estadoTareaCmd, estadoCicloCmd, estadoNuevaCmd)
}

var estadoCicloCmd = &cobra.Command{
	Use:   "ciclo <nombre> <etapa>...",
	Short: "Declara un ciclo con sus etapas (o cambia las de uno); un objetivo lo sigue con 'objetivo --ciclo'",
	Long: `Las etapas del archivo son el ciclo «proyecto». Un ciclo con nombre tiene sus propias etapas: las tareas de un
objetivo que lo sigue toman esas etapas, sin cambiar las del proyecto.

  qdd estado ciclo agentico Concepto Artefacto Medición Puerta
  qdd estado objetivo tutor --nombre "Tutor" --criterio "…" --ciclo agentico
  qdd estado ciclo proyecto Código Pruebas Producción        cambia las etapas del proyecto

Cambiar las etapas de un ciclo no pierde avance: si una tarea tiene ✅ ⏳ o ❌ en una etapa que se quita, no se escribe.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := estado.DeclararCiclo(estadoRaiz(), args[0], args[1:], time.Now()); err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[+] ciclo %s: %s en %s\n", args[0], strings.Join(args[1:], " · "), estado.Ruta)
	},
}

func runEstadoObjetivo(cmd *cobra.Command, args []string) {
	root, id, now := estadoRaiz(), args[0], time.Now()
	objetivoCambio.CambiaCiclo = cmd.Flags().Changed("ciclo")
	if err := aplicarObjetivo(root, id, objetivoCambio, objetivoEvidencia, objetivoMotivo, objetivoDescartado, objetivoReabrir, now); err != nil {
		salirCon("%v", err)
	}
	fmt.Printf("[+] objetivo %s actualizado en %s\n", id, estado.Ruta)
}

// aplicarObjetivo es lo mismo para la CLI y la tool MCP: primero los datos, después el cierre o la reapertura.
func aplicarObjetivo(root, id string, c estado.CambioObjetivo, evidencia, motivo string, descartado, reabrir bool, now time.Time) error {
	c.ID = id
	if c.Nombre != "" || c.Criterio != "" || c.Foco || c.CambiaCiclo || c.Ciclo != "" || (evidencia == "" && !descartado && !reabrir) {
		if err := estado.GuardarObjetivo(root, c, now); err != nil {
			return err
		}
	}
	if reabrir {
		return estado.ReabrirObjetivo(root, id, now)
	}
	if descartado {
		return estado.CerrarObjetivo(root, id, motivo, true, now)
	}
	if evidencia != "" {
		return estado.CerrarObjetivo(root, id, evidencia, false, now)
	}
	return nil
}

func runEstadoTarea(cmd *cobra.Command, args []string) {
	c := tareaCambio
	c.ID = args[0]
	c.CambiaObjetivo = cmd.Flags().Changed("objetivo")
	c.CambiaDependencias = cmd.Flags().Changed("depende-de")
	c.DependeDe = listaDeIDs(tareaDependeDe)
	if err := estado.GuardarTarea(estadoRaiz(), c, time.Now()); err != nil {
		salirCon("%v", err)
	}
	fmt.Printf("[+] tarea %s actualizada en %s\n", c.ID, estado.Ruta)
}

func listaDeIDs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
