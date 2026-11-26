package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'

// Revisión de toda la comunicación con QDD Móvil (2026-11-15): doctor no veía que el teléfono quitó el PC (y arrancaba un
// servicio que volvía a salir), unas credenciales o un relay rotos, un servicio que lanza un qdd borrado (el de otra
// versión de node), el canal en tiempo real caído con el latido bien, ni acciones descartadas por los relojes.

func noCorre(t *testing.T, g *CheckGroup, nombre string) {
	t.Helper()
	for _, i := range g.Items {
		if i.Name == nombre {
			t.Fatalf("no debía revisar %q: %+v", nombre, i)
		}
	}
}

func TestDoctorMovil_UnPCQuitadoPideEmparejarYNoArrancaElServicio_REM15(t *testing.T) {
	m := movilPrueba(t)
	emparejarPrueba(t)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	if err := remoto.GuardarQuitado("qdd:pc:t", time.Now()); err != nil {
		t.Fatal(err)
	}
	g := buildMovilChecks(m.raiz, time.Now())
	item := itemMovil(t, g, itemMovilQuitado)
	if item.Success || !strings.Contains(item.Error, "qdd login") {
		t.Fatalf("%+v", item)
	}
	noCorre(t, g, itemMovilCorre)
	repairMovil(m.raiz, g, time.Now())
	if len(*m.corridos) != 0 {
		t.Fatalf("arrancarlo no sirve: el relay lo vuelve a rechazar: %v", *m.corridos)
	}
}

func TestDoctorMovil_LaMarcaDeOtroEmparejamientoNoCuenta_REM15(t *testing.T) {
	m := movilPrueba(t)
	movilSano(t, m, time.Now())
	if err := remoto.GuardarQuitado("qdd:pc:anterior", time.Now()); err != nil {
		t.Fatal(err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, time.Now()), itemMovilQuitado); !item.Success {
		t.Fatalf("emparejar de nuevo da otro tópico: %+v", item)
	}
}

func escribirRemoto(t *testing.T, nombre, contenido string) string {
	t.Helper()
	dir, err := remoto.DirRemoto()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, nombre)
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestDoctorMovil_CredencialesRotasNoSeConfundenConSinEmparejar_REM16(t *testing.T) {
	m := movilPrueba(t)
	ruta := escribirRemoto(t, "credenciales.json", "{roto")
	g := buildMovilChecks(m.raiz, time.Now())
	item := itemMovil(t, g, itemMovilArchivos)
	if item.Success || !strings.Contains(item.Error, "credenciales.json") {
		t.Fatalf("%+v", item)
	}
	repairMovil(m.raiz, g, time.Now())
	if _, err := os.Stat(ruta + ".ilegible"); err != nil {
		t.Fatalf("el archivo roto se aparta, no se borra: %v", err)
	}
	if remoto.CredencialesIlegibles() {
		t.Fatal("después del fix ya no hay credenciales rotas")
	}
}

func TestDoctorMovil_EmparejadoSinRelayLoPideAlDirectorio_REM16(t *testing.T) {
	m := movilPrueba(t)
	emparejarPrueba(t)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	g := buildMovilChecks(m.raiz, time.Now())
	if item := itemMovil(t, g, itemMovilArchivos); item.Success || !strings.Contains(item.Error, "falta relay.json") {
		t.Fatalf("%+v", item)
	}
	if consultas != 0 {
		t.Fatal("diagnosticar no usa la red")
	}
	repairMovil(m.raiz, g, time.Now())
	c, err := remoto.LeerRelay()
	if err != nil || c.URL != relayNuevoPrueba.URL {
		t.Fatalf("quedó el relay del directorio: %+v %v", c, err)
	}
}

func TestDoctorMovil_UnRelayRotoSeApartaYSeRecupera_REM16(t *testing.T) {
	m := movilPrueba(t)
	emparejarPrueba(t)
	ruta := escribirRemoto(t, "relay.json", "{roto")
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	g := buildMovilChecks(m.raiz, time.Now())
	if item := itemMovil(t, g, itemMovilArchivos); item.Success || !strings.Contains(item.Error, "no se lee") {
		t.Fatalf("%+v", item)
	}
	repairMovil(m.raiz, g, time.Now())
	if _, err := os.Stat(ruta + ".ilegible"); err != nil {
		t.Fatalf("el roto queda aparte: %v", err)
	}
	if remoto.RelayNoSeLee() != "" {
		t.Fatal("relay.json quedó bueno")
	}
}

func TestDoctorMovil_ElServicioQueLanzaUnQddBorradoSeReescribe_REM17(t *testing.T) {
	m := movilPrueba(t)
	emparejarPrueba(t)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	s, _ := remotoServicio()
	s.Bin = "/no/existe/node/26.2.0/bin/qdd"
	if _, err := s.Instalar(remotoEjecutor); err != nil {
		t.Fatal(err)
	}
	*m.corridos = nil
	ahora := filepath.Join(t.TempDir(), "qdd")
	if err := os.WriteFile(ahora, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	conBin := s
	conBin.Bin = ahora
	remotoServicio = func() (remoto.Servicio, error) { return conBin, nil }
	g := buildMovilChecks(m.raiz, time.Now())
	item := itemMovil(t, g, itemMovilBinario)
	if item.Success || !strings.Contains(item.Error, "/no/existe/node/26.2.0/bin/qdd") {
		t.Fatalf("%+v", item)
	}
	repairMovil(m.raiz, g, time.Now())
	if conBin.BinarioInstalado() != ahora {
		t.Fatalf("el servicio lanza el qdd de ahora: %q", conBin.BinarioInstalado())
	}
	reinicios := 0
	for _, c := range *m.corridos {
		if strings.Contains(c, "restart") {
			reinicios++
		}
	}
	if reinicios != 1 {
		t.Fatalf("reescribirlo ya lo arranca; REM-06 no lo reinicia otra vez: %v", *m.corridos)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, time.Now()), itemMovilBinario); !item.Success {
		t.Fatalf("%+v", item)
	}
}

func TestDoctorMovil_ElServicioDeOtroQDDHomeNoSeToca_REM17(t *testing.T) {
	m := movilPrueba(t)
	emparejarPrueba(t)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	s, _ := remotoServicio()
	otro := s
	otro.Bin, otro.QDDHome = "/no/existe/qdd", "/otro/home"
	if _, err := otro.Instalar(remotoEjecutor); err != nil {
		t.Fatal(err)
	}
	g := buildMovilChecks(m.raiz, time.Now())
	if item := itemMovil(t, g, itemMovilBinario); !item.Success {
		t.Fatalf("lo arregla quien corre con ese QDD_HOME: %+v", item)
	}
}

func TestDoctorMovil_CanalCaidoConLatidoBienSeVeYSeReiniciaUnaVez_REM18(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	s, _ := remotoServicio()
	if _, err := s.Instalar(remotoEjecutor); err != nil {
		t.Fatal(err)
	}
	if err := remoto.GuardarLatido(now.Add(-10 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := remoto.GuardarPlan(remoto.PlanGratis(), now.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := remoto.GuardarConexion(false, now.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	*m.corridos = nil
	g := buildMovilChecks(m.raiz, now)
	item := itemMovil(t, g, itemMovilCanal)
	if item.Success || !strings.Contains(item.Error, "caído hace 10 min") {
		t.Fatalf("%+v", item)
	}
	repairMovil(m.raiz, g, now)
	if len(*m.corridos) == 0 || !strings.Contains(strings.Join(*m.corridos, ";"), "restart") {
		t.Fatalf("se reinicia: %v", *m.corridos)
	}
	reinicios := strings.Count(strings.Join(*m.corridos, ";"), "restart")
	if reinicios != 1 {
		t.Fatalf("REM-07 y REM-18 reinician una sola vez: %v", *m.corridos)
	}
}

func TestDoctorMovil_UnaCaidaCortaOUnidoNoAlarma_REM18(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	if err := remoto.GuardarConexion(false, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilCanal); !item.Success {
		t.Fatalf("el canal se reconecta solo en segundos: %+v", item)
	}
	if err := remoto.GuardarConexion(true, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilCanal); !item.Success {
		t.Fatalf("%+v", item)
	}
}

func TestDoctorMovil_SinConexionJSONNoOpinaDelCanal_REM18(t *testing.T) {
	m := movilPrueba(t)
	movilSano(t, m, time.Now())
	noCorre(t, buildMovilChecks(m.raiz, time.Now()), itemMovilCanal)
}

func TestDoctorMovil_AccionesDescartadasPorLaHoraDicenElDesfase_REM19(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now().UTC().Truncate(time.Second)
	movilSano(t, m, now)
	telefono := now.Add(-12 * time.Minute).Format("2006-01-02T15:04:05Z")
	for _, e := range []remoto.EntradaLog{
		{En: now.Add(-30 * time.Hour), Estado: remoto.EstadoDescartada, Resultado: "acción vieja (2020-01-01T00:00:00Z)"},
		{En: now, Estado: remoto.EstadoDescartada, Resultado: "acción vieja (" + telefono + ")"},
		{En: now, Estado: remoto.EstadoDescartada, Resultado: "firma inválida"},
	} {
		if err := remoto.Anotar(e); err != nil {
			t.Fatal(err)
		}
	}
	g := buildMovilChecks(m.raiz, now)
	item := itemMovil(t, g, itemMovilReloj)
	if item.Success || !strings.Contains(item.Error, "1 acciones") || !strings.Contains(item.Error, "12 min adelantado") {
		t.Fatalf("%+v", item)
	}
	repairMovil(m.raiz, g, now)
	if len(*m.corridos) != 0 {
		t.Fatalf("el reloj lo corrige la persona: %v", *m.corridos)
	}
}

// qdd-movile (2026-11-15): la hora de una acción la sella el relay, así que REM-19 mide este PC contra el relay.
func TestDoctorMovil_ElRelojSeMideContraElRelayAntesDePerderAcciones_REM19(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	if err := remoto.GuardarLatidoConHora(now, -3*time.Minute, true); err != nil {
		t.Fatal(err)
	}
	item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilReloj)
	if item.Success || !strings.Contains(item.Error, "3 min atrasado respecto del relay") || strings.Contains(item.Error, "descartaron") {
		t.Fatalf("avisa sin haber perdido nada todavía: %+v", item)
	}
	if err := remoto.GuardarLatidoConHora(now, 20*time.Second, true); err != nil {
		t.Fatal(err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilReloj); !item.Success {
		t.Fatalf("20 s no es un reloj corrido: %+v", item)
	}
}

func TestDoctorMovil_ConElRelojBienLasDescartadasApuntanAlCanal_REM19(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now().UTC().Truncate(time.Second)
	movilSano(t, m, now)
	if err := remoto.GuardarLatidoConHora(now, 5*time.Second, true); err != nil {
		t.Fatal(err)
	}
	if err := remoto.Anotar(remoto.EntradaLog{En: now, Estado: remoto.EstadoDescartada, Resultado: "acción vieja (" + now.Add(-7*time.Minute).Format("2006-01-02T15:04:05Z") + ")"}); err != nil {
		t.Fatal(err)
	}
	item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilReloj)
	if item.Success || !strings.Contains(item.Error, "llegaron tarde") || !strings.Contains(item.Error, "REM-18") {
		t.Fatalf("%+v", item)
	}
}

func TestDoctorMovil_SolicitudesApartadasSeNombran_REM10(t *testing.T) {
	m := movilPrueba(t)
	movilSano(t, m, time.Now())
	ruta := escribirRemoto(t, "solicitudes.json.ilegible", "{roto")
	item := itemMovil(t, buildMovilChecks(m.raiz, time.Now()), itemMovilColas)
	if item.Success || !strings.Contains(item.Error, ruta) {
		t.Fatalf("%+v", item)
	}
}

func TestRemotoCorrer_ElRelayQueYaNoReconoceElPCDejaLaMarcaParaDoctor_REM15(t *testing.T) {
	m := movilPrueba(t)
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"28000","message":"credenciales de la máquina inválidas"}`))
	}))
	defer relay.Close()
	emparejarPrueba(t)
	guardarRelayPrueba(t, remoto.ConfigRelay{URL: relay.URL, Clave: "sb_publishable_a", Propio: true})
	var errores strings.Builder
	if code := remotoCorrer(&errores); code != 0 {
		t.Fatalf("salida %d: %s", code, errores.String())
	}
	if _, ok := remoto.QuitadoEn("qdd:pc:t"); !ok {
		t.Fatalf("falta la marca: %s", errores.String())
	}
	if unido, _, err := remoto.LeerConexion(); err != nil || unido {
		t.Fatalf("el canal nunca se unió: %v %v", unido, err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, time.Now()), itemMovilQuitado); item.Success {
		t.Fatalf("%+v", item)
	}
}
