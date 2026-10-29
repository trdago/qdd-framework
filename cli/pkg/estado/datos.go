package estado

import "time"

// DatosPanel es lo que 'qdd estado ver --json' entrega a quien dibuja el estado fuera de la CLI (el mod de Claude Code,
// RFC 0006 §3.4): ya resuelto, para que el que dibuja no interprete reglas ni lea el YAML.
type DatosPanel struct {
	Formato      int           `json:"formato"`
	Actualizado  string        `json:"actualizado,omitempty"`
	Resumen      string        `json:"resumen"`
	Etapas       []string      `json:"etapas"`
	Abreviaturas []string      `json:"abreviaturas"`
	Frentes      []FrentePanel `json:"frentes"`
	Esperas      []Pendiente   `json:"esperas"`
	Fallas       []Pendiente   `json:"fallas"`
}

// FrentePanel es una fila: un símbolo por etapa, en el orden de Etapas.
type FrentePanel struct {
	ID       string   `json:"id"`
	Nombre   string   `json:"nombre"`
	Estado   string   `json:"estado"`
	Simbolos []string `json:"simbolos"`
}

// Pendiente es un ⏳ (qué espera y de quién) o un ❌ (la falla nombrada).
type Pendiente struct {
	Frente  string `json:"frente"`
	Etapa   string `json:"etapa"`
	Texto   string `json:"texto"`
	DeQuien string `json:"de_quien,omitempty"`
}

// Datos arma DatosPanel con los frentes abiertos y los cerrados hace menos de RecienCerrado.
func Datos(e Estado, now time.Time) DatosPanel {
	d := DatosPanel{
		Formato: e.Formato, Actualizado: e.Actualizado, Resumen: Statusline(e, now), Etapas: e.Etapas,
		Abreviaturas: abreviaturas(e.Etapas), Frentes: []FrentePanel{}, Esperas: []Pendiente{}, Fallas: []Pendiente{},
	}
	for _, f := range e.Frentes {
		if !f.EnAlcance(AlcanceAbiertos, now) {
			continue
		}
		fila := FrentePanel{ID: f.ID, Nombre: f.Nombre, Estado: f.Estado, Simbolos: make([]string, 0, len(e.Etapas))}
		for _, etapa := range e.Etapas {
			c := f.Celdas[etapa]
			fila.Simbolos = append(fila.Simbolos, simbolo(c))
			switch NormalizarEstado(c.Estado) {
			case Espera:
				d.Esperas = append(d.Esperas, Pendiente{Frente: f.Nombre, Etapa: etapa, Texto: c.Bloqueo, DeQuien: c.DeQuien})
			case Falla:
				d.Fallas = append(d.Fallas, Pendiente{Frente: f.Nombre, Etapa: etapa, Texto: c.Falla})
			}
		}
		d.Frentes = append(d.Frentes, fila)
	}
	return d
}
