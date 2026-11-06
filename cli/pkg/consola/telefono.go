package consola

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/candado"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Ejecutar desde el teléfono» (solo Pro, decisión de la persona, 2026-11-03): lo que el teléfono manda a
// ejecutar en un proyecto queda en una cola local por repositorio (<QDD_HOME>/claude/telefono/, RutaLocal, como los
// pedidos), una sola vez por solicitud. Un comando qdd lo corre qdd remoto (sin shell) y anota aquí cómo terminó; una
// instrucción para la IA espera a que una sesión viva de Claude Code del proyecto la tome (el mod qdd-tareas, con
// TomarInstrucciones) y se la entregue como un mensaje de la persona que dice que viene del teléfono. El panel lo
// muestra como un pedido de origen telefono, sin tecla, solo a quien lo pide (--con telefono: qdd remoto); nunca es
// una acción de qdd consola accion. Las sesiones vivas (LatirSesion) dicen si el proyecto acepta instrucciones.

const (
	// OrigenTelefono es el origen de un pedido que mandó un teléfono.
	OrigenTelefono = "telefono"
	// ConTelefono es lo que pide qdd consola panel --con para recibir lo mandado desde el teléfono.
	ConTelefono = "telefono"

	// EjecutarQDD y EjecutarIA son los tipos de esta cola: un comando qdd o una instrucción para la IA. EjecutarFlujo no
	// entra en ella: una edición de un flujo se aplica al recibirla y queda en el registro del flujo.
	EjecutarQDD   = "qdd"
	EjecutarIA    = "ia"
	EjecutarFlujo = "flujo"

	// EstadoVencida: una instrucción que ninguna sesión tomó a tiempo. EstadoRetirada: la retiró el teléfono antes.
	EstadoVencida  = "vencida"
	EstadoRetirada = "retirada"

	// VidaInstruccion es lo más que una instrucción espera una sesión de Claude Code abierta en el proyecto.
	VidaInstruccion = 30 * time.Minute
	// TopeEntrega es cuánto espera una instrucción tomada a que su sesión confirme que Claude Code la aceptó (FND-138).
	TopeEntrega = 15 * time.Minute
	// TopeComandoTelefono es lo más que corre un comando qdd mandado desde el teléfono (qdd remoto lo corta).
	TopeComandoTelefono = 10 * time.Minute
	// VidaSesion: una sesión que no latió (qdd consola telefono latir) en este tiempo ya no cuenta como abierta.
	VidaSesion = 30 * time.Second
	// MaxEjecucionesCola es el tope de la cola (las más nuevas).
	MaxEjecucionesCola = 10
	// maxDetalleTelefono es lo que se muestra en el panel de una instrucción (el teléfono tiene el texto entero).
	maxDetalleTelefono = 500

	prefijoTelefono = "telefono:"
)

// ErrSinSesion: tomar o cerrar sin decir qué sesión de Claude Code es.
var ErrSinSesion = errors.New("falta la sesión de Claude Code")

// Ejecucion es lo que un teléfono mandó a ejecutar en el proyecto.
type Ejecucion struct {
	Solicitud   string     `json:"solicitud"`
	Tipo        string     `json:"tipo"`
	Texto       string     `json:"texto"`
	Dispositivo string     `json:"dispositivo"`
	Nombre      string     `json:"nombre,omitempty"`
	Estado      string     `json:"estado"`
	Codigo      int        `json:"codigo,omitempty"`
	Salida      string     `json:"salida,omitempty"` // el final de la salida, ya sin secretos (lo pone qdd remoto)
	Motivo      string     `json:"motivo,omitempty"`
	Sesion      string     `json:"sesion,omitempty"` // la sesión de Claude Code que recibió la instrucción
	Creada      time.Time  `json:"creada"`
	Inicio      *time.Time `json:"inicio,omitempty"`
	Fin         *time.Time `json:"fin,omitempty"`
}

func (e Ejecucion) abierta() bool {
	return e.Estado == EstadoPendiente || e.Estado == EstadoEjecutando
}

// RegistrarEjecucion deja lo que mandó el teléfono en la cola: un comando qdd queda ejecutando, una instrucción
// pendiente de una sesión. Una solicitud que ya está devuelve lo que hay, sin cambiarlo.
func RegistrarEjecucion(root string, e Ejecucion, now time.Time) (Ejecucion, error) {
	if e.Tipo != EjecutarQDD && e.Tipo != EjecutarIA {
		return Ejecucion{}, fmt.Errorf("tipo «%s» desconocido: solo qdd o ia", e.Tipo)
	}
	if strings.TrimSpace(e.Solicitud) == "" || strings.TrimSpace(e.Texto) == "" {
		return Ejecucion{}, errors.New("falta la solicitud o el texto")
	}
	inicio := now.UTC()
	e.Creada, e.Estado, e.Inicio, e.Fin = inicio, EstadoPendiente, nil, nil
	if e.Tipo == EjecutarQDD {
		e.Estado, e.Inicio = EstadoEjecutando, &inicio
	}
	out := e
	err := conTelefono(root, now, func(cola []Ejecucion) ([]Ejecucion, error) {
		if i := indiceEjecucion(cola, e.Solicitud); i >= 0 {
			out = cola[i]
			return cola, nil
		}
		return append([]Ejecucion{e}, cola...), nil
	})
	return out, err
}

// TerminarEjecucion anota cómo terminó un comando qdd: hecho si salió con 0, con falla si no.
func TerminarEjecucion(root, solicitud string, codigo int, salida string, now time.Time) error {
	return cambiarEjecucion(root, solicitud, now, func(e *Ejecucion) error {
		if e.Estado != EstadoEjecutando {
			return fmt.Errorf("la ejecución %s ya terminó (%s)", solicitud, e.Estado)
		}
		fin := now.UTC()
		e.Estado, e.Codigo, e.Salida, e.Fin = EstadoHecho, codigo, salida, &fin
		if codigo != 0 {
			e.Estado = EstadoFallo
		}
		return nil
	})
}

// TomarInstrucciones es la toma de un panel anterior a FND-138, que no confirma: las da por entregadas al tomarlas.
func TomarInstrucciones(root, sesion string, now time.Time) ([]Ejecucion, error) {
	return tomar(root, sesion, now, EstadoHecho)
}

// TomarParaEntregar da a la sesión las instrucciones que esperan: cada una a una sola sesión, una sola vez. Quedan
// ejecutando hasta que la sesión confirme la entrega (ConfirmarEntrega) o diga que falló (FallarEntrega).
func TomarParaEntregar(root, sesion string, now time.Time) ([]Ejecucion, error) {
	return tomar(root, sesion, now, EstadoEjecutando)
}

func tomar(root, sesion string, now time.Time, estado string) ([]Ejecucion, error) {
	sesion = sesionValida(sesion)
	if sesion == "" {
		return nil, ErrSinSesion
	}
	var tomadas []Ejecucion
	err := conTelefono(root, now, func(cola []Ejecucion) ([]Ejecucion, error) {
		for i, e := range cola {
			if e.Tipo != EjecutarIA || e.Estado != EstadoPendiente {
				continue
			}
			inicio := now.UTC()
			cola[i].Estado, cola[i].Sesion, cola[i].Inicio = estado, sesion, &inicio
			if estado == EstadoHecho {
				cola[i].Fin = &inicio
			}
			tomadas = append(tomadas, cola[i])
		}
		return cola, nil
	})
	sort.Slice(tomadas, func(i, j int) bool { return tomadas[i].Creada.Before(tomadas[j].Creada) })
	return tomadas, err
}

// ConfirmarEntrega: Claude Code aceptó la instrucción que esta sesión tomó.
func ConfirmarEntrega(root, solicitud, sesion string, now time.Time) error {
	sesion = sesionValida(sesion)
	return cambiarEjecucion(root, solicitud, now, func(e *Ejecucion) error {
		if sesion == "" || e.Sesion != sesion || e.Estado != EstadoEjecutando {
			return fmt.Errorf("la instrucción %s no espera confirmación de esta sesión", solicitud)
		}
		fin := now.UTC()
		e.Estado, e.Fin = EstadoHecho, &fin
		return nil
	})
}

// FallarEntrega: la sesión que tomó la instrucción no pudo dársela a la IA.
func FallarEntrega(root, solicitud, sesion, motivo string, now time.Time) error {
	sesion = sesionValida(sesion)
	return cambiarEjecucion(root, solicitud, now, func(e *Ejecucion) error {
		if sesion == "" || e.Sesion != sesion || (e.Estado != EstadoHecho && e.Estado != EstadoEjecutando) {
			return fmt.Errorf("la instrucción %s no la tomó esta sesión", solicitud)
		}
		fin := now.UTC()
		e.Estado, e.Motivo, e.Fin = EstadoFallo, strings.TrimSpace(motivo), &fin
		return nil
	})
}

// RetirarInstruccion saca una instrucción que todavía espera una sesión (el «Rechazar» del teléfono).
func RetirarInstruccion(root, solicitud, motivo string, now time.Time) (Ejecucion, error) {
	var out Ejecucion
	err := cambiarEjecucion(root, solicitud, now, func(e *Ejecucion) error {
		if e.Tipo != EjecutarIA || e.Estado != EstadoPendiente {
			return fmt.Errorf("ya no espera: %s", e.Estado)
		}
		fin := now.UTC()
		e.Estado, e.Motivo, e.Fin = EstadoRetirada, strings.TrimSpace(motivo), &fin
		out = *e
		return nil
	})
	return out, err
}

func cambiarEjecucion(root, solicitud string, now time.Time, cambiar func(*Ejecucion) error) error {
	return conTelefono(root, now, func(cola []Ejecucion) ([]Ejecucion, error) {
		i := indiceEjecucion(cola, solicitud)
		if i < 0 {
			return nil, fmt.Errorf("no hay una ejecución %s en este PC", solicitud)
		}
		if err := cambiar(&cola[i]); err != nil {
			return nil, err
		}
		return cola, nil
	})
}

// EjecucionDe lee una ejecución a la hora now, sin escribir.
func EjecucionDe(root, solicitud string, now time.Time) (Ejecucion, bool) {
	cola := EjecucionesA(root, now)
	i := indiceEjecucion(cola, solicitud)
	if i < 0 {
		return Ejecucion{}, false
	}
	return cola[i], true
}

// EjecucionesA es la cola a la hora now, la más nueva primero.
func EjecucionesA(root string, now time.Time) []Ejecucion {
	return normalizarTelefono(leerTelefono(root, false), now)
}

// MensajeInstruccion es lo que la IA recibe: dice que viene del teléfono de la persona y cuál, y trae el texto tal
// cual. Empieza con 📱, así que Claude Code nunca lo lee como un comando de shell («!») ni uno suyo («/»).
func MensajeInstruccion(e Ejecucion) string {
	nombre := strings.TrimSpace(e.Nombre)
	if nombre == "" {
		nombre = "su teléfono"
	}
	return "📱 La persona te escribe desde «" + nombre + "» (QDD Móvil). Atiéndelo como un mensaje suyo:\n\n" + e.Texto
}

func indiceEjecucion(cola []Ejecucion, solicitud string) int {
	for i, e := range cola {
		if e.Solicitud == solicitud {
			return i
		}
	}
	return -1
}

// normalizarTelefono vence las instrucciones que ninguna sesión tomó en VidaInstruccion, da por cortado un comando
// que lleva más que su tope (qdd remoto se cayó a mitad), quita lo cerrado hace más de VidaHecho y deja las
// MaxEjecucionesCola más nuevas.
func normalizarTelefono(cola []Ejecucion, now time.Time) []Ejecucion {
	out := make([]Ejecucion, 0, len(cola))
	for _, e := range cola {
		if e.Tipo == EjecutarIA && e.Estado == EstadoPendiente && now.Sub(e.Creada) > VidaInstruccion {
			fin := e.Creada.Add(VidaInstruccion)
			e.Estado, e.Fin, e.Motivo = EstadoVencida, &fin, "ninguna sesión de Claude Code se abrió en el proyecto en "+VidaInstruccion.String()
		}
		if e.Tipo == EjecutarIA && e.Estado == EstadoEjecutando && e.Inicio != nil && now.Sub(*e.Inicio) > TopeEntrega {
			fin := e.Inicio.Add(TopeEntrega)
			e.Estado, e.Fin, e.Motivo = EstadoFallo, &fin, "la sesión la tomó y no confirmó que Claude Code la aceptara: puede no haber llegado"
		}
		if e.Tipo != EjecutarIA && e.Estado == EstadoEjecutando && now.Sub(e.Creada) > TopeComandoTelefono+time.Minute {
			fin := e.Creada.Add(TopeComandoTelefono)
			e.Estado, e.Fin, e.Motivo = EstadoFallo, &fin, "se cortó sin terminar (qdd remoto se cerró mientras corría)"
		}
		if !e.abierta() && e.Fin != nil && now.Sub(*e.Fin) > VidaHecho {
			continue
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Creada.After(out[j].Creada) })
	if len(out) > MaxEjecucionesCola {
		return out[:MaxEjecucionesCola]
	}
	return out
}

func rutaTelefono(root string) (string, error) {
	return RutaLocal(root, "telefono")
}

// ColasApartadas son las colas del teléfono de este proyecto que se apartaron por ilegibles (qdd doctor, REM-10).
func ColasApartadas(root string) []string {
	out := []string{}
	for _, nombre := range []string{"preguntas", "telefono", "turnos"} {
		ruta, err := RutaLocal(root, nombre)
		if err != nil {
			continue
		}
		if _, err := os.Stat(ruta + ".ilegible"); err == nil {
			out = append(out, ruta+".ilegible")
		}
	}
	return out
}

// leerTelefono lee la cola; una ilegible se aparta como .ilegible (no se pierde) si apartar, y se empieza de cero.
func leerTelefono(root string, apartar bool) []Ejecucion {
	var cola []Ejecucion
	ruta, err := rutaTelefono(root)
	if err != nil {
		return nil
	}
	if !leerArchivoJSON(ruta, &cola) && apartar {
		_ = os.Rename(ruta, ruta+".ilegible")
	}
	return cola
}

// conTelefono lee, cambia y guarda la cola con el archivo bloqueado (pkg/candado): qdd remoto que registra y termina,
// y el mod que toma las instrucciones, nunca se pisan, y una instrucción no se entrega dos veces.
func conTelefono(root string, now time.Time, cambiar func([]Ejecucion) ([]Ejecucion, error)) error {
	ruta, err := rutaTelefono(root)
	if err != nil {
		return err
	}
	return conArchivo(ruta, "la cola del teléfono", func() error {
		cola, err := cambiar(normalizarTelefono(leerTelefono(root, true), now))
		if err != nil {
			return err
		}
		return guardarJSON(ruta, cola)
	})
}

// accionesDelTelefono muestra cada ejecución como un pedido de origen telefono, sin tecla (no se confirma en el PC):
// un comando qdd ejecutando, hecho o con falla; una instrucción armada mientras espera una sesión, hecha al
// entregarse, y con falla si venció, se retiró o no se pudo entregar.
func accionesDelTelefono(root string, now time.Time) []Accion {
	var out []Accion
	for _, e := range EjecucionesA(root, now) {
		out = append(out, accionDelTelefono(root, e))
	}
	return out
}

func accionDelTelefono(root string, e Ejecucion) Accion {
	a := Accion{
		ID: prefijoTelefono + e.Solicitud, Tipo: TipoPedido, Etiqueta: etiquetaTelefono(e), Detalle: "qdd " + e.Texto,
		Carpeta: root, Estado: e.Estado, Resultado: ultimasLineas(e.Salida, lineasSalida), Inicio: e.Inicio, Fin: e.Fin,
		Origen: OrigenTelefono, Dispositivo: e.Nombre, Solicitud: e.Solicitud,
	}
	if e.Motivo != "" {
		a.Resultado = e.Motivo
	}
	if e.Tipo == EjecutarQDD {
		return a
	}
	a.Detalle = cortar(e.Texto, maxDetalleTelefono)
	if e.Estado == EstadoPendiente {
		a.Estado, a.Aviso = EstadoArmado, "espera una sesión de Claude Code abierta en el proyecto"
	}
	if e.Estado == EstadoEjecutando {
		a.Aviso = "entregándola a la sesión de Claude Code"
	}
	if e.Estado == EstadoHecho {
		a.Resultado = "entregada a la sesión de Claude Code"
	}
	if e.Estado == EstadoVencida || e.Estado == EstadoRetirada {
		a.Estado = EstadoFallo
	}
	return a
}

// etiquetaTelefono es «qdd <comando>» o la primera línea de la instrucción, hasta MaxMotivo.
func etiquetaTelefono(e Ejecucion) string {
	if e.Tipo == EjecutarQDD {
		return cortar("qdd "+e.Texto, MaxMotivo)
	}
	primera, _, _ := strings.Cut(strings.TrimSpace(e.Texto), "\n")
	return cortar(strings.TrimSpace(primera), MaxMotivo)
}

func cortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// SesionViva es una sesión interactiva de Claude Code abierta en el proyecto y su último latido.
type SesionViva struct {
	Sesion string    `json:"sesion"`
	Vista  time.Time `json:"vista"`
}

func rutaSesiones(root string) (string, error) {
	return RutaLocal(root, "sesiones")
}

// LatirSesion anota que la sesión sigue abierta en el proyecto (el mod, cada pocos segundos).
func LatirSesion(root, sesion string, now time.Time) error {
	return conSesiones(root, sesion, now, func(vivas []SesionViva, s string) []SesionViva {
		return append(vivas, SesionViva{Sesion: s, Vista: now.UTC()})
	})
}

// CerrarSesion la quita en el acto (el mod, al terminar la sesión).
func CerrarSesion(root, sesion string, now time.Time) error {
	return conSesiones(root, sesion, now, func(vivas []SesionViva, _ string) []SesionViva { return vivas })
}

// HaySesion dice si alguna sesión de Claude Code latió en el proyecto en los últimos VidaSesion.
func HaySesion(root string, now time.Time) bool {
	ruta, err := rutaSesiones(root)
	if err != nil {
		return false
	}
	var vivas []SesionViva
	leerArchivoJSON(ruta, &vivas)
	return len(sesionesVivas(vivas, "", now)) > 0
}

// sesionesVivas son las que latieron hace menos de VidaSesion, sin sacar.
func sesionesVivas(vivas []SesionViva, sacar string, now time.Time) []SesionViva {
	var out []SesionViva
	for _, v := range vivas {
		if v.Sesion == sacar || now.Sub(v.Vista) > VidaSesion || v.Vista.Sub(now) > VidaSesion {
			continue
		}
		out = append(out, v)
	}
	return out
}

func conSesiones(root, sesion string, now time.Time, cambiar func([]SesionViva, string) []SesionViva) error {
	sesion = sesionValida(sesion)
	if sesion == "" {
		return ErrSinSesion
	}
	ruta, err := rutaSesiones(root)
	if err != nil {
		return err
	}
	return conArchivo(ruta, "las sesiones abiertas", func() error {
		var vivas []SesionViva
		leerArchivoJSON(ruta, &vivas)
		return guardarJSON(ruta, cambiar(sesionesVivas(vivas, sesion, now), sesion))
	})
}

// leerArchivoJSON lee ruta en v; dice false si el archivo existe pero no se entiende.
func leerArchivoJSON(ruta string, v any) bool {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return true
	}
	return json.Unmarshal(b, v) == nil
}

// conArchivo corre cambiar con ruta bloqueada (pkg/candado), después de crear su carpeta.
func conArchivo(ruta, que string, cambiar func() error) error {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	soltar, err := candado.Tomar(ruta+".lock", 2*time.Second, que)
	if err != nil {
		return err
	}
	defer soltar()
	return cambiar()
}

// guardarJSON escribe v en ruta de forma atómica (tmp + rename), 0600.
func guardarJSON(ruta string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return escribirSiCambio(ruta, b)
}

// escribirSiCambio no toca un archivo que ya dice lo mismo: quien vigila su carpeta (qdd remoto y el flujo de cada panel)
// vería un cambio que no hubo y volvería a leer todos los proyectos (FND-161).
func escribirSiCambio(ruta string, b []byte) error {
	if actual, err := os.ReadFile(ruta); err == nil && bytes.Equal(actual, b) {
		return nil
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}
