//go:build linux || darwin

package rendimiento

import (
	"os"
	"runtime"
	"syscall"
)

// memoriaDurante: en Unix la memoria máxima la da el sistema al terminar el proceso.
func memoriaDurante(int) *float64 { return nil }

// memoriaAlTerminar es Maxrss: kilobytes en Linux, bytes en macOS.
func memoriaAlTerminar(estado *os.ProcessState) *float64 {
	if estado == nil {
		return nil
	}
	uso, ok := estado.SysUsage().(*syscall.Rusage)
	if !ok || uso.Maxrss <= 0 {
		return nil
	}
	mb := float64(uso.Maxrss) / 1024
	if runtime.GOOS == "darwin" {
		mb /= 1024
	}
	return &mb
}
