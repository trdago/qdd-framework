package ambientes

// Rule: Prohibido usar 'else'

// Acciones de un paso del plan.
const (
	Crear   = "crear"
	Cambiar = "cambiar"
	Igual   = "igual"
)

// Paso es lo que haría levantar con un registro declarado; Actual es lo que hay hoy cuando se cambia.
type Paso struct {
	Accion   string    `json:"accion"`
	Registro Registro  `json:"registro"`
	Actual   *Registro `json:"actual,omitempty"`
}

// PlanDNS compara lo declarado con lo que hay, por tipo y nombre. Lo que existe y no se declara no aparece: levantar
// nunca borra lo que el archivo no nombra.
func PlanDNS(dns DNS, actuales []Registro) []Paso {
	hay := map[string]Registro{}
	for _, r := range actuales {
		hay[r.Tipo+" "+completo(r.Nombre, dns.Zona)] = r
	}
	var pasos []Paso
	for _, r := range dns.Registros {
		actual, ok := hay[r.Tipo+" "+r.Nombre]
		if !ok {
			pasos = append(pasos, Paso{Accion: Crear, Registro: r})
			continue
		}
		if actual.Valor == r.Valor && actual.Proxied == r.Proxied {
			pasos = append(pasos, Paso{Accion: Igual, Registro: r})
			continue
		}
		pasos = append(pasos, Paso{Accion: Cambiar, Registro: r, Actual: &actual})
	}
	return pasos
}
