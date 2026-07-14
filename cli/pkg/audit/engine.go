package audit

import (
	"context"
	"fmt"

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
		fmt.Printf("[+] Engine: %d reglas remotas dinámicas cargadas desde Cloud Wisdom (v%s)\n", len(manifest.Rules), manifest.Version)
	}

	e.runCoreChecks(p, &allViolations)
	e.runAdvancedChecks(p, &allViolations)

	// Filtro de supresión Enterprise (Grandfathering)
	allViolations = FilterIgnoredViolations(allViolations)

	return allViolations
}

func (e *EngineCoordinator) runCoreChecks(p QDDPolicies, violations *[]Violation) {
	*violations = append(*violations, RunMetaSafeguardCheck(e.cwd)...)
	*violations = append(*violations, RunWCAGCheck(e.cwd)...)
	*violations = append(*violations, RunISO9241Check(e.cwd)...)

	if p.OWASP {
		*violations = append(*violations, RunOwaspCheck(e.cwd)...)
	}

	if p.CleanCode {
		*violations = append(*violations, e.processCleanCodeViolations(p)...)
	}

	*violations = append(*violations, RunTwelveFactorCheck(e.cwd)...)
	*violations = append(*violations, RunCoverageCheck(e.cwd)...)
	*violations = append(*violations, CheckDatabasePerformance(e.cwd)...)
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
