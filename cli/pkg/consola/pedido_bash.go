package consola

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// Rule: Prohibido usar 'else'
// FND-197 (MEJ-058): en Windows un pedido corre en cmd.exe, y ahí «bash» es el lanzador de WSL (System32), no el bash
// de Git que usa la sesión de Claude Code. El pedido lleva la carpeta de ese bash primero en su PATH, así «bash x.sh»
// corre donde la IA lo probó. Y lo que escribe wsl.exe (UTF-16LE) se guarda legible.

// entornoDelPedido pone la carpeta de bash primero en el PATH del pedido, solo en Windows y si hay bash de Git. En
// Windows el nombre de la variable no distingue mayúsculas (suele ser «Path»).
func entornoDelPedido(env []string, goos, bash string) []string {
	if goos != "windows" || bash == "" {
		return env
	}
	out := append([]string{}, env...)
	for i, v := range out {
		nombre, valor, ok := strings.Cut(v, "=")
		if !ok || !strings.EqualFold(nombre, "PATH") {
			continue
		}
		out[i] = nombre + "=" + carpetaWindows(bash) + ";" + valor
		return out
	}
	return append(out, "PATH="+carpetaWindows(bash))
}

// carpetaWindows es la carpeta de una ruta de Windows aunque qdd corra en otro sistema (las pruebas).
func carpetaWindows(ruta string) string {
	i := strings.LastIndexAny(ruta, `\/`)
	if i < 0 {
		return "."
	}
	return ruta[:i]
}

// bashDeGit es el bash que usa Claude Code en Windows: el de CLAUDE_CODE_GIT_BASH_PATH si existe; si no, el de la
// instalación de Git (Git\cmd\git.exe o Git\mingw64\bin\git.exe → Git\bin\bash.exe). Vacío si no lo encuentra.
func bashDeGit(getenv func(string) string, buscar func(string) (string, error), existe func(string) bool) string {
	if propio := getenv("CLAUDE_CODE_GIT_BASH_PATH"); propio != "" && existe(propio) {
		return propio
	}
	git, err := buscar("git")
	if err != nil {
		return ""
	}
	dir := filepath.Dir(git)
	for range 3 {
		dir = filepath.Dir(dir)
		bash := filepath.Join(dir, "bin", "bash.exe")
		if existe(bash) {
			return bash
		}
	}
	return ""
}

func bashDeEsteEquipo() string {
	existe := func(r string) bool {
		info, err := os.Stat(r)
		return err == nil && !info.IsDir()
	}
	return bashDeGit(os.Getenv, exec.LookPath, existe)
}

// salidaLegible decodifica la salida que viene en UTF-16LE (con BOM, o con un byte nulo en casi todas las posiciones
// impares, como la de wsl.exe); lo demás queda como está.
func salidaLegible(b []byte) string {
	if bytes.HasPrefix(b, []byte{0xFF, 0xFE}) {
		return desdeUTF16LE(b[2:])
	}
	if !pareceUTF16LE(b) {
		return string(b)
	}
	return desdeUTF16LE(b)
}

func pareceUTF16LE(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	nulos := 0
	for i := 1; i < len(b); i += 2 {
		if b[i] == 0 {
			nulos++
		}
	}
	return nulos*10 >= (len(b)/2)*8
}

func desdeUTF16LE(b []byte) string {
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u = append(u, uint16(b[i])|uint16(b[i+1])<<8)
	}
	return string(utf16.Decode(u))
}
