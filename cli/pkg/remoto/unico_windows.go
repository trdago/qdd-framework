//go:build windows

package remoto

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// bloquearArchivo toma el candado exclusivo sin esperar (LockFileEx): lo suelta cerrar el archivo o morir el proceso.
func bloquearArchivo(f *os.File) error {
	err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, new(windows.Overlapped))
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return errOcupado
	}
	return err
}
