//go:build windows

package herramientas

import "os/exec"

// enGrupo: en Windows, detenerla corta el proceso de la herramienta (exec.CommandContext).
func enGrupo(c *exec.Cmd) {}
