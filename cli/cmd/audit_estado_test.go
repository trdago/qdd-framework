package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Criterio de éxito de la ficha estado: «qdd audit bloquea un estado.yaml que no cumple EPF-C01…C07». Se prueba de caja
// negra: «qdd audit» de verdad (otro proceso, con sus hooks y su os.Exit) en un proyecto que, salvo el estado, aprueba.

const estadoConEvidencia = `formato: 1
etapas: [Código, Pruebas]
frentes:
  - id: a
    nombre: A
    estado: abierto
    celdas:
      Código: {estado: "✅", evidencia: "commit e29393e"}
      Pruebas: {estado: "—"}
`

// proyectoQueApruebaElAudit arma un proyecto que pasa «qdd audit» con el estado que se le da.
func proyectoQueApruebaElAudit(t *testing.T, estadoYAML string) string {
	t.Helper()
	dir := t.TempDir()
	archivos := map[string]string{
		".qdd/config.yaml":         "project: prueba\nlanguages: [Go]\narchitecture: CLI\ntest_commands: [\"true\"]\n",
		".qdd/project/estado.yaml": estadoYAML,
		"docs/adr/.keep":           "",
		"e2e/.keep":                "",
	}
	for nombre, contenido := range archivos {
		camino := filepath.Join(dir, nombre)
		if err := os.MkdirAll(filepath.Dir(camino), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(camino, []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git := exec.Command("git", "init", "-q", dir)
	git.Env = sinGitDelPadre()
	if salida, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, salida)
	}
	return dir
}

// sinGitDelPadre quita GIT_DIR y compañía: un hook de git que corre las pruebas no debe apuntar al proyecto de prueba
// al repositorio de QDD.
func sinGitDelPadre() []string {
	var env []string
	for _, v := range os.Environ() {
		if strings.HasPrefix(v, "GIT_") {
			continue
		}
		env = append(env, v)
	}
	return env
}

// correrAudit corre «qdd audit» en dir como otro proceso y devuelve su salida y su código.
func correrAudit(t *testing.T, dir string) (string, int) {
	t.Helper()
	c := exec.Command(os.Args[0], "-test.run=^TestAuditAyudante$")
	c.Env = append(sinGitDelPadre(), "QDD_AUDIT_AYUDANTE="+dir, "QDD_HOME="+t.TempDir())
	salida, err := c.CombinedOutput()
	if err == nil {
		return string(salida), 0
	}
	salio, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("no corrió: %v", err)
	}
	return string(salida), salio.ExitCode()
}

// TestAuditAyudante es «qdd audit» dentro del proceso hijo de correrAudit; solo, no hace nada.
func TestAuditAyudante(t *testing.T) {
	dir := os.Getenv("QDD_AUDIT_AYUDANTE")
	if dir == "" {
		t.Skip("solo corre como proceso hijo de correrAudit")
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	rootCmd.SetArgs([]string{"audit"})
	Execute()
	os.Exit(0)
}

func TestAudit_BloqueaUnEstadoQueNoCumpleEPF(t *testing.T) {
	sano := proyectoQueApruebaElAudit(t, estadoConEvidencia)
	if salida, codigo := correrAudit(t, sano); codigo != 0 {
		t.Fatalf("el proyecto de prueba aprueba con un estado válido (código %d):\n%s", codigo, salida)
	}
	sinEvidencia := strings.Replace(estadoConEvidencia, `, evidencia: "commit e29393e"`, "", 1)
	roto := proyectoQueApruebaElAudit(t, sinEvidencia)
	salida, codigo := correrAudit(t, roto)
	if codigo == 0 {
		t.Fatalf("un ✅ sin evidencia en estado.yaml bloquea «qdd audit»:\n%s", salida)
	}
	if !strings.Contains(salida, "[EPF-01-CHECK]") || !strings.Contains(salida, "EPF-C02") {
		t.Fatalf("«qdd audit» nombra la regla que bloquea:\n%s", salida)
	}
}
