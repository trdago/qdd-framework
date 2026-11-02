package audit

import "github.com/qdd-framework/qdd/pkg/blueprints"

// RunBlueprintCheck verifica la integridad y vigencia de los blueprints del
// proyecto (ver blueprints.Check) y la conformidad de los módulos que los
// aplican o implementan (ver blueprints.CheckConformidad). Sin blueprints
// importados ni módulos que los implementen no reporta nada.
func RunBlueprintCheck(cwd string) []Violation {
	var violations []Violation
	bloquean, sinMedir := blueprints.CheckConformidad(cwd)
	for _, p := range append(blueprints.Check(cwd), bloquean...) {
		violations = append(violations, violacionDe(p, false))
	}
	for _, p := range append(sinMedir, blueprints.Esperas(cwd)...) {
		violations = append(violations, violacionDe(p, true))
	}
	return violations
}

func violacionDe(p blueprints.Problema, recomendacion bool) Violation {
	return Violation{Category: "BLUEPRINT", RuleID: p.Regla, Description: p.Descripcion, File: p.Archivo, Recomendacion: recomendacion}
}
