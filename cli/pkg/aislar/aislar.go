// Package aislar corre las pruebas de un paquete sin acceso a las carpetas de QDD de la persona (FND-246): sin él,
// una prueba que no fijaba QDD_HOME escribía en ~/.local/share/qdd (525 archivos de turnos en un PC).
package aislar

import "os"

// Rule: Prohibido usar 'else'

// Variable es la que elige la carpeta central de QDD (blueprints.HomeEnv; aquí sin importarlo, para que blueprints
// también pueda usar este paquete en sus pruebas).
const Variable = "QDD_HOME"

// Pruebas es lo que corre las de un paquete: el *testing.M de su TestMain (sin importar testing en este paquete, que
// no es de pruebas).
type Pruebas interface{ Run() int }

// Correr corre las pruebas con un QDD_HOME temporal que se borra al terminar, para usar en TestMain:
// os.Exit(aislar.Correr(m)). Lo que las pruebas lanzan (un qdd hijo) lo hereda. Una prueba que fija su propio
// QDD_HOME con t.Setenv sigue mandando.
func Correr(m Pruebas) int {
	dir, err := os.MkdirTemp("", "qdd-pruebas-")
	if err != nil {
		return 1
	}
	defer os.RemoveAll(dir)
	os.Setenv(Variable, dir)
	return m.Run()
}
