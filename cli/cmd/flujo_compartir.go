package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/invitaciones"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd flujo compartir (RFC 0025): el archivo HTML para conversar un flujo con alguien sin QDD. Sin --rol es de solo
// lectura. Con --rol crea una invitación en el relay y el archivo propone: su token va solo dentro del archivo, nunca
// en la pantalla ni en el PC. El texto pasa por la limpieza de qdd reportar (valores del baúl, credenciales, la ruta
// del proyecto, la carpeta y el nombre del usuario).

var (
	flujoPara    string
	flujoSalida  string
	flujoRol     string
	flujoDias    int
	flujoTope    int
	flujoRevocar string
)

var flujoCompartirCmd = &cobra.Command{
	Use:   "compartir <flujo> [--para <nombre>] [--rol cliente|experto] [--dias <n>] [--tope <n>] [--salida <archivo.html>]",
	Short: "Escribe un archivo HTML del flujo para conversarlo con alguien sin QDD; con --rol, quien lo abra propone cambios",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		root := estadoRaiz()
		if flujoRevocar != "" {
			revocarInvitacion(cmd, root)
			return
		}
		if len(args) != 1 {
			salirCon("falta el flujo: qdd flujo compartir <flujo>")
		}
		f, err := flujo.Cargar(root, args[0])
		if err != nil {
			salirCon("%v", err)
		}
		ahora := time.Now()
		foto := fotoLimpia(root, f, ahora)
		if flujoRol == "" {
			escribirArchivo(cmd, root, f, mustArchivo(flujo.ArchivoHTML(foto)))
			fmt.Fprintf(cmd.OutOrStdout(), "    Archivo de solo lectura (r%d): se abre en cualquier navegador; adjúntalo en un correo o un chat.\n", f.Revision)
			return
		}
		inv, html := invitar(root, f, foto, ahora)
		escribirArchivo(cmd, root, f, html)
		fmt.Fprintf(cmd.OutOrStdout(), "    Invitación %s (%s, vence el %s, hasta %d propuestas): quien abra el archivo propone,\n"+
			"    y cada propuesta llega a tu panel para aprobarla o descartarla. El archivo lleva la llave de la invitación:\n"+
			"    mándaselo solo a %s. Para cerrarla: qdd flujo compartir --revocar %s\n", inv.ID, inv.Rol, inv.Vence.Local().Format("02/01 15:04"), flujoTope, paraQuien(), inv.ID)
	},
}

func mustArchivo(html []byte, err error) []byte {
	if err != nil {
		salirCon("%v", err)
	}
	return html
}

func paraQuien() string {
	if flujoPara == "" {
		return "quien invitas"
	}
	return flujoPara
}

// invitar crea la invitación en el relay, la anota en el PC (sin el token) y arma el archivo que propone.
func invitar(root string, f flujo.Flujo, foto flujo.Foto, ahora time.Time) (invitaciones.Invitacion, []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cliente, err := clienteDeInvitaciones(ctx)
	if err != nil {
		salirCon("%v", err)
	}
	datos, err := json.Marshal(foto)
	if err != nil {
		salirCon("%v", err)
	}
	vence := ahora.Add(time.Duration(flujoDias) * 24 * time.Hour)
	creada, err := cliente.Invitar(ctx, invitaciones.Nueva{Flujo: f.ID, Rol: flujoRol, Foto: datos, Vence: vence, Tope: flujoTope}, ahora)
	if err != nil {
		salirCon("%v", err)
	}
	inv := invitaciones.Invitacion{ID: creada.Invitacion, Flujo: f.ID, Rol: flujoRol, Para: flujoPara, Vence: vence, Creada: ahora}
	if err := invitaciones.Anotar(root, inv); err != nil {
		salirCon("%v", err)
	}
	e := flujo.Enlace{Relay: cliente.Funciones(), Clave: cliente.Relay.Clave, Token: creada.Token, Rol: flujoRol, Invitacion: creada.Invitacion}
	return inv, mustArchivo(flujo.ArchivoParaProponer(foto, e))
}

func revocarInvitacion(cmd *cobra.Command, root string) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cliente, err := clienteDeInvitaciones(ctx)
	if err != nil {
		salirCon("%v", err)
	}
	if err := cliente.Revocar(ctx, flujoRevocar, time.Now()); err != nil {
		salirCon("%v", err)
	}
	if err := invitaciones.MarcarRevocada(root, flujoRevocar); err != nil {
		salirCon("%v", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "[+] Invitación %s cerrada: su archivo se sigue leyendo, pero ya no propone.\n", flujoRevocar)
}

func escribirArchivo(cmd *cobra.Command, root string, f flujo.Flujo, html []byte) {
	destino := archivoDeCompartir(root, f.ID)
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		salirCon("%v", err)
	}
	if err := os.WriteFile(destino, html, 0o600); err != nil {
		salirCon("%v", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "[+] %s\n", destino)
}

// archivoDeCompartir es --salida o ./tmp/flujo-<id>[-<para>].html en el proyecto.
func archivoDeCompartir(root, id string) string {
	if flujoSalida != "" {
		return flujoSalida
	}
	nombre := "flujo-" + id
	if para := flujo.IDGrupo(flujoPara); flujoPara != "" && para != "grupo-" {
		nombre += "-" + para[len("grupo-"):]
	}
	return filepath.Join(root, "tmp", nombre+".html")
}

func init() {
	f := flujoCompartirCmd.Flags()
	f.StringVar(&flujoPara, "para", "", "con quién se conversa (va en el nombre del archivo)")
	f.StringVar(&flujoSalida, "salida", "", "dónde escribir el archivo (por defecto ./tmp/flujo-<id>-<para>.html)")
	f.StringVar(&flujoRol, "rol", "", "cliente o experto: crea una invitación y el archivo propone por el relay")
	f.IntVar(&flujoDias, "dias", 14, "cuántos días vive la invitación (1 a 90)")
	f.IntVar(&flujoTope, "tope", 50, "cuántas propuestas acepta la invitación (1 a 200)")
	f.StringVar(&flujoRevocar, "revocar", "", "cierra esa invitación: su archivo deja de proponer")
	flujoCmd.AddCommand(flujoCompartirCmd)
}
