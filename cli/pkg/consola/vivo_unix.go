//go:build !windows

package consola

import (
	"errors"
	"syscall"
)

// procesoVivo dice si el proceso existe (señal 0: no le hace nada). Sin permiso para señalarlo, existe.
func procesoVivo(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
