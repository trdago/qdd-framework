package audit

import (
	"context"
	"fmt"
	"os"

	"github.com/qdd-framework/qdd/pkg/qcl/wisdom"
)

// Violation define la estructura base para cualquier problema encontrado
// en las 5 certificaciones universales de QDD.
type Violation struct {
	Category    string
	RuleID      string
	Description string
	File        string
	Line        int
	// Recomendacion indica que se informa sin bloquear el gate (adopción gradual).
	Recomendacion bool
}

// Format muestra la violación de forma legible.
func (v Violation) Format() string {
	if v.File != "" && v.Line > 0 {
		return fmt.Sprintf("[%s] %s -> %s:%d", v.RuleID, v.Description, v.File, v.Line)
	}
	if v.File != "" {
		return fmt.Sprintf("[%s] %s -> %s", v.RuleID, v.Description, v.File)
	}
	return fmt.Sprintf("[%s] %s", v.RuleID, v.Description)
}

// EngineCoordinator orquesta todas las certificaciones
type EngineCoordinator struct {
	cwd             string
	pararAlBloquear bool
}

// ReglaPruebasPendientes dice que las pruebas no se corrieron porque una regla ya bloqueaba.
const ReglaPruebasPendientes = "QA-04-PRUEBAS-PENDIENTES"

func NewEngine(cwd string) *EngineCoordinator {
	return &EngineCoordinator{cwd: cwd}
}

// ParandoAlBloquear no corre las pruebas si una regla ya bloquea el gate (RFC 0020): quien itera arregla primero lo barato. Lo
// usan qdd audit y qdd_audit. El CI y --completo corren todo, y lo que mide (report, score) nunca se detiene.
func (e *EngineCoordinator) ParandoAlBloquear() *EngineCoordinator {
	e.pararAlBloquear = true
	return e
}

// RunAll ejecuta todas las certificaciones universales y devuelve la lista de violaciones.
func (e *EngineCoordinator) RunAll() []Violation {
	var allViolations []Violation

	// Load Dynamic Policies
	p := LoadPolicies(e.cwd)

	wClient := wisdom.NewClient(e.cwd)
	manifest, _ := wClient.FetchRulesManifest(context.Background())
	if manifest != nil && len(manifest.Rules) > 0 {
		fmt.Fprintf(os.Stderr, "[+] Engine: %d reglas remotas dinámicas cargadas desde Cloud Wisdom (v%s)\n", len(manifest.Rules), manifest.Version)
	}

	e.runCoreChecks(p, &allViolations)
	e.runAdvancedChecks(p, &allViolations)
	allViolations = append(allViolations, e.pruebasOPendientes(p, allViolations)...)

	// Filtro de supresión Enterprise (Grandfathering)
	allViolations = FilterIgnoredViolations(allViolations)
	marcarRecomendaciones(p, allViolations)

	return allViolations
}

// pruebasOPendientes corre las pruebas, que van al final porque son lo caro, salvo que ya se sepa que el gate no aprueba.
func (e *EngineCoordinator) pruebasOPendientes(p QDDPolicies, reglas []Violation) []Violation {
	bloquean := bloqueantesDeReglas(p, reglas)
	if !e.pararAlBloquear || bloquean == 0 || os.Getenv("CI") != "" || os.Getenv(envPruebasCompletas) != "" {
		return RunCoverageCheck(e.cwd)
	}
	fmt.Fprintf(os.Stderr, "[QA] PRUEBAS PENDIENTES: %d regla(s) ya bloquean el gate; corrígelas primero o usa qdd audit --completo\n", bloquean)
	return []Violation{{
		Category:      "COVERAGE",
		RuleID:        ReglaPruebasPendientes,
		Description:   "No se corrieron las pruebas porque una regla ya bloquea el gate. Corrígela primero, o usa qdd audit --completo para verlo todo de una vez.",
		Recomendacion: true,
	}}
}

// bloqueantesDeReglas cuenta sobre una copia: lo mismo que hará RunAll al final, sin tocar la lista.
func bloqueantesDeReglas(p QDDPolicies, reglas []Violation) int {
	copia := FilterIgnoredViolations(append([]Violation{}, reglas...))
	marcarRecomendaciones(p, copia)
	return len(Bloqueantes(copia))
}

func (e *EngineCoordinator) runCoreChecks(p QDDPolicies, violations *[]Violation) {
	if p.OWASP {
		*violations = append(*violations, RunOwaspCheck(e.cwd)...)
	}

	if p.CleanCode {
		*violations = append(*violations, e.processCleanCodeViolations(p)...)
	}

	*violations = append(*violations, RunTwelveFactorCheck(e.cwd)...)
	if p.DeployGateEnabled() {
		*violations = append(*violations, RunDeployGateCheck(e.cwd)...)
	}
	if p.VersioningEnabled() {
		*violations = append(*violations, RunVersioningCheck(e.cwd)...)
		*violations = append(*violations, RunUsoReleaseCheck(e.cwd, p.UsoMinimoHoras)...)
	}
	*violations = append(*violations, RunModuleHeartCheck(e.cwd, p.ModuleHeart)...)
	*violations = append(*violations, RunBlueprintCheck(e.cwd)...)
	*violations = append(*violations, RunHerramientasCheck(e.cwd)...)
	*violations = append(*violations, RunDockerDesarrolloCheck(e.cwd)...)
	*violations = append(*violations, RunEstadoCheck(e.cwd)...)
	*violations = append(*violations, RunFindingsCheck(e.cwd)...)
	*violations = append(*violations, CheckDatabasePerformance(e.cwd)...)
	if p.DataUniqueness {
		*violations = append(*violations, CheckDataUniqueness(e.cwd)...)
	}
}

func (e *EngineCoordinator) runAdvancedChecks(p QDDPolicies, violations *[]Violation) {
	if p.Traceability {
		*violations = append(*violations, RunTraceabilityCheck(e.cwd)...)
	}

	if p.BeyondLimits {
		*violations = append(*violations, RunBeyondLimitsCheck(e.cwd)...)
	}

	if p.Enterprise {
		*violations = append(*violations, RunEnterpriseCheck(e.cwd)...)
	}

	if p.AIGovernance {
		*violations = append(*violations, RunAIGovernanceCheck(e.cwd)...)
	}
}

func (e *EngineCoordinator) processCleanCodeViolations(p QDDPolicies) []Violation {
	var results []Violation
	ccViolations := RunCleanCodeCheck(e.cwd)
	for _, v := range ccViolations {
		if !p.ZeroElse && v.RuleID == "CLEAN-01-NO-ELSE" {
			continue
		}
		results = append(results, v)
	}
	return results
}
