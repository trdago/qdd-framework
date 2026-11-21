//go:build windows

package consola

import (
	"errors"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// noLoDejoSalir: Windows responde ERROR_ACCESS_DENIED cuando el job de quien lanza no permite CREATE_BREAKAWAY_FROM_JOB.
func noLoDejoSalir(c *exec.Cmd, err error) bool {
	if c.SysProcAttr == nil || c.SysProcAttr.CreationFlags&windows.CREATE_BREAKAWAY_FROM_JOB == 0 {
		return false
	}
	return errors.Is(err, windows.ERROR_ACCESS_DENIED)
}

// sinSalirDelJob es el mismo comando sin pedir salir del job: un Cmd que falló al iniciar no se vuelve a usar.
func sinSalirDelJob(c *exec.Cmd) *exec.Cmd {
	attr := *c.SysProcAttr
	attr.CreationFlags &^= windows.CREATE_BREAKAWAY_FROM_JOB
	otro := exec.Command(c.Path, c.Args[1:]...)
	otro.Dir, otro.Env, otro.Stdin, otro.Stdout, otro.Stderr, otro.ExtraFiles = c.Dir, c.Env, c.Stdin, c.Stdout, c.Stderr, c.ExtraFiles
	otro.SysProcAttr = &syscall.SysProcAttr{}
	*otro.SysProcAttr = attr
	return otro
}
