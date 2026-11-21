package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// envComoQDD hace que el binario de pruebas sea qdd (TestMain): lo que se prueba es el código de salida real.
const envComoQDD = "QDD_PRUEBA_COMO_QDD"

// proyectoConSuiteRota es un proyecto que pasa las reglas baratas, salvo un finding sin estado, y cuya única suite
// deja una marca y sale con 3.
func proyectoConSuiteRota(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	archivos := map[string]string{
		".qdd/state.json":                    `{"status":"initialized"}`,
		".qdd/config.yaml":                   "project: p\nlanguages: [Shell]\narchitecture: CLI\ntest_commands: [\"touch corrio; exit 3\"]\n",
		".qdd/project/findings/FND-001.yaml": "title: sin estado\n",
		"docs/adr/0001-inicio.md":            "# Inicio\n",
		"e2e/README.md":                      "e2e\n",
	}
	for ruta, contenido := range archivos {
		escribir(t, filepath.Join(dir, ruta), contenido)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"commit", "-qm", "chore: inicio"}} {
		if out, err := gitDelProyecto(dir, args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return dir
}

func gitDelProyecto(dir string, args ...string) *exec.Cmd {
	c := exec.Command("git", append([]string{"-c", "user.name=qdd", "-c", "user.email=qdd@test", "-c", "commit.gpgsign=false"}, args...)...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	return c
}

// qddAudit corre «qdd audit» en dir con un HOME y un QDD_HOME temporales, fuera del CI, y devuelve salida y código.
func qddAudit(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	home := t.TempDir()
	c := exec.Command(os.Args[0], append([]string{"audit"}, args...)...)
	c.Dir = dir
	for _, v := range os.Environ() {
		nombre, _, _ := strings.Cut(v, "=")
		if nombre == "CI" || nombre == "QDD_PRUEBAS_COMPLETAS" || strings.HasPrefix(nombre, "CLAUDE") {
			continue
		}
		c.Env = append(c.Env, v)
	}
	c.Env = append(c.Env, envComoQDD+"=1", "HOME="+home, "USERPROFILE="+home, "QDD_HOME="+filepath.Join(home, "qdd"), "XDG_CONFIG_HOME="+filepath.Join(home, ".config"))
	out, err := c.CombinedOutput()
	var salida *exec.ExitError
	if errors.As(err, &salida) {
		return string(out), salida.ExitCode()
	}
	if err != nil {
		t.Fatalf("qdd audit no corrió: %v", err)
	}
	return string(out), 0
}

// contrato: gate-pruebas#9cf2ac1d
// Caja negra del código de salida: con una regla que ya bloquea, qdd audit no corre la suite (QA-04) y sale 1; con la
// regla arreglada corre la suite rota y sale 1 por QA-01; --completo la corre aunque otra regla bloquee.
func TestAudit_ParandoAlBloquearNuncaApruebaYSaleConUno(t *testing.T) {
	dir := proyectoConSuiteRota(t)
	marca := filepath.Join(dir, "corrio")
	salida, codigo := qddAudit(t, dir)
	if codigo != 1 || !strings.Contains(salida, "FND-01-SIN-ESTADO") || !strings.Contains(salida, "QA-04-PRUEBAS-PENDIENTES") {
		t.Fatalf("con una regla que bloquea sale 1 y dice que las pruebas quedaron pendientes: %d\n%s", codigo, salida)
	}
	if _, err := os.Stat(marca); err == nil {
		t.Fatalf("no corre la suite si ya bloquea otra regla:\n%s", salida)
	}

	salida, codigo = qddAudit(t, dir, "--completo")
	if _, err := os.Stat(marca); err != nil || codigo != 1 || !strings.Contains(salida, "QA-01-ALL-TESTS-PASS") {
		t.Fatalf("--completo corre la suite aunque otra regla bloquee: %d %v\n%s", codigo, err, salida)
	}
	os.Remove(marca)

	escribir(t, filepath.Join(dir, ".qdd", "project", "findings", "FND-001.yaml"), "title: sin estado\nstatus: open\n")
	salida, codigo = qddAudit(t, dir)
	if codigo != 1 || !strings.Contains(salida, "QA-01-ALL-TESTS-PASS") || strings.Contains(salida, "QA-04-PRUEBAS-PENDIENTES") {
		t.Fatalf("sin otra regla que bloquee, corre la suite rota y sale 1 por QA-01: %d\n%s", codigo, salida)
	}
	if _, err := os.Stat(marca); err != nil {
		t.Fatalf("la suite corrió: %v", err)
	}
}
