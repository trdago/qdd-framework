//go:build windows

package cmd

import (
	"os"

	"golang.org/x/sys/windows"
)

// Rule: Prohibido usar 'else'
// habilitarVT activa las secuencias ANSI en la consola de Windows (Windows 10+); en Windows Terminal ya vienen activas.
func habilitarVT() {
	h := windows.Handle(os.Stdout.Fd())
	var modo uint32
	if err := windows.GetConsoleMode(h, &modo); err != nil {
		return
	}
	_ = windows.SetConsoleMode(h, modo|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
