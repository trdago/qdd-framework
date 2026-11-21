package foto

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'

// catDelSistema se busca antes de que una prueba cambie el PATH.
var catDelSistema, _ = exec.LookPath("cat")

// lectorQueEntrega deja en un PATH propio (sin el del sistema) un wl-paste que entrega datos, y espías con los nombres
// de otros programas que anotan si alguien los corre.
func lectorQueEntrega(t *testing.T, datos []byte) (bin, llamadas string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("lector de prueba en sh")
	}
	cat := catDelSistema
	if cat == "" {
		t.Skip("sin cat")
	}
	bin = t.TempDir()
	llamadas = filepath.Join(bin, "llamadas.log")
	archivo := filepath.Join(bin, "datos")
	if err := os.WriteFile(archivo, datos, 0o644); err != nil {
		t.Fatal(err)
	}
	espia := func(nombre, despues string) {
		s := "#!/bin/sh\necho \"" + nombre + " $*\" >> '" + llamadas + "'\n" + despues
		if err := os.WriteFile(filepath.Join(bin, nombre), []byte(s), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	espia("wl-paste", "'"+cat+"' '"+archivo+"'\n")
	for _, otro := range []string{"xclip", "sh", "bash", "git", "qdd", "claude", "xdg-open", "convert", "magick", "notify-send", "powershell.exe", "osascript", "curl"} {
		espia(otro, "exit 0\n")
	}
	t.Setenv("PATH", bin)
	return bin, llamadas
}

func entregadas(t *testing.T, root string) []string {
	t.Helper()
	entradas, _ := os.ReadDir(filepath.Join(root, filepath.FromSlash(Carpeta)))
	var out []string
	for _, e := range entradas {
		if e.Name() != ".gitignore" {
			out = append(out, e.Name())
		}
	}
	return out
}

// Ni lo que no es PNG, JPEG, GIF o WebP ni lo que pasa de 5 MB sale de Traer: ni del portapapeles ni de una captura, y
// del portapapeles no queda el archivo.
// contrato: foto#94bda7c7
func TestTraer_SoloUnaImagenDeHasta5MBLlegaALaIA(t *testing.T) {
	bmp := append([]byte("BM"), bytes.Repeat([]byte{0}, 64)...)
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	grande := append(append([]byte{}, png1x1...), bytes.Repeat([]byte{0}, MaxBytes)...)
	wayland := entorno("linux", map[string]string{"WAYLAND_DISPLAY": "w"}, t.TempDir())
	for nombre, datos := range map[string][]byte{"texto": []byte("sk-ant-no-soy-una-imagen"), "bmp": bmp, "svg": svg, "más de 5 MB": grande} {
		lectorQueEntrega(t, datos)
		root := t.TempDir()
		f, err := Traer(context.Background(), root, DePortapapeles, wayland, ahora)
		if err == nil || f.Datos != nil {
			t.Fatalf("%s del portapapeles no se entrega: %s %s %d", nombre, f.Ruta, f.Tipo, f.Bytes)
		}
		if quedan := entregadas(t, root); len(quedan) != 0 {
			t.Fatalf("%s no queda en tmp/fotos: %v", nombre, quedan)
		}

		home := t.TempDir()
		os.MkdirAll(filepath.Join(home, "Pictures"), 0o755)
		escribir(t, filepath.Join(home, "Pictures", "captura.png"), datos, ahora)
		if f, err := Traer(context.Background(), t.TempDir(), DeCaptura, entorno("linux", map[string]string{}, home), ahora); err == nil || f.Datos != nil {
			t.Fatalf("%s con nombre de captura no se entrega: %s %s %d", nombre, f.Ruta, f.Tipo, f.Bytes)
		}
	}

	imagenes := map[string][]byte{
		"image/png":  png1x1,
		"image/jpeg": append([]byte{0xff, 0xd8, 0xff, 0xe0}, bytes.Repeat([]byte{0}, 16)...),
		"image/gif":  append([]byte("GIF89a"), bytes.Repeat([]byte{0}, 16)...),
		"image/webp": append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte{0}, 16)...),
	}
	for tipo, datos := range imagenes {
		lectorQueEntrega(t, datos)
		f, err := Traer(context.Background(), t.TempDir(), DePortapapeles, wayland, ahora)
		if err != nil || f.Tipo != tipo || !bytes.Equal(f.Datos, datos) {
			t.Fatalf("%s sí se entrega: %s %s %v", tipo, f.Ruta, f.Tipo, err)
		}
	}
}

// Lo que se pega queda fuera del repositorio aunque el proyecto no ignore tmp/: git no lo ve.
// contrato: foto#a822eaed
func TestTraer_LoPegadoNuncaEntraAlRepositorio(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("lector de prueba en sh")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("sin git")
	}
	lectorQueEntrega(t, png1x1)
	root := t.TempDir()
	correrGit := func(args ...string) string {
		t.Helper()
		c := exec.Command(git, args...)
		c.Dir = root
		c.Env = append(gitenv.Limpio(os.Environ()), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "HOME="+root)
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return string(out)
	}
	correrGit("init", "-q")
	f, err := Traer(context.Background(), root, DePortapapeles, entorno("linux", map[string]string{"WAYLAND_DISPLAY": "w"}, t.TempDir()), ahora)
	if err != nil {
		t.Fatal(err)
	}
	if estado := correrGit("status", "--porcelain", "--untracked-files=all"); strings.TrimSpace(estado) != "" {
		t.Fatalf("git ve lo pegado en %s:\n%s", f.Ruta, estado)
	}
	correrGit("add", "-A")
	if indice := correrGit("ls-files"); strings.TrimSpace(indice) != "" {
		t.Fatalf("git add -A no lo agrega:\n%s", indice)
	}
}

// qdd foto solo corre el lector del portapapeles: ni una shell ni otro programa, y la última captura no corre nada.
// contrato: foto#fabe13dc
func TestTraer_SoloCorreElLectorDelPortapapeles(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, "Pictures"), 0o755)
	escribir(t, filepath.Join(home, "Pictures", "captura.png"), png1x1, ahora)
	env := entorno("linux", map[string]string{"WAYLAND_DISPLAY": "w"}, home)
	for _, datos := range [][]byte{png1x1, []byte("no es una imagen")} {
		_, llamadas := lectorQueEntrega(t, datos)
		if _, err := Traer(context.Background(), t.TempDir(), DeAuto, env, ahora); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(llamadas)
		if got := strings.TrimSpace(string(b)); got != "wl-paste --no-newline --type image/png" {
			t.Fatalf("solo el lector del portapapeles, una vez: %q", got)
		}
	}
	_, llamadas := lectorQueEntrega(t, png1x1)
	if _, err := Traer(context.Background(), t.TempDir(), DeCaptura, env, ahora); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(llamadas); len(b) != 0 {
		t.Fatalf("la captura no corre nada: %s", b)
	}
	if got := HayImagen(context.Background(), env, CorrerConsulta); got != HayNo {
		t.Fatalf("el wl-paste de prueba no lista una imagen: %s", got)
	}
	b, _ := os.ReadFile(llamadas)
	if got := strings.TrimSpace(string(b)); got != "wl-paste --list-types" {
		t.Fatalf("--hay solo pregunta los tipos al lector: %q", got)
	}
}
