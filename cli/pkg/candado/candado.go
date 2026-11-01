// Package candado da un bloqueo exclusivo entre procesos con un archivo: sirve en Linux, macOS y Windows sin
// llamadas del sistema operativo. Un archivo de bloqueo de más de Abandonado es de un proceso que murió.
package candado

import (
	"fmt"
	"os"
	"time"
)

// Abandonado es la edad desde la que un bloqueo se da por huérfano y se descarta.
const Abandonado = 10 * time.Second

// Tomar espera el bloqueo hasta espera y devuelve con qué soltarlo; que nombra lo bloqueado en el error si no llega.
func Tomar(lock string, espera time.Duration, que string) (func(), error) {
	limite := time.Now().Add(espera)
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			return func() { os.Remove(lock) }, nil
		}
		if info, errStat := os.Stat(lock); errStat == nil && time.Since(info.ModTime()) > Abandonado {
			os.Remove(lock)
			continue
		}
		if time.Now().After(limite) {
			return nil, fmt.Errorf("%s: lo usa otro proceso (%s)", que, lock)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
