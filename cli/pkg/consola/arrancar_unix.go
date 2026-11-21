//go:build !windows

package consola

import "os/exec"

// Fuera de Windows no hay jobs: separar no pide salir de nada.
func noLoDejoSalir(c *exec.Cmd, err error) bool { return false }

func sinSalirDelJob(c *exec.Cmd) *exec.Cmd { return c }
