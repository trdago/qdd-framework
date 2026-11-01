//go:build !windows

package herramientas

import (
	"os/exec"
	"syscall"
)

// enGrupo corre la herramienta en su propio grupo de procesos: detenerla (■ en el panel) corta también lo que ella lanzó.
func enGrupo(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		return syscall.Kill(-c.Process.Pid, syscall.SIGTERM)
	}
}
