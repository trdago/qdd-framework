package baul

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

// Rule: Prohibido usar 'else'
// El valor vive en el llavero del sistema (Secret Service, Llavero de macOS, Administrador de credenciales) o en
// 1Password. QDD no cifra por su cuenta ni guarda copias. aws-sso guarda solo el perfil, en el índice.

const servicioLlavero = "qdd-baul"

// binOp es la CLI de 1Password; las pruebas la cambian por una falsa.
var binOp = "op"

func tituloOp(nombre string) string {
	return "qdd-baul-" + nombre
}

// Guardar deja el valor en su lugar y la entrada en el índice, verificada si se puede.
func Guardar(e Entrada, valor string, ahora time.Time) error {
	e, err := completar(e)
	if err != nil {
		return err
	}
	if strings.TrimSpace(valor) == "" {
		return fmt.Errorf("el valor está vacío")
	}
	anterior, errAnterior := Buscar(e.Nombre)
	if e.Direccion == "" && errAnterior == nil {
		e.Direccion = anterior.Direccion
	}
	if err := direccionLibre(e); err != nil {
		return err
	}
	if e.Tipo == "aws-sso" {
		e.Perfil = strings.TrimSpace(valor)
	}
	if e.Tipo != "aws-sso" {
		reemplaza := errAnterior == nil && anterior.Donde == EnUnoPass && anterior.Tipo != "aws-sso"
		if err := guardarValor(e, valor, reemplaza); err != nil {
			return err
		}
	}
	e.Guardado = ahora.UTC()
	if err := cambiarEntrada(e.Nombre, func(x *Entrada) { *x = e }); err != nil {
		return err
	}
	// Modificar a otro lugar (o a aws-sso, que no guarda valor) no deja una copia olvidada del valor anterior.
	if errAnterior == nil && (anterior.Donde != e.Donde || e.Tipo == "aws-sso") {
		_ = borrarValor(anterior)
	}
	return Anotar(Linea{Accion: "guardar", Token: e.Nombre}, ahora)
}

// Quitar borra el valor de su lugar, la entrada del índice y los permisos y solicitudes del token, y lo anota.
func Quitar(nombre string, ahora time.Time) error {
	e, err := Buscar(nombre)
	if err != nil {
		return err
	}
	if err := borrarValor(e); err != nil {
		return err
	}
	if err := conBloqueo(func() error {
		indice, err := Indice()
		if err != nil {
			return err
		}
		return escribirJSON("indice.json", slices.DeleteFunc(indice, func(x Entrada) bool { return x.Nombre == nombre }))
	}); err != nil {
		return err
	}
	if err := conPermisos(ahora, func(ps *permisos) error {
		ps.Permisos = slices.DeleteFunc(ps.Permisos, func(p Permiso) bool { return p.Token == nombre })
		ps.Solicitudes = slices.DeleteFunc(ps.Solicitudes, func(s Solicitud) bool { return s.Token == nombre })
		return nil
	}); err != nil {
		return err
	}
	return Anotar(Linea{Accion: "quitar", Token: nombre}, ahora)
}

// borrarValor saca el valor de su lugar. Uno que ya no estaba no es una falla.
func borrarValor(e Entrada) error {
	if e.Tipo == "aws-sso" {
		return nil
	}
	if e.Donde == EnLlavero {
		err := keyring.Delete(servicioLlavero, e.Nombre)
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	c := exec.Command(binOp, "item", "delete", tituloOp(e.Nombre))
	var errSalida bytes.Buffer
	c.Stderr = &errSalida
	if err := c.Run(); err != nil {
		return fmt.Errorf("1Password no lo borró: %v %s", err, strings.TrimSpace(errSalida.String()))
	}
	return nil
}

// guardarValor deja el valor en el llavero o en 1Password. En 1Password, reemplazar un ítem que ya existe no puede
// crear otro con el mismo título (FND-202): 1Password los deja convivir y después no sabe cuál leer.
func guardarValor(e Entrada, valor string, reemplaza bool) error {
	if e.Donde == EnLlavero {
		return keyring.Set(servicioLlavero, e.Nombre, valor)
	}
	if !reemplaza {
		return guardarEnOp(tituloOp(e.Nombre), valor)
	}
	return reemplazarEnOp(tituloOp(e.Nombre), valor)
}

// reemplazarEnOp crea el ítem nuevo con otro título, borra el anterior y le pone el título: el valor nunca va en los
// argumentos, y si algo falla antes de borrar, el anterior sigue intacto.
func reemplazarEnOp(titulo, valor string) error {
	nuevo := titulo + ".nuevo"
	_ = correrOp("item", "delete", nuevo) // uno que quedó de un intento cortado
	if err := guardarEnOp(nuevo, valor); err != nil {
		return err
	}
	if err := correrOp("item", "delete", titulo); err != nil {
		_ = correrOp("item", "delete", nuevo)
		return fmt.Errorf("1Password no borró el valor anterior, que sigue igual: %w", err)
	}
	if err := correrOp("item", "edit", nuevo, "--title", titulo); err != nil {
		return fmt.Errorf("1Password guardó el valor nuevo como %q pero no le cambió el título: %w", nuevo, err)
	}
	return nil
}

func correrOp(args ...string) error {
	c := exec.Command(binOp, args...)
	var errSalida bytes.Buffer
	c.Stderr = &errSalida
	if err := c.Run(); err != nil {
		return fmt.Errorf("%v %s", err, strings.TrimSpace(errSalida.String()))
	}
	return nil
}

// guardarEnOp crea el ítem con la plantilla por la entrada estándar: el valor nunca va en los argumentos.
func guardarEnOp(titulo, valor string) error {
	plantilla := map[string]any{
		"title":    titulo,
		"category": "API_CREDENTIAL",
		"fields":   []map[string]string{{"id": "credential", "label": "credential", "type": "CONCEALED", "value": valor}},
	}
	b, err := json.Marshal(plantilla)
	if err != nil {
		return err
	}
	c := exec.Command(binOp, "item", "create", "--format", "json")
	c.Stdin = bytes.NewReader(b)
	var errSalida bytes.Buffer
	c.Stderr = &errSalida
	if err := c.Run(); err != nil {
		return fmt.Errorf("1Password no lo guardó: %v %s", err, strings.TrimSpace(errSalida.String()))
	}
	return nil
}

// leerValor trae el valor para un comando. Nunca se imprime.
func leerValor(e Entrada) (string, error) {
	if e.Tipo == "aws-sso" {
		return e.Perfil, nil
	}
	if e.Donde == EnLlavero {
		return keyring.Get(servicioLlavero, e.Nombre)
	}
	c := exec.Command(binOp, "item", "get", tituloOp(e.Nombre), "--fields", "label=credential", "--reveal")
	var salida, errSalida bytes.Buffer
	c.Stdout, c.Stderr = &salida, &errSalida
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("1Password no lo entregó: %v %s", err, strings.TrimSpace(errSalida.String()))
	}
	return strings.TrimRight(salida.String(), "\r\n"), nil
}

// ValoresDelLlavero da los valores guardados en el llavero, solo para taparlos (un reporte a QDD, RFC 0024). Los de
// 1Password no se leen: pedirían desbloquearlo para nada. Lo que no se puede leer se salta.
func ValoresDelLlavero() []string {
	indice, err := Indice()
	if err != nil {
		return nil
	}
	var valores []string
	for _, e := range indice {
		if e.Donde != EnLlavero || e.Tipo == "aws-sso" {
			continue
		}
		v, err := keyring.Get(servicioLlavero, e.Nombre)
		if err != nil || v == "" {
			continue
		}
		valores = append(valores, v)
	}
	return valores
}
