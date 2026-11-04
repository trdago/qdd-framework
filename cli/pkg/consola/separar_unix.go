//go:build !windows

package consola

import (
	"context"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

const topeProbarScope = 3 * time.Second

// opcionesScope: sin --expand-environment=no, systemd-run cambiaría un $ de la carpeta o del comando. Un systemd que no la
// conoce falla en la prueba y el pedido se lanza como siempre.
var opcionesScope = []string{"--user", "--scope", "--quiet", "--collect", "--expand-environment=no"}

// probarScope se puede reemplazar en las pruebas: lo real depende de la máquina.
var probarScope = scopeDisponible

// separar deja el proceso en su propia sesión: no recibe las señales de la terminal ni del grupo del panel.
func separar(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// fueraDeLaUnidad saca al pedido del cgroup de quien lo lanza (FND-129): Setsid no lo hace, y lanzado desde una unidad de
// systemd moriría al detenerse esa unidad. systemd-run --scope reemplaza su proceso por el comando, así que el PID, la
// carpeta y el entorno son los mismos. Sin systemd de usuario, se lanza como siempre.
func fueraDeLaUnidad(bin string, args []string) (string, []string) {
	if runtime.GOOS != "linux" || !probarScope() {
		return bin, args
	}
	return "systemd-run", append(append([]string{}, opcionesScope...), append([]string{bin}, args...)...)
}

// scopeDisponible lo prueba de verdad: tener el programa no basta si no hay bus de usuario (contenedores, CI, cron).
func scopeDisponible() bool {
	ctx, cancelar := context.WithTimeout(context.Background(), topeProbarScope)
	defer cancelar()
	return exec.CommandContext(ctx, "systemd-run", append(append([]string{}, opcionesScope...), "true")...).Run() == nil
}
