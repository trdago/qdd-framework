package remoto

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Lo que qdd doctor necesita para diagnosticar la comunicación con el teléfono sin usar la red: el servicio deja en
// <QDD_HOME>/remoto si el canal en tiempo real está unido y si el relay dejó de reconocer este PC; lo demás se lee de los
// archivos que ya existen (credenciales, relay, servicio instalado y log.jsonl).

// conexionGuardada es <QDD_HOME>/remoto/conexion.json: si el canal en tiempo real está unido y desde cuándo. El latido
// va por HTTP y puede seguir respondiendo con el websocket caído, y sin canal el teléfono no recibe nada.
type conexionGuardada struct {
	Formato   int       `json:"formato"`
	Conectado bool      `json:"conectado"`
	Desde     time.Time `json:"desde"`
}

// GuardarConexion anota un cambio del canal (0600).
func GuardarConexion(conectado bool, now time.Time) error {
	return guardarPrivado("conexion.json", conexionGuardada{Formato: 1, Conectado: conectado, Desde: now.UTC()})
}

// LeerConexion es lo que dejó GuardarConexion; un servicio de una versión anterior no deja nada.
func LeerConexion() (bool, time.Time, error) {
	var c conexionGuardada
	if err := leerPrivado("conexion.json", &c); err != nil {
		return false, time.Time{}, err
	}
	return c.Conectado, c.Desde, nil
}

// quitadoGuardado es <QDD_HOME>/remoto/quitado.json: el relay respondió que no conoce este tópico (28000), o sea, el
// teléfono quitó este PC. Va con el tópico: emparejar de nuevo da otro y la marca deja de valer sola.
type quitadoGuardado struct {
	Formato int       `json:"formato"`
	Topico  string    `json:"topico"`
	En      time.Time `json:"en"`
}

// GuardarQuitado anota que el relay ya no reconoce el tópico.
func GuardarQuitado(topico string, now time.Time) error {
	return guardarPrivado("quitado.json", quitadoGuardado{Formato: 1, Topico: topico, En: now.UTC()})
}

// QuitadoEn dice cuándo el relay dejó de reconocer este tópico; false si no pasó o si la marca es de otro emparejamiento.
func QuitadoEn(topico string) (time.Time, bool) {
	var q quitadoGuardado
	if err := leerPrivado("quitado.json", &q); err != nil {
		return time.Time{}, false
	}
	return q.En, q.Topico != "" && q.Topico == topico
}

// CredencialesIlegibles: credenciales.json existe y no se lee. Para el resto de qdd eso es «sin emparejar», y doctor
// diría «empareja» cuando lo que pasó es que el archivo se rompió.
func CredencialesIlegibles() bool {
	var c Credenciales
	err := leerPrivado("credenciales.json", &c)
	return err != nil && !errors.Is(err, os.ErrNotExist)
}

// ApartarCredenciales deja el archivo roto como credenciales.json.ilegible (no se pierde) y devuelve dónde quedó.
func ApartarCredenciales() (string, error) {
	ruta, err := rutaRemoto("credenciales.json")
	if err != nil {
		return "", err
	}
	return ruta + ".ilegible", os.Rename(ruta, ruta+".ilegible")
}

// RelayNoSeLee dice por qué relay.json no sirve («» si sirve): sin él, qdd remoto correr sale con error al arrancar.
func RelayNoSeLee() string {
	_, err := LeerRelay()
	if err == nil {
		return ""
	}
	if errors.Is(err, os.ErrNotExist) {
		return "falta relay.json"
	}
	return "relay.json no se lee"
}

// RecuperarRelay pide el relay al directorio firmado cuando relay.json falta o se rompió; uno roto se aparta antes.
func RecuperarRelay(consultar func(ConfigRelay) (ConfigRelay, bool, error)) (ConfigRelay, error) {
	ruta, err := rutaRemoto("relay.json")
	if err != nil {
		return ConfigRelay{}, err
	}
	if _, err := os.Stat(ruta); err == nil {
		if err := os.Rename(ruta, ruta+".ilegible"); err != nil {
			return ConfigRelay{}, err
		}
	}
	c, _, err := consultar(ConfigRelay{})
	return c, err
}

var (
	reExecStart = regexp.MustCompile(`(?m)^ExecStart=("(?:[^"\\]|\\.)*"|\S+)`)
	reArgsMac   = regexp.MustCompile(`<key>ProgramArguments</key>\s*<array>\s*<string>([^<]*)</string>`)
	reCmdWin    = regexp.MustCompile(`<Command>([^<]*)</Command>`)
)

// BinarioInstalado es el qdd que lanza el servicio instalado («» si no hay servicio o no se reconoce).
func (s Servicio) BinarioInstalado() string {
	archivo, err := s.Archivo()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(archivo)
	if err != nil {
		return ""
	}
	return binarioDe(s.SO, b)
}

func binarioDe(so string, b []byte) string {
	if so == "linux" {
		m := reExecStart.FindStringSubmatch(string(b))
		if m == nil {
			return ""
		}
		return sinComillasSystemd(m[1])
	}
	if so == "darwin" {
		m := reArgsMac.FindStringSubmatch(string(b))
		if m == nil {
			return ""
		}
		return desescaparXML(m[1])
	}
	m := reCmdWin.FindStringSubmatch(desdeUTF16(b))
	if m == nil {
		return ""
	}
	return desescaparXML(m[1])
}

func sinComillasSystemd(v string) string {
	if len(v) < 2 || !strings.HasPrefix(v, `"`) || !strings.HasSuffix(v, `"`) {
		return v
	}
	return strings.NewReplacer(`\\`, `\`, `\"`, `"`, "%%", "%").Replace(v[1 : len(v)-1])
}

func desescaparXML(v string) string {
	return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'", "&amp;", "&").Replace(v)
}

// BinarioPerdido: el servicio lanza un qdd que ya no está, como el de una versión de node que se desinstaló. Así no
// arranca nunca, y reiniciarlo no sirve: hay que volver a escribirlo con el qdd de ahora.
func (s Servicio) BinarioPerdido() (string, bool) {
	bin := s.BinarioInstalado()
	if bin == "" {
		return "", false
	}
	_, err := os.Stat(bin)
	return bin, errors.Is(err, os.ErrNotExist)
}

// ventanaDesfase son las acciones descartadas que cuentan para el reloj: una vieja de hace días no dice nada de hoy.
const ventanaDesfase = 24 * time.Hour

var reAccionVieja = regexp.MustCompile(`vieja \((\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ)\)`)

// DesfaseReloj lee log.jsonl: una acción descartada por vieja trae la hora que le puso el relay al recibirla (now() de
// Postgres; el teléfono no encola, reintenta unos 20 s y se rinde, según qdd-movile), y la línea la del PC. Devuelve la
// diferencia de la más reciente de las últimas 24 h (PC menos relay, más lo que tardó el canal) y cuántas hubo.
func DesfaseReloj(now time.Time) (time.Duration, int) {
	ruta, err := rutaRemoto("log.jsonl")
	if err != nil {
		return 0, 0
	}
	f, err := os.Open(ruta)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	var ultima time.Time
	var desfase time.Duration
	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var e EntradaLog
		if json.Unmarshal(sc.Bytes(), &e) != nil || e.Estado != EstadoDescartada || now.Sub(e.En) > ventanaDesfase {
			continue
		}
		m := reAccionVieja.FindStringSubmatch(e.Resultado)
		if m == nil {
			continue
		}
		relay, err := time.Parse(formatoEn, m[1])
		if err != nil {
			continue
		}
		n++
		if e.En.After(ultima) {
			ultima, desfase = e.En, e.En.Sub(relay)
		}
	}
	return desfase, n
}

// BinarioPerdidoDeEsteHome es BinarioPerdido solo para el servicio de este QDD_HOME: el de otro lo arregla quien corre
// con ese, y reescribirlo desde aquí le cambiaría qué publica.
func (s Servicio) BinarioPerdidoDeEsteHome() (string, bool) {
	bin, perdido := s.BinarioPerdido()
	if !perdido {
		return "", false
	}
	archivo, _ := s.Archivo()
	b, err := os.ReadFile(archivo)
	if err != nil || !s.esDeEsteHome(string(b)) {
		return "", false
	}
	return bin, true
}

// SolicitudesApartadas es solicitudes.json.ilegible si leerSolicitudes lo apartó: las acciones que ahí estaban podrían
// repetirse si el teléfono las reenvía.
func SolicitudesApartadas() string {
	ruta, err := rutaRemoto("solicitudes.json.ilegible")
	if err != nil {
		return ""
	}
	if _, err := os.Stat(ruta); err != nil {
		return ""
	}
	return ruta
}
