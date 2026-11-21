package cmd

// Rule: Prohibido usar 'else'

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/bugreport"
)

// FND-286: el arreglo del sandbox se copiaba con rsync --delete y, si el agente
// había borrado la evidencia en el sandbox, también se borraba la original.
// contrato: bugreport#63466c6d
// contrato: bugreport#45193dd3
func TestApplySandboxFix_ConservaLaEvidenciaYMarcaResuelto(t *testing.T) {
	if _, err := exec.LookPath("rsync"); err != nil {
		t.Skip("sin rsync")
	}
	cwd := t.TempDir()
	filed, err := bugreport.File(cwd, bugreport.Report{Command: "svc", ExitCode: 1, Output: "panic: boom"})
	if err != nil {
		t.Fatal(err)
	}
	sandbox := t.TempDir()
	if out, err := exec.Command("rsync", "-a", cwd+"/", sandbox+"/").CombinedOutput(); err != nil {
		t.Fatalf("rsync: %v %s", err, out)
	}
	// El agente «limpia» la evidencia en el sandbox y arregla el código.
	if err := os.RemoveAll(filepath.Join(sandbox, ".qdd", "project", "evidence")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sandbox, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	applySandboxFix(sandbox, cwd, []string{"svc"}, repairVerdict{Fixed: true, Summary: "arreglado", TestAdded: true}, filed)

	if _, err := os.Stat(filepath.Join(cwd, "main.go")); err != nil {
		t.Fatalf("el arreglo del sandbox no llegó al proyecto: %v", err)
	}
	data, err := os.ReadFile(filed.EvidencePath)
	if err != nil {
		t.Fatalf("la evidencia original se borró al aplicar el arreglo: %v", err)
	}
	if !strings.Contains(string(data), "panic: boom") {
		t.Errorf("la evidencia cambió: %q", data)
	}
	finding, err := os.ReadFile(filed.FindingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(finding), "status: resolved") {
		t.Errorf("tras la reparación confirmada el finding no quedó resuelto:\n%s", finding)
	}
}
