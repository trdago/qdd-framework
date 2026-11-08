// Package baul guarda los tokens de la persona en el llavero del sistema o en 1Password y los presta a un comando,
// con permiso por sesión y por fin, la salida tapada y un registro encadenado (RFC 0023, fase 1).
package baul

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/candado"
)

// Rule: Prohibido usar 'else'

const (
	EnLlavero = "llavero"
	EnUnoPass = "1password"
)

// Tipos: el tipo dice en qué variable de entorno va el token. «otro» la trae en la entrada.
var Tipos = map[string]string{
	"cloudflare": "CLOUDFLARE_API_TOKEN",
	"github":     "GITHUB_TOKEN",
	"supabase":   "SUPABASE_ACCESS_TOKEN",
	"resend":     "RESEND_API_KEY",
	"aws-sso":    "AWS_PROFILE",
	"otro":       "",
}

// Entrada es lo que el índice sabe de un token. Nunca el valor.
type Entrada struct {
	Nombre     string     `json:"nombre"`
	Tipo       string     `json:"tipo"`
	Donde      string     `json:"donde"`
	Variable   string     `json:"variable"`
	Perfil     string     `json:"perfil,omitempty"` // aws-sso: el perfil de aws sso, no una llave
	Alcance    string     `json:"alcance,omitempty"`
	Duenio     string     `json:"duenio,omitempty"`
	Vence      *time.Time `json:"vence,omitempty"`
	Verificado *time.Time `json:"verificado,omitempty"`
	Rechazado  bool       `json:"rechazado,omitempty"`
	Guardado   time.Time  `json:"guardado"`
	Usado      *time.Time `json:"usado,omitempty"`
}

var nombreValido = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)

var variableValida = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// ErrNoExiste: no hay un token con ese nombre en el baúl.
var ErrNoExiste = errors.New("no hay un token con ese nombre en el baúl (qdd baul)")

func carpeta() (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "baul")
	return dir, os.MkdirAll(dir, 0o700)
}

func rutaEn(nombre string) (string, error) {
	dir, err := carpeta()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, nombre), nil
}

// conBloqueo corre f con el baúl bloqueado: un índice, unos permisos y un registro a la vez.
func conBloqueo(f func() error) error {
	lock, err := rutaEn("baul.lock")
	if err != nil {
		return err
	}
	soltar, err := candado.Tomar(lock, 5*time.Second, "el baúl")
	if err != nil {
		return err
	}
	defer soltar()
	return f()
}

func leerJSON(nombre string, v any) error {
	ruta, err := rutaEn(nombre)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func escribirJSON(nombre string, v any) error {
	ruta, err := rutaEn(nombre)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := ruta + ".nuevo"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

// Indice devuelve los tokens del baúl, sin valores.
func Indice() ([]Entrada, error) {
	var indice []Entrada
	return indice, leerJSON("indice.json", &indice)
}

// Buscar devuelve la entrada de un token.
func Buscar(nombre string) (Entrada, error) {
	indice, err := Indice()
	if err != nil {
		return Entrada{}, err
	}
	i := slices.IndexFunc(indice, func(e Entrada) bool { return e.Nombre == nombre })
	if i < 0 {
		return Entrada{}, ErrNoExiste
	}
	return indice[i], nil
}

// cambiarEntrada aplica f a la entrada del token (o agrega una nueva) con el baúl bloqueado.
func cambiarEntrada(nombre string, f func(*Entrada)) error {
	return conBloqueo(func() error {
		indice, err := Indice()
		if err != nil {
			return err
		}
		i := slices.IndexFunc(indice, func(e Entrada) bool { return e.Nombre == nombre })
		if i < 0 {
			indice = append(indice, Entrada{Nombre: nombre})
			i = len(indice) - 1
		}
		f(&indice[i])
		return escribirJSON("indice.json", indice)
	})
}

// completar valida la entrada y le pone la variable de su tipo.
func completar(e Entrada) (Entrada, error) {
	if !nombreValido.MatchString(e.Nombre) {
		return e, fmt.Errorf("nombre %q: usa minúsculas, números, - y _ (hasta 40)", e.Nombre)
	}
	variable, ok := Tipos[e.Tipo]
	if !ok {
		return e, fmt.Errorf("tipo %q: usa cloudflare, github, supabase, resend, aws-sso u otro", e.Tipo)
	}
	if variable != "" {
		e.Variable = variable
	}
	if !variableValida.MatchString(e.Variable) {
		return e, fmt.Errorf("el tipo «otro» necesita la variable de entorno donde va (--variable NOMBRE)")
	}
	if e.Donde != EnLlavero && e.Donde != EnUnoPass {
		return e, fmt.Errorf("dónde guardarlo %q: llavero o 1password", e.Donde)
	}
	return e, nil
}
