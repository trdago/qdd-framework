package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Rule: Prohibido usar 'else'
// Panel en vivo de 'qdd estado watch' (RFC 0006 §3.2): pantalla alternativa, se redibuja cuando cambia el archivo o el
// tamaño de la terminal (sondeo, sin inotify: igual en Linux, macOS y Windows) y restaura la terminal al salir.

const (
	altScreenOn  = "\x1b[?1049h\x1b[?25l"
	altScreenOff = "\x1b[?25h\x1b[?1049l"
	limpiar      = "\x1b[H\x1b[2J"
)

func runEstadoWatch(cmd *cobra.Command, args []string) {
	root := estadoRaiz()
	ruta := filepath.Join(root, estado.Ruta)
	habilitarVT()
	fd := int(os.Stdout.Fd())

	salir := make(chan struct{}, 1)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() { <-sig; salir <- struct{}{} }()
	restaurar := leerTeclaQ(salir)
	defer restaurar()

	fmt.Print(altScreenOn)
	defer fmt.Print(altScreenOff)

	var ultimo string
	tick := time.NewTicker(estadoIntervalo)
	defer tick.Stop()
	for {
		ancho, alto := tamano(fd)
		mtime := ""
		if info, err := os.Stat(ruta); err == nil {
			mtime = info.ModTime().String()
		}
		huella := fmt.Sprintf("%s|%dx%d|%s", mtime, ancho, alto, time.Now().Format("15:04"))
		if huella != ultimo {
			ultimo = huella
			dibujar(root, ancho, alto)
		}
		select {
		case <-salir:
			return
		case <-tick.C:
		}
	}
}

func tamano(fd int) (int, int) {
	w, h, err := term.GetSize(fd)
	if err != nil || w <= 0 {
		return 80, 24
	}
	return w, h
}

func dibujar(root string, ancho, alto int) {
	var lineas []string
	e, err := estado.Load(root)
	if err != nil {
		lineas = []string{"Estado por frente · " + filepath.Base(root), "", estado.Truncar(fmt.Sprintf("Sin %s legible: %v", estado.Ruta, err), ancho), "", "Créalo con 'qdd estado init'."}
	}
	if err == nil {
		lineas = estado.Panel(e, filepath.Base(root), ancho, time.Now())
	}
	if len(lineas) > alto-1 && alto > 2 {
		lineas = append(lineas[:alto-2], "…")
	}
	lineas = append(lineas, "", "q para salir")
	fmt.Print(limpiar + strings.Join(lineas, "\r\n"))
}

// leerTeclaQ pone la entrada en modo crudo (si es una terminal) y avisa en salir al leer «q» o Ctrl-C. Devuelve cómo
// restaurar la terminal.
func leerTeclaQ(salir chan struct{}) func() {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return func() {}
	}
	estadoPrevio, err := term.MakeRaw(fd)
	if err != nil {
		return func() {}
	}
	go func() {
		r := bufio.NewReader(os.Stdin)
		for {
			b, err := r.ReadByte()
			if err != nil {
				return
			}
			if b == 'q' || b == 'Q' || b == 3 {
				salir <- struct{}{}
				return
			}
		}
	}()
	return func() { _ = term.Restore(fd, estadoPrevio) }
}
