package consola

import (
	"errors"
	"fmt"
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
	Inicio    *time.Time `json:"inicio,omitempty"`
	Fin       *time.Time `json:"fin,omitempty"`
	Confirmar bool       `json:"confirmar"`
	Args      []string   `json:"-"`
	ficha     string
	pedido    string
	tarea     string // id del frente que descarta la ✕ (§3.9)
	programa  string // abridor de «Abrir carpeta» (§3.9)
}

// DatosConsola es lo que 'qdd consola panel --json' entrega al panel en una sola lectura.
type DatosConsola struct {
	Formato  int                `json:"formato"`
	Version  string             `json:"version,omitempty"` // versión de qdd, que el panel muestra atenuada (§3.9)
	Carpeta  string             `json:"carpeta"`           // raíz del proyecto, junto al botón «Abrir carpeta» (§3.9)
	Estado   *estado.DatosPanel `json:"estado"`
	Motivo   string             `json:"motivo,omitempty"`
	Acciones []Accion           `json:"acciones"`
}

// Panel arma los datos del panel: el estado por frente (o el motivo por el que no hay) y las acciones de ahora.
func Panel(root string, now time.Time) DatosConsola {
	d := DatosConsola{Formato: 1, Carpeta: root}
	e, err := estado.Load(root)
	if errors.Is(err, os.ErrNotExist) {
		d.Motivo = estado.ErrSinArchivo.Error()
		d.Acciones = Acciones(root, false)
		return d
	}
	if err != nil {
		d.Motivo = err.Error()
		d.Acciones = Acciones(root, false)
		return d
	}
	datos := estado.DatosEnCurso(e, estado.LeerActividad(root), now, MaxTareas)
	d.Estado = &datos
	d.Acciones = Acciones(root, true)
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
