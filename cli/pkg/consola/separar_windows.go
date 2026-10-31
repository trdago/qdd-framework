//go:build windows

package consola

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// separar deja el proceso sin consola y en su propio grupo: cerrar la terminal o el panel no lo corta.
func separar(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS}
}
