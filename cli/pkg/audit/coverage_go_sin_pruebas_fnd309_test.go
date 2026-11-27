package audit

import (
	"path/filepath"
	"testing"
)

// FND-309: un proyecto Go sin ningún _test.go pasaba el gate: «go test ./...» sale con 0 sin pruebas. Ahora es QA-00,
// como en npm y pytest. Bloquea en los proyectos que qdd init crea desde este cambio (go_sin_pruebas: bloquea en
// policies.yaml) y avisa en los que ya usaban QDD (el modo goal eligió la recomendada, no la persona, 2026-11-26).
// contrato: gate-pruebas#c2360cac

func proyectoGoSinPruebas(t *testing.T, politicas string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main() {}\n")
	if politicas != "" {
		writeFile(t, filepath.Join(dir, ".qdd", "policies.yaml"), politicas)
	}
	return dir
}

func reglaQA00(violations []Violation) (Violation, bool) {
	for _, v := range violations {
		if v.RuleID == "QA-00-NO-TEST-COMMAND" {
			return v, true
		}
	}
	return Violation{}, false
}

func TestCoverage_GoSinPruebasEnUnProyectoNuevoBloquea_FND309(t *testing.T) {
	dir := proyectoGoSinPruebas(t, "go_sin_pruebas: bloquea\n")
	v, ok := reglaQA00(RunCoverageCheck(dir))
	if !ok || v.Recomendacion {
		t.Fatalf("un proyecto nuevo sin pruebas no pasa el gate: %+v", v)
	}
}

func TestCoverage_GoSinPruebasEnUnProyectoQueYaUsabaQDDAvisa_FND309(t *testing.T) {
	dir := proyectoGoSinPruebas(t, "zero_else: true\n")
	v, ok := reglaQA00(RunCoverageCheck(dir))
	if !ok || !v.Recomendacion {
		t.Fatalf("en un proyecto que ya usaba QDD se avisa sin bloquear: %+v", v)
	}
	if got := labels(detectTestCommands(dir)); got != "" {
		t.Fatalf("sin pruebas no corre go test: «ok» sin pruebas no es una prueba que pasó: %q", got)
	}
}

// Con npm y Go, que npm tenga pruebas no tapa que Go no tiene ninguna.
func TestCoverage_GoSinPruebasJuntoAOtraSuiteTambienSeVe_FND309(t *testing.T) {
	dir := proyectoGoSinPruebas(t, "go_sin_pruebas: bloquea\n")
	writeFile(t, filepath.Join(dir, "package.json"), `{"scripts":{"test":"jest"}}`)
	v, ok := violacionGoSinPruebas(dir)
	if !ok || v.Recomendacion {
		t.Fatalf("Go sin pruebas se ve aunque haya otra suite: %+v", v)
	}
}

func TestCoverage_GoConUnaPruebaCorreGoTest_FND309(t *testing.T) {
	dir := proyectoGoSinPruebas(t, "go_sin_pruebas: bloquea\n")
	writeFile(t, filepath.Join(dir, "pkg", "a", "a_test.go"), "package a\n")
	if _, ok := violacionGoSinPruebas(dir); ok {
		t.Fatal("con un _test.go hay pruebas")
	}
	if got := labels(detectTestCommands(dir)); got != "go test ./..." {
		t.Fatalf("corre go test: %q", got)
	}
}

// Las pruebas en vendor/ o en carpetas ocultas son de otros: no cuentan.
func TestCoverage_PruebasDeVendorNoCuentan_FND309(t *testing.T) {
	dir := proyectoGoSinPruebas(t, "go_sin_pruebas: bloquea\n")
	writeFile(t, filepath.Join(dir, "vendor", "lib", "lib_test.go"), "package lib\n")
	writeFile(t, filepath.Join(dir, ".claude", "worktrees", "w", "x_test.go"), "package x\n")
	if _, ok := violacionGoSinPruebas(dir); !ok {
		t.Fatal("vendor y carpetas ocultas no son pruebas del proyecto")
	}
}

// Quien declara cómo correr sus pruebas en config.yaml decide: QDD no adivina.
func TestCoverage_ConPruebasDeclaradasNoSeMira_FND309(t *testing.T) {
	dir := proyectoGoSinPruebas(t, "go_sin_pruebas: bloquea\n")
	writeFile(t, filepath.Join(dir, ".qdd", "config.yaml"), "test_commands:\n  - comando: \"true\"\n")
	if _, ok := violacionGoSinPruebas(dir); ok {
		t.Fatal("con test_commands declarados no se mira")
	}
}

// En un monorepo (FND-047) el módulo Go de una subcarpeta sin pruebas también se ve.
func TestCoverage_GoSinPruebasEnUnaSubcarpeta_FND309(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "backend", "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "frontend", "package.json"), `{"scripts":{"test":"vitest run"}}`)
	if _, ok := violacionGoSinPruebas(dir); !ok {
		t.Fatal("backend/ sin pruebas se ve")
	}
}
