//go:build windows

package consola

import "os/exec"

// desprender no hace nada en Windows: explorer entrega la ventana al proceso del escritorio y termina.
func desprender(c *exec.Cmd) {}
