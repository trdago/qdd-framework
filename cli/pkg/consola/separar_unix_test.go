//go:build !windows

package consola

import "golang.org/x/sys/unix"

// separado dice si el proceso está en otra sesión que este.
func separado(pid int) bool {
	suya, err := unix.Getsid(pid)
	mia, err2 := unix.Getsid(0)
	return err == nil && err2 == nil && suya != mia
}
