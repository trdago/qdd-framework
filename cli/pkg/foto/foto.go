// Package foto trae una imagen a la consola (pedido del usuario, 2026-10-31): una captura pegada en la terminal llegó
// como bytes de PNG y la IA no la pudo ver. qdd lee el portapapeles con la herramienta de cada sistema (wl-paste o
// xclip en Linux, PowerShell en Windows, osascript en macOS) o toma la última captura de pantalla, y la deja donde la
// IA la puede abrir: un archivo en tmp/fotos o, por MCP, la imagen misma.
package foto

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

// MaxBytes es el tope de una imagen: lo que la IA acepta por imagen.
const MaxBytes = 5 << 20

// Carpeta es donde queda la imagen del portapapeles, dentro de ./tmp (lo temporal; git no la ve).
const Carpeta = "tmp/fotos"

// De dónde se trae la imagen.
const (
	DeAuto         = "auto" // el portapapeles y, si no tiene una imagen, la última captura
	DePortapapeles = "portapapeles"
	DeCaptura      = "captura"
)

// Espera es cuánto se le da al lector del portapapeles.
const Espera = 10 * time.Second

// Entorno es el sistema donde corre qdd: se pasa explícito para probar cada sistema desde cualquiera.
type Entorno struct {
	GOOS string
	Home string
	Env  func(string) string
}

// Lector es un comando que lee una imagen del portapapeles: a stdout, o al archivo de la variable QDD_FOTO si ADestino
// (la ruta nunca va dentro del script, así no hay comillas que escapar).
type Lector struct {
	Nombre   string
	Argv     []string
	ADestino bool
}

// Foto es la imagen traída.
type Foto struct {
	Ruta   string
	Fuente string
	Tipo   string
	Bytes  int
	Datos  []byte
}

const scriptWindows = `Add-Type -AssemblyName System.Windows.Forms,System.Drawing; ` +
	`$i = [System.Windows.Forms.Clipboard]::GetImage(); if ($i -eq $null) { [Console]::Error.WriteLine('el portapapeles no tiene una imagen'); exit 3 }; ` +
	`$i.Save($env:QDD_FOTO, [System.Drawing.Imaging.ImageFormat]::Png)`

// Lectores devuelve los lectores del portapapeles del sistema, en orden de preferencia.
func Lectores(e Entorno) []Lector {
	switch e.GOOS {
	case "windows":
		return []Lector{{Nombre: "PowerShell", Argv: []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", scriptWindows}, ADestino: true}}
	case "darwin":
		return []Lector{{Nombre: "osascript", ADestino: true, Argv: []string{"osascript",
			"-e", `set f to POSIX file (system attribute "QDD_FOTO")`,
			"-e", `set d to (the clipboard as «class PNGf»)`,
			"-e", `set r to open for access f with write permission`,
			"-e", `write d to r`,
			"-e", `close access r`}}}
	}
	var out []Lector
	if e.Env("WAYLAND_DISPLAY") != "" {
		out = append(out, Lector{Nombre: "wl-paste", Argv: []string{"wl-paste", "--no-newline", "--type", "image/png"}})
	}
	if e.Env("DISPLAY") != "" {
		out = append(out, Lector{Nombre: "xclip", Argv: []string{"xclip", "-selection", "clipboard", "-t", "image/png", "-o"}})
	}
	return out
}

// CarpetasDeCapturas son las carpetas donde cada sistema deja las capturas de pantalla.
func CarpetasDeCapturas(e Entorno) []string {
	h := e.Home
	switch e.GOOS {
	case "windows":
		return []string{filepath.Join(h, "Pictures", "Screenshots"), filepath.Join(h, "OneDrive", "Pictures", "Screenshots"), filepath.Join(h, "Pictures")}
	case "darwin":
		return []string{filepath.Join(h, "Desktop"), filepath.Join(h, "Pictures", "Screenshots")}
	}
	var out []string
	if d := e.Env("XDG_PICTURES_DIR"); d != "" {
		out = append(out, d)
	}
	return append(out, filepath.Join(h, "Pictures"), filepath.Join(h, "Pictures", "Screenshots"),
		filepath.Join(h, "Imágenes"), filepath.Join(h, "Imágenes", "Capturas de pantalla"))
}

var extensiones = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}

// UltimaCaptura es la imagen más nueva de las carpetas (sin entrar en subcarpetas).
func UltimaCaptura(carpetas []string) (string, time.Time, error) {
	var ruta string
	var cuando time.Time
	for _, dir := range carpetas {
		entradas, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, en := range entradas {
			if en.IsDir() || !extensiones[strings.ToLower(filepath.Ext(en.Name()))] {
				continue
			}
			info, err := en.Info()
			if err != nil || !info.ModTime().After(cuando) {
				continue
			}
			ruta, cuando = filepath.Join(dir, en.Name()), info.ModTime()
		}
	}
	if ruta == "" {
		return "", time.Time{}, fmt.Errorf("no hay capturas de pantalla en %s", strings.Join(carpetas, ", "))
	}
	return ruta, cuando, nil
}

var tipos = map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}

// Validar dice el tipo de la imagen; rechaza lo que no es una imagen y lo que pasa de MaxBytes.
func Validar(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("no hay una imagen (vacío)")
	}
	if len(data) > MaxBytes {
		return "", fmt.Errorf("la imagen pesa %.1f MB; el máximo es %d MB", float64(len(data))/(1<<20), MaxBytes>>20)
	}
	tipo := http.DetectContentType(data)
	if _, ok := tipos[tipo]; !ok {
		return "", fmt.Errorf("no es una imagen (%s)", tipo)
	}
	return tipo, nil
}

// Traer busca la imagen según de (auto, portapapeles o captura).
func Traer(ctx context.Context, root, de string, e Entorno, now time.Time) (Foto, error) {
	if de == "" {
		de = DeAuto
	}
	if de != DeAuto && de != DePortapapeles && de != DeCaptura {
		return Foto{}, fmt.Errorf("de %q: usa auto, portapapeles o captura", de)
	}
	var fallas []string
	if de != DeCaptura {
		f, err := delPortapapeles(ctx, root, e, now)
		if err == nil || de == DePortapapeles {
			return f, err
		}
		fallas = append(fallas, err.Error())
	}
	f, err := deCaptura(e, now)
	if err == nil {
		return f, nil
	}
	return Foto{}, errors.New(strings.Join(append(fallas, err.Error()), "; "))
}

func delPortapapeles(ctx context.Context, root string, e Entorno, now time.Time) (Foto, error) {
	lectores := Lectores(e)
	if len(lectores) == 0 {
		return Foto{}, errors.New("portapapeles: no hay con qué leerlo; instala wl-clipboard (Wayland) o xclip (X11)")
	}
	dir := filepath.Join(root, filepath.FromSlash(Carpeta))
	if err := prepararCarpeta(dir); err != nil {
		return Foto{}, err
	}
	var fallas []string
	for _, l := range lectores {
		f, err := leer(ctx, l, dir, now)
		if err == nil {
			return f, nil
		}
		fallas = append(fallas, l.Nombre+": "+err.Error())
	}
	return Foto{}, errors.New("portapapeles: " + strings.Join(fallas, "; "))
}

// prepararCarpeta crea tmp/fotos con un .gitignore propio: lo que se pega nunca entra al repositorio.
func prepararCarpeta(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	marcador := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(marcador); err == nil {
		return nil
	}
	return os.WriteFile(marcador, []byte("# Imágenes traídas con qdd foto: temporales, git no las ve.\n*\n"), 0o644)
}

func leer(ctx context.Context, l Lector, dir string, now time.Time) (Foto, error) {
	ctx, cancel := context.WithTimeout(ctx, Espera)
	defer cancel()
	base := filepath.Join(dir, now.Format("2006-01-02_15-04-05"))
	cmd := exec.CommandContext(ctx, l.Argv[0], l.Argv[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	destino := base + ".png"
	if l.ADestino {
		cmd.Env = append(os.Environ(), "QDD_FOTO="+destino)
	}
	if err := cmd.Run(); err != nil {
		os.Remove(destino)
		return Foto{}, motivo(err, stderr.String())
	}
	data := stdout.Bytes()
	if l.ADestino {
		leido, err := os.ReadFile(destino)
		if err != nil {
			return Foto{}, err
		}
		data = leido
	}
	tipo, err := Validar(data)
	if err != nil {
		os.Remove(destino)
		return Foto{}, err
	}
	ruta := base + tipos[tipo]
	if err := os.WriteFile(ruta, data, 0o644); err != nil {
		return Foto{}, err
	}
	return Foto{Ruta: ruta, Fuente: "portapapeles (" + l.Nombre + ")", Tipo: tipo, Bytes: len(data), Datos: data}, nil
}

// motivo es lo que dijo el lector, o el error del sistema si no dijo nada.
func motivo(err error, stderr string) error {
	if s := strings.TrimSpace(stderr); s != "" {
		return errors.New(s)
	}
	return err
}

func deCaptura(e Entorno, now time.Time) (Foto, error) {
	ruta, cuando, err := UltimaCaptura(CarpetasDeCapturas(e))
	if err != nil {
		return Foto{}, err
	}
	data, err := os.ReadFile(ruta)
	if err != nil {
		return Foto{}, err
	}
	tipo, err := Validar(data)
	if err != nil {
		return Foto{}, fmt.Errorf("%s: %w", ruta, err)
	}
	return Foto{Ruta: ruta, Fuente: "última captura (" + hace(now.Sub(cuando)) + ")", Tipo: tipo, Bytes: len(data), Datos: data}, nil
}

// hace dice cuánto pasó, redondeado hacia abajo: «hace 2 min», «hace 3 h», «hace 2 d».
func hace(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("hace %d min", int(max(d, 0).Minutes()))
	}
	if d < 48*time.Hour {
		return fmt.Sprintf("hace %d h", int(d.Hours()))
	}
	return fmt.Sprintf("hace %d d", int(d.Hours()/24))
}

// EntornoReal es el sistema donde corre qdd.
func EntornoReal(goos string) Entorno {
	home, _ := os.UserHomeDir()
	return Entorno{GOOS: goos, Home: home, Env: os.Getenv}
}

// Diagnostico dice si este sistema tiene con qué leer el portapapeles (para doctor). Sin escritorio no hay
// portapapeles que leer y no es una falla: qdd foto usa la última captura.
func Diagnostico(e Entorno, buscar func(string) (string, error)) (bool, string) {
	lectores := Lectores(e)
	if len(lectores) == 0 {
		return true, "sin escritorio: qdd foto usa la última captura de pantalla"
	}
	var nombres []string
	for _, l := range lectores {
		nombres = append(nombres, l.Nombre)
		if _, err := buscar(l.Argv[0]); err == nil {
			return true, "portapapeles con " + l.Nombre
		}
	}
	if e.GOOS == "linux" {
		return false, "no está " + strings.Join(nombres, " ni ") + ": instala wl-clipboard (Wayland) o xclip (X11); mientras, qdd foto usa la última captura"
	}
	return false, "no está " + strings.Join(nombres, " ni ") + ": qdd foto usa la última captura"
}
