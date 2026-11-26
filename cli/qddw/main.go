// Command qddw es el lanzador sin ventana de qdd en Windows (sprint 9, FND-238): la tarea «QDD remoto» lo lanza al
// iniciar sesión en vez de qdd.exe. Se compila como programa de ventanas (banderasRelease), así que Windows nunca le
// abre una consola, y corre el qdd.exe de su misma carpeta con los mismos argumentos, también sin ventana. Sale con el
// código de qdd: la tarea lo relanza si termina con error. qdd.exe lo lleva dentro y lo deja a su lado
// (pkg/remoto/lanzador.go); solo usa la biblioteca estándar y x/sys para seguir siendo chico.
package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

// Rule: Prohibido usar 'else'

const (
	// envSinConsola le dice a qdd que no tiene consola: escribe su registro en un archivo (remoto.EnvSinConsola).
	envSinConsola = "QDD_SIN_CONSOLA"
	// codigoSinQDD: no hay qdd en su carpeta. La tarea apunta a una versión que ya no está, y doctor lo ve (REM-04).
	codigoSinQDD = 2
	// banderasRelease son las de publish.yml: programa de ventanas y sin símbolos.
	banderasRelease = "-H=windowsgui -s -w"
)

func main() {
	propio, err := os.Executable()
	if err != nil {
		os.Exit(codigoSinQDD)
	}
	os.Exit(correr(propio, os.Args[1:]))
}

// correr lanza el qdd de la carpeta de propio y espera a que termine.
func correr(propio string, args []string) int {
	qdd := filepath.Join(filepath.Dir(propio), nombreQDD)
	if _, err := os.Stat(qdd); err != nil {
		return codigoSinQDD
	}
	c := exec.Command(qdd, args...)
	c.Env = append(os.Environ(), envSinConsola+"=1")
	return codigoDe(lanzar(c))
}

func codigoDe(err error) int {
	var salida *exec.ExitError
	if errors.As(err, &salida) {
		return salida.ExitCode()
	}
	if err != nil {
		return 1
	}
	return 0
}
