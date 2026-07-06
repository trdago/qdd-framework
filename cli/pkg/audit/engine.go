package audit

import (
	"fmt"
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

	// 1. OWASP
	if p.OWASP {
		allViolations = append(allViolations, RunOwaspCheck(e.cwd)...)
	}
	
	// 2. Clean Code
	if p.CleanCode {
		ccViolations := RunCleanCodeCheck(e.cwd)
		for _, v := range ccViolations {
			// ZeroElse toggle overrides the NO-ELSE rule
			if !p.ZeroElse && v.RuleID == "CLEAN-01-NO-ELSE" {
				continue
			}
			allViolations = append(allViolations, v)
		}
	}

	// 3. Twelve Factor
	allViolations = append(allViolations, RunTwelveFactorCheck(e.cwd)...)

	// 4. Coverage
	allViolations = append(allViolations, RunCoverageCheck(e.cwd)...)

	// 5. Traceability
	if p.Traceability {
		allViolations = append(allViolations, RunTraceabilityCheck(e.cwd)...)
	}
	
	// 6. Beyond Limits (NASA/Netflix/DoD)
	if p.BeyondLimits {
		allViolations = append(allViolations, RunBeyondLimitsCheck(e.cwd)...)
	}

	// 7. Enterprise Scale (Monolith/Complexity)
	if p.Enterprise {
		allViolations = append(allViolations, RunEnterpriseCheck(e.cwd)...)
	}

	// Filtro de supresión Enterprise (Grandfathering)
	allViolations = FilterIgnoredViolations(allViolations)

	return allViolations
}
