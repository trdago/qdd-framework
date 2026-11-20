package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/baul"
	"github.com/zalando/go-keyring"
)

// Rule: Prohibido usar 'else'
// RFC 0023: guardar pide el valor en la terminal de la persona; sin terminal (la herramienta Bash de una IA, un pedido,
// el teléfono) se niega y dice dónde correrlo.
func TestBaulGuardar_SinTerminalSeNiega(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "entrada")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("cf-SECRETO\n")
	f.Seek(0, 0)
	defer f.Close()
	_, err = leerValorOculto(f, "cloudflare")
	if err == nil || !strings.Contains(err.Error(), "tu terminal") {
		t.Fatalf("sin terminal no se lee el valor: %v", err)
	}
}

func TestDoctorBaul_SinBaulNoApareceYConTokensSi(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if g := buildBaulChecks(root); len(g.Items) != 0 {
		t.Fatalf("sin baúl ni .env el grupo no aparece: %+v", g.Items)
	}
	os.WriteFile(root+"/.env", []byte("GITHUB_TOKEN=ghp_valor_de_prueba\n"), 0o600)
	g := buildBaulChecks(root)
	if len(g.Items) != 1 || !g.Items[0].Success || strings.Contains(g.Items[0].Error, "ghp_valor") {
		t.Fatalf("un aviso sin el valor: %+v", g.Items)
	}
}

// El baúl desde Config en el panel (pedido de la persona, 2026-11-09: «agregar, quitar, modificar»).
type copiadoFalso struct{ valor string }

func (c *copiadoFalso) Leer() (string, error) { return c.valor, nil }
func (c *copiadoFalso) Vaciar() error {
	c.valor = ""
	return nil
}

func baulDePrueba(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
}

func TestBaulPanel_GuardarDelPortapapelesListarYQuitar(t *testing.T) {
	baulDePrueba(t)
	const valor = "valor-de-prueba-123"
	copiado := &copiadoFalso{valor: valor}
	var salida bytes.Buffer
	e := baul.Entrada{Nombre: "mio", Tipo: "otro", Donde: baul.EnLlavero, Variable: "MI_TOKEN"}
	if err := guardarDelPortapapeles(&salida, e, copiado); err != nil {
		t.Fatal(err)
	}
	if copiado.valor != "" || strings.Contains(salida.String(), valor) {
		t.Fatalf("vacía el portapapeles y nunca muestra el valor: %q", salida.String())
	}
	var lista bytes.Buffer
	listarBaulJSON(&lista)
	var d struct {
		Tokens       []baul.Entrada `json:"tokens"`
		Tipos        []string       `json:"tipos"`
		Portapapeles bool           `json:"portapapeles"`
	}
	if err := json.Unmarshal(lista.Bytes(), &d); err != nil || len(d.Tokens) != 1 || d.Tokens[0].Nombre != "mio" || len(d.Tipos) != len(baul.Tipos) {
		t.Fatalf("la lista para el panel trae los tokens y los tipos: %s %v", lista.String(), err)
	}
	if strings.Contains(lista.String(), valor) {
		t.Fatalf("la lista nunca trae un valor: %s", lista.String())
	}
	if err := quitarDelBaul("mio", false, nil); err == nil {
		t.Fatal("sin --si ni terminal, quitar no borra")
	}
	if err := quitarDelBaul("mio", false, func(string) bool { return false }); err == nil {
		t.Fatal("si la persona dice que no, no borra")
	}
	if err := quitarDelBaul("mio", true, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := baul.Buscar("mio"); err == nil {
		t.Fatal("quitado")
	}
	lista.Reset()
	listarBaulJSON(&lista)
	if !strings.Contains(lista.String(), `"tokens": []`) && !strings.Contains(lista.String(), `"tokens":[]`) {
		t.Fatalf("un baúl vacío es una lista vacía, no null: %s", lista.String())
	}
}

// Pedido de la persona (2026-11-13): la lista muestra la dirección estándar de cada acceso, y guardar la recibe con
// --direccion ambiente/servidor/servicio/clase/cuenta.
func TestBaulLista_MuestraLaDireccion(t *testing.T) {
	baulDePrueba(t)
	e := baul.Entrada{Nombre: "pg-app", Tipo: "otro", Donde: baul.EnLlavero, Variable: "PGPASSWORD", Direccion: "prod/db-prod-01/notificador/postgres/app_user"}
	var salida bytes.Buffer
	if err := guardarDelPortapapeles(&salida, e, &copiadoFalso{valor: "clave-de-prueba-123"}); err != nil {
		t.Fatal(err)
	}
	var lista bytes.Buffer
	listarBaul(&lista)
	if !strings.Contains(lista.String(), "DIRECCIÓN") || !strings.Contains(lista.String(), "prod/db-prod-01/notificador/postgres/app_user") {
		t.Fatalf("la lista muestra la dirección: %s", lista.String())
	}
	if f := baulGuardarCmd.Flags().Lookup("direccion"); f == nil {
		t.Fatal("guardar recibe --direccion")
	}
}

// Pedido de la persona (2026-11-20): Config cambia el nombre de un token con qdd baul renombrar, y arma la dirección con
// los ambientes que da qdd.
func TestBaulPanel_RenombrarYAmbientes(t *testing.T) {
	baulDePrueba(t)
	e := baul.Entrada{Nombre: "mio", Tipo: "otro", Donde: baul.EnLlavero, Variable: "MI_TOKEN"}
	var salida bytes.Buffer
	if err := guardarDelPortapapeles(&salida, e, &copiadoFalso{valor: "valor-de-prueba-123"}); err != nil {
		t.Fatal(err)
	}
	salida.Reset()
	if err := renombrarEnBaul(&salida, "mio", "mio-prod"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(salida.String(), "mio-prod") || strings.Contains(salida.String(), "valor-de-prueba-123") {
		t.Fatalf("dice el nombre nuevo, nunca el valor: %q", salida.String())
	}
	var lista bytes.Buffer
	listarBaulJSON(&lista)
	var d struct {
		Tokens    []baul.Entrada `json:"tokens"`
		Ambientes []string       `json:"ambientes"`
	}
	if err := json.Unmarshal(lista.Bytes(), &d); err != nil || len(d.Tokens) != 1 || d.Tokens[0].Nombre != "mio-prod" {
		t.Fatalf("la lista trae el nombre nuevo: %s %v", lista.String(), err)
	}
	if strings.Join(d.Ambientes, ",") != strings.Join(baul.Ambientes, ",") {
		t.Fatalf("la lista trae los ambientes de la dirección: %v", d.Ambientes)
	}
}

// Pedido de la persona (2026-11-20): agregar pregunta correo, sitio, cuenta, el token y una descripción. Agregar nunca
// pisa un acceso que ya existe, y editar cambia los datos sin pedir el token.
func TestBaulPanel_AgregarNoPisaYEditarNoPideElToken(t *testing.T) {
	baulDePrueba(t)
	e := baul.Entrada{Nombre: "cloudflare-trdago", Tipo: "cloudflare", Donde: baul.EnLlavero, Correo: "yo@x.cl", Cuenta: "trdago"}
	if err := baul.Guardar(e, "valor-de-prueba-123", time.Now()); err != nil {
		t.Fatal(err)
	}
	copiado := &copiadoFalso{valor: "otro-valor-1234567"}
	var salida bytes.Buffer
	if err := agregarDelPortapapeles(&salida, e, copiado); err == nil || !strings.Contains(err.Error(), "ya existe") {
		t.Fatalf("agregar no pisa uno que ya existe: %v", err)
	}
	if copiado.valor == "" {
		t.Fatal("se niega antes de leer el portapapeles: lo copiado sigue ahí")
	}
	cmd := baulEditarCmd
	for _, f := range []string{"correo", "url", "cuenta", "descripcion", "ambiente", "vence"} {
		if cmd.Flags().Lookup(f) == nil || baulGuardarCmd.Flags().Lookup(f) == nil {
			t.Fatalf("guardar y editar reciben --%s", f)
		}
	}
	if err := cmd.Flags().Set("descripcion", "DNS de los proyectos"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Flags().Set("descripcion", ""); cmd.Flags().Lookup("descripcion").Changed = false })
	salida.Reset()
	if err := editarEnBaul(&salida, cmd, "cloudflare-trdago"); err != nil {
		t.Fatal(err)
	}
	g, _ := baul.Buscar("cloudflare-trdago")
	if g.Descripcion != "DNS de los proyectos" || g.Correo != "yo@x.cl" {
		t.Fatalf("editar cambia solo lo que se dice: %+v", g)
	}
	var lista bytes.Buffer
	listarBaulJSON(&lista)
	var d struct {
		Sitios []baul.SitioConocido `json:"sitios"`
	}
	if err := json.Unmarshal(lista.Bytes(), &d); err != nil || len(d.Sitios) != len(baul.Sitios) || d.Sitios[0].Etiqueta != "Cloudflare" {
		t.Fatalf("la lista para el panel trae los sitios: %s %v", lista.String(), err)
	}
	lista.Reset()
	listarBaul(&lista)
	if !strings.Contains(lista.String(), "trdago") || !strings.Contains(lista.String(), "yo@x.cl") {
		t.Fatalf("la lista muestra la cuenta y el correo: %s", lista.String())
	}
}

// La línea de una verificación dice solo lo que se sabe: sin «dueño -» ni «vence -» (2026-11-20).
func TestLineaVerificacion_SoloLoQueSeSabe(t *testing.T) {
	vence := time.Date(2027, 1, 31, 15, 0, 0, 0, time.UTC)
	casos := []struct {
		e    baul.Entrada
		dice string
	}{
		{baul.Entrada{Alcance: "token active"}, "[✓] cf: la API lo acepta · token active"},
		{baul.Entrada{Duenio: "yo@x.cl", Alcance: "repo", Vence: &vence}, "[✓] cf: la API lo acepta · dueño yo@x.cl · repo · vence 2027-01-31"},
	}
	for _, c := range casos {
		if got := lineaVerificacion("cf", c.e); got != c.dice {
			t.Errorf("%+v: %q, quería %q", c.e, got, c.dice)
		}
	}
}
