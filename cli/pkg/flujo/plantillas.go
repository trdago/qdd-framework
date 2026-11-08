package flujo

import (
	"slices"
	"strings"
)

// Rule: Prohibido usar 'else'
// Plantillas (RFC 0025, decisión de la persona 2026-11-08: «un flujo, tipos por plantilla»). Una página web, un flujo
// de datos o un sistema de agentes son el mismo grafo: la plantilla solo trae los tipos de nodo, su ícono, su leyenda y
// el nombre de cada nivel N1 · N2 · N3. Todas traen «resultado», el círculo con que un flujo empieza y termina, así que
// el panel y el teléfono dibujan cualquier plantilla sin conocerla: resultado es círculo y el resto, caja.

const PlantillaProceso = "proceso"

// TipoNodo es un tipo de nodo de una plantilla: ID va en el YAML y en las operaciones, Nombre es lo que se lee.
type TipoNodo struct {
	ID      string `json:"id"`
	Nombre  string `json:"nombre"`
	Icono   string `json:"icono"`
	Explica string `json:"explica"`
}

// Plantilla es un dato, no código: sus tipos y el nombre de sus tres niveles.
type Plantilla struct {
	ID      string     `json:"id"`
	Nombre  string     `json:"nombre"`
	Niveles [3]string  `json:"niveles"`
	Tipos   []TipoNodo `json:"tipos"`
	// Clases son las que se sugieren para un grupo; se puede usar cualquier otra.
	Clases []string `json:"clases"`
}

var resultadoComun = TipoNodo{ID: TipoResultado, Nombre: "resultado", Icono: "◯", Explica: "Un inicio, un fin o un resultado."}

// Plantillas son las que qdd conoce, en el orden en que se ofrecen.
var Plantillas = []Plantilla{
	{ID: PlantillaProceso, Nombre: "Proceso", Niveles: [3]string{"Hitos", "Pasos", "Desvíos"}, Clases: []string{"fase", "área", "cola"}, Tipos: []TipoNodo{
		{ID: TipoPaso, Nombre: "paso", Icono: "▭", Explica: "Algo que alguien hace."},
		resultadoComun,
	}},
	{ID: "web", Nombre: "Página web", Niveles: [3]string{"Páginas", "Secciones", "Elementos"}, Clases: []string{"sección del sitio", "plantilla", "estado"}, Tipos: []TipoNodo{
		{ID: "pagina", Nombre: "página", Icono: "▤", Explica: "Una pantalla con su propia dirección."},
		{ID: "seccion", Nombre: "sección", Icono: "▥", Explica: "Una parte de la página: cabecera, listado, formulario."},
		{ID: "componente", Nombre: "componente", Icono: "◫", Explica: "Un elemento que se usa: un botón, una tarjeta, un menú."},
		{ID: "enlace", Nombre: "enlace", Icono: "↗", Explica: "Lo que lleva de un lugar a otro."},
		resultadoComun,
	}},
	{ID: "datos", Nombre: "Flujo de datos", Niveles: [3]string{"Fuentes", "Transformaciones", "Detalle"}, Clases: []string{"ambiente", "capa", "cola"}, Tipos: []TipoNodo{
		{ID: "fuente", Nombre: "fuente", Icono: "⛁", Explica: "De dónde salen los datos."},
		{ID: "transformacion", Nombre: "transformación", Icono: "⚙", Explica: "Lo que se les hace a los datos."},
		{ID: "regla", Nombre: "regla", Icono: "⚖", Explica: "Una condición que los datos tienen que cumplir."},
		{ID: "destino", Nombre: "destino", Icono: "⛃", Explica: "Dónde quedan los datos."},
		resultadoComun,
	}},
	{ID: "agentes", Nombre: "Sistema de agentes", Niveles: [3]string{"Agentes", "Herramientas", "Mensajes"}, Clases: []string{"equipo", "entorno", "fase"}, Tipos: []TipoNodo{
		{ID: "agente", Nombre: "agente", Icono: "◉", Explica: "Un agente con su propio objetivo."},
		{ID: "herramienta", Nombre: "herramienta", Icono: "🔧", Explica: "Algo que un agente puede usar."},
		{ID: "memoria", Nombre: "memoria", Icono: "▣", Explica: "Lo que se guarda entre pasos."},
		{ID: "persona", Nombre: "persona", Icono: "☺", Explica: "Una persona que decide o responde."},
		resultadoComun,
	}},
}

// PlantillaPorID busca una plantilla; vacía es proceso.
func PlantillaPorID(id string) (Plantilla, bool) {
	if id == "" {
		id = PlantillaProceso
	}
	i := slices.IndexFunc(Plantillas, func(p Plantilla) bool { return p.ID == id })
	if i < 0 {
		return Plantilla{}, false
	}
	return Plantillas[i], true
}

// PlantillaDe es la plantilla del flujo; una que no existe se trata como proceso (Validar ya la rechaza).
func PlantillaDe(f Flujo) Plantilla {
	p, ok := PlantillaPorID(f.Plantilla)
	if !ok {
		p, _ = PlantillaPorID(PlantillaProceso)
	}
	return p
}

// Tiene dice si el tipo es de la plantilla.
func (p Plantilla) Tiene(tipo string) bool {
	return slices.ContainsFunc(p.Tipos, func(t TipoNodo) bool { return t.ID == tipo })
}

// listaDeTipos es «paso o resultado», «pagina, seccion, … o resultado».
func (p Plantilla) listaDeTipos() string {
	ids := make([]string, len(p.Tipos))
	for i, t := range p.Tipos {
		ids[i] = t.ID
	}
	if len(ids) < 2 {
		return strings.Join(ids, "")
	}
	return strings.Join(ids[:len(ids)-1], ", ") + " o " + ids[len(ids)-1]
}

// NombresDeNivel son los nombres de N1, N2 y N3 en la plantilla del flujo.
func NombresDeNivel(f Flujo) [3]string {
	return PlantillaDe(f).Niveles
}

// tipoConocido: el tipo es de alguna plantilla. Una operación se lee sin el flujo; al guardar decide la del flujo.
func tipoConocido(tipo string) bool {
	return slices.ContainsFunc(Plantillas, func(p Plantilla) bool { return p.Tiene(tipo) })
}

func problemasDePlantilla(f Flujo) []string {
	if _, ok := PlantillaPorID(f.Plantilla); ok {
		return nil
	}
	ids := make([]string, len(Plantillas))
	for i, p := range Plantillas {
		ids[i] = p.ID
	}
	return []string{"plantilla «" + f.Plantilla + "»: usa " + strings.Join(ids, ", ")}
}
