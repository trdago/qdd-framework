package estado

import (
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// Qué tareas puede mover la IA (MEJ-055, FND-216): el modo goal le recuerda a la IA solo esas, y si no queda ninguna,
// avisa a la persona una vez. Las etapas van en orden, así que manda la primera sin terminar: si es un ⏳ de la persona o
// de un tercero, lo que viene después también espera. Lo que la IA puede mover es una etapa sin empezar o fallida, un ⏳
// suyo sin verificación automática, o una tarea con todo hecho que falta cerrar. Una tarea que depende de otra abierta
// espera a esa otra. La Validación real sin empezar espera un uso real (EPF-C04: su evidencia no es la del despliegue),
// no a la IA.

// EsperaAjena es la etapa que tiene detenida una tarea, esperando a alguien que no es la IA.
type EsperaAjena struct {
	Frente  Frente
	Etapa   string
	Bloqueo string
	DeQuien string
}

// aliasIA son los de_quien con que una sesión se nombra a sí misma o al equipo con la persona («nosotros»).
var aliasIA = map[string]bool{"ia": true, "ai": true, "claude": true, "agente": true, "asistente": true, "nosotros": true}

// UsoReal es de quién espera una Validación real sin empezar.
const UsoReal = "uso real"

// etapaDeUsoReal: la etapa de la ficha «software» que solo se cumple con un uso real.
const etapaDeUsoReal = "validación real"

// reSesion: el id corto de una sesión del proyecto (qdd-6e para el proyecto qdd).
var reSesion = regexp.MustCompile(`^-[0-9a-f]{1,8}$`)

// DeLaIA dice si un ⏳ es de la IA: un alias de la IA, el nombre del proyecto o una sesión suya («qdd-6e» en qdd). Una
// sesión de otro proyecto («qdd-movile-13»), la persona o un tercero no lo son; lo que no se reconoce, tampoco.
func DeLaIA(deQuien, proyecto string) bool {
	quien := strings.ToLower(strings.TrimSpace(deQuien))
	quien = strings.TrimPrefix(strings.TrimPrefix(quien, "la "), "el ")
	if quien == "" || DeLaPersona(quien) {
		return false
	}
	if aliasIA[quien] {
		return true
	}
	proyecto = strings.ToLower(proyecto)
	if proyecto == "" || !strings.HasPrefix(quien, proyecto) {
		return false
	}
	resto := strings.TrimPrefix(quien, proyecto)
	return resto == "" || reSesion.MatchString(resto)
}

// Movibles separa las tareas abiertas que la IA puede mover de las etapas que esperan a otros. proyecto es el nombre de
// la carpeta, para reconocer sus sesiones.
func (e Estado) Movibles(proyecto string) ([]Frente, []EsperaAjena) {
	abiertas := map[string]bool{}
	for _, f := range e.Abiertos() {
		abiertas[f.ID] = true
	}
	var movibles []Frente
	var esperas []EsperaAjena
	for _, f := range e.Abiertos() {
		if dependeDeAbierta(f, abiertas) {
			continue
		}
		espera, detenida := primeraAjena(f, e.EtapasDe(f), proyecto)
		if detenida {
			esperas = append(esperas, espera)
			continue
		}
		movibles = append(movibles, f)
	}
	return movibles, esperas
}

func dependeDeAbierta(f Frente, abiertas map[string]bool) bool {
	for _, id := range f.DependeDe {
		if abiertas[id] {
			return true
		}
	}
	return false
}

// primeraAjena mira la primera etapa sin terminar: si es un ⏳ que no es de la IA (o uno de la IA que se responde solo),
// la tarea está detenida esperando a otro.
func primeraAjena(f Frente, etapas []string, proyecto string) (EsperaAjena, bool) {
	for _, etapa := range etapas {
		c := f.Celdas[etapa]
		estado := NormalizarEstado(c.Estado)
		if strings.EqualFold(strings.TrimSpace(etapa), etapaDeUsoReal) && (estado == SinEmpezar || estado == "") {
			return EsperaAjena{Frente: f, Etapa: etapa, DeQuien: UsoReal}, true
		}
		switch estado {
		case Hecho, NoAplica:
			continue
		case Espera:
			if DeLaIA(c.DeQuien, proyecto) && strings.TrimSpace(c.Verifica) == "" {
				return EsperaAjena{}, false
			}
			return EsperaAjena{Frente: f, Etapa: etapa, Bloqueo: c.Bloqueo, DeQuien: c.DeQuien}, true
		}
		return EsperaAjena{}, false
	}
	return EsperaAjena{}, false
}
