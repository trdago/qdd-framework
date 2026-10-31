package consola

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-089 (captura del usuario, 2026-10-31): el panel corría el pedido como hijo suyo y lo esperaba. Un
// /reload-plugins mató el proceso entre el merge y el git push; el panel quedó «ejecutando…» y bloqueado. Ahora el panel
// lanza el pedido y vuelve en el acto: «qdd consola correr» lo corre en su propia sesión del sistema (no muere con el
// panel ni con la terminal), anota su PID y el del comando, y deja el resultado en la cola, que el panel lee.

// Lanzar marca el pedido como ejecutando (con la cola bloqueada: no corre dos veces) y lo deja corriendo aparte, con
// «bin consola correr <id>»; vuelve sin esperarlo. Una acción que no es un pedido se ejecuta como siempre.
func (a Accion) Lanzar(root, bin string, salida io.Writer) int {
	if a.Tipo != TipoPedido {
		return a.Ejecutar(root, bin, salida)
	}
	if err := marcarEjecutando(root, a.pedido, time.Now()); err != nil {
		fmt.Fprintf(salida, "[!] %v\n", err)
		return 1
	}
	c := exec.Command(bin, ArgsCorrer(root, a.pedido)...)
	c.Dir = root
	separar(c)
	if err := c.Start(); err != nil {
		terminar(root, a.pedido, 1, "no se pudo lanzar: "+err.Error())
		fmt.Fprintf(salida, "[!] no se pudo lanzar el pedido: %v\n", err)
		return 1
	}
	anotarLanzado(root, a.pedido, c.Process.Pid)
	_ = c.Process.Release()
	fmt.Fprintf(salida, "[+] En marcha: %s (el panel muestra el resultado)\n", a.Etiqueta)
	return 0
}

// ArgsCorrer son los argumentos de «qdd consola correr»: la raíz va explícita, no se deduce de la carpeta.
func ArgsCorrer(root, id string) []string {
	return []string{"consola", "correr", "--raiz", root, id}
}

// EsCorrer reconoce los argumentos de ArgsCorrer y devuelve la raíz y el pedido (para las pruebas, que se relanzan).
func EsCorrer(args []string) (string, string, bool) {
	if len(args) != 5 || args[0] != "consola" || args[1] != "correr" || args[2] != "--raiz" {
		return "", "", false
	}
	return args[3], args[4], true
}

// anotarLanzado deja el PID del proceso que va a correr el pedido, si todavía no lo tomó.
func anotarLanzado(root, id string, pid int) {
	conCola(root, time.Now(), func(cola []Pedido) ([]Pedido, error) {
		for i := range cola {
			if cola[i].ID == id && cola[i].Estado == EstadoEjecutando && cola[i].Hijo == 0 {
				cola[i].PID = pid
			}
		}
		return cola, nil
	})
}

// CorrerLanzado es «qdd consola correr <id>»: corre un pedido que Lanzar ya marcó como ejecutando y que nadie tomó.
// Un pedido pendiente, hecho, o que ya corre en otro proceso no se toca.
func CorrerLanzado(root, id string, salida io.Writer) int {
	err := conCola(root, time.Now(), func(cola []Pedido) ([]Pedido, error) {
		for i, p := range cola {
			if p.ID != id {
				continue
			}
			if p.Estado != EstadoEjecutando || p.Hijo != 0 {
				return nil, fmt.Errorf("el pedido %s no espera que lo corran (está %s)", id, p.Estado)
			}
			cola[i].PID, cola[i].Hijo = os.Getpid(), -1
			return cola, nil
		}
		return nil, fmt.Errorf("no hay un pedido '%s' en la cola", id)
	})
	if err != nil {
		fmt.Fprintf(salida, "[!] %v\n", err)
		return 1
	}
	for _, a := range accionesDePedidos(root) {
		if a.pedido == id {
			return correrComando(root, a, salida)
		}
	}
	return 1
}
