//go:build linux || darwin

package consola

import "golang.org/x/sys/unix"

// Rule: Prohibido usar 'else'

// usoDisco es el % usado del disco de dir, como df: usado / (usado + disponible para el usuario).
func usoDisco(dir string) (float64, bool) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return 0, false
	}
	b := uint64(st.Bsize)
	return porcentajeUsado(st.Blocks*b, st.Bfree*b, st.Bavail*b)
}
