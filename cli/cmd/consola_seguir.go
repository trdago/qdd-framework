package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// respaldoSeguir cubre lo que cambia con la hora sin que nadie escriba, y marca el latido del flujo.
const respaldoSeguir = 10 * time.Second

// errBinarioNuevo: qdd se actualizó; el flujo termina bien y el panel lo abre con el binario nuevo.
var errBinarioNuevo = errors.New("el binario de qdd cambió")

// qdd consola seguir (oculto): lo lanza el panel, un proceso por sesión en vez de uno cada 2 s (RFC 0016, MEJ-020).
var consolaSeguirCmd = &cobra.Command{
	Use:    "seguir",
	Hidden: true,
	Short:  "Lo mismo que 'consola panel --json', una línea por cada cambio (interno del panel)",
	Run:    runConsolaSeguir,
}

// qdd consola binario (oculto): con esa ruta, el panel no pasa por el envoltorio de Node en cada llamada.
var consolaBinarioCmd = &cobra.Command{
	Use:    "binario",
	Hidden: true,
	Short:  "La ruta del binario de qdd (interno del panel)",
	Run: func(cmd *cobra.Command, args []string) {
		bin, err := os.Executable()
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Println(bin)
	},
}

func init() {
	consolaSeguirCmd.Flags().StringSliceVar(&consolaCon, "con", nil, "tipos de acción opcionales que este panel dibuja")
	consolaCmd.AddCommand(consolaSeguirCmd, consolaBinarioCmd)
}

// lecturaPanel es la única fuente del JSON del panel: 'consola panel --json' y 'consola seguir' entregan lo mismo.
func lecturaPanel(root string, con []string) func() ([]byte, error) {
	return func() ([]byte, error) {
		prsAlDia(root)
		versionAlDia(root)
		datos := datosPanel(root, time.Now(), con)
		conEfectos(&datos)
		return json.Marshal(datos)
	}
}

func runConsolaSeguir(cmd *cobra.Command, args []string) {
	root := estadoRaiz()
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	senal, err := consola.SenalesDelPanel(ctx, consola.CarpetasDelPanel(root), respaldoSeguir)
	if err != nil {
		salirCon("%v", err)
	}
	bin, err := os.Executable()
	if err != nil {
		salirCon("%v", err)
	}
	err = consola.Seguir(ctx, lecturaPanel(root, consolaCon), senal, emitirLinea(os.Stdout), latidoSeguir(consola.VigiaBinario(bin), os.Stdout))
	if err != nil && !errors.Is(err, errBinarioNuevo) {
		os.Exit(1)
	}
}

func emitirLinea(w io.Writer) func([]byte) error {
	return func(l []byte) error {
		_, err := fmt.Fprintf(w, "%s\n", l)
		return err
	}
}

// latidoSeguir escribe una línea vacía: si nadie lee (la sesión murió), falla y el flujo termina, en cualquier sistema.
func latidoSeguir(cambio func() bool, w io.Writer) func() error {
	return func() error {
		if cambio() {
			return errBinarioNuevo
		}
		_, err := fmt.Fprintln(w)
		return err
	}
}
