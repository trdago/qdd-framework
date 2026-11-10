package consola

import "os/exec"

// arrancar inicia un proceso que separar ya dejó aparte y devuelve el que quedó corriendo.
func arrancar(c *exec.Cmd) (*exec.Cmd, error) {
	return c, c.Start()
}
