package baul

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
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
	if e.Tipo == "aws-sso" {
		e.Perfil = strings.TrimSpace(valor)
	}
	if e.Tipo != "aws-sso" {
		if err := guardarValor(e, valor); err != nil {
			return err
		}
	}
	e.Guardado = ahora.UTC()
	if err := cambiarEntrada(e.Nombre, func(x *Entrada) { *x = e }); err != nil {
		return err
	}
	return Anotar(Linea{Accion: "guardar", Token: e.Nombre}, ahora)
}

func guardarValor(e Entrada, valor string) error {
	if e.Donde == EnLlavero {
		return keyring.Set(servicioLlavero, e.Nombre, valor)
	}
	return guardarEnOp(e, valor)
}

// guardarEnOp crea el ítem con la plantilla por la entrada estándar: el valor nunca va en los argumentos.
func guardarEnOp(e Entrada, valor string) error {
	plantilla := map[string]any{
		"title":    tituloOp(e.Nombre),
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
