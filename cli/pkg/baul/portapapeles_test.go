package baul

import (
	"errors"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// Rule: Prohibido usar 'else'
// El baúl desde Config en el panel (pedido de la persona, 2026-11-09: «agregar, quitar, modificar»; eligió que el valor
// entre por el portapapeles). qdd lee el token directo del portapapeles, lo vacía y lo guarda: el valor nunca se ve ni
// pasa por el panel, los argumentos o el chat.

type portapapelesFalso struct {
	valor   string
	err     error
	vaciado bool
}

func (p *portapapelesFalso) Leer() (string, error) { return p.valor, p.err }
func (p *portapapelesFalso) Vaciar() error {
	p.vaciado = true
	p.valor = ""
	return nil
}

func otro(nombre string) Entrada {
	return Entrada{Nombre: nombre, Tipo: "otro", Donde: EnLlavero, Variable: "MI_TOKEN"}
}

func TestPortapapeles_GuardaElTokenYLoVacia(t *testing.T) {
	entorno(t)
	p := &portapapelesFalso{valor: "  " + secreto + "\n"}
	if _, err := GuardarDelPortapapeles(otro("mio"), p, ahora); err != nil {
		t.Fatal(err)
	}
	if !p.vaciado {
		t.Fatal("el portapapeles queda vacío: el token no se queda copiado")
	}
	e, _ := Buscar("mio")
	if v, err := leerValor(e); err != nil || v != secreto {
		t.Fatalf("guarda lo copiado, sin espacios alrededor: %q %v", v, err)
	}
}

func TestPortapapeles_LoQueNoEsUnTokenNoSeGuardaPeroSeVacia(t *testing.T) {
	entorno(t)
	for _, copiado := range []string{"", "   ", "una frase con espacios", "dos\nlíneas", strings.Repeat("x", 5000)} {
		p := &portapapelesFalso{valor: copiado}
		_, err := GuardarDelPortapapeles(otro("mio"), p, ahora)
		if err == nil {
			t.Fatalf("%q no es un token", copiado)
		}
		if strings.Contains(err.Error(), copiado) && strings.TrimSpace(copiado) != "" {
			t.Fatalf("el error no repite lo copiado: %v", err)
		}
		if !p.vaciado {
			t.Fatalf("lo leído se vacía igual: %q", copiado)
		}
	}
	if _, err := Buscar("mio"); !errors.Is(err, ErrNoExiste) {
		t.Fatal("no quedó nada guardado")
	}
	if _, err := GuardarDelPortapapeles(otro("mio"), &portapapelesFalso{err: errors.New("sin wl-paste")}, ahora); err == nil {
		t.Fatal("sin portapapeles no guarda")
	}
}

func TestModificar_ElValorNuevoReemplazaAlAnteriorYCambiarDeLugarNoDejaCopia(t *testing.T) {
	entorno(t)
	if _, err := GuardarDelPortapapeles(otro("mio"), &portapapelesFalso{valor: "viejo-123"}, ahora); err != nil {
		t.Fatal(err)
	}
	if _, err := GuardarDelPortapapeles(otro("mio"), &portapapelesFalso{valor: "nuevo-456"}, ahora); err != nil {
		t.Fatal(err)
	}
	e, _ := Buscar("mio")
	if v, _ := leerValor(e); v != "nuevo-456" {
		t.Fatalf("modificar deja el valor nuevo: %q", v)
	}
	sso := Entrada{Nombre: "mio", Tipo: "aws-sso", Donde: EnLlavero}
	if _, err := GuardarDelPortapapeles(sso, &portapapelesFalso{valor: "mi-perfil"}, ahora); err != nil {
		t.Fatal(err)
	}
	if _, err := keyring.Get(servicioLlavero, "mio"); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("el valor anterior no queda olvidado en el llavero: %v", err)
	}
}

func TestQuitar_BorraElValorLaEntradaYSusPermisos(t *testing.T) {
	root := entorno(t)
	guardarCloudflare(t)
	if err := conPermisos(ahora, func(ps *permisos) error {
		ps.Permisos = append(ps.Permisos, Permiso{Sesion: "s1", Proyecto: root, Token: "cf", Fin: "deploy", Dado: ahora})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := Quitar("cf", ahora); err != nil {
		t.Fatal(err)
	}
	if _, err := Buscar("cf"); !errors.Is(err, ErrNoExiste) {
		t.Fatal("la entrada sale del índice")
	}
	if _, err := keyring.Get(servicioLlavero, "cf"); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("el valor sale del llavero: %v", err)
	}
	var ps permisos
	if err := leerJSON("permisos.json", &ps); err != nil || len(ps.Permisos) != 0 {
		t.Fatalf("sin token no quedan permisos: %+v %v", ps, err)
	}
	lineas, _ := Registro()
	if len(lineas) == 0 || lineas[len(lineas)-1].Accion != "quitar" {
		t.Fatalf("quitar queda en el registro: %+v", lineas)
	}
	if err := Quitar("cf", ahora); !errors.Is(err, ErrNoExiste) {
		t.Fatalf("quitar uno que no está lo dice: %v", err)
	}
}

func TestPortapapeles_LosComandosDeCadaSistema(t *testing.T) {
	sin := func(string) string { return "" }
	casos := []struct {
		goos string
		env  func(string) string
		leer string
	}{
		{"linux", func(k string) string { return map[string]string{"WAYLAND_DISPLAY": "wayland-0"}[k] }, "wl-paste"},
		{"linux", func(k string) string { return map[string]string{"DISPLAY": ":0"}[k] }, "xclip"},
		{"darwin", sin, "pbpaste"},
		{"windows", sin, "powershell.exe"},
	}
	for _, c := range casos {
		leer, vaciar := comandosPortapapeles(c.goos, c.env)
		if len(leer) == 0 || leer[0] != c.leer || len(vaciar) == 0 {
			t.Fatalf("%s: %v %v", c.goos, leer, vaciar)
		}
	}
	if leer, _ := comandosPortapapeles("linux", sin); leer != nil {
		t.Fatal("sin escritorio no hay portapapeles")
	}
}
