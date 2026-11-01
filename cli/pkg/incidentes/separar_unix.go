//go:build !windows

package incidentes

import (
	"os/exec"
	"syscall"
)

// separar deja el proceso en su propia sesión: no recibe las señales de la terminal ni del grupo del panel.
func separar(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
