package remoto

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): qdd remoto activar deja «qdd remoto correr» como servicio del usuario, que arranca con la sesión y
// se reinicia si se cae: systemd --user en Linux, launchd en macOS y una tarea programada al iniciar sesión en Windows.
// El servicio lleva el PATH de quien lo activó: los pedidos corren con la shell y necesitan las mismas herramientas
// (git, gh, npm) que en la terminal.

const (
	nombreUnidad = "qdd-remoto.service"
	etiquetaMac  = "dev.qdd.remoto"
	tareaWindows = "QDD remoto"
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
  <Triggers><LogonTrigger><Enabled>true</Enabled></LogonTrigger></Triggers>
  <Principals><Principal id="Author"><LogonType>InteractiveToken</LogonType><RunLevel>LeastPrivilege</RunLevel></Principal></Principals>
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
