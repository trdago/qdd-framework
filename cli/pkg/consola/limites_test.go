package consola

import (
	"encoding/json"
	"fmt"
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
	alGuardar := time.Unix(1792900000, 0) // antes del reinicio de entradaBarra
	if l := LeerLimites(); l != nil {
		t.Fatalf("sin archivo no hay límites: %+v", l)
	}
	GuardarLimites([]byte(entradaBarra), alGuardar)
	l := LeerLimites()
	if l == nil || l.CincoHoras.Usado != 12.2 || l.Semana.Usado != 40 {
		t.Fatalf("queda el último valor: %+v", l)
	}
	GuardarLimites([]byte(`{"model":{"display_name":"Opus"}}`), alGuardar.Add(time.Hour))
	if l := LeerLimites(); l == nil || !l.Actualizado.Equal(alGuardar) {
		t.Fatalf("una entrada sin límites no borra lo último: %+v", l)
	}
}

func TestLimites_NoEscribeCadaDosSegundos(t *testing.T) {
	homeDeLimites(t)
	alGuardar := time.Unix(1792900000, 0) // antes del reinicio de entradaBarra
	GuardarLimites([]byte(entradaBarra), alGuardar)
	GuardarLimites([]byte(entradaBarra), alGuardar.Add(2*time.Second))
	if l := LeerLimites(); !l.Actualizado.Equal(alGuardar) {
		t.Fatalf("sin cambios, la barra no escribe en cada vuelta: %v", l.Actualizado)
	}
	GuardarLimites([]byte(entradaBarra), alGuardar.Add(CadaLimites+time.Second))
	if l := LeerLimites(); !l.Actualizado.Equal(alGuardar.Add(CadaLimites + time.Second)) {
		t.Fatalf("sin cambios, sigue diciendo de cuándo es el dato: %v", l.Actualizado)
	}
	otro := strings.Replace(entradaBarra, "12.2", "13", 1)
	GuardarLimites([]byte(otro), alGuardar.Add(CadaLimites+2*time.Second))
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

// FND-130: la barra de cada sesión escribe lo último que supo su sesión, y una inactiva tiene datos viejos. El teléfono
// veía saltar la semana entre 12 %, 32 % y 56 %, y cada salto era un delta.
func entradaDeLimites(cinco, semana string) []byte {
	return []byte(`{"rate_limits":{` + strings.Trim(strings.Join([]string{cinco, semana}, ","), ",") + `}}`)
}

func tramoJSONDe(nombre string, usado float64, reinicia time.Time) string {
	return fmt.Sprintf(`"%s":{"used_percentage":%v,"resets_at":%d}`, nombre, usado, reinicia.Unix())
}

func TestLimites_UnaSesionConDatosViejosNoHaceRetroceder_FND130(t *testing.T) {
	homeDeLimites(t)
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	cinco, semana := now.Add(2*time.Hour), now.Add(72*time.Hour)
	GuardarLimites(entradaDeLimites(tramoJSONDe("five_hour", 13, cinco), tramoJSONDe("seven_day", 56, semana)), now)
	huella := LeerLimites().Huella()
	for _, vieja := range [][]byte{
		entradaDeLimites(tramoJSONDe("five_hour", 12, cinco), tramoJSONDe("seven_day", 55, semana)),
		entradaDeLimites("", tramoJSONDe("seven_day", 32, semana)),
		entradaDeLimites("", tramoJSONDe("seven_day", 12, semana)),
	} {
		now = now.Add(2 * time.Second)
		GuardarLimites(vieja, now)
		if h := LeerLimites().Huella(); h != huella {
			t.Fatalf("retrocedió: %s, se esperaba %s", h, huella)
		}
	}
}

func TestLimites_EnLaMismaVentanaLoUsadoSube_FND130(t *testing.T) {
	homeDeLimites(t)
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	semana := now.Add(72 * time.Hour)
	GuardarLimites(entradaDeLimites("", tramoJSONDe("seven_day", 56, semana)), now)
	GuardarLimites(entradaDeLimites("", tramoJSONDe("seven_day", 57, semana)), now.Add(time.Second))
	if l := LeerLimites(); l.Semana == nil || l.Semana.Usado != 57 {
		t.Fatalf("sube: %+v", l.Semana)
	}
}

func TestLimites_UnaVentanaNuevaMandaYUnaPasadaSeIgnora_FND130(t *testing.T) {
	homeDeLimites(t)
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	vieja, nueva := now.Add(time.Hour), now.Add(6*time.Hour)
	GuardarLimites(entradaDeLimites(tramoJSONDe("five_hour", 90, vieja), ""), now)
	GuardarLimites(entradaDeLimites(tramoJSONDe("five_hour", 3, nueva), ""), now.Add(time.Second))
	if l := LeerLimites(); l.CincoHoras == nil || l.CincoHoras.Usado != 3 {
		t.Fatalf("la ventana nueva empieza de nuevo, aunque use menos: %+v", l.CincoHoras)
	}
	GuardarLimites(entradaDeLimites(tramoJSONDe("five_hour", 90, vieja), ""), now.Add(2*time.Second))
	if l := LeerLimites(); l.CincoHoras.Usado != 3 {
		t.Fatalf("una sesión vieja no devuelve la ventana anterior: %+v", l.CincoHoras)
	}
}

func TestLimites_UnaVentanaQueYaPasoNoSeConservaNiSePublica_FND130(t *testing.T) {
	homeDeLimites(t)
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	cinco, semana := now.Add(time.Hour), now.Add(72*time.Hour)
	GuardarLimites(entradaDeLimites(tramoJSONDe("five_hour", 90, cinco), tramoJSONDe("seven_day", 50, semana)), now)
	despues := now.Add(2 * time.Hour)
	GuardarLimites(entradaDeLimites("", tramoJSONDe("seven_day", 51, semana)), despues)
	l := LeerLimites()
	if l.CincoHoras != nil || l.Semana == nil || l.Semana.Usado != 51 {
		t.Fatalf("la de 5 h ya se reinició: no se dice nada de ella: %+v", l)
	}
	GuardarLimites(entradaDeLimites(tramoJSONDe("five_hour", 90, cinco), tramoJSONDe("seven_day", 51, semana)), despues.Add(time.Second))
	if l := LeerLimites(); l.CincoHoras != nil {
		t.Fatalf("una entrada con la ventana pasada no la revive: %+v", l.CincoHoras)
	}
}

func TestLimites_SinFechaSeUsaLoQueVino_FND130(t *testing.T) {
	homeDeLimites(t)
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	GuardarLimites([]byte(`{"rate_limits":{"seven_day":{"used_percentage":40}}}`), now)
	GuardarLimites([]byte(`{"rate_limits":{"seven_day":{"used_percentage":30}}}`), now.Add(time.Second))
	if l := LeerLimites(); l.Semana == nil || l.Semana.Usado != 30 {
		t.Fatalf("sin fecha no se sabe si es otra ventana: %+v", l.Semana)
	}
}

func TestLimites_UnaFechaDeReinicioCasiIgualEsLaMismaVentana_FND130(t *testing.T) {
	homeDeLimites(t)
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	semana := now.Add(72 * time.Hour)
	GuardarLimites(entradaDeLimites("", tramoJSONDe("seven_day", 56, semana)), now)
	GuardarLimites(entradaDeLimites("", tramoJSONDe("seven_day", 12, semana.Add(30*time.Second))), now.Add(time.Second))
	if l := LeerLimites(); l.Semana.Usado != 56 {
		t.Fatalf("30 s de diferencia no son otra ventana: %+v", l.Semana)
	}
}
