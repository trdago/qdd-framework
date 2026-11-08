package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/qdd-framework/qdd/pkg/baul"
	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/incidentes"
	"github.com/qdd-framework/qdd/pkg/invitaciones"
	"github.com/qdd-framework/qdd/pkg/listas"
	"github.com/qdd-framework/qdd/pkg/reportes"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 3: lo que proponen los invitados llega al panel como pedidos de la persona, uno por propuesta. El
// panel llama 'qdd flujo propuestas ciclo --si-toca' en su latido; aprobar corre desde el pedido que la persona
// confirmó o en su terminal, nunca desde una IA por su cuenta ni desde el modo goal.

// pausaDelCiclo es cada cuánto el ciclo sale al relay: el contrato pide sondear cada 15 s.
const pausaDelCiclo = 15 * time.Second

var (
	propuestasSiToca bool
	propuestasJSON   bool
	propuestasMotivo string
)

// clienteDeInvitaciones habla con el relay a nombre de esta instalación; las pruebas lo cambian por uno falso.
var clienteDeInvitaciones = func(ctx context.Context) (*invitaciones.Cliente, error) {
	cfg, err := relayDeReportes(ctx)
	if err != nil {
		return nil, fmt.Errorf("no se encontró el relay de QDD: %w", err)
	}
	llave, err := reportes.Llave()
	if err != nil {
		return nil, err
	}
	return &invitaciones.Cliente{Relay: cfg, Llave: llave}, nil
}

// fotoLimpia es la foto que ve el invitado: pasa por la limpieza de 'qdd reportar'.
func fotoLimpia(root string, f flujo.Flujo, ahora time.Time) flujo.Foto {
	contexto, valores := incidentes.ContextoDe(root), baul.ValoresDelLlavero()
	return flujo.ArmarFoto(f, func(s string) string { return reportes.Limpiar(s, contexto, valores) }, ahora)
}

// panelDeConsola deja cada propuesta como un pedido de la persona: el modo goal no lo lanza.
type panelDeConsola struct {
	root  string
	ahora time.Time
}

func (p panelDeConsola) Pedir(comando, motivo string) (string, error) {
	pedido, err := consola.PedirEnOrden(p.root, comando, motivoDePedido(motivo), "", consola.Orden{DePersona: true}, p.ahora)
	return pedido.ID, err
}

func (p panelDeConsola) Estado(id, comando string) string {
	for _, pedido := range consola.PedidosA(p.root, p.ahora) {
		if pedido.ID == id && pedido.Comando == comando {
			return pedido.Estado
		}
	}
	return ""
}

func motivoDePedido(s string) string {
	r := []rune(s)
	if len(r) <= consola.MaxMotivo {
		return s
	}
	return string(r[:consola.MaxMotivo-1]) + "…"
}

func entornoDePropuestas(ctx context.Context, root string, ahora time.Time) (invitaciones.Entorno, error) {
	cliente, err := clienteDeInvitaciones(ctx)
	if err != nil {
		return invitaciones.Entorno{}, err
	}
	foto := func(id string) (json.RawMessage, error) {
		f, err := flujo.Cargar(root, id)
		if err != nil {
			return nil, err
		}
		return listas.Marshal(fotoLimpia(root, f, ahora))
	}
	return invitaciones.Entorno{Root: root, Relay: cliente, Panel: panelDeConsola{root, ahora}, Foto: foto, Ahora: ahora}, nil
}

func cicloDePropuestas(ctx context.Context, root string, ahora time.Time) (invitaciones.Resumen, error) {
	e, err := entornoDePropuestas(ctx, root, ahora)
	if err != nil {
		return invitaciones.Resumen{}, err
	}
	return invitaciones.Ciclo(ctx, e)
}

func aprobarPropuesta(ctx context.Context, root, id string, ahora time.Time) (string, error) {
	e, err := entornoDePropuestas(ctx, root, ahora)
	if err != nil {
		return "", err
	}
	return invitaciones.Aprobar(ctx, e, id)
}

// autorizarPropuesta: una propuesta la aprueba la persona, con el pedido 📝 que confirmó o con su «sí» en la terminal.
func autorizarPropuesta(id, pedido string, preguntar func(string) bool, confirmado func(pedido, comando string) bool) error {
	if pedido != "" && confirmado(pedido, invitaciones.ComandoAprobar(id)) {
		return nil
	}
	if pedido == "" && preguntar != nil && preguntar("¿Aprobar "+id+" e integrarla al flujo?") {
		return nil
	}
	return errors.New("una propuesta la aprueba la persona: con «s  Sí» en el pedido 📝 del panel, o en su terminal")
}

var flujoPropuestasCmd = &cobra.Command{
	Use:   "propuestas",
	Short: "Lo que proponen los invitados de un flujo compartido y en qué quedó cada propuesta",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		l, err := invitaciones.Leer(estadoRaiz())
		if err != nil {
			salirCon("%v", err)
		}
		imprimirPropuestas(cmd.OutOrStdout(), estadoRaiz(), l)
	},
}

func imprimirPropuestas(w io.Writer, root string, l invitaciones.Local) {
	if propuestasJSON {
		_ = json.NewEncoder(w).Encode(l.Propuestas)
		return
	}
	if len(l.Propuestas) == 0 {
		fmt.Fprintln(w, "Todavía no llegan propuestas. 'qdd flujo compartir <flujo> --rol cliente' invita a alguien.")
		return
	}
	for _, v := range l.Propuestas {
		fmt.Fprintf(w, "%-8s %-10s %s (%s): %s\n", v.ID, v.Situacion, v.Autor, v.Rol, describirPropuesta(root, v))
	}
}

func describirPropuesta(root string, v invitaciones.Vista) string {
	o, err := invitaciones.LeerOperacion(v.Operacion)
	if err != nil {
		return v.Razon
	}
	f, _ := flujo.Cargar(root, o.Flujo)
	return o.Describir(f)
}

var flujoPropuestasCicloCmd = &cobra.Command{
	Use:   "ciclo [--si-toca] [--json]",
	Short: "Lee las propuestas nuevas del relay, las deja en el panel y avisa lo resuelto (lo llama el panel)",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		root, ahora := estadoRaiz(), time.Now()
		if propuestasSiToca && !invitaciones.TocaCiclo(root, ahora, pausaDelCiclo) {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		res, err := cicloDePropuestas(ctx, root, ahora)
		if propuestasJSON {
			_ = json.NewEncoder(cmd.OutOrStdout()).Encode(res)
		}
		if err != nil {
			salirCon("%v", err)
		}
		if !propuestasJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "[+] %d nuevas, %d en el panel, %d rechazadas solas, %d esperan turno.\n", res.Nuevas, res.Pedidas, res.Rechazadas, res.Esperan)
		}
	},
}

var flujoPropuestasAprobarCmd = &cobra.Command{
	Use:   "aprobar <PRO-n>",
	Short: "Integra la propuesta al flujo a nombre de quien la hizo: corre desde el pedido 📝 que la persona confirmó, o en su terminal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := estadoRaiz()
		confirmado := func(pedido, comando string) bool { return consola.PedidoConfirmado(root, pedido, comando, time.Now()) }
		if err := autorizarPropuesta(args[0], os.Getenv(consola.VarPedido), preguntaDeTerminal(), confirmado); err != nil {
			salirCon("%v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		linea, err := aprobarPropuesta(ctx, root, args[0], time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[+] %s\n", linea)
	},
}

var flujoPropuestasRechazarCmd = &cobra.Command{
	Use:   "rechazar <PRO-n> [--motivo <texto>]",
	Short: "Deja la propuesta fuera sin tocar el flujo; quien la hizo ve el motivo (lo decide la persona, en su terminal)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		preguntar := preguntaDeTerminal()
		if preguntar == nil || !preguntar("¿Rechazar "+args[0]+"?") {
			salirCon("una propuesta la rechaza la persona: en su terminal, o descartando su pedido 📝 en el panel")
		}
		root := estadoRaiz()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		e, err := entornoDePropuestas(ctx, root, time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		if err := invitaciones.Rechazar(ctx, e, args[0], propuestasMotivo); err != nil {
			salirCon("%v", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[+] %s rechazada: el flujo no cambia y quien la hizo lo ve en su archivo.\n", args[0])
	},
}

var flujoInvitacionesCmd = &cobra.Command{
	Use:   "invitaciones [--json]",
	Short: "Las invitaciones de este proyecto: flujo, rol, para quién, hasta cuándo y si siguen vivas",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		l, err := invitaciones.Leer(estadoRaiz())
		if err != nil {
			salirCon("%v", err)
		}
		if propuestasJSON {
			_ = json.NewEncoder(cmd.OutOrStdout()).Encode(l.Invitaciones)
			return
		}
		if len(l.Invitaciones) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Sin invitaciones. 'qdd flujo compartir <flujo> --rol cliente|experto' crea una.")
			return
		}
		ahora := time.Now()
		for _, i := range l.Invitaciones {
			viva := "vencida"
			if i.Viva(ahora) {
				viva = "viva"
			}
			if i.Revocada {
				viva = "revocada"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s  %-10s %-8s %-8s para %s, vence %s\n", i.ID, i.Flujo, i.Rol, viva, i.Para, i.Vence.Local().Format("02/01 15:04"))
		}
	},
}

func init() {
	flujoPropuestasCmd.Flags().BoolVar(&propuestasJSON, "json", false, "salida JSON")
	flujoPropuestasCicloCmd.Flags().BoolVar(&propuestasSiToca, "si-toca", false, "solo si hay invitaciones vivas o algo por avisar, y cada 15 s")
	flujoPropuestasCicloCmd.Flags().BoolVar(&propuestasJSON, "json", false, "salida JSON")
	flujoPropuestasRechazarCmd.Flags().StringVar(&propuestasMotivo, "motivo", "", "por qué no se integra (lo ve quien la propuso)")
	flujoInvitacionesCmd.Flags().BoolVar(&propuestasJSON, "json", false, "salida JSON")
	flujoPropuestasCmd.AddCommand(flujoPropuestasCicloCmd, flujoPropuestasAprobarCmd, flujoPropuestasRechazarCmd)
	flujoCmd.AddCommand(flujoPropuestasCmd, flujoInvitacionesCmd)
}
