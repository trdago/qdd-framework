// Package aislar corre las pruebas de un paquete sin acceso a las carpetas de QDD de la persona (FND-246): sin él,
// una prueba que no fijaba QDD_HOME escribía en ~/.local/share/qdd (525 archivos de turnos en un PC).
package aislar

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

// Variable es la que elige la carpeta central de QDD (blueprints.HomeEnv; aquí sin importarlo, para que blueprints
// también pueda usar este paquete en sus pruebas).
const Variable = "QDD_HOME"

// prefijo es el de la carpeta temporal que hace las veces de QDD_HOME.
const prefijo = "qdd-pruebas-"

// Pruebas es lo que corre las de un paquete: el *testing.M de su TestMain (sin importar testing en este paquete, que
// no es de pruebas).
type Pruebas interface{ Run() int }

// Correr corre las pruebas con un QDD_HOME temporal que se borra al terminar, para usar en TestMain:
// os.Exit(aislar.Correr(m)). Lo que las pruebas lanzan (un qdd hijo) lo hereda. Una prueba que fija su propio
// QDD_HOME con t.Setenv sigue mandando.
func Correr(m Pruebas) int {
	dir, propia, err := Carpeta(prefijo)
	if err != nil {
		return 1
	}
	os.Setenv(Variable, dir)
	if !propia {
		return m.Run()
	}
	defer Borrar(dir)
	return m.Run()
}

// Carpeta da el QDD_HOME temporal de un TestMain: el que heredó de su padre si es un ayudante que la prueba lanzó con
// el mismo binario (-test.run=…), o uno nuevo con ese prefijo (propia: lo borra quien lo creó). Si cada ayudante creara
// el suyo, al matarlo o salir con os.Exit quedaba en el TMPDIR de la máquina (FND-288).
func Carpeta(prefijoDeCarpeta string) (dir string, propia bool, err error) {
	heredada := os.Getenv(Variable)
	if strings.HasPrefix(filepath.Base(heredada), prefijoDeCarpeta) && existe(heredada) {
		return heredada, false, nil
	}
	dir, err = os.MkdirTemp("", prefijoDeCarpeta)
	return dir, err == nil, err
}

func existe(dir string) bool {
	_, err := os.Stat(dir)
	return err == nil
}

// Borrar insiste durante 1 s: un pedido o un qdd que lanzó una prueba puede seguir un momento en su proceso y volver a
// crear su carpeta justo después del primer borrado, y quedaba en el TMPDIR de la máquina (FND-288). Para los TestMain
// que arman su propia carpeta central.
func Borrar(dir string) {
	os.RemoveAll(dir)
	for fin := time.Now().Add(time.Second); time.Now().Before(fin); {
		time.Sleep(100 * time.Millisecond)
		os.RemoveAll(dir)
	}
}
