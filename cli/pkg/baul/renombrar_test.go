package baul

import (
	"errors"
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-20): en Config del panel se pueden «quitar, agregar, modificar los nombres». Renombrar
// mueve el valor sin mostrarlo, sin pedirlo de nuevo y sin dejar una copia con el nombre anterior.

func TestRenombrar_EnElLlaveroMueveElValorYLosPermisos(t *testing.T) {
	entorno(t)
	guardarCloudflare(t)
	if err := conPermisos(ahora, func(ps *permisos) error {
		ps.Permisos = append(ps.Permisos, Permiso{Sesion: "s1", Proyecto: raizActual, Token: "cf", Fin: "dns", Dado: ahora})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := Renombrar("cf", "cf-prod", ahora); err != nil {
		t.Fatal(err)
	}
	if _, err := Buscar("cf"); !errors.Is(err, ErrNoExiste) {
		t.Fatalf("el nombre anterior ya no está: %v", err)
	}
	e, err := Buscar("cf-prod")
	if err != nil || e.Tipo != "cloudflare" || e.Variable != "CLOUDFLARE_API_TOKEN" {
		t.Fatalf("la entrada conserva todo menos el nombre: %+v %v", e, err)
	}
	v, err := leerValor(e)
	if err != nil || v != secreto {
		t.Fatalf("el valor quedó con el nombre nuevo: %v", err)
	}
	if _, err := keyring.Get(servicioLlavero, "cf"); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("no queda una copia con el nombre anterior: %v", err)
	}
	ok, err := tienePermiso("s1", 0, raizActual, "cf-prod", "dns", ahora)
	if err != nil || !ok {
		t.Fatalf("el permiso sigue al token: %v %v", ok, err)
	}
	lineas, _ := Registro()
	ultima := lineas[len(lineas)-1]
	if ultima.Accion != "renombrar" || ultima.Token != "cf-prod" || ultima.Antes != "cf" {
		t.Fatalf("queda anotado con el nombre anterior: %+v", ultima)
	}
	if err := VerificarRegistro(); err != nil {
		t.Fatal(err)
	}
}

func TestRenombrar_NoPisaOtroNiAceptaUnNombreInvalido(t *testing.T) {
	entorno(t)
	guardarCloudflare(t)
	if err := Guardar(Entrada{Nombre: "gh", Tipo: "github", Donde: EnLlavero}, "ghp-OTRO-1234567890", ahora); err != nil {
		t.Fatal(err)
	}
	if err := Renombrar("cf", "gh", ahora); err == nil {
		t.Fatal("no pisa un token que ya existe")
	}
	if err := Renombrar("cf", "Con Espacios", ahora); err == nil {
		t.Fatal("el nombre nuevo sigue las reglas de nombre")
	}
	if err := Renombrar("no-hay", "otro", ahora); !errors.Is(err, ErrNoExiste) {
		t.Fatalf("uno que no existe: %v", err)
	}
	if err := Renombrar("cf", "cf", ahora); err == nil {
		t.Fatal("el mismo nombre no es un cambio")
	}
	v, _ := keyring.Get(servicioLlavero, "gh")
	if v != "ghp-OTRO-1234567890" {
		t.Fatal("el otro token quedó intacto")
	}
}

func TestRenombrar_EnUnoPasswordCambiaElTituloSinTocarElValor(t *testing.T) {
	entorno(t)
	items := opReal(t)
	if err := Guardar(Entrada{Nombre: "pg-app", Tipo: "otro", Variable: "PGPASSWORD", Donde: EnUnoPass}, "valor-uno-123456", ahora); err != nil {
		t.Fatal(err)
	}
	if err := Renombrar("pg-app", "pg-prod", ahora); err != nil {
		t.Fatal(err)
	}
	quedan, _ := os.ReadDir(items)
	if len(quedan) != 1 {
		t.Fatalf("un solo ítem: quedan %d", len(quedan))
	}
	e, _ := Buscar("pg-prod")
	v, err := leerValor(e)
	if err != nil || v != "valor-uno-123456" {
		t.Fatalf("se lee con el título nuevo: %v", err)
	}
}
