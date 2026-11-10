package consola

import "os/exec"

// Rule: Prohibido usar 'else'
// arrancar inicia un proceso que separar ya dejó aparte y devuelve el que quedó corriendo. En Windows, un job que no
// deja salir a sus procesos rechaza el lanzamiento (FND-193): se lanza otra vez dentro del job, como antes.
func arrancar(c *exec.Cmd) (*exec.Cmd, error) {
	err := c.Start()
	if err == nil || !noLoDejoSalir(c, err) {
		return c, err
	}
	otro := sinSalirDelJob(c)
	return otro, otro.Start()
}
