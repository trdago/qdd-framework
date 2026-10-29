package estado

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// Lector automático de las invariantes del blueprint estado-por-frente (escenarios EPF-C01…C07). EPF-C08 (texto y
// tabla no se contradicen) necesita leer la prosa y queda como revisión humana.

// Violacion es un escenario de conformidad que no se cumple.
type Violacion struct {
	ID      string `json:"id"`
	Frente  string `json:"frente,omitempty"`
	Etapa   string `json:"etapa,omitempty"`
	Mensaje string `json:"mensaje"`
}

func (v Violacion) String() string {
	lugar := v.Frente
	if v.Etapa != "" {
		lugar += " › " + v.Etapa
	}
	if lugar == "" {
		return fmt.Sprintf("[%s] %s", v.ID, v.Mensaje)
	}
	return fmt.Sprintf("[%s] %s: %s", v.ID, lugar, v.Mensaje)
}

// secretoODatoPersonal: forma de credencial, correo o RUT chileno (EPF-C07).
var secretoODatoPersonal = regexp.MustCompile(`(?i)(sk-[a-z0-9_-]{16,}|ghp_[a-z0-9]{16,}|github_pat_[a-z0-9_]{16,}|AKIA[0-9A-Z]{16}|xox[baprs]-[a-z0-9-]{10,}|(password|passwd|secret|token|api[_-]?key)\s*[:=]\s*\S+|[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}|\b\d{1,2}\.\d{3}\.\d{3}-[\dkK]\b)`)

// Check revisa el archivo. prev es la versión anterior (p. ej. la de git HEAD) para EPF-C05; nil si no hay.
func Check(e Estado, prev *Estado) []Violacion {
	var vs []Violacion
	vistos := map[string]bool{}
	for _, f := range e.Frentes {
		vs = append(vs, checkFrente(e, f, vistos)...)
		vistos[f.ID] = true
	}
	if prev == nil {
		return vs
	}
	for _, f := range prev.Frentes {
		if e.buscar(f.ID) < 0 {
			vs = append(vs, Violacion{ID: "EPF-C05", Frente: f.ID, Mensaje: "el frente desapareció: ciérralo con 'qdd estado cerrar' y su motivo, no lo borres"})
		}
	}
	return vs
}

func checkFrente(e Estado, f Frente, vistos map[string]bool) []Violacion {
	var vs []Violacion
	if strings.TrimSpace(f.ID) == "" {
		return []Violacion{{ID: "EPF-C01", Frente: f.Nombre, Mensaje: "frente sin id"}}
	}
	if vistos[f.ID] {
		vs = append(vs, Violacion{ID: "EPF-C01", Frente: f.ID, Mensaje: "id repetido"})
	}
	vs = append(vs, checkCierre(f)...)
	vs = append(vs, checkTexto(f.ID, "", f.Nombre, f.MotivoCierre)...)
	for nombre := range f.Celdas {
		if !e.tieneEtapa(nombre) {
			vs = append(vs, Violacion{ID: "EPF-C01", Frente: f.ID, Etapa: nombre, Mensaje: "celda de una etapa que no está en «etapas»"})
		}
	}
	for _, etapa := range e.Etapas {
		c, ok := f.Celdas[etapa]
		if !ok {
			vs = append(vs, Violacion{ID: "EPF-C01", Frente: f.ID, Etapa: etapa, Mensaje: "falta la celda: cada celda tiene exactamente un estado"})
			continue
		}
		vs = append(vs, checkCelda(f.ID, etapa, c)...)
	}
	return append(vs, checkDesplegadoNoEsValidado(f)...)
}

func checkCierre(f Frente) []Violacion {
	switch f.Estado {
	case EstadoAbierto:
		return nil
	case EstadoCerrado, EstadoDescartado:
		if strings.TrimSpace(f.MotivoCierre) == "" {
			return []Violacion{{ID: "EPF-C05", Frente: f.ID, Mensaje: "frente " + f.Estado + " sin motivo_cierre: hay que decir por qué"}}
		}
		return nil
	}
	return []Violacion{{ID: "EPF-C01", Frente: f.ID, Mensaje: fmt.Sprintf("estado del frente %q: debe ser abierto, cerrado o descartado", f.Estado)}}
}

func checkCelda(frente, etapa string, c Celda) []Violacion {
	vs := checkTexto(frente, etapa, c.Evidencia, c.Bloqueo, c.DeQuien, c.Falla, c.RequiereAprobacion)
	switch NormalizarEstado(c.Estado) {
	case Hecho:
		if strings.TrimSpace(c.Evidencia) == "" {
			vs = append(vs, Violacion{ID: "EPF-C02", Frente: frente, Etapa: etapa, Mensaje: "✅ sin evidencia (commit, versión, n/n pruebas o id de corrida)"})
		}
		if c.RequiereAprobacion != "" && !c.Aprobado {
			vs = append(vs, Violacion{ID: "EPF-C06", Frente: frente, Etapa: etapa, Mensaje: "requiere aprobación de " + c.RequiereAprobacion + " y todavía no la tiene: no puede estar en ✅"})
		}
	case Espera:
		if strings.TrimSpace(c.Bloqueo) == "" || strings.TrimSpace(c.DeQuien) == "" {
			vs = append(vs, Violacion{ID: "EPF-C03", Frente: frente, Etapa: etapa, Mensaje: "⏳ debe decir qué espera (bloqueo) y de quién (de_quien)"})
		}
	case Falla:
		if strings.TrimSpace(c.Falla) == "" {
			vs = append(vs, Violacion{ID: "EPF-C01", Frente: frente, Etapa: etapa, Mensaje: "❌ sin nombrar la falla"})
		}
	case NoAplica:
	default:
		vs = append(vs, Violacion{ID: "EPF-C01", Frente: frente, Etapa: etapa, Mensaje: fmt.Sprintf("estado %q: debe ser uno de ✅ ⏳ ❌ —", c.Estado)})
	}
	return vs
}

// checkDesplegadoNoEsValidado (EPF-C04): con etapas «software», Validación real en ✅ necesita su propia evidencia
// (no la del despliegue); con etapas «modelos», Producción en ✅ necesita la Puerta en ✅.
func checkDesplegadoNoEsValidado(f Frente) []Violacion {
	var vs []Violacion
	prod, hayProd := f.Celdas["Producción"]
	if !hayProd || NormalizarEstado(prod.Estado) != Hecho {
		return nil
	}
	val, hayVal := f.Celdas["Validación real"]
	if hayVal && NormalizarEstado(val.Estado) == Hecho && strings.TrimSpace(val.Evidencia) == strings.TrimSpace(prod.Evidencia) {
		vs = append(vs, Violacion{ID: "EPF-C04", Frente: f.ID, Etapa: "Validación real", Mensaje: "desplegado no es validado: la Validación real necesita evidencia de un uso real, no la del despliegue"})
	}
	puerta, hayPuerta := f.Celdas["Puerta"]
	if hayPuerta && NormalizarEstado(puerta.Estado) != Hecho {
		vs = append(vs, Violacion{ID: "EPF-C04", Frente: f.ID, Etapa: "Producción", Mensaje: "Producción en ✅ sin la Puerta en ✅ con su evidencia"})
	}
	return vs
}

func checkTexto(frente, etapa string, textos ...string) []Violacion {
	for _, t := range textos {
		if secretoODatoPersonal.MatchString(t) {
			return []Violacion{{ID: "EPF-C07", Frente: frente, Etapa: etapa, Mensaje: "parece una credencial o un dato personal: no va en la tabla"}}
		}
	}
	return nil
}
