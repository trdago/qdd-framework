package baul

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-20): agregar un acceso pregunta lo que uno sabe de él, no cómo lo guarda qdd: correo,
// sitio (de una lista, o la URL si es otro), usuario o id de cuenta, el token o la contraseña y una descripción opcional.

func TestAcceso_GuardaCorreoSitioCuentaYDescripcion(t *testing.T) {
	entorno(t)
	e := Entrada{Nombre: "sendgrid-ventas", Tipo: "otro", Donde: EnLlavero, URL: " App.SendGrid.com/settings ", Correo: " Yo@X.cl ", Cuenta: " ventas ", Descripcion: " envíos del sitio ", Ambiente: "prod"}
	if err := Guardar(e, "SG.valor-1234567890", ahora); err != nil {
		t.Fatal(err)
	}
	g, _ := Buscar("sendgrid-ventas")
	if g.URL != "https://app.sendgrid.com/settings" || g.Correo != "yo@x.cl" || g.Cuenta != "ventas" || g.Descripcion != "envíos del sitio" || g.Ambiente != "prod" {
		t.Fatalf("se normaliza lo escrito: %+v", g)
	}
	if g.Variable != "SENDGRID_TOKEN" {
		t.Fatalf("«otro» con URL saca su variable del dominio: %q", g.Variable)
	}
	if Sitio(g) != "app.sendgrid.com" {
		t.Fatalf("el sitio de «otro» es el host de su URL: %q", Sitio(g))
	}
	cf := Entrada{Nombre: "cf", Tipo: "cloudflare", Donde: EnLlavero}
	if Sitio(cf) != "cloudflare.com" {
		t.Fatalf("el sitio de un tipo conocido sale de la lista: %q", Sitio(cf))
	}
}

func TestAcceso_RechazaCorreoUrlYAmbienteInvalidos(t *testing.T) {
	entorno(t)
	malas := map[string]Entrada{
		"correo sin @":          {Nombre: "a", Tipo: "cloudflare", Donde: EnLlavero, Correo: "yo.x.cl"},
		"correo con espacios":   {Nombre: "a", Tipo: "cloudflare", Donde: EnLlavero, Correo: "yo @x.cl"},
		"otro sin URL ni var":   {Nombre: "a", Tipo: "otro", Donde: EnLlavero},
		"URL sin host":          {Nombre: "a", Tipo: "otro", Donde: EnLlavero, URL: "https://"},
		"ambiente inventado":    {Nombre: "a", Tipo: "cloudflare", Donde: EnLlavero, Ambiente: "produccion"},
		"descripción en líneas": {Nombre: "a", Tipo: "cloudflare", Donde: EnLlavero, Descripcion: "uno\ndos"},
	}
	for caso, e := range malas {
		if err := Guardar(e, "valor-1234567890", ahora); err == nil {
			t.Errorf("%s: se rechaza", caso)
		}
	}
}

func TestAcceso_ModificarElTokenConservaLoQueNoSeDice(t *testing.T) {
	entorno(t)
	e := Entrada{Nombre: "cf", Tipo: "cloudflare", Donde: EnLlavero, Correo: "yo@x.cl", Cuenta: "trdago", Descripcion: "DNS", Ambiente: "global"}
	if err := Guardar(e, secreto, ahora); err != nil {
		t.Fatal(err)
	}
	if err := Guardar(Entrada{Nombre: "cf", Tipo: "cloudflare", Donde: EnLlavero}, "cf-OTRO-1234567890", ahora); err != nil {
		t.Fatal(err)
	}
	g, _ := Buscar("cf")
	if g.Correo != "yo@x.cl" || g.Cuenta != "trdago" || g.Descripcion != "DNS" || g.Ambiente != "global" {
		t.Fatalf("cambiar el token no borra el resto: %+v", g)
	}
}

func TestEditar_CambiaLosDatosSinTocarElValor(t *testing.T) {
	entorno(t)
	guardarCloudflare(t)
	correo, cuenta, vacia := "otra@x.cl", "trdago", ""
	if err := Editar("cf", Cambios{Correo: &correo, Cuenta: &cuenta}, ahora); err != nil {
		t.Fatal(err)
	}
	descripcion := "tokens de DNS"
	if err := Editar("cf", Cambios{Descripcion: &descripcion}, ahora); err != nil {
		t.Fatal(err)
	}
	g, _ := Buscar("cf")
	if g.Correo != "otra@x.cl" || g.Cuenta != "trdago" || g.Descripcion != "tokens de DNS" {
		t.Fatalf("editar cambia solo lo que se dice: %+v", g)
	}
	if err := Editar("cf", Cambios{Descripcion: &vacia}, ahora); err != nil {
		t.Fatal(err)
	}
	g, _ = Buscar("cf")
	if g.Descripcion != "" || g.Correo != "otra@x.cl" {
		t.Fatalf("una descripción vacía la borra: %+v", g)
	}
	v, err := leerValor(g)
	if err != nil || v != secreto {
		t.Fatalf("el valor no cambia: %v", err)
	}
	malo := "sin-arroba"
	if err := Editar("cf", Cambios{Correo: &malo}, ahora); err == nil {
		t.Fatal("editar valida como guardar")
	}
	if err := Editar("no-hay", Cambios{Correo: &correo}, ahora); !errors.Is(err, ErrNoExiste) {
		t.Fatalf("uno que no existe: %v", err)
	}
	lineas, _ := Registro()
	if lineas[len(lineas)-1].Accion != "editar" {
		t.Fatalf("queda anotado: %+v", lineas[len(lineas)-1])
	}
}

func TestCorreoDistinto_AvisaSoloSiLaAPIDioUnCorreo(t *testing.T) {
	casos := []struct {
		e     Entrada
		aviso bool
	}{
		{Entrada{Nombre: "cf", Correo: "yo@x.cl", Duenio: "otro@x.cl"}, true},
		{Entrada{Nombre: "cf", Correo: "yo@x.cl", Duenio: "Yo@X.cl"}, false},
		{Entrada{Nombre: "gh", Correo: "yo@x.cl", Duenio: "trdago"}, false},
		{Entrada{Nombre: "cf", Duenio: "otro@x.cl"}, false},
	}
	for _, c := range casos {
		if got := CorreoDistinto(c.e) != ""; got != c.aviso {
			t.Errorf("%+v: aviso=%v", c.e, got)
		}
	}
}

// Investigación de estándares (2026-11-20): CXF tiene expiryDate, OWASP pide registrar cuándo rotar y NIST SP 800-57 el
// cryptoperiod. Cuando la API no dice cuándo vence, la persona puede declararlo, y doctor avisa igual.
func TestVenceDeclarado_DoctorAvisaYVerificarNoLoBorra(t *testing.T) {
	entorno(t)
	vence, err := FechaDeclarada("2026-11-15")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FechaDeclarada("25/11/2026"); err == nil {
		t.Fatal("la fecha va como AAAA-MM-DD")
	}
	e := Entrada{Nombre: "mio", Tipo: "otro", Donde: EnLlavero, URL: "mi-api.cl", VenceDeclarado: vence}
	if err := Guardar(e, "valor-1234567890", ahora); err != nil {
		t.Fatal(err)
	}
	g, _ := Buscar("mio")
	if VenceEn(g) == nil || VenceEn(g).Format("2006-01-02") != "2026-11-15" {
		t.Fatalf("vence cuando la persona dijo: %+v", g)
	}
	indice, _ := Indice()
	d := diagnosticarVigentes(indice, ahora)
	if !strings.Contains(d.Detalle, "mio vence el 2026-11-15") {
		t.Fatalf("doctor avisa del vencimiento declarado: %+v", d)
	}
	fecha := "2027-01-31"
	if err := Editar("mio", Cambios{Vence: &fecha}, ahora); err != nil {
		t.Fatal(err)
	}
	g, _ = Buscar("mio")
	if VenceEn(g).Format("2006-01-02") != "2027-01-31" {
		t.Fatalf("editar cambia la fecha: %+v", g)
	}
	api := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	g.Vence = &api
	if !VenceEn(g).Equal(api) {
		t.Fatal("la fecha de la API manda sobre la declarada")
	}
}
