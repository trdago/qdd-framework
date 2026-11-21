package consola

import (
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-197 (MEJ-058): en Windows un pedido corre en cmd.exe, y ahí «bash» es C:\Windows\System32\bash.exe, el lanzador de
// WSL, no el bash de Git que usa la sesión. Con WSL detenido el pedido moría por timeout, y su error (UTF-16LE) se veía
// como «L\0a\0 \0o\0p…».

func TestEntornoDelPedido_EnWindowsBashEsElDeGit_FND197(t *testing.T) {
	bash := `C:\Program Files\Git\bin\bash.exe`
	env := []string{"USERNAME=ana", `Path=C:\Windows\system32;C:\Program Files\Git\cmd`}
	got := entornoDelPedido(env, "windows", bash)
	if got[1] != `Path=C:\Program Files\Git\bin;C:\Windows\system32;C:\Program Files\Git\cmd` || got[0] != "USERNAME=ana" {
		t.Fatalf("la carpeta del bash de Git va primero en el PATH (Path, sin importar mayúsculas): %v", got)
	}
	if got := entornoDelPedido(env, "linux", bash); got[1] != env[1] {
		t.Fatalf("fuera de Windows no cambia: %v", got)
	}
	if got := entornoDelPedido(env, "windows", ""); got[1] != env[1] {
		t.Fatalf("sin bash de Git no cambia: %v", got)
	}
}

func TestBashDeGit_LoBuscaJuntoAGit_FND197(t *testing.T) {
	git := filepath.Join("C:", "Git", "cmd", "git.exe")
	bash := filepath.Join("C:", "Git", "bin", "bash.exe")
	existe := func(r string) bool { return r == bash }
	sinEnv := func(string) string { return "" }
	if got := bashDeGit(sinEnv, func(string) (string, error) { return git, nil }, existe); got != bash {
		t.Fatalf("Git\\cmd\\git.exe → Git\\bin\\bash.exe: %q", got)
	}
	mingw := filepath.Join("C:", "Git", "mingw64", "bin", "git.exe")
	if got := bashDeGit(sinEnv, func(string) (string, error) { return mingw, nil }, existe); got != bash {
		t.Fatalf("Git\\mingw64\\bin\\git.exe → Git\\bin\\bash.exe: %q", got)
	}
	propio := filepath.Join("D:", "tools", "bash.exe")
	conEnv := func(k string) string {
		if k == "CLAUDE_CODE_GIT_BASH_PATH" {
			return propio
		}
		return ""
	}
	existeTodo := func(string) bool { return true }
	if got := bashDeGit(conEnv, func(string) (string, error) { return git, nil }, existeTodo); got != propio {
		t.Fatalf("el que Claude Code tiene configurado gana: %q", got)
	}
	if got := bashDeGit(sinEnv, func(string) (string, error) { return git, nil }, func(string) bool { return false }); got != "" {
		t.Fatalf("sin bash de Git, nada: %q", got)
	}
}

func TestSalidaLegible_UTF16DeWSLSeLee_FND197(t *testing.T) {
	wsl := []byte("L\x00a\x00 \x00o\x00p\x00e\x00r\x00a\x00c\x00i\x00\xf3\x00n\x00\r\x00\n\x00")
	if got := salidaLegible(wsl); got != "La operación\r\n" {
		t.Fatalf("UTF-16LE sin BOM: %q", got)
	}
	if got := salidaLegible(append([]byte{0xFF, 0xFE}, wsl...)); got != "La operación\r\n" {
		t.Fatalf("UTF-16LE con BOM: %q", got)
	}
	if got := salidaLegible([]byte("listo: operación hecha\n")); got != "listo: operación hecha\n" {
		t.Fatalf("UTF-8 queda igual: %q", got)
	}
}
