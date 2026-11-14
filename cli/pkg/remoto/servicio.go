package remoto

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): qdd remoto activar deja «qdd remoto correr» como servicio del usuario, que arranca con la sesión y
// se reinicia si se cae: systemd --user en Linux, launchd en macOS y una tarea programada al iniciar sesión en Windows.
// En Windows, RestartOnFailure solo reintenta una tarea que no pudo arrancar, no un programa que sale con error: la
// tarea lleva además un disparador cada 5 minutos que la lanza si no corre (FND-204).
// El servicio lleva el PATH de quien lo activó: los pedidos corren con la shell y necesitan las mismas herramientas
// (git, gh, npm) que en la terminal.

const (
	nombreUnidad = "qdd-remoto.service"
	etiquetaMac  = "dev.qdd.remoto"
	tareaWindows = "QDD remoto"
	// tareaAnterior es el nombre del archivo de la tarea, con el que se registró a mano mientras qdd no podía (FND-195).
	tareaAnterior = "qdd-remoto"
	// reintentoTarea lanza la tarea cada 5 minutos; con IgnoreNew, si ya corre no pasa nada (FND-204).
	reintentoTarea = `<TimeTrigger><Repetition><Interval>PT5M</Interval><StopAtDurationEnd>false</StopAtDurationEnd></Repetition><StartBoundary>2026-01-01T00:00:00</StartBoundary><Enabled>true</Enabled></TimeTrigger>`
	// Lo que hace que el sistema no mate lo lanzado por el servicio cuando este termina (FND-128).
	marcaSystemd = "KillMode=process"
	marcaLaunchd = "<key>AbandonProcessGroup</key><true/>"
)

// Servicio es lo necesario para escribir el servicio: el sistema, el binario de qdd, la carpeta del usuario, QDD_HOME
// (si se fijó) y el PATH.
type Servicio struct {
	SO      string
	Bin     string
	Usuario string // carpeta del usuario (os.UserHomeDir)
	QDDHome string // vacío: el de siempre
	Path    string
	DirQDD  string // <QDD_HOME>/remoto, donde va la tarea de Windows
	// Cuenta es DOMINIO\usuario en Windows: la tarea es solo suya. Sin usuario vale para cualquiera del equipo, y Windows
	// pide administrador para registrarla (FND-195).
	Cuenta string
}

// Archivo es dónde queda el servicio.
func (s Servicio) Archivo() (string, error) {
	if s.SO == "linux" {
		return filepath.Join(s.Usuario, ".config", "systemd", "user", nombreUnidad), nil
	}
	if s.SO == "darwin" {
		return filepath.Join(s.Usuario, "Library", "LaunchAgents", etiquetaMac+".plist"), nil
	}
	if s.SO == "windows" {
		return filepath.Join(s.DirQDD, "qdd-remoto.xml"), nil
	}
	return "", fmt.Errorf("qdd remoto no sabe dejar un servicio en %s: corre 'qdd remoto correr' a mano", s.SO)
}

// Contenido es el archivo del servicio.
func (s Servicio) Contenido() ([]byte, error) {
	if s.SO == "linux" {
		return []byte(s.unidadSystemd()), nil
	}
	if s.SO == "darwin" {
		return []byte(s.plist()), nil
	}
	if s.SO == "windows" {
		return utf16ConBOM(s.tareaXML()), nil
	}
	_, err := s.Archivo()
	return nil, err
}

// DejaVivoLoLanzado revisa el archivo instalado: al reiniciarse, el servicio no mata lo que lanzó (FND-128). qdd remoto
// termina cuando ve un qdd nuevo, y un pedido que está instalando ese qdd tiene que seguir corriendo.
func (s Servicio) DejaVivoLoLanzado() (bool, bool, error) {
	archivo, err := s.Archivo()
	if err != nil {
		return false, false, err
	}
	b, err := os.ReadFile(archivo)
	if errors.Is(err, os.ErrNotExist) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if s.SO == "linux" {
		return true, strings.Contains(string(b), marcaSystemd), nil
	}
	if s.SO == "darwin" {
		return true, strings.Contains(string(b), marcaLaunchd), nil
	}
	return true, true, nil
}

// RenovarDejaVivo le agrega la marca al archivo instalado y deja lo demás como está: el binario, el PATH y QDD_HOME son
// los que eligió la persona al activarlo, no los de quien corre doctor. En systemd no reinicia el servicio.
func (s Servicio) RenovarDejaVivo(correr Ejecutor) error {
	archivo, err := s.Archivo()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(archivo)
	if err != nil {
		return err
	}
	nuevo, comandos, err := s.conMarca(string(b), archivo)
	if err != nil {
		return err
	}
	if err := os.WriteFile(archivo, []byte(nuevo), 0o600); err != nil {
		return err
	}
	for i, c := range comandos {
		err := correr(c[0], c[1:]...)
		if err != nil && !(s.SO == "darwin" && i == 0) {
			return fmt.Errorf("%s: %w", strings.Join(c, " "), err)
		}
	}
	return nil
}

func (s Servicio) conMarca(contenido, archivo string) (string, [][]string, error) {
	seccion, marca := "[Service]\n", marcaSystemd+"\n"
	comandos := [][]string{{"systemctl", "--user", "daemon-reload"}}
	if s.SO == "darwin" {
		seccion, marca = "<dict>\n", "\t"+marcaLaunchd+"\n"
		comandos = [][]string{{"launchctl", "unload", archivo}, {"launchctl", "load", "-w", archivo}}
	}
	if !strings.Contains(contenido, seccion) {
		return "", nil, fmt.Errorf("%s no tiene %s: corre qdd remoto activar", archivo, strings.TrimSpace(seccion))
	}
	return strings.Replace(contenido, seccion, seccion+marca, 1), comandos, nil
}

func (s Servicio) unidadSystemd() string {
	var env strings.Builder
	if s.QDDHome != "" {
		fmt.Fprintf(&env, "Environment=%s\n", comillasSystemd("QDD_HOME="+s.QDDHome))
	}
	if s.Path != "" {
		fmt.Fprintf(&env, "Environment=%s\n", comillasSystemd("PATH="+s.Path))
	}
	return "[Unit]\nDescription=QDD remoto: el panel de este PC en QDD Móvil\nAfter=network-online.target\nWants=network-online.target\n\n" +
		"[Service]\nExecStart=" + comillasSystemd(s.Bin) + " remoto correr\n" + env.String() +
		"Restart=on-failure\nRestartSec=15\n" + marcaSystemd + "\n\n[Install]\nWantedBy=default.target\n"
}

func comillasSystemd(v string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "%", "%%").Replace(v) + `"`
}

func (s Servicio) plist() string {
	var env strings.Builder
	if s.QDDHome != "" {
		env.WriteString("\t\t<key>QDD_HOME</key><string>" + escaparXML(s.QDDHome) + "</string>\n")
	}
	if s.Path != "" {
		env.WriteString("\t\t<key>PATH</key><string>" + escaparXML(s.Path) + "</string>\n")
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key><string>` + etiquetaMac + `</string>
	<key>ProgramArguments</key>
	<array><string>` + escaparXML(s.Bin) + `</string><string>remoto</string><string>correr</string></array>
	<key>EnvironmentVariables</key>
	<dict>
` + env.String() + `	</dict>
	<key>RunAtLoad</key><true/>
	<key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
	<key>ThrottleInterval</key><integer>15</integer>
	` + marcaLaunchd + `
</dict>
</plist>
`
}

func (s Servicio) tareaXML() string {
	return `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo><Description>QDD remoto: el panel de este PC en QDD Móvil</Description></RegistrationInfo>
  <Triggers><LogonTrigger><Enabled>true</Enabled>` + s.usuarioTarea() + `</LogonTrigger>` + reintentoTarea + `</Triggers>
  <Principals><Principal id="Author">` + s.usuarioTarea() + `<LogonType>InteractiveToken</LogonType><RunLevel>LeastPrivilege</RunLevel></Principal></Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Hidden>true</Hidden>
    <RestartOnFailure><Interval>PT1M</Interval><Count>999</Count></RestartOnFailure>
  </Settings>
  <Actions Context="Author"><Exec><Command>` + escaparXML(s.Bin) + `</Command><Arguments>remoto correr</Arguments></Exec></Actions>
</Task>
`
}

func (s Servicio) usuarioTarea() string {
	if s.Cuenta == "" {
		return ""
	}
	return "<UserId>" + escaparXML(s.Cuenta) + "</UserId>"
}

func escaparXML(v string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(v)
}

func utf16ConBOM(s string) []byte {
	u := utf16.Encode([]rune(s))
	out := []byte{0xFF, 0xFE}
	for _, r := range u {
		out = append(out, byte(r), byte(r>>8))
	}
	return out
}

// ComandosReiniciar: encender ya reinicia en Linux y macOS. En Windows, /Run sobre una tarea que corre no la reinicia, así
// que primero se termina.
func (s Servicio) ComandosReiniciar() [][]string {
	if s.SO == "windows" {
		return [][]string{{"schtasks", "/End", "/TN", tareaWindows}, {"schtasks", "/Run", "/TN", tareaWindows}}
	}
	return s.Comandos(true)
}

// Comandos son lo que se corre para encender (despues de escribir el archivo) o apagar el servicio.
func (s Servicio) Comandos(encender bool) [][]string {
	archivo, _ := s.Archivo()
	if s.SO == "linux" && encender {
		return [][]string{{"systemctl", "--user", "daemon-reload"}, {"systemctl", "--user", "enable", nombreUnidad}, {"systemctl", "--user", "restart", nombreUnidad}}
	}
	if s.SO == "linux" {
		return [][]string{{"systemctl", "--user", "disable", "--now", nombreUnidad}, {"systemctl", "--user", "daemon-reload"}}
	}
	if s.SO == "darwin" && encender {
		return [][]string{{"launchctl", "unload", archivo}, {"launchctl", "load", "-w", archivo}}
	}
	if s.SO == "darwin" {
		return [][]string{{"launchctl", "unload", "-w", archivo}}
	}
	if s.SO == "windows" && encender {
		return [][]string{{"schtasks", "/Create", "/TN", tareaWindows, "/XML", archivo, "/F"}, {"schtasks", "/Run", "/TN", tareaWindows}}
	}
	if s.SO == "windows" {
		return [][]string{{"schtasks", "/End", "/TN", tareaWindows}, {"schtasks", "/Delete", "/TN", tareaWindows, "/F"}}
	}
	return nil
}

// esDeEsteHome mira el QDD_HOME del archivo instalado. En Windows el archivo ya vive en <QDD_HOME>/remoto.
func (s Servicio) esDeEsteHome(contenido string) bool {
	if s.SO == "windows" {
		return true
	}
	if s.QDDHome == "" {
		return !strings.Contains(contenido, "QDD_HOME")
	}
	if s.SO == "darwin" {
		return strings.Contains(contenido, "<key>QDD_HOME</key><string>"+escaparXML(s.QDDHome)+"</string>")
	}
	return strings.Contains(contenido, comillasSystemd("QDD_HOME="+s.QDDHome))
}

// Ejecutor corre un comando del sistema (en pruebas, uno que solo anota).
type Ejecutor func(nombre string, args ...string) error

// Instalar escribe el servicio y lo enciende. Un primer comando que falla al apagar lo que no había (launchctl unload)
// no es un error.
func (s Servicio) Instalar(correr Ejecutor) (string, error) {
	archivo, err := s.Archivo()
	if err != nil {
		return "", err
	}
	contenido, err := s.Contenido()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(archivo), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(archivo, contenido, 0o600); err != nil {
		return "", err
	}
	for i, c := range s.Comandos(true) {
		err := correr(c[0], c[1:]...)
		if err != nil && !(s.SO == "darwin" && i == 0) {
			return archivo, fmt.Errorf("%s: %w", strings.Join(c, " "), err)
		}
	}
	return archivo, nil
}

// reComandoTarea es el programa que lanza la tarea de Windows.
var reComandoTarea = regexp.MustCompile(`<Command>[^<]*</Command>`)

// TareaSinEstaVersion: la tarea de Windows lanza otro binario que el de esta versión. Un binario compilado a mano no
// cuenta: no es el que la persona instaló.
func (s Servicio) TareaSinEstaVersion() bool {
	if !consola.BinarioPorVersion(s.Bin) {
		return false
	}
	xml, ok := s.tareaInstalada()
	return ok && !strings.Contains(xml, s.comandoTarea())
}

// TareaSinCuenta: la tarea de Windows instalada se dispara para cualquier usuario, así que registrarla pide administrador
// y, sin él, quedó solo el archivo (FND-195).
func (s Servicio) TareaSinCuenta() bool {
	if s.Cuenta == "" {
		return false
	}
	xml, ok := s.tareaInstalada()
	return ok && !strings.Contains(xml, "<UserId>")
}

func (s Servicio) tareaInstalada() (string, bool) {
	if s.SO != "windows" {
		return "", false
	}
	archivo, err := s.Archivo()
	if err != nil {
		return "", false
	}
	b, err := os.ReadFile(archivo)
	if err != nil {
		return "", false
	}
	return desdeUTF16(b), true
}

func (s Servicio) comandoTarea() string {
	return "<Command>" + escaparXML(s.Bin) + "</Command>"
}

// TareaSinReintento: la tarea de Windows instalada solo arranca al iniciar sesión, así que si qdd remoto termina (al ver
// una versión nueva, salida 3) nada lo vuelve a lanzar hasta el próximo inicio de sesión (FND-204).
func (s Servicio) TareaSinReintento() bool {
	xml, ok := s.tareaInstalada()
	return ok && !strings.Contains(xml, "<TimeTrigger>")
}

// RenovarTarea pone al día la tarea de Windows y deja lo demás como está: el binario de esta versión, porque su carpeta
// cambia con cada una (FND-142), el usuario, si le faltaba (FND-195), y el disparador que la relanza (FND-204). Después
// la arranca: lo que corría termina al ver la versión nueva, y RestartOnFailure no lo relanza. Si todavía corre, /Run no
// abre otra (IgnoreNew) y el disparador la lanza a los 5 minutos de terminar. Sin tarea instalada, o ya al día, no hace
// nada.
func (s Servicio) RenovarTarea(correr Ejecutor) error {
	version, cuenta, reintento := s.TareaSinEstaVersion(), s.TareaSinCuenta(), s.TareaSinReintento()
	if !version && !cuenta && !reintento {
		return nil
	}
	xml, _ := s.tareaInstalada()
	if version {
		xml = reComandoTarea.ReplaceAllLiteralString(xml, s.comandoTarea())
	}
	if cuenta {
		xml = strings.Replace(xml, "</LogonTrigger>", s.usuarioTarea()+"</LogonTrigger>", 1)
		xml = strings.Replace(xml, `<Principal id="Author">`, `<Principal id="Author">`+s.usuarioTarea(), 1)
	}
	if reintento {
		xml = strings.Replace(xml, "</Triggers>", reintentoTarea+"</Triggers>", 1)
	}
	archivo, err := s.Archivo()
	if err != nil {
		return err
	}
	if err := os.WriteFile(archivo, utf16ConBOM(xml), 0o600); err != nil {
		return err
	}
	if err := correr("schtasks", "/Create", "/TN", tareaWindows, "/XML", archivo, "/F"); err != nil {
		return err
	}
	if err := correr("schtasks", "/Run", "/TN", tareaWindows); err != nil {
		return fmt.Errorf("schtasks /Run: %w", err)
	}
	return nil
}

// Consulta corre un comando del sistema y devuelve su salida (en pruebas, una fija).
type Consulta func(nombre string, args ...string) (string, error)

// TareaSobrante: queda la tarea «qdd-remoto» que lanza qdd remoto correr, registrada a mano (FND-195). Al iniciar sesión
// compite con la de qdd y puede lanzar una versión ya desinstalada (FND-204). Una con ese nombre que lanza otra cosa no
// es de qdd.
func (s Servicio) TareaSobrante(consultar Consulta) bool {
	if s.SO != "windows" {
		return false
	}
	salida, err := consultar("schtasks", "/Query", "/TN", tareaAnterior, "/XML")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ReplaceAll(salida, "\x00", ""), "<Arguments>remoto correr</Arguments>")
}

// QuitarTareaSobrante la apaga y la borra. Que no estuviera corriendo no es un error.
func (s Servicio) QuitarTareaSobrante(correr Ejecutor) error {
	_ = correr("schtasks", "/End", "/TN", tareaAnterior)
	return correr("schtasks", "/Delete", "/TN", tareaAnterior, "/F")
}

// ComandosAlQuitarse: el teléfono quitó este PC y qdd remoto termina con 0. systemd y launchd no reinician esa salida,
// pero en Windows el disparador lo relanzaría cada 5 minutos: la tarea se desactiva hasta que activar la vuelve a
// registrar (FND-204).
func (s Servicio) ComandosAlQuitarse() [][]string {
	if s.SO != "windows" {
		return nil
	}
	return [][]string{{"schtasks", "/Change", "/TN", tareaWindows, "/DISABLE"}}
}

// desdeUTF16 lee lo que escribió utf16ConBOM.
func desdeUTF16(b []byte) string {
	u := make([]uint16, 0, len(b)/2)
	for i := 2; i+1 < len(b); i += 2 {
		u = append(u, uint16(b[i])|uint16(b[i+1])<<8)
	}
	return string(utf16.Decode(u))
}

// ErrServicioDeOtroHome: el servicio instalado publica otro QDD_HOME; quien lo quiera quitar debe correr con ese.
var ErrServicioDeOtroHome = errors.New("el servicio de este PC publica otro QDD_HOME")

// Quitar apaga el servicio y borra su archivo, si es de este QDD_HOME (FND-136). Lo que ya no estaba no es un error.
func (s Servicio) Quitar(correr Ejecutor) error {
	archivo, err := s.Archivo()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(archivo)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !s.esDeEsteHome(string(b)) {
		return ErrServicioDeOtroHome
	}
	for _, c := range s.Comandos(false) {
		_ = correr(c[0], c[1:]...)
	}
	return os.Remove(archivo)
}
