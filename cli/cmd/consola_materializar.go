package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/materializar"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Lo que cambia en el panel se materializa como las ventanas del usuario (hypr-materialize): qdd calcula los cuadros
// una vez por cambio y el panel solo los reproduce (pkg/materializar).

var consolaMaterializarCmd = &cobra.Command{
	Use:   "materializar",
	Short: "Los cuadros con que aparece (o se va) un texto del panel (JSON por la entrada estándar)",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		os.Exit(consolaMaterializar(os.Stdin, os.Stdout, os.Stderr, materializar.EstiloDe(home)))
	},
}

var consolaEfectosCmd = &cobra.Command{
	Use:   "efectos [on|off]",
	Short: "Enciende o apaga los efectos del panel (encendidos por defecto); sin argumento, dice cómo están",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(consolaEfectos(args, os.Stdout))
	},
}

func init() {
	consolaCmd.AddCommand(consolaMaterializarCmd, consolaEfectosCmd)
}

// pedidoMaterializar es lo que manda el panel: las líneas tal como las dibuja y hacia dónde va el efecto. El acento y
// las duraciones, si no vienen, son los del estilo del usuario.
type pedidoMaterializar struct {
	Lineas []materializar.Linea `json:"lineas"`
	materializar.Opciones
}

type respuestaMaterializar struct {
	CuadroMs int                    `json:"cuadro_ms"`
	Cuadros  [][]materializar.Linea `json:"cuadros"`
}

func consolaMaterializar(in io.Reader, out, errs io.Writer, estilo materializar.Estilo) int {
	var p pedidoMaterializar
	if err := json.NewDecoder(in).Decode(&p); err != nil {
		fmt.Fprintf(errs, "[!] la entrada no es un pedido de cuadros: %v\n", err)
		return 1
	}
	o := p.Opciones
	if o.Acento == "" {
		o.Acento = estilo.Acento
	}
	if o.DuracionMs <= 0 {
		o.DuracionMs = duracionDe(estilo, o.Abrir)
	}
	if o.CuadroMs <= 0 {
		o.CuadroMs = materializar.CuadroMs
	}
	r := respuestaMaterializar{CuadroMs: o.CuadroMs, Cuadros: materializar.Cuadros(p.Lineas, o)}
	if err := json.NewEncoder(out).Encode(r); err != nil {
		fmt.Fprintf(errs, "[!] %v\n", err)
		return 1
	}
	return 0
}

func duracionDe(e materializar.Estilo, abrir bool) int {
	if abrir {
		return e.AbrirMs
	}
	return e.CerrarMs
}

func consolaEfectos(args []string, out io.Writer) int {
	qddHome, err := blueprints.Home()
	if err != nil {
		fmt.Fprintf(out, "[!] %v\n", err)
		return 1
	}
	if len(args) == 1 && args[0] != "on" && args[0] != "off" {
		fmt.Fprintf(out, "[!] usa 'qdd consola efectos on' o 'qdd consola efectos off'\n")
		return 1
	}
	if len(args) == 1 {
		if err := materializar.Guardar(qddHome, args[0] == "on"); err != nil {
			fmt.Fprintf(out, "[!] %v\n", err)
			return 1
		}
	}
	home, _ := os.UserHomeDir()
	e := materializar.Leer(qddHome, home)
	fmt.Fprintf(out, "[+] Efectos del panel %s · %s\n", encendidos(e.Activo), e.Estilo)
	return 0
}

func encendidos(activo bool) string {
	if activo {
		return "encendidos"
	}
	if os.Getenv(materializar.EnvEfectos) == "off" {
		return "apagados (" + materializar.EnvEfectos + "=off)"
	}
	return "apagados"
}

// conEfectos suma al panel si materializa y con qué estilo.
func conEfectos(d *consola.DatosConsola) {
	qddHome, err := blueprints.Home()
	if err != nil {
		return
	}
	home, _ := os.UserHomeDir()
	e := materializar.Leer(qddHome, home)
	d.Efectos = &e
}

const nombreEfectos = "Efectos del panel (qdd consola efectos)"

// checkEfectos dice si el panel materializa lo que cambia y con qué estilo; un interruptor ilegible es una falla que
// --fix deja encendido.
func checkEfectos() *CheckItem {
	qddHome, err := blueprints.Home()
	if err != nil {
		return &CheckItem{Name: nombreEfectos, Error: err.Error()}
	}
	if err := materializar.Revisar(qddHome); err != nil {
		return &CheckItem{Name: nombreEfectos, Error: err.Error() + " (qdd doctor --fix lo deja encendido)"}
	}
	home, _ := os.UserHomeDir()
	e := materializar.Leer(qddHome, home)
	return &CheckItem{Name: nombreEfectos, Success: true, Error: encendidos(e.Activo) + " · " + e.Estilo.String()}
}

// repararEfectos deja legible un interruptor roto (encendido) y lo registra como autorreparación.
func repararEfectos() {
	fmt.Println("  [~] Dejando legible el interruptor de efectos del panel...")
	qddHome, err := blueprints.Home()
	if err == nil {
		err = materializar.Reparar(qddHome)
	}
	hecho := []string{"[+] efectos encendidos: " + materializar.RutaEfectos(qddHome)}
	if err != nil {
		hecho = nil
	}
	imprimirReparado(hecho, err)
	registrarAutorreparacion("doctor: "+nombreEfectos, "el interruptor de efectos del panel estaba ilegible", hecho)
}
