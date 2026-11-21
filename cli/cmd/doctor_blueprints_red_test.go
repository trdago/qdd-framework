package cmd

// Rule: Prohibido usar 'else'

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// gitQueAnota pone en el PATH un git que anota cada llamada y sigue con el git de verdad; devuelve el archivo de notas.
func gitQueAnota(t *testing.T) string {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("sin git")
	}
	dir := t.TempDir()
	notas := filepath.Join(dir, "llamadas.log")
	script := "#!/bin/sh\necho \"$@\" >> " + notas + "\nexec " + real + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return notas
}

// En un proyecto sin blueprints (aunque tenga un remote), ni el diagnóstico ni la reparación del grupo de blueprints
// llaman a un comando de git que use la red.
// contrato: doctor#0b914dc4
func TestDoctor_SinBlueprintsElGrupoNoUsaLaRed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("el git que anota es un script de sh")
	}
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "git@github.com:acme/proyecto.git"}} {
		c := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	t.Chdir(root)
	notas := gitQueAnota(t)

	g := buildBlueprintChecks(root)
	for _, item := range g.Items {
		if !item.Success {
			t.Errorf("sin blueprints, %s falla: %s", item.Name, item.Error)
		}
	}
	repairBlueprints(root, g)
	blueprints.Reparar(root)

	data, _ := os.ReadFile(notas)
	for _, linea := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		for _, red := range []string{"ls-remote", "fetch", "clone", "pull", "push", "remote update"} {
			if strings.Contains(" "+linea+" ", " "+red+" ") {
				t.Errorf("sin blueprints el grupo usó la red: git %s", linea)
			}
		}
	}
}
