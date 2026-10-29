package audit

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/qdd-framework/qdd/pkg/contexto"
)

// EstadoCobertura dice si un principio se pudo revisar (RFC 0005 §3.4).
type EstadoCobertura string

const (
	Evaluado    EstadoCobertura = "evaluado"
	Parcial     EstadoCobertura = "parcial"
	NoEvaluado  EstadoCobertura = "no_evaluado"
	Desactivado EstadoCobertura = "desactivado"
)

// UnidadNoEvaluada es una unidad (o el repositorio) que un principio no pudo
// revisar, con el motivo y qué hacer.
type UnidadNoEvaluada struct {
	Ruta   string
	Motivo string
}

// CoberturaPrincipio es lo que se revisó de un principio.
type CoberturaPrincipio struct {
	Principio   Principio
	Estado      EstadoCobertura
	Evaluadas   []string
	NoEvaluadas []UnidadNoEvaluada
	Limite      string
	Violaciones int
}

// Cobertura resume qué principios se pudieron revisar. Un principio sin
// violaciones que no se evaluó no está aprobado: no se sabe.
type Cobertura struct {
	Principios   []CoberturaPrincipio
	Aplicables   int
	Evaluados    int
	Parciales    int
	NoEvaluados  int
	Desactivados int
}

// Resultado es lo que devuelve una auditoría: las violaciones y qué se revisó.
type Resultado struct {
	Violaciones []Violation
	Cobertura   Cobertura
}

// CalcularCobertura dice, para cada principio, en qué unidades del Contexto
// tiene un detector que lo revise. Solo informa: no cambia el puntaje ni lo
// que bloquea.
func CalcularCobertura(cwd string, p QDDPolicies, ctx contexto.Contexto, violations []Violation) Cobertura {
	porPrincipio := map[string]int{}
	for _, v := range violations {
		if pr, ok := PrincipioDeRegla(v.RuleID); ok {
			porPrincipio[pr.ID]++
		}
	}
	var c Cobertura
	for _, pr := range Principios() {
		cp := coberturaDePrincipio(cwd, p, ctx, pr)
		cp.Violaciones = porPrincipio[pr.ID]
		c.Principios = append(c.Principios, cp)
		sumarCobertura(&c, cp.Estado)
	}
	return c
}

func sumarCobertura(c *Cobertura, e EstadoCobertura) {
	if e == Desactivado {
		c.Desactivados++
		return
	}
	c.Aplicables++
	switch e {
	case Evaluado:
		c.Evaluados++
	case Parcial:
		c.Parciales++
	default:
		c.NoEvaluados++
	}
}

func coberturaDePrincipio(cwd string, p QDDPolicies, ctx contexto.Contexto, pr Principio) CoberturaPrincipio {
	d := detectores[pr.ID]
	cp := CoberturaPrincipio{Principio: pr, Limite: d.limite}
	if !politicasActivas(p, pr.Politicas) {
		cp.Estado = Desactivado
		return cp
	}
	switch pr.Alcance {
	case AlcanceUnidad:
		cubrirUnidades(&cp, ctx, d.stacks)
	case AlcancePruebas:
		cubrirPruebas(&cp, cwd, ctx)
	default:
		cubrirRepositorio(&cp, cwd, d)
	}
	cp.Estado = estadoDe(cp)
	return cp
}

func estadoDe(cp CoberturaPrincipio) EstadoCobertura {
	if len(cp.NoEvaluadas) == 0 {
		return Evaluado
	}
	if len(cp.Evaluadas) == 0 {
		return NoEvaluado
	}
	return Parcial
}

// cubrirUnidades revisa cada stack de cada unidad: una unidad Go y Node con un
// detector que solo lee Go queda evaluada en Go y no evaluada en Node.
func cubrirUnidades(cp *CoberturaPrincipio, ctx contexto.Contexto, stacks []string) {
	lee := map[string]bool{}
	for _, s := range stacks {
		lee[s] = true
	}
	for _, u := range ctx.Unidades {
		if len(u.Stacks) == 0 {
			cp.NoEvaluadas = append(cp.NoEvaluadas, UnidadNoEvaluada{Ruta: u.Ruta, Motivo: "no se reconoce su stack (ningún manifiesto conocido)"})
			continue
		}
		for _, s := range u.Stacks {
			if lee[s] {
				cp.Evaluadas = append(cp.Evaluadas, u.Ruta+" ("+s+")")
				continue
			}
			cp.NoEvaluadas = append(cp.NoEvaluadas, UnidadNoEvaluada{Ruta: u.Ruta, Motivo: "sin detector para " + s})
		}
	}
}

// cubrirPruebas: una unidad está evaluada si el gate ejecuta un comando de
// pruebas en su carpeta. Las pruebas de una subcarpeta con su propio
// manifiesto son de otra unidad.
func cubrirPruebas(cp *CoberturaPrincipio, cwd string, ctx contexto.Contexto) {
	conPruebas := map[string]bool{}
	for _, tc := range resolveTestCommands(cwd) {
		conPruebas[carpetaDeComando(cwd, tc)] = true
	}
	for _, u := range ctx.Unidades {
		if conPruebas[u.Ruta] {
			cp.Evaluadas = append(cp.Evaluadas, u.Ruta)
			continue
		}
		cp.NoEvaluadas = append(cp.NoEvaluadas, UnidadNoEvaluada{
			Ruta:   u.Ruta,
			Motivo: "el gate no ejecuta pruebas en esta unidad: declara su comando en test_commands de .qdd/config.yaml",
		})
	}
}

var prefijoCd = regexp.MustCompile(`^cd\s+("[^"]+"|'[^']+'|\S+)\s*&&`)

// carpetaDeComando devuelve la carpeta, relativa a la raíz, donde corre un
// comando de pruebas. Un comando declarado como "cd <ruta> && ..." corre en
// <ruta>.
func carpetaDeComando(cwd string, tc testCommand) string {
	dir := tc.Dir
	if m := prefijoCd.FindStringSubmatch(strings.TrimSpace(tc.Label)); m != nil {
		dir = filepath.Join(tc.Dir, strings.Trim(m[1], `"'`))
	}
	rel, err := filepath.Rel(cwd, dir)
	if err != nil {
		return "."
	}
	return filepath.ToSlash(rel)
}

func cubrirRepositorio(cp *CoberturaPrincipio, cwd string, d detector) {
	if d.evaluable == nil {
		cp.Evaluadas = []string{"."}
		return
	}
	ok, motivo := d.evaluable(cwd)
	if ok {
		cp.Evaluadas = []string{"."}
		return
	}
	cp.NoEvaluadas = []UnidadNoEvaluada{{Ruta: ".", Motivo: motivo}}
}

func politicasActivas(p QDDPolicies, claves []string) bool {
	for _, k := range claves {
		if !politicaActiva(p, k) {
			return false
		}
	}
	return true
}

func politicaActiva(p QDDPolicies, clave string) bool {
	activas := map[string]bool{
		"owasp":           p.OWASP,
		"clean_code":      p.CleanCode,
		"zero_else":       p.ZeroElse,
		"beyond_limits":   p.BeyondLimits,
		"traceability":    p.Traceability,
		"enterprise":      p.Enterprise,
		"ai_governance":   p.AIGovernance,
		"data_uniqueness": p.DataUniqueness,
		"deploy_gate":     p.DeployGateEnabled(),
	}
	return activas[clave]
}
