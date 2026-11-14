package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): activar, emparejar, desactivar y correr. Lo que habla con el sistema (el servicio, la carpeta del
// usuario) se puede cambiar en las pruebas, que nunca instalan nada.

var (
	remotoRelayURL   string
	remotoClave      string
	remotoSinServ    bool
	remotoEjecutor   remoto.Ejecutor = ejecutarSistema
	remotoConsultar  remoto.Consulta = consultarSistema
	remotoUsuario                    = os.UserHomeDir
	remotoPlazo                      = remoto.VigenciaOferta
	remotoCadaEspera time.Duration

	remotoConsultarRelay = remoto.DirectorioDeFabrica().Consultar
)

// consultarSistema corre un comando que solo lee y devuelve lo que imprimió.
func consultarSistema(nombre string, args ...string) (string, error) {
	out, err := exec.Command(nombre, args...).Output()
	return string(out), err
}

func ejecutarSistema(nombre string, args ...string) error {
	out, err := exec.Command(nombre, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

var remotoActivarCmd = &cobra.Command{
	Use:   "activar [--relay <url> --clave <publicable>]",
	Short: "Guarda el relay, empareja este PC con QDD Móvil y deja el servicio que lo publica",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(remotoActivar(cmd.Context(), os.Stdout, os.Stderr))
	},
}

var remotoEmparejarCmd = &cobra.Command{
	Use:   "emparejar",
	Short: "Muestra el QR para QDD Móvil y espera a que el teléfono lo acepte (5 min, un solo uso)",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(remotoEmparejar(cmd.Context(), os.Stdout, os.Stderr))
	},
}

var remotoDesactivarCmd = &cobra.Command{
	Use:   "desactivar",
	Short: "Saca este PC del relay, detiene y borra el servicio y las credenciales",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(remotoDesactivar(cmd.Context(), os.Stdout, os.Stderr))
	},
}

var remotoCorrerCmd = &cobra.Command{
	Use:    "correr",
	Short:  "El proceso que publica este PC en QDD Móvil (lo corre el servicio)",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(remotoCorrer(os.Stderr))
	},
}

func init() {
	remotoActivarCmd.Flags().StringVar(&remotoRelayURL, "relay", "", "URL del relay (https://<ref>.supabase.co); sin ella, la guardada")
	remotoActivarCmd.Flags().StringVar(&remotoClave, "clave", "", "clave publicable del relay (sb_publishable_… o el JWT anon); nunca la secreta")
	remotoActivarCmd.Flags().BoolVar(&remotoSinServ, "sin-servicio", false, "no deja el servicio: córrelo tú con 'qdd remoto correr'")
	remotoCmd.AddCommand(remotoActivarCmd, remotoEmparejarCmd, remotoDesactivarCmd, remotoCorrerCmd)
}

func contexto(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// relayDeFlags: las banderas mandan y dejan el relay como propio; si no, lo guardado, confirmado con el directorio; la
// primera vez, lo que diga el directorio.
func relayDeFlags(errores io.Writer) (remoto.ConfigRelay, error) {
	if remotoRelayURL != "" || remotoClave != "" {
		return guardarYLeerRelay(remoto.ConfigRelay{URL: remotoRelayURL, Clave: remotoClave, Propio: true})
	}
	c, err := remoto.LeerRelay()
	if err == nil {
		return relayConfirmado(context.Background(), c, errores), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return c, err
	}
	c, err = remotoConsultarRelay(context.Background(), 0)
	if err != nil {
		return c, fmt.Errorf("falta el relay: %v; vuelve a intentarlo o usa 'qdd login --relay <url> --clave <publicable>'", err)
	}
	return guardarYLeerRelay(c)
}

// relayConfirmado es el relay con el que se sigue. Nunca calla (FND-171): quien va a escanear un código tiene que saber si
// nada confirmó el relay o si se reemplazó el que tenía guardado como propio.
func relayConfirmado(ctx context.Context, actual remoto.ConfigRelay, errores io.Writer) remoto.ConfigRelay {
	c, cambio, err := remoto.ConfirmarRelay(ctx, remotoConsultarRelay, actual, estaEmparejado())
	if err != nil {
		fmt.Fprintf(errores, "[!] no se pudo confirmar el relay con el directorio (%v): sigue con el guardado, que puede estar retirado. Si el teléfono no conecta, usa 'qdd doctor --fix' desde otra red.\n", err)
		return c
	}
	if cambio && actual.Propio {
		fmt.Fprintln(errores, "[i] El relay guardado como propio no coincide con el directorio y este PC no estaba emparejado: se reemplazó. Si el tuyo es otro, usa 'qdd login --relay <url> --clave <publicable>'.")
	}
	return c
}

func estaEmparejado() bool {
	_, err := remoto.LeerCredenciales()
	return err == nil
}

func guardarYLeerRelay(c remoto.ConfigRelay) (remoto.ConfigRelay, error) {
	if err := remoto.GuardarRelay(c); err != nil {
		return c, err
	}
	return remoto.LeerRelay()
}

func remotoActivar(ctx context.Context, salida, errores io.Writer) int {
	ctx = contexto(ctx)
	c, err := relayDeFlags(errores)
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintf(salida, "[+] Relay: %s (clave publicable guardada en <QDD_HOME>/remoto/relay.json, 0600)\n", c.URL)
	cred, err := remoto.LeerCredenciales()
	if err != nil {
		cred, err = emparejarYGuardar(ctx, c, salida)
	}
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintf(salida, "[+] Este PC es «%s» en QDD Móvil.\n", cred.Nombre)
	if remotoSinServ {
		fmt.Fprintln(salida, "[i] Sin servicio: corre 'qdd remoto correr' para publicarlo.")
		return 0
	}
	s, err := servicioRemoto()
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	archivo, err := s.Instalar(remotoEjecutor)
	if err != nil {
		fmt.Fprintf(errores, "[!] el servicio quedó en %s pero no se pudo encender: %v\n    Córrelo a mano con 'qdd remoto correr'.\n", archivo, err)
		return 1
	}
	fmt.Fprintf(salida, "[+] Servicio en %s: publica este PC mientras tu sesión está abierta.\n", archivo)
	if s.SO == "linux" {
		fmt.Fprintln(salida, "[i] Para que siga sin sesión abierta: loginctl enable-linger (decisión tuya; qdd no lo cambia).")
	}
	return 0
}

func emparejarYGuardar(ctx context.Context, c remoto.ConfigRelay, salida io.Writer) (remoto.Credenciales, error) {
	cred, err := remoto.Emparejar(ctx, remoto.NuevoRelay(c), remoto.EstaMaquina().Nombre, salida, remotoCadaEspera, remotoPlazo)
	if err != nil {
		return cred, err
	}
	if err := remoto.GuardarCredenciales(cred); err != nil {
		return cred, err
	}
	fmt.Fprintln(salida, "[+] Emparejado: credenciales en <QDD_HOME>/remoto/credenciales.json (0600).")
	return cred, nil
}

func remotoEmparejar(ctx context.Context, salida, errores io.Writer) int {
	ctx = contexto(ctx)
	if cred, err := remoto.LeerCredenciales(); err == nil {
		fmt.Fprintf(errores, "[!] Este PC ya está emparejado como «%s»: cualquier teléfono con tu cuenta lo ve. Para emparejarlo de nuevo, primero 'qdd remoto desactivar'.\n", cred.Nombre)
		return 1
	}
	c, err := remoto.LeerRelay()
	if err != nil {
		fmt.Fprintln(errores, "[!] falta el relay: usa 'qdd login'")
		return 1
	}
	if _, err := emparejarYGuardar(ctx, relayConfirmado(ctx, c, errores), salida); err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintln(salida, "[i] Este PC no publica hasta tener su servicio: corre 'qdd login'.")
	return 0
}

func remotoDesactivar(ctx context.Context, salida, errores io.Writer) int {
	ctx = contexto(ctx)
	cred, errCred := remoto.LeerCredenciales()
	c, errRelay := remoto.LeerRelay()
	if errCred == nil && errRelay == nil {
		_, err := remoto.NuevoRelay(c).Desactivar(ctx, cred.Topico, cred.Firma)
		if err != nil && !errors.Is(err, remoto.ErrCredenciales) {
			fmt.Fprintf(errores, "[!] no se pudo sacar el PC del relay (%v): quítalo desde el teléfono («Quitar este PC»).\n", err)
		}
		fmt.Fprintln(salida, "[+] El PC salió del relay: su canal ya no admite a nadie.")
	}
	quitarServicio(salida, errores)
	if err := remoto.BorrarCredenciales(); err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintln(salida, "[+] Credenciales borradas. La config del relay queda para volver a activar.")
	return 0
}

func quitarServicio(salida, errores io.Writer) {
	s, err := servicioRemoto()
	if err != nil {
		return
	}
	err = s.Quitar(remotoEjecutor)
	if errors.Is(err, remoto.ErrServicioDeOtroHome) {
		fmt.Fprintln(salida, "[i] El servicio de este PC publica otro QDD_HOME: no se tocó.")
		return
	}
	if err != nil {
		fmt.Fprintf(errores, "[!] servicio: %v\n", err)
		return
	}
	fmt.Fprintln(salida, "[+] Servicio detenido.")
}

// servicioRemoto arma el servicio de este sistema con el binario que corre y el PATH de la terminal.
func servicioRemoto() (remoto.Servicio, error) {
	usuario, err := remotoUsuario()
	if err != nil {
		return remoto.Servicio{}, err
	}
	bin, err := os.Executable()
	if err != nil {
		return remoto.Servicio{}, err
	}
	if real, err := filepath.EvalSymlinks(bin); err == nil {
		bin = real
	}
	dir, err := remoto.DirRemoto()
	if err != nil {
		return remoto.Servicio{}, err
	}
	return remoto.Servicio{SO: runtime.GOOS, Bin: bin, Usuario: usuario, QDDHome: os.Getenv(blueprints.HomeEnv), Path: os.Getenv("PATH"), DirQDD: dir, Cuenta: cuentaWindows()}, nil
}

// pausarAlQuitarse: el teléfono quitó este PC; en Windows la tarea se desactiva para que el disparador de cada 5
// minutos no lo vuelva a lanzar (FND-204). qdd remoto activar la registra de nuevo. Una falla no cambia la salida.
func pausarAlQuitarse() {
	s, err := remotoServicio()
	if err != nil {
		return
	}
	for _, c := range s.ComandosAlQuitarse() {
		_ = remotoEjecutor(c[0], c[1:]...)
	}
}

// cuentaWindows es DOMINIO\usuario de quien corre qdd en Windows: la tarea de inicio es solo suya (FND-195). Fuera de
// Windows no hace falta.
func cuentaWindows() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.Username
}

// remotoCorrer es el servicio: uno por máquina, hasta una señal o hasta que el relay ya no reconozca el PC.
func remotoCorrer(errores io.Writer) int {
	c, err := remoto.LeerRelay()
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	cred, err := remoto.LeerCredenciales()
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	soltar, err := unaSolaVez()
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	defer soltar()
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	c, _, err = remoto.ActualizarRelay(ctx, remotoConsultarRelay, c)
	if err != nil {
		fmt.Fprintf(errores, "[i] sigue con el relay guardado: %v\n", err)
	}
	canal := remoto.NuevoCanal(remoto.ConfigCanal{Relay: c.URL, Clave: c.Clave, Topico: cred.Topico, PresenceKey: "pc"})
	go canal.Correr(ctx)
	bin, _ := os.Executable()
	b := &remoto.Bucle{QDD: Version, Cred: cred, T: canal, R: remoto.NuevoRelay(c), Bin: bin, Log: errores}
	b.AlPublicar = func(ok bool) {
		if err := remoto.GuardarPublicacion(ok, time.Now()); err != nil {
			fmt.Fprintf(errores, "[i] no se anotó el último envío: %v\n", err)
		}
	}
	b.Consultar = func(ctx context.Context) (bool, error) {
		actual, err := remoto.LeerRelay()
		if err != nil {
			return false, err
		}
		_, cambio, err := remoto.ActualizarRelay(ctx, remotoConsultarRelay, actual)
		return cambio, err
	}
	err = b.Correr(ctx)
	if errors.Is(err, remoto.ErrRelayCambio) {
		fmt.Fprintf(errores, "[i] %v.\n", err)
		return codigoReiniciar
	}
	if errors.Is(err, remoto.ErrCredenciales) {
		fmt.Fprintln(errores, "[i] El relay ya no reconoce este PC: se quitó desde el teléfono. Vuelve a emparejarlo con 'qdd remoto desactivar' y 'qdd remoto activar'.")
		pausarAlQuitarse()
		return 0
	}
	if errors.Is(err, remoto.ErrActualizado) {
		fmt.Fprintf(errores, "[i] %v.\n", err)
		return codigoReiniciar
	}
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	return 0
}

// codigoReiniciar es la salida con la que el servicio vuelve a arrancar qdd: así toma la versión nueva tras actualizar
// qdd (FND-120). systemd y launchd reinician ante una salida distinta de 0; en Windows, RestartOnFailure no, y lo
// relanzan el /Run de la actualización o el disparador de cada 5 minutos (FND-204).
const codigoReiniciar = 3

// unaSolaVez deja <QDD_HOME>/remoto/correr.pid con este proceso; si otro vivo ya corre, es un error.
func unaSolaVez() (func(), error) {
	dir, err := remoto.DirRemoto()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	ruta := filepath.Join(dir, "correr.pid")
	if b, err := os.ReadFile(ruta); err == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		if pid > 0 && pid != os.Getpid() && consola.ProcesoVivo(pid) {
			return nil, fmt.Errorf("qdd remoto ya corre en esta máquina (pid %d)", pid)
		}
	}
	if err := os.WriteFile(ruta, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		return nil, err
	}
	return func() { os.Remove(ruta) }, nil
}
