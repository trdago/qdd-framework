package baul

import (
	"errors"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-13): cada acceso se nombra de forma estándar, ambiente/servidor/servicio/clase/cuenta, «para que
// no se genere 2 veces la misma entrada para el mismo usuario, servicio, servidor o producto».

func TestDireccion_SeNormaliza(t *testing.T) {
	casos := map[string]string{
		"prod/db-prod-01/notificador/postgres/app_user":           "prod/db-prod-01/notificador/postgres/app_user",
		" Prod / DB-Prod-01 / Notificador / Postgres / App_User ": "prod/db-prod-01/notificador/postgres/app_user",
		"global/https://github.com/qdd-framework/token/trdago":    "global/github.com/qdd-framework/token/trdago",
	}
	for entrada, want := range casos {
		got, err := NormalizarDireccion(entrada)
		if err != nil || got != want {
			t.Errorf("%q: got %q %v, want %q", entrada, got, err, want)
		}
	}
	for _, mala := range []string{"prod/db/notificador/postgres", "prod/db//postgres/app", "prod/a/b/c/d/e", "prod/db/not ificador/postgres/app", "produccion/db/notificador/postgres/app"} {
		if _, err := NormalizarDireccion(mala); err == nil {
			t.Errorf("%q: una dirección incompleta o con espacios se rechaza", mala)
		}
	}
}

func TestDireccion_LaMismaNoSeGuardaDosVeces(t *testing.T) {
	entorno(t)
	latir(t)
	primera := Entrada{Nombre: "pg-notificador", Tipo: "otro", Variable: "PGPASSWORD", Donde: EnLlavero, Direccion: "prod/db-prod-01/notificador/postgres/app_user"}
	if err := Guardar(primera, "clave-uno-123456", ahora); err != nil {
		t.Fatal(err)
	}
	otra := Entrada{Nombre: "notificador-pg", Tipo: "otro", Variable: "PGPASSWORD", Donde: EnLlavero, Direccion: "PROD/DB-Prod-01/Notificador/postgres/APP_USER"}
	err := Guardar(otra, "clave-dos-123456", ahora)
	if !errors.Is(err, ErrDireccionRepetida) {
		t.Fatalf("la misma dirección con otro nombre se rechaza: %v", err)
	}
	if _, err := Buscar("notificador-pg"); !errors.Is(err, ErrNoExiste) {
		t.Fatalf("la repetida no queda en el índice: %v", err)
	}
	if _, err := leerValor(otra); err == nil {
		t.Fatal("la repetida no deja su valor en el llavero")
	}
	if err := Guardar(primera, "clave-rotada-123456", ahora); err != nil {
		t.Fatalf("guardar de nuevo el mismo nombre es modificarla (rotar la clave): %v", err)
	}
	e, _ := Buscar("pg-notificador")
	if e.Direccion != "prod/db-prod-01/notificador/postgres/app_user" {
		t.Fatalf("el índice guarda la dirección normalizada: %q", e.Direccion)
	}
}

func TestDireccion_OtraCuentaUOtroAmbienteEsOtraEntrada(t *testing.T) {
	entorno(t)
	latir(t)
	for _, e := range []Entrada{
		{Nombre: "pg-app", Tipo: "otro", Variable: "PGPASSWORD", Donde: EnLlavero, Direccion: "prod/db-prod-01/notificador/postgres/app_user"},
		{Nombre: "pg-admin", Tipo: "otro", Variable: "PGPASSWORD", Donde: EnLlavero, Direccion: "prod/db-prod-01/notificador/postgres/admin"},
		{Nombre: "pg-app-qa", Tipo: "otro", Variable: "PGPASSWORD", Donde: EnLlavero, Direccion: "qa/db-prod-01/notificador/postgres/app_user"},
		{Nombre: "sin-direccion", Tipo: "otro", Variable: "X_TOKEN", Donde: EnLlavero},
	} {
		if err := Guardar(e, "valor-de-"+e.Nombre, ahora); err != nil {
			t.Fatalf("%s: %v", e.Nombre, err)
		}
	}
}

func TestDireccion_ModificarSinDireccionLaConserva(t *testing.T) {
	entorno(t)
	latir(t)
	e := Entrada{Nombre: "pg-app", Tipo: "otro", Variable: "PGPASSWORD", Donde: EnLlavero, Direccion: "prod/db-prod-01/notificador/postgres/app_user"}
	if err := Guardar(e, "clave-uno-123456", ahora); err != nil {
		t.Fatal(err)
	}
	e.Direccion = ""
	if err := Guardar(e, "clave-dos-123456", ahora); err != nil {
		t.Fatal(err)
	}
	if x, _ := Buscar("pg-app"); x.Direccion != "prod/db-prod-01/notificador/postgres/app_user" {
		t.Fatalf("rotar la clave sin decir la dirección no la borra: %q", x.Direccion)
	}
}
