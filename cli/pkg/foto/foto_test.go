package foto

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Pedido del usuario (2026-10-31): una captura pegada en la consola llegó como bytes de PNG y la IA no la pudo ver.
// qdd foto trae la imagen del portapapeles o la última captura, en Linux, Windows y macOS, para que la IA la vea.

// png1x1 es un PNG válido de 1×1.
var png1x1 = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

var ahora = time.Date(2026, 10, 31, 0, 10, 0, 0, time.UTC)

func entorno(goos string, vars map[string]string, home string) Entorno {
	return Entorno{GOOS: goos, Home: home, Env: func(k string) string { return vars[k] }}
}

func nombres(ls []Lector) string {
	var out []string
	for _, l := range ls {
		out = append(out, l.Argv[0])
	}
	return strings.Join(out, ",")
}

func TestLectores_UnoPorSistema(t *testing.T) {
	casos := []struct {
		goos string
		vars map[string]string
		want string
	}{
		{"linux", map[string]string{"WAYLAND_DISPLAY": "wayland-1", "DISPLAY": ":0"}, "wl-paste,xclip"},
		{"linux", map[string]string{"DISPLAY": ":0"}, "xclip"},
		{"linux", map[string]string{}, ""},
		{"windows", map[string]string{}, "powershell.exe"},
		{"darwin", map[string]string{}, "osascript"},
	}
	for _, c := range casos {
		if got := nombres(Lectores(entorno(c.goos, c.vars, "/h"))); got != c.want {
			t.Errorf("%s %v: %q, quería %q", c.goos, c.vars, got, c.want)
		}
	}
}

func TestLectores_LaRutaVaPorVariableNoDentroDelScript(t *testing.T) {
	for _, goos := range []string{"windows", "darwin"} {
		ls := Lectores(entorno(goos, nil, "/h"))
		if len(ls) != 1 || !ls[0].ADestino {
			t.Fatalf("%s escribe en un archivo: %+v", goos, ls)
		}
		if strings.Contains(strings.Join(ls[0].Argv, " "), "/h") {
			t.Fatalf("%s: la ruta no se mete en el script (comillas): %v", goos, ls[0].Argv)
		}
	}
}

func TestCarpetasDeCapturas_PorSistema(t *testing.T) {
	linux := CarpetasDeCapturas(entorno("linux", map[string]string{"XDG_PICTURES_DIR": "/fotos"}, "/h"))
	if linux[0] != "/fotos" || !contiene(linux, filepath.Join("/h", "Pictures")) {
		t.Fatalf("linux: XDG primero y ~/Pictures: %v", linux)
	}
	win := CarpetasDeCapturas(entorno("windows", nil, "/h"))
	if !contiene(win, filepath.Join("/h", "Pictures", "Screenshots")) || !contiene(win, filepath.Join("/h", "OneDrive", "Pictures", "Screenshots")) {
		t.Fatalf("windows: Screenshots y la de OneDrive: %v", win)
	}
	if mac := CarpetasDeCapturas(entorno("darwin", nil, "/h")); !contiene(mac, filepath.Join("/h", "Desktop")) {
		t.Fatalf("macOS guarda en el escritorio: %v", mac)
	}
}

func contiene(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func TestUltimaCaptura_LaImagenMasNuevaDeTodasLasCarpetas(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	escribir(t, filepath.Join(a, "vieja.png"), png1x1, ahora.Add(-time.Hour))
	escribir(t, filepath.Join(b, "nueva.jpg"), png1x1, ahora.Add(-time.Minute))
	escribir(t, filepath.Join(b, "notas.txt"), []byte("no"), ahora)
	ruta, cuando, err := UltimaCaptura([]string{a, filepath.Join(a, "no-existe"), b})
	if err != nil || filepath.Base(ruta) != "nueva.jpg" || !cuando.Equal(ahora.Add(-time.Minute)) {
		t.Fatalf("la imagen más nueva, no el .txt: %q %v %v", ruta, cuando, err)
	}
	if _, _, err := UltimaCaptura([]string{t.TempDir()}); err == nil {
		t.Fatal("sin imágenes es un error que lo dice")
	}
}

func escribir(t *testing.T, ruta string, data []byte, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(ruta, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(ruta, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestValidar_SoloImagenesYConTope(t *testing.T) {
	if tipo, err := Validar(png1x1); err != nil || tipo != "image/png" {
		t.Fatalf("un PNG: %q %v", tipo, err)
	}
	if _, err := Validar([]byte("hola, no soy una imagen")); err == nil {
		t.Fatal("un texto no es una imagen")
	}
	if _, err := Validar(nil); err == nil {
		t.Fatal("vacío es un portapapeles sin imagen")
	}
	grande := append(append([]byte{}, png1x1...), bytes.Repeat([]byte{0}, MaxBytes)...)
	if _, err := Validar(grande); err == nil || !strings.Contains(err.Error(), "MB") {
		t.Fatalf("más de MaxBytes se rechaza diciendo el tamaño: %v", err)
	}
}

// Un wl-paste de prueba en el PATH: el producto ejecuta el lector real del sistema.
func TestTraer_DelPortapapelesQuedaEnTmpFotos(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("lector de prueba en sh")
	}
	bin := t.TempDir()
	png := filepath.Join(bin, "img.png")
	os.WriteFile(png, png1x1, 0o644)
	os.WriteFile(filepath.Join(bin, "wl-paste"), []byte("#!/bin/sh\ncat '"+png+"'\n"), 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	root := t.TempDir()
	f, err := Traer(context.Background(), root, DePortapapeles, entorno("linux", map[string]string{"WAYLAND_DISPLAY": "w"}, t.TempDir()), ahora)
	if err != nil {
		t.Fatal(err)
	}
	if f.Fuente != "portapapeles (wl-paste)" || f.Tipo != "image/png" || !strings.HasPrefix(f.Ruta, filepath.Join(root, "tmp", "fotos")) {
		t.Fatalf("queda en tmp/fotos con su fuente: %+v", f)
	}
	if data, _ := os.ReadFile(f.Ruta); !bytes.Equal(data, png1x1) {
		t.Fatal("la imagen se guarda tal cual")
	}
	if _, err := os.Stat(filepath.Join(root, "tmp", "fotos", ".gitignore")); err != nil {
		t.Fatal("tmp/fotos queda ignorada por git")
	}
}

func TestTraer_SinImagenEnElPortapapelesUsaLaUltimaCaptura(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("lector de prueba en sh")
	}
	bin := t.TempDir()
	os.WriteFile(filepath.Join(bin, "wl-paste"), []byte("#!/bin/sh\necho 'No suitable type of content copied' >&2\nexit 1\n"), 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, "Pictures"), 0o755)
	escribir(t, filepath.Join(home, "Pictures", "captura.png"), png1x1, ahora.Add(-2*time.Minute))
	env := entorno("linux", map[string]string{"WAYLAND_DISPLAY": "w"}, home)
	f, err := Traer(context.Background(), t.TempDir(), DeAuto, env, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if f.Fuente != "última captura (hace 2 min)" || filepath.Base(f.Ruta) != "captura.png" {
		t.Fatalf("la captura, en su lugar y con su edad: %+v", f)
	}
	if _, err := Traer(context.Background(), t.TempDir(), DePortapapeles, env, ahora); err == nil || !strings.Contains(err.Error(), "No suitable") {
		t.Fatalf("pedir solo el portapapeles sin imagen falla con lo que dijo el lector: %v", err)
	}
}

func TestTraer_SinLectorNiCapturasDiceQueInstalar(t *testing.T) {
	_, err := Traer(context.Background(), t.TempDir(), DeAuto, entorno("linux", map[string]string{}, t.TempDir()), ahora)
	if err == nil || !strings.Contains(err.Error(), "wl-clipboard") {
		t.Fatalf("dice qué falta: %v", err)
	}
	if _, err := Traer(context.Background(), t.TempDir(), "camara", entorno("linux", nil, "/h"), ahora); err == nil {
		t.Fatal("una fuente desconocida se rechaza")
	}
}

func TestDiagnostico_DiceSiHayConQueLeerElPortapapeles(t *testing.T) {
	con := func(hay ...string) func(string) (string, error) {
		return func(n string) (string, error) {
			for _, h := range hay {
				if h == n {
					return "/usr/bin/" + n, nil
				}
			}
			return "", os.ErrNotExist
		}
	}
	wayland := entorno("linux", map[string]string{"WAYLAND_DISPLAY": "w"}, "/h")
	if ok, texto := Diagnostico(wayland, con("wl-paste")); !ok || !strings.Contains(texto, "wl-paste") {
		t.Fatalf("con wl-paste: %v %q", ok, texto)
	}
	if ok, texto := Diagnostico(wayland, con()); ok || !strings.Contains(texto, "wl-clipboard") {
		t.Fatalf("Wayland sin wl-paste es una falla que dice qué instalar: %v %q", ok, texto)
	}
	if ok, texto := Diagnostico(entorno("linux", map[string]string{}, "/h"), con()); !ok || !strings.Contains(texto, "captura") {
		t.Fatalf("sin escritorio (SSH) no hay portapapeles que leer, y no es una falla: %v %q", ok, texto)
	}
	if ok, _ := Diagnostico(entorno("windows", nil, "/h"), con("powershell.exe")); !ok {
		t.Fatal("Windows trae PowerShell")
	}
}
