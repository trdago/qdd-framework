package baul

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Rule: Prohibido usar 'else'
// La dirección nombra un acceso de forma estándar: ambiente/servidor/servicio/clase/cuenta (pedido de la persona,
// 2026-11-13). Ambiente es uno de Ambientes; clase es qué es (token, postgres, oracle, ssh, client…) y cuenta, el usuario
// o identificador. Una dirección es de una sola entrada: lo mismo no se guarda dos veces con nombres distintos.

// NivelesDireccion son las partes de una dirección, en orden.
var NivelesDireccion = []string{"ambiente", "servidor", "servicio", "clase", "cuenta"}

// Ambientes son los que vale el primer nivel: una lista cerrada, para que «prod» y «produccion» no sean dos entradas.
var Ambientes = []string{"dev", "qa", "prod", "global"}

var (
	parteValida = regexp.MustCompile(`^[a-z0-9][a-z0-9._@:+-]*$`)
	esquema     = regexp.MustCompile(`^[a-z][a-z0-9+.-]*://`)
)

// ErrDireccionRepetida: otra entrada ya tiene esa dirección.
var ErrDireccionRepetida = errors.New("esa dirección ya está en el baúl")

// NormalizarDireccion deja la dirección en minúsculas, sin espacios alrededor de cada parte ni el esquema del servidor
// (https://), y rechaza una que no tenga sus cinco partes o un ambiente que no está en Ambientes.
func NormalizarDireccion(d string) (string, error) {
	d = strings.ToLower(strings.TrimSpace(d))
	partes := strings.SplitN(d, "/", 2)
	if len(partes) == 2 {
		d = strings.TrimSpace(partes[0]) + "/" + esquema.ReplaceAllString(strings.TrimSpace(partes[1]), "")
	}
	partes = strings.Split(d, "/")
	if len(partes) != len(NivelesDireccion) {
		return "", fmt.Errorf("dirección %q: van %d partes, %s", d, len(NivelesDireccion), strings.Join(NivelesDireccion, "/"))
	}
	for i, p := range partes {
		partes[i] = strings.TrimSpace(p)
		if !parteValida.MatchString(partes[i]) {
			return "", fmt.Errorf("dirección %q: el %s %q no vale (minúsculas, números y . _ @ : + -, sin espacios)", d, NivelesDireccion[i], partes[i])
		}
	}
	if !slices.Contains(Ambientes, partes[0]) {
		return "", fmt.Errorf("dirección %q: el ambiente %q no vale (%s)", d, partes[0], strings.Join(Ambientes, ", "))
	}
	return strings.Join(partes, "/"), nil
}

// direccionLibre dice si la dirección de e no la tiene otra entrada del índice.
func direccionLibre(e Entrada) error {
	if e.Direccion == "" {
		return nil
	}
	indice, err := Indice()
	if err != nil {
		return err
	}
	for _, x := range indice {
		if x.Direccion == e.Direccion && x.Nombre != e.Nombre {
			return fmt.Errorf("%w como «%s» (%s): para cambiar su valor, guárdala con ese nombre", ErrDireccionRepetida, x.Nombre, x.Direccion)
		}
	}
	return nil
}
