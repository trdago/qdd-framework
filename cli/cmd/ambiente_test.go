package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/ambientes"
	"github.com/qdd-framework/qdd/pkg/baul"
)

// Rule: Prohibido usar 'else'
// RFC 0023, fase 3 (solo plan; lo eligió el modo goal con la opción recomendada): 'qdd ambiente levantar dev' lee el
// DNS en Cloudflare con el token del baúl y muestra el plan. No cambia nada.

type lectorFalso struct {
	registros []ambientes.Registro
	zona      string
}

func (l *lectorFalso) Registros(_ context.Context, zona string) ([]ambientes.Registro, error) {
	l.zona = zona
	return l.registros, nil
}

func proyectoConAmbientes(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	contenido := "formato: 1\nambientes:\n  dev:\n    dns:\n      zona: ejemplo.cl\n      registros:\n        - {tipo: CNAME, nombre: app, valor: app-dev.pages.dev, proxied: true}\n        - {tipo: A, nombre: api, valor: 203.0.113.10}\n"
	if err := os.WriteFile(filepath.Join(root, ambientes.Ruta), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func conLector(t *testing.T, l *lectorFalso) {
	t.Helper()
	antes := lectorDeAmbiente
	lectorDeAmbiente = func(string) lectorDNS { return l }
	t.Cleanup(func() { lectorDeAmbiente = antes })
}

func TestAmbienteLevantar_MuestraElPlanYNoCambiaNada(t *testing.T) {
	t.Setenv(variableCloudflare, "tok")
	l := &lectorFalso{registros: []ambientes.Registro{{Tipo: "CNAME", Nombre: "app.ejemplo.cl", Valor: "viejo.pages.dev", Proxied: true}}}
	conLector(t, l)
	var out, errs bytes.Buffer
	if c := ambienteLevantar(proyectoConAmbientes(t), "dev", &out, &errs); c != 0 {
		t.Fatalf("sale con 0: %d %s", c, errs.String())
	}
	texto := out.String()
	for _, debe := range []string{"ejemplo.cl", "cambiar", "app.ejemplo.cl", "viejo.pages.dev", "crear", "api.ejemplo.cl", "Solo plan"} {
		if !strings.Contains(texto, debe) {
			t.Fatalf("el plan dice %q:\n%s", debe, texto)
		}
	}
	if l.zona != "ejemplo.cl" {
		t.Fatalf("lee la zona del ambiente: %q", l.zona)
	}
}

func TestAmbienteLevantar_UnAmbienteQueNoEstaDeclaradoLoDice(t *testing.T) {
	t.Setenv(variableCloudflare, "tok")
	conLector(t, &lectorFalso{})
	var out, errs bytes.Buffer
	if c := ambienteLevantar(proyectoConAmbientes(t), "prod", &out, &errs); c == 0 || !strings.Contains(errs.String(), "dev") {
		t.Fatalf("prod no está: nombra los que sí: %d %q", c, errs.String())
	}
}

func TestAmbienteLevantar_SinTokenLoPideAlBaulConSuFin(t *testing.T) {
	t.Setenv(variableCloudflare, "")
	t.Setenv(variableAmbienteDelBaul, "")
	antesEntrada, antesUsar := entradaCloudflareDeAmbiente, usarBaul
	t.Cleanup(func() { entradaCloudflareDeAmbiente, usarBaul = antesEntrada, antesUsar })
	entradaCloudflareDeAmbiente = func(amb string) (baul.Entrada, bool) {
		return baul.Entrada{Nombre: "cloudflare-" + amb, Tipo: "cloudflare"}, true
	}
	var usado baul.OpcionesUsar
	usarBaul = func(o baul.OpcionesUsar) (int, error) {
		usado = o
		return 0, nil
	}
	var out, errs bytes.Buffer
	if c := ambienteLevantar(proyectoConAmbientes(t), "dev", &out, &errs); c != 0 {
		t.Fatalf("la vuelta con el token salió bien: %d %s", c, errs.String())
	}
	if usado.Nombre != "cloudflare-dev" || usado.Fin != FinAmbientes || len(usado.Comando) == 0 {
		t.Fatalf("pide al baúl el token del ambiente, con su fin, para correr esta misma vuelta: %+v", usado)
	}
}

func TestAmbienteLevantar_SinCloudflareEnElBaulDiceComoGuardarlo(t *testing.T) {
	t.Setenv(variableCloudflare, "")
	t.Setenv(variableAmbienteDelBaul, "")
	antes := entradaCloudflareDeAmbiente
	t.Cleanup(func() { entradaCloudflareDeAmbiente = antes })
	entradaCloudflareDeAmbiente = func(string) (baul.Entrada, bool) { return baul.Entrada{}, false }
	var out, errs bytes.Buffer
	if c := ambienteLevantar(proyectoConAmbientes(t), "dev", &out, &errs); c == 0 || !strings.Contains(errs.String(), "qdd baul guardar") {
		t.Fatalf("dice cómo guardar el token, sin pedirlo por el chat: %d %q", c, errs.String())
	}
}

func TestCloudflareDeAmbiente_PrefiereLaEntradaDeEseAmbiente(t *testing.T) {
	indice := []baul.Entrada{
		{Nombre: "cf-general", Tipo: "cloudflare"},
		{Nombre: "cf-prod", Tipo: "cloudflare", Direccion: "prod/cloudflare.com/dns/token/mi-organizacion"},
		{Nombre: "cf-dev-rechazado", Tipo: "cloudflare", Direccion: "dev/cloudflare.com/dns/token/mi-organizacion", Rechazado: true},
	}
	if e, _ := cloudflareDe(indice, "prod"); e.Nombre != "cf-prod" {
		t.Fatalf("la de prod para prod: %+v", e)
	}
	if e, _ := cloudflareDe(indice, "dev"); e.Nombre != "cf-general" {
		t.Fatalf("sin una de dev que sirva, la general: %+v", e)
	}
	if _, ok := cloudflareDe(nil, "dev"); ok {
		t.Fatal("sin entradas, ninguna")
	}
}
