//go:build unix

package remoto

import (
	"errors"
	"os"
	"syscall"
)

// bloquearArchivo toma el candado exclusivo sin esperar (flock): lo suelta cerrar el archivo o morir el proceso.
func bloquearArchivo(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return errOcupado
	}
	return err
}
