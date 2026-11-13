package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/qdd-framework/qdd/pkg/baul"
	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/listas"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Rule: Prohibido usar 'else'
// qdd baul (RFC 0023, fase 1): los tokens de la persona, guardados una vez en el llavero o en 1Password y prestados a un
// comando con permiso por sesión y por fin. La IA nunca ve un valor: no hay «qdd baul ver».

var (
	baulTipo      string
	baulEn        string
	baulVariable  string
	baulDireccion string
	baulPara      string
	// Desde Config en el panel (pedido de la persona, 2026-11-09).
	baulPortapapeles bool
	baulSi           bool
	baulJSON         bool
	// portapapeles es el del sistema; las pruebas lo cambian.
	portapapeles = baul.PortapapelesDelSistema
)

var baulCmd = &cobra.Command{
	Use:   "baul",
	Short: "Baúl de accesos: guarda tokens en el llavero o en 1Password y los presta a un comando, con permiso por sesión",
	Long: `Guarda cada token una vez, en el llavero del sistema (por defecto) o en 1Password, y lo presta a un comando solo
con permiso de la persona: por sesión, por proyecto, por token y por fin, hasta que se cierra la consola.

  qdd baul                                     lista los tokens (nunca un valor)
  qdd baul guardar <nombre> --tipo <tipo>      pide el valor oculto, en tu terminal
  qdd baul guardar <nombre> --tipo <tipo> --portapapeles
                                               lo lee del portapapeles y lo vacía (Config en el panel)
  qdd baul quitar <nombre> [--si]              lo borra de su lugar, del índice y de los permisos
  qdd baul verificar <nombre>                  lo prueba contra su API y anota dueño, alcance y vencimiento
  qdd baul usar <nombre> --para "<fin>" -- <comando>
  qdd baul registro                            cada uso, encadenado

Tipos: cloudflare, github, supabase, resend, aws-sso (guarda el perfil de aws sso) y otro (con --variable).

Una IA usa 'qdd baul usar'. Sin permiso, deja un pedido «🔑» en el panel y sale con 75; corre de nuevo cuando la persona
diga «Sí». El token va solo al entorno del comando y su salida sale con los valores del baúl tapados.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		if baulJSON {
			listarBaulJSON(os.Stdout)
			return
		}
		listarBaul(os.Stdout)
	},
}

// listarBaulJSON es la lista para Config en el panel: los tokens (nunca un valor), los tipos y si este sistema puede
// leer el portapapeles.
func listarBaulJSON(w io.Writer) {
	indice, err := baul.Indice()
	if err != nil {
		salirCon("%v", err)
	}
	tipos := make([]string, 0, len(baul.Tipos))
	for t := range baul.Tipos {
		tipos = append(tipos, t)
	}
	slices.Sort(tipos)
	leer, _ := portapapeles().(interface{ Disponible() bool })
	out, err := listas.Marshal(map[string]any{"tokens": indice, "tipos": tipos, "portapapeles": leer != nil && leer.Disponible()})
	if err != nil {
		salirCon("%v", err)
	}
	fmt.Fprintln(w, string(out))
}

func listarBaul(w io.Writer) {
	indice, err := baul.Indice()
	if err != nil {
		salirCon("%v", err)
	}
	if len(indice) == 0 {
		fmt.Fprintln(w, "El baúl está vacío. Guarda un token en tu terminal: qdd baul guardar <nombre> --tipo <tipo>")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "NOMBRE\tDIRECCIÓN\tTIPO\tDÓNDE\tDUEÑO\tALCANCE\tVENCE\tUSADO")
	for _, e := range indice {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", e.Nombre, guion(e.Direccion), e.Tipo, e.Donde, guion(e.Duenio), guion(alcanceDe(e)), fecha(e.Vence), fecha(e.Usado))
	}
	tw.Flush()
}

func alcanceDe(e baul.Entrada) string {
	if e.Rechazado {
		return "RECHAZADO: vuelve a guardarlo"
	}
	return e.Alcance
}

func guion(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func fecha(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Local().Format("2006-01-02")
}

var baulGuardarCmd = &cobra.Command{
	Use:   "guardar <nombre>",
	Short: "Guarda un token: pide el valor oculto en tu terminal (nunca por argumentos, variables ni el chat)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		e := baul.Entrada{Nombre: args[0], Tipo: baulTipo, Donde: baulEn, Variable: baulVariable, Direccion: baulDireccion}
		if baulPortapapeles {
			if err := guardarDelPortapapeles(os.Stdout, e, portapapeles()); err != nil {
				salirCon("%v", err)
			}
			return
		}
		valor, err := leerValorOculto(os.Stdin, baulTipo)
		if err != nil {
			salirCon("%v", err)
		}
		if baulTipo != "otro" {
			if _, err := baul.VerificarValor(baulTipo, valor); errors.Is(err, baul.ErrRechazado) {
				salirCon("no se guardó: %v", err)
			}
		}
		if err := baul.Guardar(e, valor, time.Now()); err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[✓] %s quedó en el %s.\n", args[0], baulEn)
		if baulTipo == "otro" {
			return
		}
		mostrarVerificacion(args[0])
	},
}

// guardarDelPortapapeles guarda lo copiado (lo vacía siempre que lo leyó) y dice cómo quedó, sin el valor.
func guardarDelPortapapeles(w io.Writer, e baul.Entrada, p baul.Portapapeles) error {
	aviso, err := baul.GuardarDelPortapapeles(e, p, time.Now())
	if aviso != "" {
		fmt.Fprintf(w, "[!] %s\n", aviso)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "[✓] %s quedó en el %s.\n", e.Nombre, e.Donde)
	if e.Tipo == "otro" || e.Tipo == "aws-sso" {
		return nil
	}
	mostrarVerificacionEn(w, e.Nombre)
	return nil
}

var baulQuitarCmd = &cobra.Command{
	Use:   "quitar <nombre>",
	Short: "Borra un token de su lugar, del índice y de sus permisos (pregunta en tu terminal, o --si)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var preguntar func(string) bool
		if term.IsTerminal(int(os.Stdin.Fd())) {
			preguntar = preguntarEnTerminal
		}
		if err := quitarDelBaul(args[0], baulSi, preguntar); err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[✓] %s salió del baúl.\n", args[0])
	},
}

// quitarDelBaul borra con --si (el panel lo pasa tras «¿Seguro?») o con el sí de la persona en su terminal.
func quitarDelBaul(nombre string, si bool, preguntar func(string) bool) error {
	if _, err := baul.Buscar(nombre); err != nil {
		return err
	}
	if !si && preguntar == nil {
		return fmt.Errorf("quitar %s borra el token: confírmalo con --si", nombre)
	}
	if !si && !preguntar("¿Quitar "+nombre+" del baúl? Se borra su valor") {
		return fmt.Errorf("no se quitó %s", nombre)
	}
	return baul.Quitar(nombre, time.Now())
}

// leerValorOculto pide el valor en la terminal de la persona. Sin terminal se niega: una IA, un pedido o el teléfono
// no pueden guardar un token. aws-sso pide el perfil, que no es secreto, con eco.
func leerValorOculto(entrada *os.File, tipo string) (string, error) {
	fd := int(entrada.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("guardar pide el valor en tu terminal, con entrada oculta. Aquí no hay una: córrelo tú en tu terminal (nunca pegues el token en el chat)")
	}
	if tipo == "aws-sso" {
		fmt.Fprint(os.Stderr, "Perfil de aws sso (el de ~/.aws/config): ")
		linea, err := bufio.NewReader(entrada).ReadString('\n')
		return strings.TrimSpace(linea), err
	}
	fmt.Fprint(os.Stderr, "Valor (no se ve al escribir): ")
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	return strings.TrimSpace(string(b)), err
}

func mostrarVerificacion(nombre string) {
	mostrarVerificacionEn(os.Stdout, nombre)
}

func mostrarVerificacionEn(w io.Writer, nombre string) {
	e, err := baul.Verificar(nombre, time.Now())
	if errors.Is(err, baul.ErrRechazado) {
		fmt.Fprintf(w, "[✗] %s: la API lo rechaza. Vuelve a guardarlo con un token vigente.\n", nombre)
		return
	}
	if err != nil {
		fmt.Fprintf(w, "[i] %s sin verificar: %v\n", nombre, err)
		return
	}
	fmt.Fprintf(w, "[✓] %s: dueño %s · %s · vence %s\n", nombre, guion(e.Duenio), guion(e.Alcance), fecha(e.Vence))
}

var baulVerificarCmd = &cobra.Command{
	Use:   "verificar <nombre>",
	Short: "Prueba el token contra su API (solo lectura) y anota dueño, alcance y vencimiento",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := baul.Buscar(args[0]); err != nil {
			salirCon("%v", err)
		}
		mostrarVerificacion(args[0])
	},
}

var baulUsarCmd = &cobra.Command{
	Use:   "usar <nombre> --para \"<fin>\" -- <comando> [args…]",
	Short: "Corre un comando con el token en su entorno, si la persona lo permitió para esta sesión y este fin",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		root := estado.ProyectoDe(cwd)
		if root == "" {
			root = cwd
		}
		o := baul.OpcionesUsar{Root: root, Nombre: args[0], Fin: baulPara, Sesion: estado.SesionActual(), Comando: args[1:], Entrada: os.Stdin, Salida: os.Stdout, Ahora: time.Now()}
		if o.Sesion == "" && term.IsTerminal(int(os.Stdin.Fd())) {
			o.Terminal = &baul.Terminal{PID: os.Getppid(), Preguntar: preguntarEnTerminal}
		}
		codigo, err := baul.Usar(o)
		if errors.Is(err, baul.ErrSinPermiso) {
			fmt.Fprintf(os.Stderr, "[⏳] %v\n", err)
			os.Exit(codigo)
		}
		if err != nil {
			salirCon("%v", err)
		}
		os.Exit(codigo)
	},
}

func preguntarEnTerminal(pregunta string) bool {
	fmt.Fprintf(os.Stderr, "%s [s/N] ", pregunta)
	linea, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	r := strings.ToLower(strings.TrimSpace(linea))
	return r == "s" || r == "si" || r == "sí"
}

var baulPermitirCmd = &cobra.Command{
	Use:    "permitir <solicitud>",
	Short:  "Da el permiso de una solicitud: corre solo desde el pedido «🔑» que la persona confirmó en el panel",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		root := estado.ProyectoDe(cwd)
		if root == "" {
			root = cwd
		}
		confirmado := func(pedido, comando string) bool { return consola.PedidoConfirmado(root, pedido, comando, time.Now()) }
		if err := baul.Permitir(root, args[0], os.Getenv(consola.VarPedido), confirmado, time.Now()); err != nil {
			salirCon("%v", err)
		}
		fmt.Println("[✓] Permitido hasta que se cierre esa consola.")
	},
}

var baulRegistroCmd = &cobra.Command{
	Use:   "registro",
	Short: "Cada uso del baúl: quién, en qué proyecto, qué token, para qué y cómo terminó (nunca un valor)",
	Run: func(cmd *cobra.Command, args []string) {
		lineas, err := baul.Registro()
		if err != nil {
			salirCon("%v", err)
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "CUÁNDO\tQUIÉN\tACCIÓN\tTOKEN\tFIN\tCÓDIGO")
		for _, l := range lineas {
			codigo := "-"
			if l.Codigo != nil {
				codigo = fmt.Sprint(*l.Codigo)
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", l.En.Local().Format("2006-01-02 15:04"), guion(l.Quien), l.Accion, l.Token, guion(l.Fin), codigo)
		}
		tw.Flush()
		if err := baul.VerificarRegistro(); err != nil {
			salirCon("%v", err)
		}
	},
}

func init() {
	baulGuardarCmd.Flags().StringVar(&baulTipo, "tipo", "", "cloudflare, github, supabase, resend, aws-sso u otro")
	baulGuardarCmd.Flags().StringVar(&baulEn, "en", baul.EnLlavero, "llavero o 1password")
	baulGuardarCmd.Flags().StringVar(&baulVariable, "variable", "", "con --tipo otro: la variable de entorno donde va")
	baulGuardarCmd.Flags().StringVar(&baulDireccion, "direccion", "", "ambiente/servidor/servicio/clase/cuenta (p. ej. prod/db-01/notificador/postgres/app_user; ambiente: dev, qa, prod o global): una dirección es de una sola entrada")
	baulGuardarCmd.Flags().BoolVar(&baulPortapapeles, "portapapeles", false, "lee el valor del portapapeles y lo vacía (lo usa Config en el panel)")
	_ = baulGuardarCmd.MarkFlagRequired("tipo")
	baulQuitarCmd.Flags().BoolVar(&baulSi, "si", false, "sin preguntar (el panel lo pasa tras «¿Seguro?»)")
	baulCmd.Flags().BoolVar(&baulJSON, "json", false, "la lista para el panel, en JSON")
	baulUsarCmd.Flags().StringVar(&baulPara, "para", "", "el fin: para qué se usa (el permiso es por fin)")
	_ = baulUsarCmd.MarkFlagRequired("para")
	baulCmd.AddCommand(baulGuardarCmd, baulQuitarCmd, baulVerificarCmd, baulUsarCmd, baulPermitirCmd, baulRegistroCmd)
	rootCmd.AddCommand(baulCmd)
}
