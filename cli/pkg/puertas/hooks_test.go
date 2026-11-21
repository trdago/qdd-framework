package puertas

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalarHooks_CreaLosQueFaltan(t *testing.T) {
	dir := repoDePrueba(t)
	cambios, err := InstalarHooks(dir)
	if err != nil || len(cambios) != 2 {
		t.Fatalf("crea pre-commit y commit-msg: %v %v", cambios, err)
	}
	for _, hook := range []string{"pre-commit", "commit-msg"} {
		ruta := filepath.Join(dir, ".git", "hooks", hook)
		datos, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(datos), "#!/bin/sh\n") || !strings.Contains(string(datos), "qdd puertas hook "+hook) {
			t.Errorf("%s:\n%s", hook, datos)
		}
		if info, _ := os.Stat(ruta); info.Mode()&0o111 == 0 {
			t.Errorf("%s queda ejecutable", hook)
		}
	}
	if cambios, _ := InstalarHooks(dir); len(cambios) != 0 {
		t.Fatalf("instalar dos veces no cambia nada: %v", cambios)
	}
	for _, e := range EstadoHooks(dir) {
		if e.Estado != HookAlDia {
			t.Errorf("recién instalado está al día: %+v", e)
		}
	}
}

// Los hooks conviven con lo que ya hay: el bloque va después del shebang (un exit 0 al final no lo salta) y lo ajeno
// queda igual.
// contrato: puertas#77eb9692
func TestInstalarHooks_NoTocaLoAjeno(t *testing.T) {
	dir := repoDePrueba(t)
	ajeno := "#!/usr/bin/env bash\nnpx lint-staged\nexit 0\n"
	escribir(t, dir, ".git/hooks/pre-commit", ajeno)
	viejo := "#!/bin/sh\n" + marcaInicio + "\nqdd puertas hook commit-msg-viejo\n" + marcaFin + "\necho mío\n"
	escribir(t, dir, ".git/hooks/commit-msg", viejo)
	estados := map[string]string{}
	for _, e := range EstadoHooks(dir) {
		estados[e.Hook] = e.Estado
	}
	if estados["pre-commit"] != HookFalta || estados["commit-msg"] != HookViejo {
		t.Fatalf("diagnostica falta y viejo: %v", estados)
	}
	if _, err := InstalarHooks(dir); err != nil {
		t.Fatal(err)
	}
	pre, _ := os.ReadFile(filepath.Join(dir, ".git/hooks/pre-commit"))
	if !strings.HasPrefix(string(pre), "#!/usr/bin/env bash\n"+marcaInicio) || !strings.HasSuffix(string(pre), "npx lint-staged\nexit 0\n") {
		t.Fatalf("el bloque va tras el shebang y lo ajeno sigue:\n%s", pre)
	}
	msg, _ := os.ReadFile(filepath.Join(dir, ".git/hooks/commit-msg"))
	if strings.Contains(string(msg), "commit-msg-viejo") || strings.Count(string(msg), marcaInicio) != 1 || !strings.HasSuffix(string(msg), "echo mío\n") {
		t.Fatalf("el bloque viejo se reemplaza una vez y lo ajeno sigue:\n%s", msg)
	}
}

func TestInstalarHooks_ConHusky(t *testing.T) {
	dir := repoDePrueba(t)
	git(t, dir, "config", "core.hooksPath", ".husky/_")
	escribir(t, dir, ".husky/pre-commit", "npm test\n")
	if _, err := InstalarHooks(dir); err != nil {
		t.Fatal(err)
	}
	pre, _ := os.ReadFile(filepath.Join(dir, ".husky/pre-commit"))
	if !strings.HasPrefix(string(pre), marcaInicio) || !strings.HasSuffix(string(pre), "npm test\n") {
		t.Fatalf("con husky va en .husky/<hook>, sin shebang si no tenía:\n%s", pre)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git/hooks/pre-commit")); !os.IsNotExist(err) {
		t.Fatal("con husky no se escribe en .git/hooks")
	}
}

// Husky deja core.hooksPath absoluto, apuntando a otro checkout: los hooks van en el .husky/ de este.
func TestInstalarHooks_ConHuskyAbsoluto(t *testing.T) {
	dir := repoDePrueba(t)
	otro := t.TempDir()
	git(t, dir, "config", "core.hooksPath", filepath.Join(otro, ".husky", "_"))
	if _, err := InstalarHooks(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".husky", "commit-msg")); err != nil {
		t.Fatalf("va en el .husky de este worktree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(otro, ".husky")); !os.IsNotExist(err) {
		t.Fatal("no toca el otro checkout")
	}
}

func TestMensajeSinComentarios(t *testing.T) {
	got := MensajeSinComentarios("fix: x (FND-1)\n\n# Please enter the commit message\n# On branch main\n")
	if got != "fix: x (FND-1)" {
		t.Fatalf("%q", got)
	}
}

// Invariante: una puerta que no puede decidir deja pasar. El hook corre de verdad con sh -e (como husky) y un qdd falso.
// contrato: puertas#91d6f84a
func TestHook_SoloElRechazoCortaElCommit(t *testing.T) {
	dir := repoDePrueba(t)
	if _, err := InstalarHooks(dir); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	casos := map[string]bool{"exit 0": true, "exit 1": true, "echo 'Error: unknown command' >&2; exit 1": true, "exit 2": false}
	for qdd, pasa := range casos {
		escribir(t, bin, "qdd", "#!/bin/sh\n"+qdd+"\n")
		os.Chmod(filepath.Join(bin, "qdd"), 0o755)
		c := exec.Command("sh", "-e", filepath.Join(dir, ".git", "hooks", "pre-commit"))
		c.Env = append(os.Environ(), "PATH="+bin+":/usr/bin:/bin")
		if err := c.Run(); (err == nil) != pasa {
			t.Errorf("qdd que hace %q: pasa=%v, err=%v", qdd, pasa, err)
		}
	}
	c := exec.Command("sh", "-e", filepath.Join(dir, ".git", "hooks", "commit-msg"), "msg")
	c.Env = append(os.Environ(), "PATH=/nada")
	if err := c.Run(); err != nil {
		t.Errorf("sin qdd en el PATH, pasa: %v", err)
	}
}

func TestHooks_SeResuelvenEnLaRaiz(t *testing.T) {
	dir := repoDePrueba(t)
	escribir(t, dir, "sub/x.go", "package x\n")
	if EsRaiz(filepath.Join(dir, "sub")) || !EsRaiz(dir) {
		t.Fatal("solo la raíz del repositorio es raíz")
	}
	git(t, dir, "config", "core.hooksPath", ".husky/_")
	if _, err := InstalarHooks(filepath.Join(dir, "sub")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", ".husky")); !os.IsNotExist(err) {
		t.Fatal("desde una subcarpeta, los hooks van en la raíz")
	}
	if _, err := os.Stat(filepath.Join(dir, ".husky", "pre-commit")); err != nil {
		t.Fatal(err)
	}
}
