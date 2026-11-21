package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// Los verificadores reales leen sin escribir: se prueban con un programa falso en vez de npm, gh y git.

func conPrograma(t *testing.T, salida string, err error) *[][]string {
	t.Helper()
	var llamadas [][]string
	antes := correrVerificacion
	correrVerificacion = func(dir, programa string, args ...string) (string, error) {
		llamadas = append(llamadas, append([]string{programa}, args...))
		return salida, err
	}
	t.Cleanup(func() { correrVerificacion = antes })
	return &llamadas
}

func TestVerificarNpm_LaVersionPublicada(t *testing.T) {
	llamadas := conPrograma(t, "2.10.0-rc.43", nil)
	ok, visto, err := verificarNpm(t.TempDir(), "qdd-framework@2.10.0-rc.43")
	if err != nil || !ok || visto != "npm tiene qdd-framework@2.10.0-rc.43" {
		t.Fatalf("npm la tiene: %v %q %v", ok, visto, err)
	}
	if (*llamadas)[0][1] != "view" {
		t.Fatalf("solo lee: %v", *llamadas)
	}
	conPrograma(t, "", errors.New("E404"))
	if ok, _, err := verificarNpm(t.TempDir(), "qdd-framework@9.9.9"); ok || err != nil {
		t.Fatalf("una versión que npm no tiene es «todavía no», no un error: %v %v", ok, err)
	}
}

func TestVerificarTag_EnOrigin(t *testing.T) {
	conPrograma(t, "abc\trefs/tags/v1", nil)
	if ok, _, err := verificarTag(t.TempDir(), "v1"); !ok || err != nil {
		t.Fatalf("el tag está: %v %v", ok, err)
	}
	conPrograma(t, "", nil)
	if ok, _, _ := verificarTag(t.TempDir(), "v2"); ok {
		t.Fatal("sin tag, todavía no")
	}
}

func TestVerificarArchivo_SoloDentroDelProyecto(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "listo.txt"), []byte("x"), 0o600)
	if ok, _, _ := verificarArchivo(root, "listo.txt"); !ok {
		t.Fatal("el archivo existe")
	}
	if ok, _, _ := verificarArchivo(root, "../../../../../../etc/passwd"); ok {
		t.Fatal("una ruta que sale del proyecto se mira dentro de él")
	}
}
