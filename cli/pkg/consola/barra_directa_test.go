package consola

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Decisión de la persona (2026-11-05): la barra corre cada 2 s por sesión y arrancar node era más de la mitad de su
// CPU en reposo (2,27 → 1,04 % de un núcleo, medido en Linux). Llama al binario por su ruta y, si la ruta ya no
// existe, a qdd como antes.

const binarioDePrueba = "/opt/node/lib/node_modules/qdd-framework/bin/qdd"

func comandoGuardado(t *testing.T, c Consola) map[string]any {
	t.Helper()
	settings := leerJSON(t, filepath.Join(c.ClaudeDir, "settings.json"))
	var barra map[string]any
	if err := json.Unmarshal([]byte(settings["statusLine"]), &barra); err != nil {
		t.Fatalf("statusLine ilegible: %v: %v", err, settings)
	}
	return barra
}

func settingsConBarra(comando string, extra string) string {
	return `{"statusLine":{"type":"command","command":` + jsonStr(comando) + extra + `}}`
}

func TestBarra_ConBinarioLoLlamaDirectoConQDDDeRespaldo(t *testing.T) {
	c := nuevaConsola(t, "2.2.0", &claudeFalso{})
	c.Binario = binarioDePrueba
	if _, err := c.Activar(true); err != nil {
		t.Fatal(err)
	}
	want := `"` + binarioDePrueba + `" consola statusline || qdd consola statusline`
	if got := comandoGuardado(t, c)["command"]; got != want {
		t.Fatalf("comando: %q, quiero %q", got, want)
	}
	if d := c.Diagnosticar(); !d[1].OK {
		t.Fatalf("doctor la reconoce: %+v", d)
	}
	if _, err := c.Desactivar(); err != nil {
		t.Fatal(err)
	}
	if _, ok := leerJSON(t, filepath.Join(c.ClaudeDir, "settings.json"))["statusLine"]; ok {
		t.Fatal("desactivar la quita como a la barra de siempre")
	}
}

func TestBarra_SinBinarioUsaQDDComoSiempre(t *testing.T) {
	c := nuevaConsola(t, "2.2.0", &claudeFalso{})
	if _, err := c.Activar(true); err != nil {
		t.Fatal(err)
	}
	if got := comandoGuardado(t, c)["command"]; got != ComandoBarra {
		t.Fatalf("un binario sin npm no deja su ruta en settings.json: %q", got)
	}
}

func TestBarra_AlActualizarRenuevaLaRutaYConservaLoDemas(t *testing.T) {
	for nombre, antes := range map[string]string{
		"ruta vieja": `"/vieja/node_modules/qdd-framework/bin/qdd" consola statusline || qdd consola statusline`,
		"sin ruta":   ComandoBarra,
	} {
		t.Run(nombre, func(t *testing.T) {
			c := nuevaConsola(t, "2.2.0", &claudeFalso{})
			c.Binario = binarioDePrueba
			escribirArchivo(t, filepath.Join(c.ClaudeDir, "settings.json"), settingsConBarra(antes, `,"refreshInterval":7`))
			if _, err := c.Activar(false); err != nil {
				t.Fatal(err)
			}
			if d := c.diagnosticarBarra(); d.OK {
				t.Fatalf("doctor avisa que hay que renovarla: %+v", d)
			}
			if _, err := c.Activar(true); err != nil {
				t.Fatal(err)
			}
			barra := comandoGuardado(t, c)
			if !strings.HasPrefix(barra["command"].(string), `"`+binarioDePrueba+`"`) || barra["refreshInterval"] != float64(7) {
				t.Fatalf("ruta nueva y el resto igual: %+v", barra)
			}
			if d := c.diagnosticarBarra(); !d.OK {
				t.Fatalf("doctor queda conforme: %+v", d)
			}
			reg, _ := c.LeerRegistro()
			if len(reg.BarraAnterior) != 0 {
				t.Fatalf("la barra de QDD no se guarda como la anterior de la persona: %s", reg.BarraAnterior)
			}
		})
	}
}

func TestBarra_SinBinarioNoTocaUnaDirectaYaPuesta(t *testing.T) {
	c := nuevaConsola(t, "2.2.0", &claudeFalso{})
	directa := `"` + binarioDePrueba + `" consola statusline || qdd consola statusline`
	escribirArchivo(t, filepath.Join(c.ClaudeDir, "settings.json"), settingsConBarra(directa, ""))
	if _, err := c.Activar(true); err != nil {
		t.Fatal(err)
	}
	if got := comandoGuardado(t, c)["command"]; got != directa {
		t.Fatalf("un qdd compilado a mano no cambia la barra instalada: %q", got)
	}
	if d := c.diagnosticarBarra(); !d.OK {
		t.Fatalf("y doctor no se queja: %+v", d)
	}
}

func TestBarra_UnaPropiaParecidaNoSeToca(t *testing.T) {
	for _, propia := range []string{
		`echo hola; "/x/qdd" consola statusline || qdd consola statusline`,
		`"/x/qdd" consola statusline || qdd consola statusline; echo hola`,
		`"/x/qdd" "/y" consola statusline || qdd consola statusline`,
		`"" consola statusline || qdd consola statusline`,
	} {
		c := nuevaConsola(t, "2.2.0", &claudeFalso{})
		c.Binario = binarioDePrueba
		escribirArchivo(t, filepath.Join(c.ClaudeDir, "settings.json"), settingsConBarra(propia, ""))
		if _, err := c.Activar(true); err != nil {
			t.Fatal(err)
		}
		if got := comandoGuardado(t, c)["command"]; got != propia {
			t.Fatalf("una barra propia nunca se reemplaza: %q → %q", propia, got)
		}
	}
}

func TestBarra_UnaRutaConCaracteresDeShellNoSeEscribe(t *testing.T) {
	for _, ruta := range []string{`/a"b/qdd`, `/a$HOME/qdd`, "/a`id`/qdd", `/a%PATH%/qdd`, "/a\nb/qdd"} {
		if got := (Consola{Binario: ruta}).comandoBarra(); got != ComandoBarra {
			t.Fatalf("%q no se escribe en un comando de shell: %q", ruta, got)
		}
	}
	if got := (Consola{Binario: `C:\Users\Ana María\qdd\bin\2.10.0\qdd.exe`}).comandoBarra(); !strings.HasPrefix(got, `"C:/Users/Ana María/qdd/bin/2.10.0/qdd.exe" consola`) {
		t.Fatalf("en Windows va con barras normales y entre comillas, con espacios: %q", got)
	}
}

// falsoQDD compila en dir un ejecutable real que imprime marca y repite su entrada estándar: Git Bash no encuentra un
// qdd.cmd al llamar a qdd, y la prueba tiene que valer igual en todas las shells.
func falsoQDD(t *testing.T, dir, nombre, marca string) string {
	t.Helper()
	fuente := filepath.Join(t.TempDir(), "main.go")
	escribirArchivo(t, fuente, `package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	fmt.Println("`+marca+`")
	io.Copy(os.Stdout, os.Stdin)
}
`)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, nombre)
	if runtime.GOOS == "windows" {
		ruta += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", ruta, fuente).CombinedOutput(); err != nil {
		t.Fatalf("no compiló el qdd falso: %v: %s", err, out)
	}
	return ruta
}

// Se corre de verdad en cada shell donde Claude Code puede correrla (en Windows, también cmd): las comillas y el
// || son justo lo que cambia entre ellas. La carpeta lleva un espacio, como un usuario de Windows con dos nombres.
func TestBarra_ElComandoCorreEnLaShellYElRespaldoRecibeLaEntrada(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "con espacio")
	directo := falsoQDD(t, filepath.Join(dir, "version"), "qdd-directo", "DIRECTO")
	respaldo := filepath.Join(dir, "path")
	falsoQDD(t, respaldo, "qdd", "RESPALDO")
	t.Setenv("PATH", respaldo+string(os.PathListSeparator)+os.Getenv("PATH"))

	for shell, correr := range shellsDeLaBarra() {
		t.Run(shell, func(t *testing.T) {
			conRuta := Consola{Binario: directo}.comandoBarra()
			out := correr(t, conRuta, `{"model":"opus"}`)
			if !strings.Contains(out, "DIRECTO") || strings.Contains(out, "RESPALDO") || !strings.Contains(out, `"model"`) {
				t.Fatalf("con la ruta vigente corre solo el binario y recibe el JSON: %q", out)
			}
			sinRuta := Consola{Binario: filepath.Join(dir, "borrada", "qdd")}.comandoBarra()
			out = correr(t, sinRuta, `{"model":"opus"}`)
			if !strings.Contains(out, "RESPALDO") || !strings.Contains(out, `"model"`) {
				t.Fatalf("sin la ruta corre qdd y el JSON le llega entero: %q", out)
			}
		})
	}
}
