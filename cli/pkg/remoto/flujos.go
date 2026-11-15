package remoto

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'
// Flujos en QDD Móvil (pedido de qdd-movile, 2026-11-06). El panel lleva solo el resumen y cada flujo viaja en su propio
// evento: un delta manda el panel entero, así que con los nodos adentro cada avance de una tarea reenviaría todos.

// FlujoTelefono es lo que el teléfono necesita para dibujar un flujo con su propio dibujo.
type FlujoTelefono struct {
	ID          string         `json:"id"`
	Nombre      string         `json:"nombre"`
	Descripcion string         `json:"descripcion,omitempty"`
	Huella      string         `json:"huella"`
	Nodos       []NodoTelefono `json:"nodos"`

	Comentarios []flujo.Comentario `json:"comentarios,omitempty"` // preguntas y respuestas sobre el flujo completo
}

// NodoTelefono es un nodo con el estado de su tarea cuando el tablero visible no la trae (MEJ-083, propuesta de
// qdd-movile): cerrada, descartada u oculta. Si la tarea está en los frentes del panel, o no existe, no viene el campo.
type NodoTelefono struct {
	flujo.Nodo
	TareaEstado *TareaEstado `json:"tarea_estado,omitempty"`
}

// TareaEstado es el estado de una tarea fuera del tablero: abierto, cerrado o descartado, y sus celdas.
type TareaEstado struct {
	Estado   string   `json:"estado"`
	Simbolos []string `json:"simbolos"`
}

// MensajeFlujo es el evento «flujo».
type MensajeFlujo struct {
	Version int           `json:"version"`
	Clave   string        `json:"clave"`
	Flujo   FlujoTelefono `json:"flujo"`
}

// tareasVisibles son las tareas de los frentes del panel: el teléfono lee su estado de ahí, al día.
func tareasVisibles(d *estado.DatosPanel) map[string]bool {
	out := map[string]bool{}
	if d == nil {
		return out
	}
	for _, f := range d.Frentes {
		out[f.ID] = true
	}
	return out
}

// fueraDelTablero da el estado de una tarea que existe y no está en el tablero visible; nil si está a la vista o no
// existe. Lee el estado una sola vez.
func fueraDelTablero(root string, visibles map[string]bool) func(string) *TareaEstado {
	cargar := sync.OnceValues(func() (estado.Estado, error) { return estado.Load(root) })
	return func(tarea string) *TareaEstado {
		if tarea == "" || visibles[tarea] {
			return nil
		}
		e, err := cargar()
		if err != nil {
			return nil
		}
		simbolos, ok := e.Simbolos(tarea)
		if !ok {
			return nil
		}
		i := slices.IndexFunc(e.Frentes, func(f estado.Frente) bool { return f.ID == tarea })
		return &TareaEstado{Estado: estadoDeFrente(e.Frentes[i].Estado), Simbolos: simbolos}
	}
}

func estadoDeFrente(s string) string {
	if s == "" {
		return estado.EstadoAbierto
	}
	return s
}

// huellaFueraDelTablero: la huella del teléfono depende del archivo y del estado de las tareas fuera del tablero, que
// viaja en el flujo. Lo que está a la vista no cuenta: la app lo lee de los frentes del panel, y si la huella dependiera
// de eso, cada avance gastaría un mensaje con el flujo entero. Una tarea que se cierra sale del tablero y cambia la huella.
func huellaFueraDelTablero(fuera func(string) *TareaEstado) func(string) string {
	return func(tarea string) string {
		te := fuera(tarea)
		if te == nil {
			return ""
		}
		return te.Estado + " " + strings.Join(te.Simbolos, " ")
	}
}

// resumenDeFlujos es el campo flujos del panel que se publica; [] dice que este qdd sabe de flujos y el proyecto no tiene.
func resumenDeFlujos(root string, visibles map[string]bool) json.RawMessage {
	lista := flujo.Listar(root, huellaFueraDelTablero(fueraDelTablero(root, visibles)))
	if len(lista) > 0 {
		o := OcultadorDe(root)
		for i, r := range lista {
			lista[i].Nombre, lista[i].Problema = o.Ocultar(r.Nombre), o.Ocultar(r.Problema)
		}
	}
	b, err := json.Marshal(lista)
	if err != nil {
		return nil
	}
	return b
}

// resumenesDe lee el resumen de un panel armado por PanelDe; sin panel, o sin el campo, no hay nada que mandar.
func resumenesDe(p ProyectoEstado) []flujo.Resumen {
	if p.Panel == nil {
		return nil
	}
	var lista []flujo.Resumen
	if json.Unmarshal(p.Panel.Flujos, &lista) != nil {
		return nil
	}
	return lista
}

// mandarFlujos manda los flujos cuya huella todavía no tienen los teléfonos; un archivo roto no se manda: el resumen ya
// dice su problema. Si un envío falla, base en falso hace que el próximo estado completo los mande todos.
func (b *Bucle) mandarFlujos(ctx context.Context, e Estado) {
	vigentes := map[string]string{}
	for _, p := range e.Proyectos {
		for _, r := range resumenesDe(p) {
			if r.Problema != "" {
				continue
			}
			clave := p.Clave + "\x00" + r.ID
			if b.flujos[clave] == r.Huella {
				vigentes[clave] = r.Huella
				continue
			}
			m, err := b.mensajeFlujo(p, r)
			if err != nil {
				b.logf("flujo %s: %v", r.ID, err)
				continue
			}
			if !b.broadcast(ctx, "flujo", m) {
				b.base = false
				return
			}
			vigentes[clave] = r.Huella
		}
	}
	b.flujos = vigentes
}

// mensajeFlujo carga el flujo y tapa lo que parezca un secreto (inv. 16). La huella es la del resumen: si el archivo
// cambió entre medio, el próximo resumen trae otra y el flujo se vuelve a mandar.
func (b *Bucle) mensajeFlujo(p ProyectoEstado, r flujo.Resumen) (MensajeFlujo, error) {
	f, err := flujo.Cargar(p.Ruta, r.ID)
	if err != nil {
		return MensajeFlujo{}, err
	}
	o := b.ocultador(p.Ruta)
	for i, n := range f.Nodos {
		f.Nodos[i].Nombre, f.Nodos[i].Descripcion, f.Nodos[i].Tarea = o.Ocultar(n.Nombre), o.Ocultar(n.Descripcion), o.Ocultar(n.Tarea)
		for j, c := range n.Comentarios {
			f.Nodos[i].Comentarios[j].Texto, f.Nodos[i].Comentarios[j].DeNodo = o.Ocultar(c.Texto), o.Ocultar(c.DeNodo)
		}
		for j, it := range n.Items {
			f.Nodos[i].Items[j].Texto, f.Nodos[i].Items[j].Evidencia = o.Ocultar(it.Texto), o.Ocultar(it.Evidencia)
		}
	}
	for j, c := range f.Comentarios {
		f.Comentarios[j].Texto = o.Ocultar(c.Texto)
	}
	fuera := fueraDelTablero(p.Ruta, p.visibles)
	nodos := make([]NodoTelefono, len(f.Nodos))
	for i, n := range f.Nodos {
		nodos[i] = NodoTelefono{Nodo: n, TareaEstado: fuera(n.Tarea)}
	}
	telefono := FlujoTelefono{ID: r.ID, Nombre: o.Ocultar(f.Nombre), Descripcion: o.Ocultar(f.Descripcion), Huella: r.Huella, Nodos: nodos, Comentarios: f.Comentarios}
	return MensajeFlujo{Version: VersionProtocolo, Clave: p.Clave, Flujo: telefono}, nil
}
