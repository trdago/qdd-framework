package audit

import (
	"fmt"

	"github.com/qdd-framework/qdd/pkg/uso"
)

// Rule: Prohibido usar 'else'
// VER-05-UNPROVEN-RELEASE (pedido del usuario, 2026-10-30): con la política uso_minimo_horas, una versión que no es
// un fix (minor o major) sale primero como preview (vX.Y.Z-rc.N, npm --tag preview) y pasa a latest solo con la
// evidencia de las horas de uso real medidas (.qdd/project/evidence/uso/vX.Y.Z.yaml, de 'qdd uso evidencia'). Un fix
// y una preview no esperan.

// RunUsoReleaseCheck aplica VER-05 a la versión declarada, si se está publicando.
func RunUsoReleaseCheck(cwd string, minimo float64) []Violation {
	if minimo <= 0 {
		return nil
	}
	source, raw := declaredVersion(cwd)
	current, ok := parseSemver(raw)
	if !ok || current.isPre {
		return nil
	}
	tags := mergedSemverTags(cwd)
	if len(tags) == 0 || !releasePending(cwd, current, tags) {
		return nil
	}
	base, ok := baseTag(tags, current)
	if !ok || actualBump(base, current) < bumpMinor {
		return nil
	}
	ev, err := uso.LeerEvidencia(cwd, current.String())
	if err != nil {
		return []Violation{versionViolation("VER-05-UNPROVEN-RELEASE", source, fmt.Sprintf(
			"%s no es un fix y no tiene uso medido: publícala primero como preview (v%s-rc.1), úsala %.0f h en proyectos reales y deja las horas con 'qdd uso evidencia %s'.",
			current.raw, current, minimo, current))}
	}
	if ev.Horas < minimo {
		return []Violation{versionViolation("VER-05-UNPROVEN-RELEASE", uso.RutaEvidencia+"/v"+current.String()+".yaml", fmt.Sprintf(
			"%s tiene %.2f h de uso real medido y la política pide %.0f h antes de pasar a latest: sigue usando la preview.",
			current.raw, ev.Horas, minimo))}
	}
	return nil
}
