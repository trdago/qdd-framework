//go:build windows

package consola

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// separar deja el proceso en su propio grupo y con una consola oculta: cerrar la terminal o el panel no lo corta, y
// los gh o git que lance heredan esa consola en vez de abrir cada uno una ventana visible (FND-122: con
// el proceso separado sin consola y Windows le daba una ventana nueva a cada hijo).
func separar(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW}
}

// fueraDeLaUnidad no hace nada en Windows: CREATE_NEW_PROCESS_GROUP ya lo separa y no hay cgroups (FND-129).
func fueraDeLaUnidad(bin string, args []string) (string, []string) {
	return bin, args
}
