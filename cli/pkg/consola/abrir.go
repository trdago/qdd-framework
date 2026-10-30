package consola

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9: «Abrir carpeta» abre la raíz del proyecto con el gestor de archivos que el sistema tenga configurado
// (xdg-open en Linux y BSD, open en macOS, explorer en Windows). No escribe en el proyecto: no pide confirmación.

const (
	TipoAbrir = "abrir"

	// esperaAbridor es cuánto se espera al abridor antes de dejarlo seguir solo: un error rápido (sin aplicación para
	// carpetas) se informa; si sigue vivo, el gestor ya está abriendo.
	esperaAbridor = 3 * time.Second
)

// Abridor es el programa que abre una carpeta con la aplicación predeterminada.
type Abridor struct {
	Programa string
	Args     []string // van antes de la carpeta
}

// buscarAbridor devuelve el abridor del sistema, o false si no está instalado. Los tests lo reemplazan.
var buscarAbridor = abridorDelSistema

func abridorDelSistema() (Abridor, bool) {
	a := Abridor{Programa: "xdg-open"}
	if runtime.GOOS == "darwin" {
		a = Abridor{Programa: "open"}
	}
	if runtime.GOOS == "windows" {
		a = Abridor{Programa: "explorer"}
	}
	ruta, err := exec.LookPath(a.Programa)
	if err != nil {
		return Abridor{}, false
	}
	a.Programa = ruta
	return a, true
}

func accionAbrir(root string) (Accion, bool) {
	ab, ok := buscarAbridor()
	if !ok {
		return Accion{}, false
	}
	return Accion{
		ID: "abrir-carpeta", Tipo: TipoAbrir, Etiqueta: "Abrir carpeta",
		Detalle: nombrePrograma(ab.Programa) + " " + root, Args: append(append([]string{}, ab.Args...), root), programa: ab.Programa,
	}, true
}

func nombrePrograma(ruta string) string {
	for i := len(ruta) - 1; i >= 0; i-- {
		if ruta[i] == '/' || ruta[i] == '\\' {
			return ruta[i+1:]
		}
	}
	return ruta
}

// abrir lanza el abridor desprendido (sesión propia: sobrevive a qdd y al tope del panel). Si termina con error dentro
// de esperaAbridor, lo informa; si sigue abriendo, lo deja seguir.
func abrir(a Accion, salida io.Writer) int {
	c := exec.Command(a.programa, a.Args...)
	desprender(c)
	if err := c.Start(); err != nil {
		fmt.Fprintf(salida, "[!] no se pudo ejecutar %s: %v\n", a.Detalle, err)
		return 1
	}
	termino := make(chan error, 1)
	go func() { termino <- c.Wait() }()
	select {
	case err := <-termino:
		if err != nil {
			fmt.Fprintf(salida, "[!] %s falló: %v (¿hay una aplicación predeterminada para carpetas?)\n", a.Detalle, err)
			return 1
		}
	case <-time.After(esperaAbridor):
	}
	fmt.Fprintf(salida, "[+] Carpeta abierta: %s\n", a.Detalle)
	return 0
}
