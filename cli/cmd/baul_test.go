package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

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
