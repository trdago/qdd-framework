package audit

import (
	"context"
	"fmt"
	"os"

	"github.com/qdd-framework/qdd/pkg/contexto"
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
	// Principio, PorQue y Categoria dicen qué protege la regla y por qué
	// (RFC 0005 §3.1). Los completa el motor desde principios.yaml.
	Principio string
	PorQue    string
	Categoria string
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
	cwd string
}

func NewEngine(cwd string) *EngineCoordinator {
	return &EngineCoordinator{cwd: cwd}
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

	// Filtro de supresión Enterprise (Grandfathering)
	allViolations = FilterIgnoredViolations(allViolations)
	marcarRecomendaciones(p, allViolations)
	conPrincipio(allViolations)

	return allViolations
}

// Evaluar ejecuta la auditoría y dice además qué se pudo revisar en cada
// unidad del proyecto (RFC 0005 §3.4).
func (e *EngineCoordinator) Evaluar() Resultado {
	violations := e.RunAll()
	return Resultado{
		Violaciones: violations,
		Cobertura:   CalcularCobertura(e.cwd, LoadPolicies(e.cwd), contexto.Leer(e.cwd), violations),
	}
}

func (e *EngineCoordinator) runCoreChecks(p QDDPolicies, violations *[]Violation) {
	if p.OWASP {
		*violations = append(*violations, RunOwaspCheck(e.cwd)...)
	}

	if p.CleanCode {
		*violations = append(*violations, e.processCleanCodeViolations(p)...)
	}

	*violations = append(*violations, RunTwelveFactorCheck(e.cwd)...)
	*violations = append(*violations, RunCoverageCheck(e.cwd)...)
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
