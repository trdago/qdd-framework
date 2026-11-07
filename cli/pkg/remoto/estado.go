// Package remoto es el lado PC de la consola remota de QDD (RFC 0013, QDD Móvil): un solo proceso por máquina publica
// el panel de todos sus proyectos por un canal de Supabase Realtime y ejecuta las acciones firmadas que llegan.
package remoto

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«qdd remoto estado»: lo que el PC publica, sin red. El mismo JSON es el contenido del evento estado.

// VersionProtocolo va en cada mensaje: un cliente que no la entiende lo dice y no adivina.
const VersionProtocolo = 1

// ErrCarpetaNoExiste es el error de un proyecto del registro cuya carpeta ya no está.
const ErrCarpetaNoExiste = "la carpeta ya no existe"

// Maquina es el PC que publica.
type Maquina struct {
	Nombre string `json:"nombre"`
	SO     string `json:"so"`
}

// ProyectoEstado es un proyecto publicado: su panel (el de qdd consola panel --json) o el error por el que no lo hay.
type ProyectoEstado struct {
	Clave  string                `json:"clave"`
	Nombre string                `json:"nombre"`
	Ruta   string                `json:"ruta"`
	Panel  *consola.DatosConsola `json:"panel,omitempty"`
	Error  string                `json:"error,omitempty"`
	// ReglasAtrasadas son los archivos con el bloque de reglas de un qdd anterior (integration.BloquesViejos; pedido de
	// qdd-movile, 2026-11-07). Sin el campo no se midió (la carpeta no está); [] es al día.
	ReglasAtrasadas *[]string `json:"reglas_atrasadas,omitempty"`
}

// Estado es lo que publica el PC: qdd remoto estado --json y el evento estado (con Seq).
type Estado struct {
	Version      int              `json:"version"`
	QDD          string           `json:"qdd"`
	Maquina      Maquina          `json:"maquina"`
	En           time.Time        `json:"en"`
	Seq          uint64           `json:"seq,omitempty"`
	FueraDelPlan int              `json:"fuera_del_plan,omitempty"`
	Limites      *consola.Limites `json:"limites,omitempty"`
	Proyectos    []ProyectoEstado `json:"proyectos"`
}

// EstaMaquina es el nombre y el sistema del PC.
func EstaMaquina() Maquina {
	nombre, err := os.Hostname()
	if err != nil || nombre == "" {
		nombre = "PC"
	}
	return Maquina{Nombre: nombre, SO: runtime.GOOS}
}

// Armar arma el estado de los proyectos del registro (ya ordenado por visto, el más reciente primero). limite es el
// de proyectos del plan (0: sin límite): se publican los más recientes y FueraDelPlan dice cuántos quedaron fuera.
func Armar(qdd string, reg registro.Registro, now time.Time, limite int) Estado {
	e := Estado{Version: VersionProtocolo, QDD: qdd, Maquina: EstaMaquina(), En: now.UTC(), Limites: consola.LeerLimites(), Proyectos: []ProyectoEstado{}}
	for _, p := range Publicables(reg, limite) {
		e.Proyectos = append(e.Proyectos, PanelDe(qdd, p, now))
	}
	e.FueraDelPlan = len(reg.Proyectos) - len(e.Proyectos)
	return e
}

// Publicables son los proyectos que entran en el plan: los más recientes hasta limite (0: todos).
func Publicables(reg registro.Registro, limite int) []registro.Proyecto {
	if limite <= 0 || len(reg.Proyectos) <= limite {
		return reg.Proyectos
	}
	return reg.Proyectos[:limite]
}

// PanelDe es el proyecto con su panel, exactamente consola.Panel (con la versión, como qdd consola panel --json) con
// las preguntas de la consola y lo mandado desde el teléfono (opcionales, sin secretos), o con el error si la carpeta
// ya no está. El ejecutar del panel depende del plan: lo pone ConEjecutar.
func PanelDe(qdd string, p registro.Proyecto, now time.Time) ProyectoEstado {
	out := ProyectoEstado{Clave: p.Clave, Nombre: filepath.Base(p.Ruta), Ruta: p.Ruta}
	if info, err := os.Stat(p.Ruta); err != nil || !info.IsDir() {
		out.Error = ErrCarpetaNoExiste
		return out
	}
	reglas := append([]string{}, integration.BloquesViejos(p.Ruta)...)
	out.ReglasAtrasadas = &reglas
	panel := consola.Panel(p.Ruta, now, consola.TipoPregunta, consola.ConTelefono)
	panel.Flujos = resumenDeFlujos(p.Ruta)
	sinEsperasSinSesion(p.Ruta, &panel, now)
	ocultarDelPanel(p.Ruta, &panel)
	panel.Version = qdd
	out.Panel = &panel
	return out
}

// sinEsperasSinSesion: un ⏳ lo resuelve la IA con la respuesta; sin una sesión abierta no hay quien, y el teléfono no lo
// ofrece (FND-124). Las preguntas de la consola siguen: tienen su propia vida.
func sinEsperasSinSesion(root string, panel *consola.DatosConsola, now time.Time) {
	if consola.HaySesion(root, now) {
		return
	}
	var quedan []consola.Accion
	for _, a := range panel.Acciones {
		if a.Tipo == consola.TipoPregunta && a.Origen == consola.OrigenEstado {
			continue
		}
		quedan = append(quedan, a)
	}
	panel.Acciones = quedan
}

// Linea es el proyecto en una línea (qdd remoto estado sin --json): nombre, foco, tareas, esperan y fallan.
func Linea(p ProyectoEstado) string {
	if p.Error != "" {
		return p.Nombre + " · ✕ " + p.Error
	}
	if p.Panel == nil || p.Panel.Estado == nil {
		return p.Nombre + " · sin estado"
	}
	d := p.Panel.Estado
	foco := "sin foco"
	for _, o := range d.Objetivos {
		if o.Foco {
			foco = "foco: " + o.Nombre
		}
	}
	return fmt.Sprintf("%s · %s · %d tareas · %d esperan · %d fallan", p.Nombre, foco, d.Tareas, d.Esperan, d.Fallan)
}
