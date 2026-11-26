package consola

import "os/exec"

// soltar deja correr a un hijo ya lanzado sin esperarlo, y lo recoge cuando termina. Soltarlo con Release no lo recoge en
// Linux: el hijo quedaba zombie mientras viviera quien lo lanzó (FND-179).
func soltar(c *exec.Cmd) {
	go func() { _ = c.Wait() }()
}
