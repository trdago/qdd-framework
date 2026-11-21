package cmd

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'

// redVigilada cambia el transporte HTTP de todo el proceso por uno que cuenta y no sale: el relay, el directorio, la
// sonda y Realtime lo usan, así que cualquier uso de la red de doctor queda contado.
type redVigilada struct{ usos *int }

func (r redVigilada) RoundTrip(*http.Request) (*http.Response, error) {
	*r.usos++
	return nil, errors.New("la prueba no tiene red")
}

func vigilarRed(t *testing.T) *int {
	t.Helper()
	usos := 0
	antes := http.DefaultTransport
	http.DefaultTransport = redVigilada{usos: &usos}
	t.Cleanup(func() { http.DefaultTransport = antes })
	return &usos
}

// Un PC con todo lo que doctor puede encontrar mal a la vez: el relay nunca confirmado, la sonda caída en el canal, el
// reloj corrido y una cola apartada. El diagnóstico no usa la red; --fix solo consulta el directorio firmado, no sondea,
// y lo que es de la persona (el reloj, la cola) se dice y no se intenta.
// contrato: remoto#f6eb38fa
func TestDoctorMovil_DiagnosticaSinRedYFixSoloConsultaElDirectorio(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	servicioSanoPrueba(t, now)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayAntiguoPrueba)
	sondeos := sondaPrueba(t, remoto.ResultadoSonda{OK: true})
	guardarSondaPrueba(t, remoto.ResultadoSonda{Tramo: remoto.TramoCanal, Motivo: remoto.MotivoDeTramo(remoto.TramoCanal), En: now.Add(-time.Minute)})
	if err := remoto.GuardarLatidoConHora(now, 10*time.Minute, true); err != nil {
		t.Fatal(err)
	}
	cola := escribirRemoto(t, "solicitudes.json.ilegible", "{roto")
	red := vigilarRed(t)
	*m.corridos = nil

	g := buildMovilChecks(m.raiz, now)
	if consultas != 0 || *sondeos != 0 || *red != 0 {
		t.Fatalf("el diagnóstico no usa la red: directorio %d, sonda %d, http %d", consultas, *sondeos, *red)
	}
	for _, nombre := range []string{itemMovilDirectorio, itemMovilSonda, itemMovilReloj, itemMovilColas} {
		if item := itemMovil(t, g, nombre); item.Success {
			t.Fatalf("%s debía verse: %+v", nombre, item)
		}
	}
	if reloj := itemMovil(t, g, itemMovilReloj); !strings.Contains(reloj.Error, "hora automática") {
		t.Fatalf("el reloj se le dice a la persona: %+v", reloj)
	}

	repairMovil(m.raiz, g, now)
	if consultas == 0 {
		t.Fatal("--fix compara el relay con el directorio firmado")
	}
	if *sondeos != 0 || *red != 0 {
		t.Fatalf("--fix no usa más red que el directorio: sonda %d, http %d", *sondeos, *red)
	}
	if len(*m.corridos) != 0 {
		t.Fatalf("el relay no cambió: nada que reiniciar, y el reloj y la cola no se intentan: %v", *m.corridos)
	}
	if desfase, _ := remoto.DesfaseRelay(); desfase < 9*time.Minute {
		t.Fatalf("el reloj lo corrige la persona: %v", desfase)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilColas); item.Success || !strings.Contains(item.Error, cola) {
		t.Fatalf("la cola apartada sigue donde estaba: %+v", item)
	}
}

// Sin --fix, el directorio tampoco se consulta aunque nunca se haya comparado.
// contrato: remoto#f6eb38fa
func TestDoctorMovil_UnRelaySinConfirmarNoSeConsultaSinFix(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	red := vigilarRed(t)
	item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilDirectorio)
	if item.Success || consultas != 0 || *red != 0 {
		t.Fatalf("lo dice sin consultar: %+v, directorio %d, http %d", item, consultas, *red)
	}
}
