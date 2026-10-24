package audit

import "github.com/qdd-framework/qdd/pkg/blueprints"

// RunBlueprintCheck verifica la integridad y vigencia de los blueprints del
// proyecto (ver blueprints.Check). Sin blueprints importados no reporta nada.
func RunBlueprintCheck(cwd string) []Violation {
	var violations []Violation
	for _, p := range blueprints.Check(cwd) {
		violations = append(violations, Violation{Category: "BLUEPRINT", RuleID: p.Regla, Description: p.Descripcion, File: p.Archivo})
	}
	return violations
}
