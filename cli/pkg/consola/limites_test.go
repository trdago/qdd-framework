package consola

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Pedido de qdd-movile (2026-11-04): la persona quiere ver en la app cuánto queda de sus límites de Claude sin gastar
// tokens. Claude Code se los entrega solo a la barra, así que la barra deja el último valor en un archivo de la máquina.

const entradaBarra = `{"cwd":"/home/x/secreto","model":{"display_name":"Opus"},"rate_limits":{
	"five_hour":{"used_percentage":12.2,"resets_at":1793000000},
	"seven_day":{"used_percentage":40,"resets_at":"2026-11-09T09:00:00-03:00"}}}`

func homeDeLimites(t *testing.T) {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
}

func TestLimites_ElPorcentajeVaTalCualYElReinicioConSuHora(t *testing.T) {
	l := LimitesDe([]byte(entradaBarra), enCola)
	if l == nil || l.CincoHoras == nil || l.CincoHoras.Usado != 12.2 || l.Semana == nil || l.Semana.Usado != 40 {
		t.Fatalf("usado es el used_percentage tal cual: %+v", l)
	}
	if l.CincoHoras.Reinicia == nil || l.CincoHoras.Reinicia.Unix() != 1793000000 {
		t.Fatalf("reinicia acepta segundos epoch: %+v", l.CincoHoras)
	}
	if l.Semana.Reinicia == nil || l.Semana.Reinicia.UTC().Format(time.RFC3339) != "2026-11-09T12:00:00Z" {
		t.Fatalf("reinicia acepta una fecha ISO 8601: %+v", l.Semana)
	}
	if !l.Actualizado.Equal(enCola) {
		t.Fatalf("actualizado es cuándo la barra lo leyó: %v", l.Actualizado)
	}
	b, _ := json.Marshal(l)
	for _, nada := range []string{"secreto", "Opus", "cwd"} {
		if strings.Contains(string(b), nada) {
			t.Fatalf("no lleva rutas ni el modelo: %s", b)
		}
	}
}

func TestLimites_LoQueNoVinoSeOmiteNuncaUnCero(t *testing.T) {
	l := LimitesDe([]byte(`{"rate_limits":{"five_hour":{"used_percentage":5}}}`), enCola)
	if l == nil || l.CincoHoras == nil || l.Semana != nil || l.CincoHoras.Reinicia != nil {
		t.Fatalf("un tramo sin reinicio vale, y el que no vino se omite: %+v", l)
	}
	b, _ := json.Marshal(l)
	if strings.Contains(string(b), "semana") || strings.Contains(string(b), "reinicia") {
		t.Fatalf("lo que no vino no viaja: %s", b)
	}
	for _, entrada := range []string{`{}`, `{"rate_limits":{}}`, `{"rate_limits":{"five_hour":{"resets_at":1}}}`, `ilegible`} {
		if l := LimitesDe([]byte(entrada), enCola); l != nil {
			t.Fatalf("%s: sin ningún tramo no hay límites: %+v", entrada, l)
		}
	}
}

func TestLimites_LaBarraLosGuardaYUnaEntradaSinLimitesNoLosBorra(t *testing.T) {
	homeDeLimites(t)
	if l := LeerLimites(); l != nil {
		t.Fatalf("sin archivo no hay límites: %+v", l)
	}
	GuardarLimites([]byte(entradaBarra), enCola)
	l := LeerLimites()
	if l == nil || l.CincoHoras.Usado != 12.2 || l.Semana.Usado != 40 {
		t.Fatalf("queda el último valor: %+v", l)
	}
	GuardarLimites([]byte(`{"model":{"display_name":"Opus"}}`), enCola.Add(time.Hour))
	if l := LeerLimites(); l == nil || !l.Actualizado.Equal(enCola) {
		t.Fatalf("una entrada sin límites no borra lo último: %+v", l)
	}
}

func TestLimites_NoEscribeCadaDosSegundos(t *testing.T) {
	homeDeLimites(t)
	GuardarLimites([]byte(entradaBarra), enCola)
	GuardarLimites([]byte(entradaBarra), enCola.Add(2*time.Second))
	if l := LeerLimites(); !l.Actualizado.Equal(enCola) {
		t.Fatalf("sin cambios, la barra no escribe en cada vuelta: %v", l.Actualizado)
	}
	GuardarLimites([]byte(entradaBarra), enCola.Add(CadaLimites+time.Second))
	if l := LeerLimites(); !l.Actualizado.Equal(enCola.Add(CadaLimites + time.Second)) {
		t.Fatalf("sin cambios, sigue diciendo de cuándo es el dato: %v", l.Actualizado)
	}
	otro := strings.Replace(entradaBarra, "12.2", "13", 1)
	GuardarLimites([]byte(otro), enCola.Add(CadaLimites+2*time.Second))
	if l := LeerLimites(); l.CincoHoras.Usado != 13 {
		t.Fatalf("un cambio se escribe en el acto: %+v", l.CincoHoras)
	}
}

func TestLimites_LaHuellaIgnoraActualizado(t *testing.T) {
	a := LimitesDe([]byte(entradaBarra), enCola)
	b := LimitesDe([]byte(entradaBarra), enCola.Add(time.Hour))
	if a.Huella() != b.Huella() {
		t.Fatalf("actualizado no cuenta como cambio: %q != %q", a.Huella(), b.Huella())
	}
	c := LimitesDe([]byte(strings.Replace(entradaBarra, "1793000000", "1793000001", 1)), enCola)
	if a.Huella() == c.Huella() {
		t.Fatal("un reinicio distinto sí cuenta")
	}
	var nada *Limites
	if nada.Huella() != "" {
		t.Fatal("sin límites la huella es vacía")
	}
}

func TestLimites_UnArchivoIlegibleEsComoNoTenerlos(t *testing.T) {
	homeDeLimites(t)
	ruta, err := rutaLimites()
	if err != nil {
		t.Fatal(err)
	}
	GuardarLimites([]byte(entradaBarra), enCola)
	if err := os.WriteFile(ruta, []byte("{roto"), 0o600); err != nil {
		t.Fatal(err)
	}
	if l := LeerLimites(); l != nil {
		t.Fatalf("ilegible: nil, nunca un valor inventado: %+v", l)
	}
	GuardarLimites([]byte(entradaBarra), enCola)
	if l := LeerLimites(); l == nil {
		t.Fatal("la barra lo vuelve a escribir")
	}
}
