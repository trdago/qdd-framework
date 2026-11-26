package consola

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/findings"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0018: lo que dibuja la vista «Proyectos». Solo lee archivos locales: no escribe en ningún proyecto, no corre
// auditorías y no usa la red. null es «sin medir»: nunca un cero ni un valor inventado.

// TopeHistorial es cuántos días guarda cada historial, los últimos: el panel tiene un tope de texto (MEJ-014).
const TopeHistorial = 30

// DatosProyectos es la salida de qdd consola proyectos --json.
type DatosProyectos struct {
	Proyectos []FilaProyecto `json:"proyectos"`
	Aviso     string         `json:"aviso,omitempty"`
}

// FilaProyecto es un proyecto. Del proyecto solo se entrega el nombre de la carpeta, nunca la ruta completa.
type FilaProyecto struct {
	Clave            string           `json:"clave"`
	Nombre           string           `json:"nombre"`
	Existe           bool             `json:"existe"`
	Puntaje          *int             `json:"puntaje"`
	Hace             string           `json:"hace"`
	FindingsAbiertos *int             `json:"findings_abiertos"`
	Esperan          *int             `json:"esperan"`
	Fallan           *int             `json:"fallan"`
	Puntajes         []PuntajeDelDia  `json:"puntajes"`
	FindingsPorFecha []FindingsDelDia `json:"findings_por_fecha"`
	FindingsSinFecha int              `json:"findings_sin_fecha"`
	// ReglasAtrasadas son los archivos con el bloque de reglas de un qdd anterior (integration.BloquesViejos):
	// qdd doctor --fix los renueva. null si la carpeta no existe.
	ReglasAtrasadas []string `json:"reglas_atrasadas"`
}

// PuntajeDelDia es el último puntaje de un día. Dos escalas no se comparan: OtraEscala lo marca aparte.
type PuntajeDelDia struct {
	Fecha      string `json:"fecha"`
	Puntaje    int    `json:"puntaje"`
	OtraEscala bool   `json:"otra_escala,omitempty"`
}

// FindingsDelDia son los findings con esa fecha. Cerrados son los resueltos y los descartados; uno sin estado no es ni
// abierto ni cerrado: lo decide una persona (pkg/findings).
type FindingsDelDia struct {
	Fecha     string `json:"fecha"`
	Abiertos  int    `json:"abiertos"`
	Cerrados  int    `json:"cerrados"`
	SinEstado int    `json:"sin_estado,omitempty"`
}

// Proyectos entrega una fila por proyecto del registro, el visto más recientemente primero. Nunca falla: el panel
// muestra el aviso.
func Proyectos(now time.Time) DatosProyectos {
	reg, err := registro.Leer()
	if err != nil {
		return DatosProyectos{Proyectos: []FilaProyecto{}, Aviso: "No se pudo leer el registro de proyectos: qdd doctor --fix lo repara."}
	}
	filas := make([]FilaProyecto, 0, len(reg.Proyectos))
	for _, p := range reg.Proyectos {
		filas = append(filas, filaDe(p, now))
	}
	return DatosProyectos{Proyectos: filas}
}

func filaDe(p registro.Proyecto, now time.Time) FilaProyecto {
	fila := FilaProyecto{
		Clave:            p.Clave,
		Nombre:           filepath.Base(p.Ruta),
		Puntajes:         []PuntajeDelDia{},
		FindingsPorFecha: []FindingsDelDia{},
	}
	if info, err := os.Stat(p.Ruta); err != nil || !info.IsDir() {
		return fila
	}
	fila.Existe = true
	llenarPuntaje(&fila, p.Ruta, now)
	llenarFindings(&fila, p.Ruta)
	llenarEstado(&fila, p.Ruta)
	fila.ReglasAtrasadas = append([]string{}, integration.BloquesViejos(p.Ruta)...)
	return fila
}

func llenarPuntaje(fila *FilaProyecto, root string, now time.Time) {
	data, err := os.ReadFile(filepath.Join(root, ".qdd", "project", "metrics", "certificate_history.json"))
	if err != nil {
		return
	}
	var historia []audit.Certificate
	if json.Unmarshal(data, &historia) != nil || len(historia) == 0 {
		return
	}
	fila.Puntajes = puntajesPorDia(historia)
	ultimo := historia[len(historia)-1]
	if ultimo.Escala != audit.EscalaActual {
		return
	}
	puntaje := ultimo.Score
	fila.Puntaje = &puntaje
	fila.Hace = haceDesde(ultimo.Timestamp, now)
}

func puntajesPorDia(historia []audit.Certificate) []PuntajeDelDia {
	dias := []PuntajeDelDia{}
	for _, cert := range historia {
		fecha, ok := diaDe(cert.Timestamp)
		if !ok {
			continue
		}
		punto := PuntajeDelDia{Fecha: fecha, Puntaje: cert.Score, OtraEscala: cert.Escala != audit.EscalaActual}
		if n := len(dias); n > 0 && dias[n-1].Fecha == fecha && dias[n-1].OtraEscala == punto.OtraEscala {
			dias[n-1] = punto
			continue
		}
		dias = append(dias, punto)
	}
	return ultimos(dias)
}

func haceDesde(timestamp string, now time.Time) string {
	cuando, err := time.Parse(time.RFC3339, timestamp)
	if err != nil || cuando.After(now) {
		return ""
	}
	return estado.Hace(now.Sub(cuando))
}

// diaDe da el día de una fecha con o sin hora. Una que no es fecha no inventa un día.
func diaDe(fecha string) (string, bool) {
	if len(fecha) < 10 {
		return "", false
	}
	if _, err := time.Parse("2006-01-02", fecha[:10]); err != nil {
		return "", false
	}
	return fecha[:10], true
}

func llenarFindings(fila *FilaProyecto, root string) {
	if _, err := os.Stat(findings.Dir(root)); err != nil {
		return
	}
	lista, err := findings.Leer(root)
	if err != nil {
		return
	}
	abiertos := findings.Contar(lista).Abiertos
	fila.FindingsAbiertos = &abiertos
	porDia := map[string]FindingsDelDia{}
	for _, f := range lista {
		dia, ok := diaDe(f.Cuando)
		if !ok {
			fila.FindingsSinFecha++
			continue
		}
		porDia[dia] = sumarFinding(porDia[dia], dia, f.Estado)
	}
	for _, dia := range porDia {
		fila.FindingsPorFecha = append(fila.FindingsPorFecha, dia)
	}
	sort.Slice(fila.FindingsPorFecha, func(i, j int) bool { return fila.FindingsPorFecha[i].Fecha < fila.FindingsPorFecha[j].Fecha })
	fila.FindingsPorFecha = ultimos(fila.FindingsPorFecha)
}

func sumarFinding(dia FindingsDelDia, fecha string, e findings.Estado) FindingsDelDia {
	dia.Fecha = fecha
	switch e {
	case findings.Abierto:
		dia.Abiertos++
	case findings.Resuelto, findings.Descartado:
		dia.Cerrados++
	default:
		dia.SinEstado++
	}
	return dia
}

func ultimos[T any](lista []T) []T {
	if len(lista) <= TopeHistorial {
		return lista
	}
	return lista[len(lista)-TopeHistorial:]
}

func llenarEstado(fila *FilaProyecto, root string) {
	e, err := estado.Load(root)
	if err != nil {
		return
	}
	totales := e.Totales()
	fila.Esperan = &totales.Esperan
	fila.Fallan = &totales.Fallan
}
