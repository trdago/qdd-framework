package remoto

import (
	"context"
	"encoding/json"

	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'
// Flujos en QDD Móvil (pedido de qdd-movile, 2026-11-06). El panel lleva solo el resumen y cada flujo viaja en su propio
// evento: un delta manda el panel entero, así que con los nodos adentro cada avance de una tarea reenviaría todos.

// FlujoTelefono es lo que el teléfono necesita para dibujar un flujo con su propio dibujo.
type FlujoTelefono struct {
	ID          string       `json:"id"`
	Nombre      string       `json:"nombre"`
	Descripcion string       `json:"descripcion,omitempty"`
	Huella      string       `json:"huella"`
	Nodos       []flujo.Nodo `json:"nodos"`

	Comentarios []flujo.Comentario `json:"comentarios,omitempty"` // preguntas y respuestas sobre el flujo completo
}

// MensajeFlujo es el evento «flujo».
type MensajeFlujo struct {
	Version int           `json:"version"`
	Clave   string        `json:"clave"`
	Flujo   FlujoTelefono `json:"flujo"`
}

// sinEstadoDeTarea deja la huella del teléfono en función del archivo: la app lee el estado de cada tarea de los frentes
// del panel, y si la huella dependiera de él, cada avance gastaría un mensaje con el flujo entero.
func sinEstadoDeTarea(string) string {
	return ""
}

// resumenDeFlujos es el campo flujos del panel que se publica; [] dice que este qdd sabe de flujos y el proyecto no tiene.
func resumenDeFlujos(root string) json.RawMessage {
	lista := flujo.Listar(root, sinEstadoDeTarea)
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
	}
	for j, c := range f.Comentarios {
		f.Comentarios[j].Texto = o.Ocultar(c.Texto)
	}
	telefono := FlujoTelefono{ID: r.ID, Nombre: o.Ocultar(f.Nombre), Descripcion: o.Ocultar(f.Descripcion), Huella: r.Huella, Nodos: f.Nodos, Comentarios: f.Comentarios}
	return MensajeFlujo{Version: VersionProtocolo, Clave: p.Clave, Flujo: telefono}, nil
}
