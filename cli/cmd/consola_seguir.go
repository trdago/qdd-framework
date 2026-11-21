package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/herramientas"
	"github.com/qdd-framework/qdd/pkg/listas"
	"github.com/qdd-framework/qdd/pkg/remoto"
)

// respaldoSeguir cubre lo que cambia con la hora sin que nadie escriba, y marca el latido del flujo.
const respaldoSeguir = 10 * time.Second

// cadaLatidoDelFlujo es lo mismo que el panel usaba con un proceso, y menos que consola.VidaSesion.
const cadaLatidoDelFlujo = 5 * time.Second

const prefijoTelefonoEnFlujo = "telefono:"

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
		return listas.Marshal(datos)
	}
}

// carpetasDelFlujo: las del panel y las herramientas, que viven en pkg/herramientas (MEJ-020).
func carpetasDelFlujo(root string) []string {
	return append(consola.CarpetasDelPanel(root), filepath.Join(root, filepath.Dir(herramientas.Archivo("x"))), filepath.Join(root, flujo.Carpeta))
}

func runConsolaSeguir(cmd *cobra.Command, args []string) {
	root := estadoRaiz()
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	senal, err := consola.SenalesDelPanel(ctx, carpetasDelFlujo(root), respaldoSeguir, consola.EventoDelPanel(root))
	if err != nil {
		salirCon("%v", err)
	}
	bin, err := os.Executable()
	if err != nil {
		salirCon("%v", err)
	}
	salida := &salidaCompartida{w: os.Stdout}
	go latirElFlujo(ctx, root, sesionDelFlujo(consolaCon), emitirLinea(salida))
	err = consola.Seguir(ctx, lecturaPanel(root, consolaCon), senal, emitirLinea(salida), latidoSeguir(consola.VigiaBinario(bin), salida))
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

// salidaCompartida: el flujo y su latido escriben por la misma salida, y una línea no debe mezclarse con otra.
type salidaCompartida struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *salidaCompartida) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

// sesionDelFlujo es la sesión de «telefono:<id>» en lo que pide el panel (MEJ-020). Va en --con porque un qdd anterior
// ignora lo que no conoce, y el panel sigue latiendo con procesos hasta que qdd dice telefono_en_flujo.
func sesionDelFlujo(con []string) string {
	for _, c := range con {
		if sesion, ok := strings.CutPrefix(c, prefijoTelefonoEnFlujo); ok {
			return sesion
		}
	}
	return ""
}

// latirElFlujo late mientras el flujo esté abierto, sin un proceso por latido.
func latirElFlujo(ctx context.Context, root, sesion string, emitir func([]byte) error) {
	if sesion == "" {
		return
	}
	cada := time.NewTicker(cadaLatidoDelFlujo)
	defer cada.Stop()
	for {
		// Si el panel ya no lee, latidoSeguir termina el flujo; aquí no hay nadie a quien avisarle.
		_ = latidoDelFlujo(root, sesion, time.Now(), emitir)
		select {
		case <-ctx.Done():
			return
		case <-cada.C:
		}
	}
}

// latidoDelFlujo hace lo mismo que «telefono latir --confirma» y entrega por el flujo. Una instrucción que no llega al
// panel queda sin entregar: el teléfono lo ve, en vez de quedar ejecutando para siempre.
// La sesión late siempre, con o sin QDD Móvil: el baúl pide su permiso a una sesión que late (FND-191).
func latidoDelFlujo(root, sesion string, now time.Time, emitir func([]byte) error) error {
	if err := consola.LatirSesion(root, sesion, now); err != nil {
		return err
	}
	if _, err := remoto.LeerCredenciales(); err != nil {
		return nil
	}
	tomadas, err := consola.TomarParaEntregar(root, sesion, now)
	if err != nil || len(tomadas) == 0 {
		return err
	}
	out := []entrega{}
	for _, e := range tomadas {
		out = append(out, entrega{Solicitud: e.Solicitud, Nombre: e.Nombre, Mensaje: consola.MensajeInstruccion(e)})
	}
	linea, err := json.Marshal(map[string][]entrega{"telefono_entregar": out})
	if err == nil {
		err = emitir(linea)
	}
	if err == nil {
		return nil
	}
	for _, e := range tomadas {
		_ = consola.FallarEntrega(root, e.Solicitud, sesion, "no llegó al panel: "+err.Error(), now)
	}
	return err
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
