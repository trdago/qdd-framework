package audit

// AdopcionGradual es el modo con que `qdd init` deja a los proyectos nuevos:
// las reglas de madurez del proceso se informan como recomendaciones y no
// bloquean. Los problemas reales (pruebas que fallan, OWASP, despliegues sin
// pruebas, código sucio) bloquean igual. Para exigirlo todo: `adopcion: estricta`
// en .qdd/policies.yaml (o quitar la clave).
const AdopcionGradual = "gradual"

// reglasDeMadurez son prácticas que un proyecto adopta con el tiempo; su
// ausencia no indica que el código esté mal.
var reglasDeMadurez = map[string]bool{
	"QA-02-E2E-TESTS":             true,
	"TRC-01-CONVENTIONAL-COMMITS": true,
	"CERT-030-MANDATORY-ADR":      true,
	"12F-01-CODEBASE":             true,
}

func marcarRecomendaciones(p QDDPolicies, violations []Violation) {
	if p.Adopcion != AdopcionGradual {
		return
	}
	for i := range violations {
		violations[i].Recomendacion = reglasDeMadurez[violations[i].RuleID]
	}
}

// Bloqueantes devuelve las violaciones que impiden aprobar el gate.
func Bloqueantes(violations []Violation) []Violation {
	var out []Violation
	for _, v := range violations {
		if !v.Recomendacion {
			out = append(out, v)
		}
	}
	return out
}

// Recomendaciones devuelve las violaciones que se informan sin bloquear.
func Recomendaciones(violations []Violation) []Violation {
	var out []Violation
	for _, v := range violations {
		if v.Recomendacion {
			out = append(out, v)
		}
	}
	return out
}

// Score es el único puntaje de calidad de QDD: lo usan audit, certify,
// evolution, report, el dashboard y el MCP. Una violación bloqueante resta 10
// y una recomendación 2, con piso en 0.
func Score(violations []Violation) int {
	bloqueantes := len(Bloqueantes(violations))
	score := 100 - bloqueantes*10 - (len(violations)-bloqueantes)*2
	if score < 0 {
		return 0
	}
	return score
}
