//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Rule: Prohibido usar 'else'

const nombreQDD = "qdd.exe"

// lanzar corre qdd sin ventana (CREATE_NO_WINDOW) dentro de un trabajo que muere con qddw: si Windows termina qddw
// (schtasks /End), termina también qdd remoto, y no queda uno huérfano. Lo que qdd lanza (los pedidos) sale del
// trabajo solo (SILENT_BREAKAWAY_OK): un pedido que instala un qdd nuevo sigue corriendo (FND-128). Si el trabajo no se
// puede crear, qdd corre igual, sin ventana.
func lanzar(c *exec.Cmd) error {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	trabajo := trabajoQueMuereConQddw()
	if err := c.Start(); err != nil {
		return err
	}
	asignar(trabajo, c.Process.Pid)
	return c.Wait()
}

// trabajoQueMuereConQddw queda abierto hasta que qddw termina: al cerrarse su handle, Windows termina lo que tiene.
func trabajoQueMuereConQddw() windows.Handle {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK},
	}
	_, err = windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if err != nil {
		windows.CloseHandle(h)
		return 0
	}
	return h
}

func asignar(trabajo windows.Handle, pid int) {
	if trabajo == 0 {
		return
	}
	p, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(p)
	_ = windows.AssignProcessToJobObject(trabajo, p)
}
