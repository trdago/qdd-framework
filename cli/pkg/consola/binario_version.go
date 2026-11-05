package consola

import (
	"path/filepath"
	"regexp"
)

// Rule: Prohibido usar 'else'
// En Windows, npm no puede reemplazar qdd.exe mientras algo lo usa (issue #107, FND-142). Por eso el instalador lo deja
// fuera del paquete, en <LOCALAPPDATA>/qdd/bin/<versión>/qdd.exe: lo que sigue corriendo se queda con su versión, y lo
// nuevo usa la nueva. Ese archivo ya no cambia al actualizar: el que cambia es <LOCALAPPDATA>/qdd/bin/actual.txt, que el
// instalador escribe al terminar. Es lo que miran el flujo del panel y qdd remoto para saber que hay otra versión.

var reBinarioPorVersion = regexp.MustCompile(`[\\/]qdd[\\/]bin[\\/]\d+\.\d+\.\d+[^\\/]*[\\/]qdd\.exe$`)

// BinarioPorVersion: bin es un qdd.exe instalado en su carpeta de versión.
func BinarioPorVersion(bin string) bool {
	return reBinarioPorVersion.MatchString(bin)
}

// RutaVigilada es el archivo que cambia cuando se instala otra versión de qdd.
func RutaVigilada(bin string) string {
	if !BinarioPorVersion(bin) {
		return bin
	}
	return filepath.Join(filepath.Dir(filepath.Dir(bin)), "actual.txt")
}
