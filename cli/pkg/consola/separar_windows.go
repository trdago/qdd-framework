//go:build windows

package consola

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// separar deja el proceso en su propio grupo y con una consola oculta: cerrar la terminal o el panel no lo corta, y
// los gh o git que lance heredan esa consola en vez de abrir cada uno una ventana visible (FND-122: con
// el proceso separado sin consola y Windows le daba una ventana nueva a cada hijo). También pide salir del job de quien
// lo lanza (FND-193): un job que mata a sus procesos al cerrarse se llevaba el pedido al cerrar Claude Code; si el job
// no lo permite, arrancar lo lanza dentro.
func separar(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW | windows.CREATE_BREAKAWAY_FROM_JOB}
}

// fueraDeLaUnidad no hace nada en Windows: CREATE_NEW_PROCESS_GROUP ya lo separa y no hay cgroups (FND-129).
func fueraDeLaUnidad(bin string, args []string) (string, []string) {
	return bin, args
}
