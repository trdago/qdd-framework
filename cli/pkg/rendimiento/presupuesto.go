package rendimiento

// Nombres de las medidas que no son latencias.
const (
	MedidaCPU       = "seguir_cpu"
	MedidaMemoria   = "seguir_memoria"
	MedidaLanzados  = "seguir_lanzados_hora"
	MedidaZombies   = "zombies"
	MedidaHuerfanos = "huerfanos"
	MedidaContexto  = "contexto_mcp"
)

// presupuesto de una medida: hasta dónde está bien, en qué unidad y qué es. falla: pasarlo es un error, no lentitud.
type presupuesto struct {
	tope    float64
	unidad  string
	que     string
	falla   bool
	windows float64 // multiplica el tope en Windows: lanzar un proceso cuesta varias veces más
}

// presupuestos provisionales (2026-11-07): medidos en Linux, 16 núcleos, con holgura. Se recalibran con las primeras
// mediciones de Windows y macOS (RFC 0022).
var presupuestos = map[string]presupuesto{
	"latencia_statusline_p95": {tope: 60, unidad: "ms", que: "la barra de estado, cada 2 s en cada sesión", windows: 5},
	"latencia_panel_p95":      {tope: 400, unidad: "ms", que: "una lectura completa del panel", windows: 3},
	"latencia_estado_p95":     {tope: 150, unidad: "ms", que: "qdd estado, el tablero en texto", windows: 3},
	"latencia_flujo_p95":      {tope: 150, unidad: "ms", que: "dibujar un flujo de 25 nodos", windows: 3},
	MedidaCPU:                 {tope: 2, unidad: "% de un núcleo", que: "qdd consola seguir, el proceso largo de cada panel", windows: 2},
	MedidaMemoria:             {tope: 80, unidad: "MB", que: "memoria máxima de qdd consola seguir", windows: 1.5},
	MedidaLanzados:            {tope: 600, unidad: "procesos/h", que: "procesos que lanza qdd consola seguir por hora", windows: 1},
	MedidaZombies:             {tope: 0, unidad: "procesos", que: "hijos terminados que nadie recogió (zombies)", falla: true, windows: 1},
	MedidaHuerfanos:           {tope: 0, unidad: "procesos", que: "procesos que siguen vivos al cerrar la consola", falla: true, windows: 1},
	MedidaContexto:            {tope: 16000, unidad: "caracteres", que: "tools/list del MCP, en el contexto de cada sesión (~4 por token)", windows: 1},
}

// Evaluar compara un valor con su presupuesto en ese sistema operativo. nil queda sin medir.
func Evaluar(nombre string, valor *float64, so string) Medida {
	p := presupuestos[nombre]
	tope := p.tope
	if so == "windows" && p.windows > 0 {
		tope *= p.windows
	}
	m := Medida{Nombre: nombre, Valor: valor, Unidad: p.unidad, Presupuesto: tope, Que: p.que, Estado: EstadoOK}
	if valor == nil {
		m.Estado = EstadoSinMedir
		return m
	}
	if *valor <= tope {
		return m
	}
	m.Estado = EstadoSobre
	if p.falla {
		m.Estado = EstadoFalla
	}
	return m
}
