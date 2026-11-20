package baul

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Agregar un acceso pregunta lo que la persona sabe de él (pedido de la persona, 2026-11-20): correo, sitio (de Sitios,
// o la URL si es otro), usuario o id de cuenta, el token o la contraseña, una descripción opcional y el ambiente.

// SitioConocido es un tipo con su nombre para la persona y su dominio.
type SitioConocido struct {
	Tipo     string `json:"tipo"`
	Etiqueta string `json:"etiqueta"`
	Host     string `json:"host"`
}

// Sitios son los que el panel ofrece en la lista; «otro» pide la URL.
var Sitios = []SitioConocido{
	{Tipo: "cloudflare", Etiqueta: "Cloudflare", Host: "cloudflare.com"},
	{Tipo: "github", Etiqueta: "GitHub", Host: "github.com"},
	{Tipo: "supabase", Etiqueta: "Supabase", Host: "supabase.com"},
	{Tipo: "resend", Etiqueta: "Resend", Host: "resend.com"},
	{Tipo: "aws-sso", Etiqueta: "AWS (perfil de SSO)", Host: "aws.amazon.com"},
}

var correoValido = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

const (
	topeCuenta      = 100
	topeDescripcion = 200
)

// Sitio da el dominio del acceso: el de su tipo, o el host de su URL si es «otro».
func Sitio(e Entrada) string {
	i := slices.IndexFunc(Sitios, func(s SitioConocido) bool { return s.Tipo == e.Tipo })
	if i >= 0 {
		return Sitios[i].Host
	}
	u, err := url.Parse(e.URL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// normalizarAcceso limpia y valida lo que la persona escribió, y saca la variable de «otro» del dominio de su URL.
func normalizarAcceso(e Entrada) (Entrada, error) {
	e.Correo = strings.ToLower(strings.TrimSpace(e.Correo))
	e.Cuenta = strings.TrimSpace(e.Cuenta)
	e.Descripcion = strings.TrimSpace(e.Descripcion)
	e.Ambiente = strings.ToLower(strings.TrimSpace(e.Ambiente))
	if e.Correo != "" && !correoValido.MatchString(e.Correo) {
		return e, fmt.Errorf("correo %q: no parece un correo", e.Correo)
	}
	if len(e.Cuenta) > topeCuenta || strings.ContainsAny(e.Cuenta, "\r\n") {
		return e, fmt.Errorf("usuario o id de cuenta: una línea de hasta %d caracteres", topeCuenta)
	}
	if len(e.Descripcion) > topeDescripcion || strings.ContainsAny(e.Descripcion, "\r\n") {
		return e, fmt.Errorf("descripción: una línea de hasta %d caracteres", topeDescripcion)
	}
	if e.Ambiente != "" && !slices.Contains(Ambientes, e.Ambiente) {
		return e, fmt.Errorf("ambiente %q: %s", e.Ambiente, strings.Join(Ambientes, ", "))
	}
	if strings.TrimSpace(e.URL) == "" {
		e.URL = ""
		return e, nil
	}
	u, err := normalizarURL(e.URL)
	if err != nil {
		return e, err
	}
	e.URL = u
	if e.Variable == "" {
		e.Variable = variableDeHost(Sitio(e))
	}
	return e, nil
}

// normalizarURL acepta «sitio.com/ruta» o una URL entera y la deja con https y el host en minúsculas.
func normalizarURL(crudo string) (string, error) {
	crudo = strings.TrimSpace(crudo)
	if !strings.Contains(crudo, "://") {
		crudo = "https://" + crudo
	}
	u, err := url.Parse(crudo)
	if err != nil || u.Hostname() == "" || strings.ContainsAny(u.Host, " \t") {
		return "", fmt.Errorf("URL %q: escribe la dirección del sitio, por ejemplo app.sendgrid.com", strings.TrimPrefix(crudo, "https://"))
	}
	u.Host = strings.ToLower(u.Host)
	u.RawQuery, u.Fragment = "", ""
	return strings.TrimSuffix(u.String(), "/"), nil
}

// variableDeHost: app.sendgrid.com → SENDGRID_TOKEN (el dominio sin subdominios ni terminación).
func variableDeHost(host string) string {
	partes := strings.Split(host, ".")
	nombre := partes[0]
	if len(partes) >= 2 {
		nombre = partes[len(partes)-2]
	}
	nombre = strings.ToUpper(regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(nombre, "_"))
	if nombre == "" || nombre[0] >= '0' && nombre[0] <= '9' {
		nombre = "SITIO_" + nombre
	}
	return nombre + "_TOKEN"
}

// conservarAcceso: modificar el token sin decir correo, URL, cuenta, descripción, ambiente o dirección los conserva,
// y la variable de «otro» también.
func conservarAcceso(e *Entrada, anterior Entrada) {
	campos := []struct{ nuevo, viejo *string }{
		{&e.Correo, &anterior.Correo}, {&e.URL, &anterior.URL}, {&e.Cuenta, &anterior.Cuenta},
		{&e.Descripcion, &anterior.Descripcion}, {&e.Ambiente, &anterior.Ambiente}, {&e.Direccion, &anterior.Direccion},
	}
	if e.Tipo == "otro" && anterior.Tipo == "otro" {
		campos = append(campos, struct{ nuevo, viejo *string }{&e.Variable, &anterior.Variable})
	}
	for _, c := range campos {
		if *c.nuevo == "" {
			*c.nuevo = *c.viejo
		}
	}
}

// Cambios son los datos que Editar cambia; nil no se toca y "" lo borra. Vence va como AAAA-MM-DD.
type Cambios struct {
	Correo, URL, Cuenta, Descripcion, Ambiente, Vence *string
}

// FechaDeclarada lee la fecha de vencimiento que escribe la persona (AAAA-MM-DD). Vacía es ninguna.
func FechaDeclarada(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, fmt.Errorf("vence %q: escribe la fecha como AAAA-MM-DD (2027-01-31)", s)
	}
	return &t, nil
}

// VenceEn: la fecha que dio la API manda; sin ella, la que declaró la persona.
func VenceEn(e Entrada) *time.Time {
	if e.Vence != nil {
		return e.Vence
	}
	return e.VenceDeclarado
}

// Editar cambia los datos de un acceso sin tocar su valor (pedido de la persona, 2026-11-20: corregir el correo o la
// descripción no pide el token de nuevo).
func Editar(nombre string, c Cambios, ahora time.Time) error {
	e, err := Buscar(nombre)
	if err != nil {
		return err
	}
	for _, p := range []struct {
		desde *string
		hacia *string
	}{{c.Correo, &e.Correo}, {c.URL, &e.URL}, {c.Cuenta, &e.Cuenta}, {c.Descripcion, &e.Descripcion}, {c.Ambiente, &e.Ambiente}} {
		if p.desde != nil {
			*p.hacia = *p.desde
		}
	}
	if c.Vence != nil {
		vence, err := FechaDeclarada(*c.Vence)
		if err != nil {
			return err
		}
		e.VenceDeclarado = vence
	}
	e, err = completar(e)
	if err != nil {
		return err
	}
	if err := cambiarEntrada(nombre, func(x *Entrada) { *x = e }); err != nil {
		return err
	}
	return Anotar(Linea{Accion: "editar", Token: nombre}, ahora)
}

// CorreoDistinto avisa cuando la API dice que el token es de otro correo que el que escribió la persona. Una API que no
// da un correo (GitHub da el usuario) no avisa.
func CorreoDistinto(e Entrada) string {
	if e.Correo == "" || !strings.Contains(e.Duenio, "@") || strings.EqualFold(e.Correo, e.Duenio) {
		return ""
	}
	return fmt.Sprintf("%s es de %s según su API, no de %s: revisa que copiaste el token correcto", e.Nombre, e.Duenio, e.Correo)
}
