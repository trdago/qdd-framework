package baul

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"
)

// Rule: Prohibido usar 'else'
// El baúl desde Config en el panel (pedido de la persona, 2026-11-09). La persona copia el token y presiona «Pegar del
// portapapeles»: qdd lo lee directo del portapapeles del sistema y lo vacía en el acto. El valor nunca se ve en el panel,
// no pasa por sus argumentos, el chat ni un pedido, y se prueba contra su API antes de guardarlo, como en la terminal.

// Portapapeles es el del sistema; las pruebas lo cambian por uno falso.
type Portapapeles interface {
	Leer() (string, error)
	Vaciar() error
}

// topeToken: más largo no es un token pegado, es otra cosa copiada.
const topeToken = 4096

// ErrSinPortapapeles: este sistema no tiene con qué leer el portapapeles.
var ErrSinPortapapeles = errors.New("no hay con qué leer el portapapeles: instala wl-clipboard (Wayland) o xclip (X11), o guárdalo en tu terminal con qdd baul guardar")

// GuardarDelPortapapeles lee el token, vacía el portapapeles y lo guarda si es un token que su API acepta. Devuelve un
// aviso si no se pudo vaciar: el token quedó guardado, pero sigue copiado.
func GuardarDelPortapapeles(e Entrada, p Portapapeles, ahora time.Time) (string, error) {
	copiado, err := p.Leer()
	if err != nil {
		return "", fmt.Errorf("no pude leer el portapapeles: %w", err)
	}
	aviso := ""
	if err := p.Vaciar(); err != nil {
		aviso = "no pude vaciar el portapapeles: cópiale otra cosa encima"
	}
	valor := strings.TrimSpace(copiado)
	if err := pareceToken(valor); err != nil {
		return aviso, err
	}
	if e.Tipo != "otro" && e.Tipo != "aws-sso" {
		if _, err := VerificarValor(e.Tipo, valor); errors.Is(err, ErrRechazado) {
			return aviso, fmt.Errorf("no se guardó: %w", err)
		}
	}
	return aviso, Guardar(e, valor, ahora)
}

// pareceToken rechaza lo que se copió por error. El mensaje nunca repite lo copiado.
func pareceToken(valor string) error {
	if valor == "" {
		return errors.New("el portapapeles está vacío: copia el token y vuelve a intentarlo")
	}
	if len(valor) > topeToken || strings.IndexFunc(valor, unicode.IsSpace) >= 0 {
		return errors.New("lo copiado no parece un token (tiene espacios, saltos de línea o es muy largo): copia solo el token")
	}
	return nil
}

// PortapapelesDelSistema lee y vacía el portapapeles con la herramienta de cada sistema.
func PortapapelesDelSistema() Portapapeles {
	leer, vaciar := comandosPortapapeles(runtime.GOOS, os.Getenv)
	return sistema{leer: leer, vaciar: vaciar}
}

type sistema struct{ leer, vaciar []string }

// Disponible: este sistema tiene con qué leer el portapapeles (el panel ofrece «Pegar del portapapeles» solo así).
func (s sistema) Disponible() bool {
	if s.leer == nil {
		return false
	}
	_, err := exec.LookPath(s.leer[0])
	return err == nil
}

func (s sistema) Leer() (string, error) {
	if s.leer == nil {
		return "", ErrSinPortapapeles
	}
	var salida, errSalida bytes.Buffer
	c := exec.Command(s.leer[0], s.leer[1:]...)
	c.Stdout, c.Stderr = &salida, &errSalida
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("%s: %v %s", s.leer[0], err, strings.TrimSpace(errSalida.String()))
	}
	return salida.String(), nil
}

func (s sistema) Vaciar() error {
	if s.vaciar == nil {
		return ErrSinPortapapeles
	}
	c := exec.Command(s.vaciar[0], s.vaciar[1:]...)
	c.Stdin = strings.NewReader("")
	return c.Run()
}

const (
	leerWindows   = `Add-Type -AssemblyName System.Windows.Forms; [Console]::Out.Write([System.Windows.Forms.Clipboard]::GetText())`
	vaciarWindows = `Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Clipboard]::Clear()`
)

// comandosPortapapeles: con qué se lee el texto copiado y con qué se vacía. Sin escritorio, nada.
func comandosPortapapeles(goos string, env func(string) string) ([]string, []string) {
	if goos == "windows" {
		ps := []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command"}
		return append(append([]string{}, ps...), leerWindows), append(append([]string{}, ps...), vaciarWindows)
	}
	if goos == "darwin" {
		return []string{"pbpaste"}, []string{"pbcopy"}
	}
	if env("WAYLAND_DISPLAY") != "" {
		return []string{"wl-paste", "--no-newline", "--type", "text/plain"}, []string{"wl-copy", "--clear"}
	}
	if env("DISPLAY") != "" {
		return []string{"xclip", "-selection", "clipboard", "-o"}, []string{"xclip", "-selection", "clipboard", "-i"}
	}
	return nil, nil
}
