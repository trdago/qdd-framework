package materializar

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Pedido del usuario (2026-11-01): lo que cambia en el panel se materializa como sus ventanas (hypr-materialize):
// caracteres que parpadean en el acento, bloques que se afinan y el texto nítido. Solo cambia la forma de aparecer:
// nada se mueve de lugar ni cambia de ancho.

var fila = []Linea{{
	{Texto: "├ ", Tenue: true},
	{Texto: "panel-arbol", Color: "#e5c07b"},
	{Texto: " ✅ ⏳ ❌", Color: "yellow"},
}}

func opciones(abrir bool) Opciones {
	return Opciones{Abrir: abrir, DuracionMs: 800, CuadroMs: 40, Acento: "#7DAEA3", Semilla: 7}
}

func TestCuadros_AbrirTerminaEnElTextoExacto(t *testing.T) {
	cuadros := Cuadros(fila, opciones(true))
	if len(cuadros) != 20 {
		t.Fatalf("0,8 s a 40 ms son 20 cuadros: %d", len(cuadros))
	}
	if got := cuadros[len(cuadros)-1]; !reflect.DeepEqual(got, juntar(fila)) {
		t.Fatalf("el último cuadro es la fila tal cual:\n%#v\n%#v", got, juntar(fila))
	}
	if texto(cuadros[0]) == texto(fila) {
		t.Fatal("el primer cuadro todavía no es el texto")
	}
}

func TestCuadros_CerrarTerminaEnBlanco(t *testing.T) {
	o := opciones(false)
	o.DuracionMs = 400
	cuadros := Cuadros(fila, o)
	if len(cuadros) != 10 {
		t.Fatalf("0,4 s a 40 ms son 10 cuadros: %d", len(cuadros))
	}
	if strings.TrimSpace(texto(cuadros[len(cuadros)-1])) != "" {
		t.Fatalf("al irse queda en blanco: %q", texto(cuadros[len(cuadros)-1]))
	}
}

func TestCuadros_NingunCuadroCambiaDeAncho(t *testing.T) {
	quiere := Ancho(texto(fila))
	for _, abrir := range []bool{true, false} {
		for i, c := range Cuadros(fila, opciones(abrir)) {
			if got := Ancho(texto(c)); got != quiere {
				t.Fatalf("abrir=%v cuadro %d mide %d, no %d: %q", abrir, i, got, quiere, texto(c))
			}
		}
	}
}

func TestCuadros_PasaPorCaracteresYBloquesEnElAcento(t *testing.T) {
	var glifos, bloques, acento bool
	for _, c := range Cuadros(fila, opciones(true)) {
		for _, s := range segmentos(c) {
			glifos = glifos || strings.ContainsAny(s.Texto, Glifos)
			bloques = bloques || strings.ContainsAny(s.Texto, "█▓▒")
			// teñido por el acento: un hex que no es el color propio de la fila
			acento = acento || strings.HasPrefix(s.Color, "#") && s.Color != "#e5c07b"
		}
	}
	if !glifos || !bloques || !acento {
		t.Fatalf("las etapas de la ventana: glifos=%v bloques=%v acento=%v", glifos, bloques, acento)
	}
}

func TestCuadros_LaMismaSemillaDaLosMismosCuadros(t *testing.T) {
	if !reflect.DeepEqual(Cuadros(fila, opciones(true)), Cuadros(fila, opciones(true))) {
		t.Fatal("determinista: la misma entrada da los mismos cuadros")
	}
	otra := opciones(true)
	otra.Semilla = 8
	if reflect.DeepEqual(Cuadros(fila, opciones(true)), Cuadros(fila, otra)) {
		t.Fatal("otra semilla, otro borde del frente")
	}
}

func TestCuadros_UnAcentoDelTemaNoSeMezcla(t *testing.T) {
	o := opciones(true)
	o.Acento = "cyan"
	for _, c := range Cuadros(fila, o) {
		for _, s := range segmentos(c) {
			if strings.HasPrefix(s.Color, "#") && s.Color != "#e5c07b" {
				t.Fatalf("con un color del tema no se inventa un hex: %q", s.Color)
			}
		}
	}
}

func TestAncho_EmojiYVariantesOcupanDos(t *testing.T) {
	casos := map[string]int{"abc": 3, "✅": 2, "⏳": 2, "❌": 2, "🎯": 2, "⚙️": 2, "▶": 1, "ｱ": 1, "·": 1}
	for s, quiere := range casos {
		if got := Ancho(s); got != quiere {
			t.Errorf("Ancho(%q) = %d, no %d", s, got, quiere)
		}
	}
}

func TestEstiloDe_TomaElAcentoYLasDuracionesDeHyprMaterialize(t *testing.T) {
	home := t.TempDir()
	hypr := filepath.Join(home, ".config", "hypr")
	escribir(t, filepath.Join(hypr, "shaders", "materialize_open.glsl"), "// @duration 0.8\nconst vec3  ACCENT = vec3(0.490, 0.682, 0.639);    // color\n")
	escribir(t, filepath.Join(hypr, "materialize.conf"), "windowrule = tag +shader_open_default:"+filepath.Join(hypr, "shaders", "materialize_open.glsl")+"@0.8, match:class .*\n"+
		"windowrule = tag +shader_close_default:"+filepath.Join(hypr, "shaders", "materialize_close.glsl")+"@0.4, match:class .*\n")
	t.Setenv("XDG_CONFIG_HOME", "")
	e := EstiloDe(home)
	if e.Acento != "#7daea3" || e.AbrirMs != 800 || e.CerrarMs != 400 || e.Origen != OrigenHypr {
		t.Fatalf("el de tus ventanas: %+v", e)
	}
}

func TestEstiloDe_SinHyprUsaElAcentoDelTema(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	e := EstiloDe(t.TempDir())
	if e.Acento != "" || e.AbrirMs != 800 || e.CerrarMs != 400 || e.Origen != OrigenPorDefecto {
		t.Fatalf("en cualquier sistema: acento del tema y 0,8 s / 0,4 s: %+v", e)
	}
}

func TestEfectos_EncendidosPorDefectoYSeApagan(t *testing.T) {
	home := t.TempDir()
	t.Setenv(EnvEfectos, "")
	if !Activos(home) {
		t.Fatal("encendidos por defecto")
	}
	if err := Guardar(home, false); err != nil {
		t.Fatal(err)
	}
	if Activos(home) {
		t.Fatal("'qdd consola efectos off' los apaga")
	}
	if err := Guardar(home, true); err != nil || !Activos(home) {
		t.Fatalf("y se vuelven a encender: %v", err)
	}
	t.Setenv(EnvEfectos, "off")
	if Activos(home) {
		t.Fatal("QDD_EFECTOS=off manda")
	}
}

func TestEfectos_UnArchivoIlegibleNoLosApagaYDoctorLoDice(t *testing.T) {
	home := t.TempDir()
	t.Setenv(EnvEfectos, "")
	escribir(t, RutaEfectos(home), "{no es json")
	if !Activos(home) {
		t.Fatal("un archivo roto no apaga nada en silencio")
	}
	if err := Revisar(home); err == nil {
		t.Fatal("doctor lo ve")
	}
	if err := Reparar(home); err != nil || Revisar(home) != nil {
		t.Fatalf("y doctor --fix lo deja encendido y legible: %v", err)
	}
}

func escribir(t *testing.T, ruta, contenido string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func segmentos(l []Linea) []Segmento {
	var todos []Segmento
	for _, linea := range l {
		todos = append(todos, linea...)
	}
	return todos
}

func texto(l []Linea) string {
	var b strings.Builder
	for i, linea := range l {
		if i > 0 {
			b.WriteString("\n")
		}
		for _, s := range linea {
			b.WriteString(s.Texto)
		}
	}
	return b.String()
}
