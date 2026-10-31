package audit

import (
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// VER-05 (pedido del usuario, 2026-10-30): una versión que no es un fix sale primero como preview (vX.Y.Z-rc.N) y
// pasa a latest recién con las horas de uso real medidas que pide la política uso_minimo_horas.

func releaseMinor(t *testing.T) string {
	root, run := releasedAt(t, "2.7.0")
	run("commit", "-q", "--allow-empty", "-m", "feat(uso): medidor de uso")
	setVersion(t, root, "2.8.0")
	run("commit", "-qam", "chore(release): v2.8.0")
	return root
}

func evidenciaDeUso(t *testing.T, root, version, horas string) {
	writeFile(t, filepath.Join(root, ".qdd", "project", "evidence", "uso", "v"+version+".yaml"), "version: "+version+"\nhoras: "+horas+"\n")
}

func TestUsoRelease_UnaMinorSinHorasMedidasNoPasaALatest(t *testing.T) {
	root := releaseMinor(t)
	if v := RunUsoReleaseCheck(root, 5); !hasRule(v, "VER-05-UNPROVEN-RELEASE") {
		t.Fatalf("sin evidencia de uso se bloquea: %v", v)
	}
	evidenciaDeUso(t, root, "2.8.0", "3.5")
	if v := RunUsoReleaseCheck(root, 5); !hasRule(v, "VER-05-UNPROVEN-RELEASE") {
		t.Fatalf("3.5 h no alcanzan las 5: %v", v)
	}
	evidenciaDeUso(t, root, "2.8.0", "5.2")
	if v := RunUsoReleaseCheck(root, 5); len(v) != 0 {
		t.Fatalf("con 5.2 h medidas pasa: %v", v)
	}
	if v := RunUsoReleaseCheck(releaseMinor(t), 0); len(v) != 0 {
		t.Fatalf("sin la política no se exige: %v", v)
	}
}

func TestUsoRelease_UnFixYUnaPreviewNoEsperan(t *testing.T) {
	root, run := releasedAt(t, "2.7.0")
	run("commit", "-q", "--allow-empty", "-m", "fix(estado): la barra")
	setVersion(t, root, "2.7.1")
	run("commit", "-qam", "chore(release): v2.7.1")
	if v := RunUsoReleaseCheck(root, 5); len(v) != 0 {
		t.Fatalf("un fix sale directo: %v", v)
	}
	pre, run2 := releasedAt(t, "2.7.0")
	run2("commit", "-q", "--allow-empty", "-m", "feat(uso): medidor de uso")
	setVersion(t, pre, "2.8.0-rc.1")
	run2("commit", "-qam", "chore(release): v2.8.0-rc.1")
	if v := RunUsoReleaseCheck(pre, 5); len(v) != 0 {
		t.Fatalf("una preview sale para medir su uso: %v", v)
	}
}
