//go:build !windows

package main

import "os/exec"

// Rule: Prohibido usar 'else'

// nombreQDD: fuera de Windows qddw solo corre en pruebas, con un qdd sin extensión.
const nombreQDD = "qdd"

func lanzar(c *exec.Cmd) error {
	return c.Run()
}
