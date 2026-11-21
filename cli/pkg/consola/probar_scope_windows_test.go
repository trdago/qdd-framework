//go:build windows

package consola

// probarScope existe en separar_unix.go; en Windows no hay systemd-run, así que las pruebas que lo apagan comparten
// esta variable sin efecto y compilan en las tres plataformas.
var probarScope = func() bool { return false }
