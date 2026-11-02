package consola

import (
	"fmt"
	"strings"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'
// Los blueprints a un clic en el panel (pedido del usuario, 2026-11-01). Leer el inventario, traer el catálogo,
// actualizar e instalar son comandos de qdd, sin IA; aplicar es un mensaje para la IA, como atender una etapa. qdd
// vuelve a calcular estas acciones antes de ejecutar una (PrefijoBlueprint).

// PrefijoBlueprint marca las acciones del inventario de blueprints: 'qdd consola accion' las busca ahí.
const PrefijoBlueprint = "blueprint:"

// DatosBlueprints es lo que 'qdd consola blueprints --json' entrega al panel.
type DatosBlueprints struct {
	Inventario blueprints.Inventario `json:"inventario"`
	Acciones   []Accion              `json:"acciones"`
}

// Blueprints arma el inventario del proyecto (sin red) y sus acciones.
func Blueprints(root string) (DatosBlueprints, error) {
	inv, err := blueprints.InventarioDe(root)
	if err != nil {
		return DatosBlueprints{}, err
	}
	return DatosBlueprints{Inventario: inv, Acciones: AccionesDeInventario(inv)}, nil
}

// AccionesBlueprints es lo que 'qdd consola accion blueprint:…' vuelve a calcular antes de ejecutar.
func AccionesBlueprints(root string) []Accion {
	d, err := Blueprints(root)
	if err != nil {
		return nil
	}
	return d.Acciones
}

// AccionesDeInventario: traer el catálogo (escribe solo la copia central, sin confirmar); actualizar un instalado
// con versión nueva e instalar uno disponible (escriben el lockfile, con «s  Sí»); y aplicar con la IA un instalado.
func AccionesDeInventario(inv blueprints.Inventario) []Accion {
	acciones := []Accion{{
		ID: PrefijoBlueprint + "sync", Tipo: TipoComando, Etiqueta: "↻ Actualizar catálogo", Detalle: "qdd blueprint sync",
		Args: []string{"blueprint", "sync"},
	}}
	acciones = append(acciones, sinIAJuntos(inv)...)
	for _, e := range inv.Entradas {
		acciones = append(acciones, accionesDe(e)...)
	}
	return acciones
}

// sinIAJuntos: con dos o más instalados cuya receta no cambió, un solo 'qdd blueprint add' los toma todos.
func sinIAJuntos(inv blueprints.Inventario) []Accion {
	var ids []string
	for _, e := range inv.Entradas {
		if e.Estado == blueprints.CatSoloLock {
			ids = append(ids, e.ID)
		}
	}
	if len(ids) < 2 {
		return nil
	}
	return []Accion{{
		ID: PrefijoBlueprint + "actualizar-sin-ia", Tipo: TipoComando, Etiqueta: fmt.Sprintf("⬆ Actualizar los %d sin la IA", len(ids)),
		Detalle: "qdd blueprint add " + strings.Join(ids, " "), Confirmar: true, Args: append([]string{"blueprint", "add"}, ids...),
	}}
}

func accionesDe(e blueprints.EntradaInventario) []Accion {
	if e.Estado == blueprints.CatDisponible {
		return []Accion{{
			ID: PrefijoBlueprint + "instalar:" + e.ID, Tipo: TipoComando, Etiqueta: "＋ Instalar «" + nombreBP(e) + "»",
			Detalle: "qdd blueprint add " + e.Origen + " " + e.ID, Confirmar: true, Args: []string{"blueprint", "add", e.Origen, e.ID},
		}}
	}
	aplicar := Accion{
		ID: PrefijoBlueprint + "aplicar:" + e.ID, Tipo: TipoAtender, Etiqueta: "✦ Aplicar «" + nombreBP(e) + "» con la IA",
		Detalle: mensajeAplicar(e),
	}
	if e.Estado != blueprints.CatSoloLock && e.Estado != blueprints.CatConCambios {
		return []Accion{aplicar}
	}
	actualizar := Accion{
		ID: PrefijoBlueprint + "actualizar:" + e.ID, Tipo: TipoComando, Etiqueta: "⬆ Actualizar «" + nombreBP(e) + "» a " + e.Publicada,
		Detalle: "qdd blueprint add " + e.ID, Confirmar: true, Args: []string{"blueprint", "add", e.ID},
	}
	return []Accion{actualizar, aplicar}
}

// mensajeAplicar es lo que recibe la IA, estilo chat y de MaxMensajeAtender como mucho: se acorta lo citado, nunca
// las instrucciones. Las preguntas de diseño las responde la persona.
func mensajeAplicar(e blueprints.EntradaInventario) string {
	nombre := "«" + recortar(nombreBP(e), 40) + "»"
	pedido := " con qdd_blueprint_apply: pregúntame sus decisiones de diseño tal como vienen y construye con pruebas primero."
	if e.Estado != blueprints.CatConCambios {
		return recortar("Aplica el blueprint "+nombre+" ("+e.ID+"@"+e.Instalada+") a este proyecto"+pedido, MaxMensajeAtender)
	}
	cambio := " La receta cambió a " + e.Publicada + ": " + recortar(strings.Join(e.Razones, "; "), maxDato) + "."
	return recortar("Actualiza "+nombre+" con qdd blueprint add "+e.ID+" y adapta su módulo"+pedido+cambio, MaxMensajeAtender)
}

func nombreBP(e blueprints.EntradaInventario) string {
	if e.Nombre != "" {
		return e.Nombre
	}
	return e.ID
}
