//go:build !windows

package consola

import (
	"os/exec"
	"syscall"
)

// desprender pone el abridor en su propia sesión para que no lo cierre el fin de qdd ni el tope del panel.
func desprender(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
