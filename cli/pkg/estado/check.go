package estado

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// Lector automático de las invariantes del blueprint estado-por-tareas (antes estado-por-frente; escenarios EPF-C01…C07). EPF-C08 (texto y
// tabla no se contradicen) necesita el texto del reporte: lo revisa CheckRespuesta (render.go).

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

// secretoODatoPersonal: la lista mínima de la decisión del blueprint (EPF-C07): tokens con prefijo conocido (sk-, ghp_,
// github_pat_, AKIA, xox, npm_), claves privadas, «Bearer …», «clave=», «token=», «password=», correos, teléfonos con
// código de país y RUT (con o sin puntos).
var secretoODatoPersonal = regexp.MustCompile(`(?i)(sk-[a-z0-9_-]{16,}|ghp_[a-z0-9]{16,}|github_pat_[a-z0-9_]{16,}|AKIA[0-9A-Z]{16}|xox[baprs]-[a-z0-9-]{10,}|\bnpm_[a-z0-9]{20,}|-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----|\bbearer\s+[a-z0-9._~+/=-]{8,}|(password|passwd|secret|token|api[_-]?key)\s*[:=]\s*\S+|\bclave\s*=\s*\S+|[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}|\+\d{1,3}(?:[\s.-]?\d){8,12}\b|\b\d{1,2}\.\d{3}\.\d{3}-[\dkK]\b|\b\d{7,8}-[\dkK]\b)`)

// Check revisa el archivo. prev es la versión anterior (p. ej. la de git HEAD) para EPF-C05; nil si no hay.
func Check(e Estado, prev *Estado) []Violacion {
	var vs []Violacion
	vistos := map[string]bool{}
	for _, f := range e.Frentes {
		vs = append(vs, checkFrente(e, f, vistos)...)
		vs = append(vs, e.checkDependencias(f)...)
		vistos[f.ID] = true
	}
	vs = append(vs, e.checkObjetivos(prev)...)
	vs = append(vs, e.checkCiclos()...)
	if prev == nil {
		return vs
	}
	for _, f := range prev.Frentes {
		if e.buscar(f.ID) < 0 {
			vs = append(vs, Violacion{ID: "EPF-C05", Frente: f.ID, Mensaje: "la tarea desapareció: ciérrala con 'qdd estado cerrar' y su motivo, no lo borres"})
		}
	}
	return vs
}

func checkFrente(e Estado, f Frente, vistos map[string]bool) []Violacion {
	var vs []Violacion
	if strings.TrimSpace(f.ID) == "" {
		return []Violacion{{ID: "EPF-C01", Frente: f.Nombre, Mensaje: "tarea sin id"}}
	}
	if vistos[f.ID] {
		vs = append(vs, Violacion{ID: "EPF-C01", Frente: f.ID, Mensaje: "id repetido"})
	}
	vs = append(vs, checkCierre(f)...)
	vs = append(vs, checkTexto(f.ID, "", f.Nombre, f.MotivoCierre)...)
	for nombre := range f.Celdas {
		if e.tieneEtapa(f, nombre) {
			continue
		}
		if e.usaCiclos() {
			vs = append(vs, Violacion{ID: "EPT-K02", Frente: f.ID, Etapa: nombre, Mensaje: "celda de una etapa que no es del ciclo «" + e.CicloDe(f) + "» de la tarea"})
			continue
		}
		vs = append(vs, Violacion{ID: "EPF-C01", Frente: f.ID, Etapa: nombre, Mensaje: "celda de una etapa que no está en «etapas»"})
	}
	for _, etapa := range e.EtapasDe(f) {
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
			return []Violacion{{ID: "EPF-C05", Frente: f.ID, Mensaje: "tarea " + f.Estado + " sin motivo_cierre: hay que decir por qué"}}
		}
		return nil
	}
	return []Violacion{{ID: "EPF-C01", Frente: f.ID, Mensaje: fmt.Sprintf("estado de la tarea %q: debe ser abierto, cerrado o descartado", f.Estado)}}
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
			vs = append(vs, Violacion{ID: "EPF-C09", Frente: frente, Etapa: etapa, Mensaje: "❌ sin nombrar la falla"})
		}
	case NoAplica, SinEmpezar:
	default:
		vs = append(vs, Violacion{ID: "EPF-C01", Frente: frente, Etapa: etapa, Mensaje: fmt.Sprintf("estado %q: debe ser uno de ✅ ⏳ ❌ — ·", c.Estado)})
	}
	return vs
}

// checkDesplegadoNoEsValidado (EPF-C04): con etapas «software», Validación real en ✅ necesita la evidencia de un uso
// real: ni la del despliegue ni solo una versión o un commit (FND-266); con etapas «modelos», Producción en ✅ necesita
// la Puerta en ✅.
func checkDesplegadoNoEsValidado(f Frente) []Violacion {
	var vs []Violacion
	prod, hayProd := f.Celdas["Producción"]
	val, hayVal := f.Celdas["Validación real"]
	if hayVal && NormalizarEstado(val.Estado) == Hecho && evidenciaDeDespliegue(val.Evidencia, prod) {
		vs = append(vs, Violacion{ID: "EPF-C04", Frente: f.ID, Etapa: "Validación real", Mensaje: "desplegado no es validado: la Validación real necesita evidencia de un uso real, no la del despliegue"})
	}
	if !hayProd || NormalizarEstado(prod.Estado) != Hecho {
		return vs
	}
	puerta, hayPuerta := f.Celdas["Puerta"]
	if hayPuerta && NormalizarEstado(puerta.Estado) != Hecho {
		vs = append(vs, Violacion{ID: "EPF-C04", Frente: f.ID, Etapa: "Producción", Mensaje: "Producción en ✅ sin la Puerta en ✅ con su evidencia"})
	}
	return vs
}

// soloDespliegue es una evidencia que nombra solo una versión o un commit: dice qué se desplegó, no que se usó.
var soloDespliegue = regexp.MustCompile(`(?i)^(?:(?:tag|versi[oó]n|release|commit)\s+)?(?:v?\d+\.\d+(?:\.\d+)?(?:-[0-9a-z.]+)?|[0-9a-f]{7,40})$`)

func evidenciaDeDespliegue(evidencia string, prod Celda) bool {
	ev := strings.TrimSpace(evidencia)
	if NormalizarEstado(prod.Estado) == Hecho && ev == strings.TrimSpace(prod.Evidencia) {
		return true
	}
	return soloDespliegue.MatchString(ev)
}

func checkTexto(frente, etapa string, textos ...string) []Violacion {
	for _, t := range textos {
		if secretoODatoPersonal.MatchString(t) {
			return []Violacion{{ID: "EPF-C07", Frente: frente, Etapa: etapa, Mensaje: "parece una credencial o un dato personal: no va en la tabla"}}
		}
	}
	return nil
}
