package consola

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qdd-framework/qdd/pkg/materializar"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.7: las acciones del panel las decide qdd. 'qdd consola panel --json' las ofrece y 'qdd consola accion <id>'
// las vuelve a calcular antes de ejecutar una; el panel solo pasa ids (blueprint consola-claude-code v3.0.0).

const (
	TipoAprobar   = "aprobar"
	TipoComando   = "comando"
	TipoDescartar = "descartar"

	// MaxTareas es cuántas tareas pendientes muestra el panel: las más nuevas (§3.9).
	MaxTareas = 4

	// MotivoDescarte es el motivo_cierre de una tarea descartada con la ✕ del panel: nunca desaparece en silencio.
	MotivoDescarte = "descartada desde el panel de Claude Code (✕)"

	// MaxFichas es el tope de fichas en borrador que se ofrecen: una tecla 1…9 por ficha.
	MaxFichas = 9

	prefijoAprobar   = "aprobar:"
	prefijoDescartar = "descartar:"
)

// Accion es un botón del panel. Args son los argumentos de qdd para un comando; no salen en el JSON. En un pedido,
// Detalle es el comando exacto y Carpeta dónde corre.
type Accion struct {
	ID        string     `json:"id"`
	Tipo      string     `json:"tipo"`
	Etiqueta  string     `json:"etiqueta"`
	Detalle   string     `json:"detalle"`
	Carpeta   string     `json:"carpeta,omitempty"`
	Tecla     string     `json:"tecla,omitempty"`
	Estado    string     `json:"estado,omitempty"`    // solo pedidos: pendiente, espera, ejecutando, hecho o fallo
	Resultado string     `json:"resultado,omitempty"` // solo pedidos: últimas líneas de la salida
	Espera    []string   `json:"espera,omitempty"`    // solo pedidos en espera: las teclas de lo que espera
	Aviso     string     `json:"aviso,omitempty"`     // solo pedidos: «todavía no», lo que dijo la última vez (FND-082)
	AvisoEn   *time.Time `json:"aviso_en,omitempty"`
	Proximo   *time.Time `json:"proximo,omitempty"`  // solo pedidos armados: cuándo vuelve a intentarlo
	Intentos  int        `json:"intentos,omitempty"` // solo pedidos armados: cuántas veces corrió
	Sesion    string     `json:"sesion,omitempty"`   // solo pedidos: la sesión de Claude Code que lo confirmó
	Avisar    bool       `json:"avisar,omitempty"`   // solo pedidos: terminó y esa sesión todavía no recibió el aviso
	Mensaje   string     `json:"mensaje,omitempty"`  // solo con avisar: lo que lee la IA de esa sesión (aviso.go)
	PR        int        `json:"pr,omitempty"`       // solo pedidos opcionales de un PR abierto (prs.go)
	// VersionNueva: solo el pedido opcional de actualizar qdd (version_nueva.go).
	VersionNueva string `json:"version_nueva,omitempty"`
	DeTarea      string `json:"tarea,omitempty"` // solo pedidos: la tarea que extiende, para que el teléfono no la ofrezca otra vez
	// Solo preguntas (preguntas.go): de dónde viene (ia, permiso o qdd), dónde se respondió (pc o telefono), sus
	// preguntas con opciones y, en un permiso, la herramienta y el comando exacto.
	Origen       string           `json:"origen,omitempty"`
	RespondidaEn string           `json:"respondida_en,omitempty"`
	Preguntas    []ItemPregunta   `json:"preguntas,omitempty"`
	Permiso      *PermisoPregunta `json:"permiso,omitempty"`
	// Solo pedidos de origen telefono (telefono.go): el nombre del teléfono que lo mandó y la solicitud con que llegó.
	Dispositivo string     `json:"dispositivo,omitempty"`
	Solicitud   string     `json:"solicitud,omitempty"`
	Inicio      *time.Time `json:"inicio,omitempty"`
	Fin         *time.Time `json:"fin,omitempty"`
	Confirmar   bool       `json:"confirmar"`
	Args        []string   `json:"-"`
	ficha       string
	pedido      string
	tarea       string // id del frente que descarta la ✕ (§3.9)
	programa    string // abridor de «Abrir carpeta» (§3.9)
}

// DatosConsola es lo que 'qdd consola panel --json' entrega al panel en una sola lectura.
type DatosConsola struct {
	Formato  int                   `json:"formato"`
	Version  string                `json:"version,omitempty"` // versión de qdd, que el panel muestra atenuada (§3.9)
	Carpeta  string                `json:"carpeta"`           // raíz del proyecto, junto al botón «Abrir carpeta» (§3.9)
	Estado   *estado.DatosPanel    `json:"estado"`
	Motivo   string                `json:"motivo,omitempty"`
	Acciones []Accion              `json:"acciones"`
	Efectos  *materializar.Efectos `json:"efectos,omitempty"` // si el panel materializa lo que cambia, y con qué estilo
	// Ejecutar dice qué acepta este PC mandar a ejecutar desde el teléfono (solo con QDD Móvil activado y Pro; lo pone
	// pkg/remoto, que sabe el plan). Sin él, el teléfono no ofrece ejecutar.
	Ejecutar *OpcionesEjecutar `json:"ejecutar,omitempty"`
	// Goal es el modo goal del proyecto (goal.go): con Activo false, este qdd lo ofrece y está apagado.
	Goal *Goal `json:"goal,omitempty"`
	// AvisoVersion: hay una versión nueva de qdd y aquí no se puede actualizar con un botón (version_nueva.go).
	AvisoVersion string `json:"aviso_version,omitempty"`
	// Herramientas (con «herramientas») es la lista de «herramienta listar»; vacía sale como []: así el panel sabe que
	// este qdd la trae y no lanza un proceso para leerla (MEJ-020).
	Herramientas json.RawMessage `json:"herramientas,omitempty"`
	// MejorasToca (con «mejoras»): si pasaron los 10 min del ciclo de mejoras. Sin el campo, el panel lo pide cada minuto.
	MejorasToca *bool `json:"mejoras_toca,omitempty"`
}

// OpcionesEjecutar es lo que el PC acepta desde el teléfono: comandos qdd (Sugeridos son los que permite, sin el
// «qdd») e instrucciones para la IA (IA: hay una sesión de Claude Code abierta en el proyecto que las recibe).
type OpcionesEjecutar struct {
	QDD       bool     `json:"qdd"`
	IA        bool     `json:"ia"`
	Sugeridos []string `json:"sugeridos"`
}

// opcionales son los tipos de acción que un panel recibe solo si dice que los dibuja (FND-092): un mod anterior no los
// conoce y los mostraría como botones sin tecla.
// Una pregunta (preguntas.go) se responde en el terminal o desde el teléfono: solo la pide qdd remoto. Lo mandado desde
// el teléfono (telefono.go) es un pedido de origen telefono y sale solo con --con telefono: el panel del PC lo
// mostraría como un pedido suyo.
var opcionales = map[string]bool{TipoAtender: true, TipoPregunta: true}

// Panel es lo que dibuja el panel. con son los tipos opcionales que ese panel sabe dibujar (qdd consola panel --con).
func Panel(root string, now time.Time, con ...string) DatosConsola {
	d := panel(root, now)
	pide := map[string]bool{}
	for _, t := range con {
		pide[t] = true
	}
	var acciones []Accion
	for _, a := range d.Acciones {
		if opcionales[a.Tipo] && !pide[a.Tipo] {
			continue
		}
		if a.Origen == OrigenTelefono && !pide[ConTelefono] {
			continue
		}
		acciones = append(acciones, a)
	}
	d.Acciones = acciones
	return d
}

// panel arma los datos del panel: el estado por frente (o el motivo por el que no hay) y las acciones de ahora.
// Las preguntas que esperan a la persona van primero, y después lo mandado desde el teléfono; ninguno de los dos es
// una acción de 'qdd consola accion' (se responden en el terminal o con ResponderPregunta; lo corre qdd remoto).
func panel(root string, now time.Time) DatosConsola {
	vista := VistaGoal(root, now)
	d := DatosConsola{Formato: 1, Carpeta: root, Goal: &vista}
	preguntas := append(accionesDePreguntas(root, now), accionesDelTelefono(root, now)...)
	e, err := estado.Load(root)
	if errors.Is(err, os.ErrNotExist) {
		d.Motivo = estado.ErrSinArchivo.Error()
		d.Acciones = append(preguntas, Acciones(root, false)...)
		return d
	}
	if err != nil {
		d.Motivo = err.Error()
		d.Acciones = append(preguntas, Acciones(root, false)...)
		return d
	}
	datos := estado.DatosEnCurso(e, estado.LeerActividad(root), now, MaxTareas)
	d.Estado = &datos
	d.Acciones = append(preguntas, Acciones(root, true)...)
	return d
}

// Acciones lista lo que el panel puede pedir: los comandos que la IA dejó para la persona (§3.9), aprobar cada ficha
// en borrador (hasta MaxFichas, por id), descartar cada tarea con su ✕ (§3.10), revisar el estado si hay estado.yaml,
// y auditar y abrir la carpeta (§3.10, si el sistema tiene abridor) si la carpeta es un proyecto QDD (tiene .qdd/).
func Acciones(root string, hayEstado bool) []Accion {
	acciones := append([]Accion{}, accionesDePedidos(root)...)
	for _, m := range borradores(root) {
		acciones = append(acciones, Accion{
			ID: prefijoAprobar + m.ID, Tipo: TipoAprobar, Etiqueta: "Aprobar «" + m.Nombre + "»",
			Detalle: "qdd modules approve " + m.ID, Confirmar: true, ficha: m.ID,
		})
	}
	acciones = append(acciones, descartes(root)...)
	acciones = append(acciones, atenciones(root)...)
	if hayEstado {
		acciones = append(acciones, Accion{
			ID: "revisar-estado", Tipo: TipoComando, Etiqueta: "Revisar estado", Detalle: "qdd estado check",
			Args: []string{"estado", "check"},
		})
	}
	if _, err := os.Stat(filepath.Join(root, ".qdd")); err != nil {
		return acciones
	}
	acciones = append(acciones, Accion{
		ID: "auditar", Tipo: TipoComando, Etiqueta: "Auditar", Detalle: "qdd audit", Confirmar: true,
		Args: []string{"audit"},
	})
	if hayEstado {
		// Cola de mejoras (pkg/mejoras): fuerza el ciclo sin esperar los 10 min; con tablero, porque entrega tareas.
		acciones = append(acciones, Accion{
			ID: "mejoras-ciclo", Tipo: TipoComando, Etiqueta: "Ciclo de mejoras", Detalle: "qdd mejoras ciclo",
			Args: []string{"mejoras", "ciclo"},
		})
		acciones = append(acciones, accionesDeGoal(root)...)
	}
	a, ok := accionAbrir(root)
	if !ok {
		return acciones
	}
	return append(acciones, a)
}

// descartes ofrece la ✕ de cada tarea que el panel muestra (§3.9): pasa a «descartado» con MotivoDescarte.
func descartes(root string) []Accion {
	e, err := estado.Load(root)
	if err != nil {
		return nil
	}
	var out []Accion
	for _, f := range estado.DatosPendientes(e, time.Now(), MaxTareas).Frentes {
		out = append(out, Accion{
			ID: prefijoDescartar + f.ID, Tipo: TipoDescartar, Etiqueta: "Descartar «" + f.Nombre + "»",
			Detalle: "qdd estado cerrar " + f.ID + " --descartado --motivo \"" + MotivoDescarte + "\"", Confirmar: true, tarea: f.ID,
		})
	}
	return out
}

// HayEstado dice si el proyecto tiene .qdd/project/estado.yaml.
func HayEstado(root string) bool {
	_, err := os.Stat(filepath.Join(root, estado.Ruta))
	return err == nil
}

func borradores(root string) []modules.Module {
	mods, _ := modules.Load(root)
	var out []modules.Module
	for _, m := range mods {
		if m.Estado == modules.EstadoBorrador {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > MaxFichas {
		return out[:MaxFichas]
	}
	return out
}

// BuscarAccion devuelve la acción con ese id; un id que qdd no ofrece ahora es un error.
func BuscarAccion(acciones []Accion, id string) (Accion, error) {
	for _, a := range acciones {
		if a.ID == id {
			return a, nil
		}
	}
	return Accion{}, fmt.Errorf("la acción '%s' no está entre las que ofrece 'qdd consola panel --json'", id)
}

// Autorizar exige el segundo paso de la persona para lo que escribe.
func Autorizar(a Accion, confirmado bool) error {
	if a.Confirmar && !confirmado {
		return fmt.Errorf("'%s' escribe en el proyecto: se ejecuta solo con --confirmado (el segundo paso del panel)", a.ID)
	}
	return nil
}

// Ejecutar corre la acción en la raíz del proyecto y devuelve su código de salida. Aprobar usa el mismo código que
// 'qdd modules approve'; un comando se ejecuta con bin (el binario de qdd en uso).
func (a Accion) Ejecutar(root, bin string, salida io.Writer) int {
	if a.Tipo == TipoAprobar {
		return aprobar(root, a.ficha, salida)
	}
	if a.Tipo == TipoPedido {
		return ejecutarPedido(root, a, salida)
	}
	if a.Tipo == TipoAbrir {
		return abrir(a, salida)
	}
	if a.Tipo == TipoDescartar {
		return descartar(root, a, salida)
	}
	if a.Tipo == TipoAtender {
		return atender(a, salida)
	}
	c := exec.Command(bin, a.Args...)
	c.Dir = root
	c.Stdout = salida
	c.Stderr = salida
	return codigoDe(c.Run(), a, salida)
}

func aprobar(root, id string, salida io.Writer) int {
	m, problemas := modules.Approve(root, id)
	if len(problemas) > 0 {
		fmt.Fprintf(salida, "[!] No se pudo aprobar '%s':\n  - %s\n", id, strings.Join(problemas, "\n  - "))
		return 1
	}
	fmt.Fprintln(salida, modules.TextoAprobada(m))
	return 0
}

func descartar(root string, a Accion, salida io.Writer) int {
	if err := estado.Cerrar(root, a.tarea, MotivoDescarte, true, time.Now()); err != nil {
		fmt.Fprintf(salida, "[!] No se pudo descartar '%s': %v\n", a.tarea, err)
		return 1
	}
	fmt.Fprintf(salida, "[+] Tarea '%s' descartada: sale del panel y queda en el archivo con su motivo.\n", a.tarea)
	return 0
}
