package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/qdd-framework/qdd/pkg/ambientes"
	"github.com/qdd-framework/qdd/pkg/baul"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// qdd ambiente (RFC 0023, fase 3): levantar un ambiente desde .qdd/ambientes.yaml. Empieza por el DNS de Cloudflare y
// solo muestra el plan (lo eligió el modo goal con la opción recomendada, 2026-11-15): lee con el token del baúl y no
// cambia nada. Sin el token en el entorno, vuelve a correr este mismo comando con 'baul usar', como compartir --enlace.

const (
	// FinAmbientes es el fin del permiso del baúl para leer el DNS de los ambientes.
	FinAmbientes = "leer el DNS de los ambientes"
	// variableAmbienteDelBaul marca la vuelta que corre con el token: si aun así no está, no se vuelve a pedir.
	variableAmbienteDelBaul = "QDD_AMBIENTE_POR_BAUL"
)

// lectorDNS es lo que levantar necesita de Cloudflare: leer.
type lectorDNS interface {
	Registros(ctx context.Context, zona string) ([]ambientes.Registro, error)
}

var (
	lectorDeAmbiente            = func(token string) lectorDNS { return ambientes.LectorDNS{Token: token} }
	entradaCloudflareDeAmbiente = func(amb string) (baul.Entrada, bool) {
		indice, err := baul.Indice()
		if err != nil {
			return baul.Entrada{}, false
		}
		return cloudflareDe(indice, amb)
	}
)

var ambienteCmd = &cobra.Command{
	Use:   "ambiente",
	Short: "Ambientes del proyecto desde .qdd/ambientes.yaml (RFC 0023): por ahora, el plan del DNS en Cloudflare",
	// Como baul: sin el gatekeeper del proyecto, porque la vuelta con el token corre como hijo de 'baul usar'.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
}

var ambienteLevantarCmd = &cobra.Command{
	Use:   "levantar <dev|qa|prod>",
	Short: "Compara el DNS declarado del ambiente con el de Cloudflare y muestra el plan; no cambia nada",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(ambienteLevantar(estadoRaiz(), args[0], cmd.OutOrStdout(), cmd.ErrOrStderr()))
	},
}

func init() {
	ambienteCmd.AddCommand(ambienteLevantarCmd)
	rootCmd.AddCommand(ambienteCmd)
}

func ambienteLevantar(root, amb string, salida, errores io.Writer) int {
	a, err := ambientes.Cargar(root)
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	declarado, ok := a.Ambientes[amb]
	if !ok {
		fmt.Fprintf(errores, "[!] %s no declara el ambiente %q; declara: %s\n", ambientes.Ruta, amb, strings.Join(declarados(a), ", "))
		return 1
	}
	token := os.Getenv(variableCloudflare)
	if token == "" {
		return levantarConElBaul(root, amb, salida, errores)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), time.Minute)
	defer cancelar()
	actuales, err := lectorDeAmbiente(token).Registros(ctx, declarado.DNS.Zona)
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	imprimirPlan(salida, amb, declarado.DNS, ambientes.PlanDNS(declarado.DNS, actuales))
	return 0
}

func declarados(a ambientes.Archivo) []string {
	var out []string
	for nombre := range a.Ambientes {
		out = append(out, nombre)
	}
	sort.Strings(out)
	return out
}

var marcasDePlan = map[string]string{ambientes.Crear: "+", ambientes.Cambiar: "~", ambientes.Igual: "="}

func imprimirPlan(salida io.Writer, amb string, dns ambientes.DNS, pasos []ambientes.Paso) {
	fmt.Fprintf(salida, "Ambiente %s · DNS de %s en Cloudflare\n", amb, dns.Zona)
	cambios := 0
	for _, p := range pasos {
		r := p.Registro
		fmt.Fprintf(salida, "  %s %-7s %-5s %s → %s%s\n", marcasDePlan[p.Accion], p.Accion, r.Tipo, r.Nombre, r.Valor, conProxy(r.Proxied))
		if p.Actual != nil {
			fmt.Fprintf(salida, "              hoy: %s%s\n", p.Actual.Valor, conProxy(p.Actual.Proxied))
		}
		if p.Accion != ambientes.Igual {
			cambios++
		}
	}
	fmt.Fprintf(salida, "Solo plan: no se cambió nada en Cloudflare (%d por crear o cambiar). Lo que la zona tiene y %s no nombra no se toca.\n", cambios, ambientes.Ruta)
}

func conProxy(proxied bool) string {
	if !proxied {
		return ""
	}
	return " (proxy)"
}

// levantarConElBaul vuelve a correr este mismo comando con el token de Cloudflare del baúl. Sin permiso, el baúl deja
// el pedido 🔑 en el panel y esto sale con 75.
func levantarConElBaul(root, amb string, salida, errores io.Writer) int {
	if os.Getenv(variableAmbienteDelBaul) != "" {
		fmt.Fprintf(errores, "[!] el baúl corrió levantar sin %s: revisa la entrada de Cloudflare con qdd baul verificar\n", variableCloudflare)
		return 1
	}
	entrada, ok := entradaCloudflareDeAmbiente(amb)
	if !ok {
		fmt.Fprintf(errores, "[!] no hay un token de Cloudflare en el baúl: guárdalo en una terminal aparte con 'qdd baul guardar cloudflare --tipo cloudflare' (con Zone:DNS:Read) y vuelve a levantar\n")
		return 1
	}
	o := baul.OpcionesUsar{Root: root, Nombre: entrada.Nombre, Fin: FinAmbientes, Sesion: estado.SesionActual(), Comando: estaMismaVuelta(),
		Entrada: os.Stdin, Salida: salida, Ahora: time.Now()}
	if o.Sesion == "" && term.IsTerminal(int(os.Stdin.Fd())) {
		o.Terminal = &baul.Terminal{PID: os.Getppid(), Preguntar: preguntarEnTerminal}
	}
	os.Setenv(variableAmbienteDelBaul, "1")
	defer os.Unsetenv(variableAmbienteDelBaul)
	codigo, err := usarBaul(o)
	if errors.Is(err, baul.ErrSinPermiso) {
		fmt.Fprintf(errores, "[⏳] Falta tu permiso para usar %s: confírmalo en «Para ti» (🔑) y vuelve a levantar. Dura lo que esta consola.\n", entrada.Nombre)
		return codigo
	}
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	return codigo
}

// cloudflareDe elige el token de Cloudflare del ambiente (su dirección empieza con él); si no hay uno que sirva, el
// primero sin ambiente. Uno rechazado nunca.
func cloudflareDe(indice []baul.Entrada, amb string) (baul.Entrada, bool) {
	sirve := slices.DeleteFunc(slices.Clone(indice), func(e baul.Entrada) bool { return e.Tipo != "cloudflare" || e.Rechazado })
	for _, e := range sirve {
		if strings.HasPrefix(e.Direccion, amb+"/") {
			return e, true
		}
	}
	for _, e := range sirve {
		if e.Direccion == "" {
			return e, true
		}
	}
	return baul.Entrada{}, false
}
