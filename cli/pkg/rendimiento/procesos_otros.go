//go:build !linux && !darwin && !windows

package rendimiento

import "errors"

// Procesos en otro sistema: sin una forma conocida de leer la tabla, lo que depende de ella queda sin medir.
func Procesos() ([]Proceso, error) {
	return nil, errors.New("este sistema no tiene cómo listar procesos")
}
