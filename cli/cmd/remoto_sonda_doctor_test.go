package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// REM-20 (pedido de la persona desde QDD Móvil, 2026-11-15): doctor dice cómo salió la última prueba del canal con el
// teléfono que corrió el servicio, sin usar la red; --fix la repite en el momento y aplica la reparación de su tramo.

func sondaPrueba(t *testing.T, r remoto.ResultadoSonda) *int {
	t.Helper()
	antes := remotoSondear
	t.Cleanup(func() { remotoSondear = antes })
	llamadas := 0
	remotoSondear = func(context.Context, remoto.ConfigRelay, remoto.Credenciales, time.Duration) remoto.ResultadoSonda {
		llamadas++
		return r
	}
	return &llamadas
}

func guardarSondaPrueba(t *testing.T, r remoto.ResultadoSonda) {
	t.Helper()
	if err := remoto.GuardarSonda(r, time.Time{}); err != nil {
		t.Fatal(err)
	}
}

// servicioSanoPrueba instala el servicio con un qdd que existe (si no, REM-17 lo reescribe y ese es el reinicio) y con
// el latido al día (si no, REM-07 reinicia): así el reinicio que mide una prueba es el del canal.
func servicioSanoPrueba(t *testing.T, now time.Time) {
	t.Helper()
	guardarRelayPrueba(t, relayAntiguoPrueba)
	s, _ := remotoServicio()
	s.Bin = filepath.Join(t.TempDir(), "qdd")
	if err := os.WriteFile(s.Bin, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	remotoServicio = func() (remoto.Servicio, error) { return s, nil }
	if _, err := s.Instalar(remotoEjecutor); err != nil {
		t.Fatal(err)
	}
	if err := remoto.GuardarLatido(now); err != nil {
		t.Fatal(err)
	}
}

func TestDoctorMovil_UnaSondaBuenaEnVerde_REM20(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	guardarSondaPrueba(t, remoto.ResultadoSonda{OK: true, Eco: true, En: now.Add(-10 * time.Minute)})
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilSonda); !item.Success {
		t.Fatalf("%+v", item)
	}
}

func TestDoctorMovil_UnaSondaQueFallaDiceElTramoSinUsarLaRed_REM20(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	llamadas := sondaPrueba(t, remoto.ResultadoSonda{OK: true})
	guardarSondaPrueba(t, remoto.ResultadoSonda{Tramo: remoto.TramoEco, Motivo: remoto.MotivoDeTramo(remoto.TramoEco), En: now.Add(-5 * time.Minute)})
	item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilSonda)
	if item.Success || !strings.Contains(item.Error, remoto.MotivoDeTramo(remoto.TramoEco)) || !strings.Contains(item.Error, "--fix") {
		t.Fatalf("dice qué tramo falló y cómo repararlo: %+v", item)
	}
	if *llamadas != 0 {
		t.Fatal("el diagnóstico no usa la red: solo --fix prueba en el momento")
	}
}

func TestDoctorMovil_UnaSondaViejaEsQueElServicioNoLaCorre_REM20(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	guardarSondaPrueba(t, remoto.ResultadoSonda{OK: true, En: now.Add(-5 * time.Hour)})
	item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilSonda)
	if item.Success || !strings.Contains(item.Error, "hace 5 h") {
		t.Fatalf("una prueba vieja no dice cómo está el canal hoy: %+v", item)
	}
}

// --fix no sondea: doctor no usa la red salvo el directorio firmado (invariante de la ficha remoto). Aplica la
// reparación de la última sonda del servicio, y el servicio la repite al arrancar.
func TestDoctorMovil_FixReiniciaSiElEcoNoVolvioSinUsarLaRed_REM20(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	servicioSanoPrueba(t, now)
	*m.corridos = nil
	llamadas := sondaPrueba(t, remoto.ResultadoSonda{OK: true})
	guardarSondaPrueba(t, remoto.ResultadoSonda{Tramo: remoto.TramoEco, Motivo: remoto.MotivoDeTramo(remoto.TramoEco), En: now.Add(-time.Minute)})
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	if *llamadas != 0 {
		t.Fatalf("--fix no sondea: %d", *llamadas)
	}
	if n := strings.Count(strings.Join(*m.corridos, ";"), "restart"); n != 1 {
		t.Fatalf("el eco no volvió: reinicia el servicio una vez para reconectar: %v", *m.corridos)
	}
	g, err := remoto.LeerSonda()
	if err != nil || g.ReinicioEn.IsZero() || g.Ultima.Tramo != remoto.TramoEco {
		t.Fatalf("anota el reinicio para el tope del servicio y conserva la prueba: %+v %v", g, err)
	}
}

func TestDoctorMovil_FixConUnaSondaViejaReiniciaParaQueElServicioLaCorra_REM20(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	servicioSanoPrueba(t, now)
	*m.corridos = nil
	llamadas := sondaPrueba(t, remoto.ResultadoSonda{OK: true})
	guardarSondaPrueba(t, remoto.ResultadoSonda{OK: true, En: now.Add(-5 * time.Hour)})
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	if *llamadas != 0 {
		t.Fatalf("--fix no sondea: %d", *llamadas)
	}
	if n := strings.Count(strings.Join(*m.corridos, ";"), "restart"); n != 1 {
		t.Fatalf("el servicio no corría la prueba: lo reinicia una vez: %v", *m.corridos)
	}
}

// Una sola política de reinicio (con REM-07 y REM-18): en una pasada de --fix el servicio se reinicia una vez, y ese
// reinicio cuenta para el tope del servicio, que así no se reinicia otra vez por el eco.
func TestDoctorMovil_FixConElCanalCaidoYElEcoPerdidoReiniciaUnaSolaVez_REM20(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	servicioSanoPrueba(t, now)
	if err := remoto.GuardarConexion(false, now.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	*m.corridos = nil
	guardarSondaPrueba(t, remoto.ResultadoSonda{Tramo: remoto.TramoEco, Motivo: remoto.MotivoDeTramo(remoto.TramoEco), En: now.Add(-time.Minute)})
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	if n := strings.Count(strings.Join(*m.corridos, ";"), "restart"); n != 1 {
		t.Fatalf("REM-18 y REM-20 reinician una sola vez: %v", *m.corridos)
	}
	g, err := remoto.LeerSonda()
	if err != nil || g.ReinicioEn.IsZero() || g.Ultima.Tramo != remoto.TramoEco {
		t.Fatalf("el reinicio de REM-18 queda para el tope del servicio y conserva la última prueba: %+v %v", g, err)
	}
	if remoto.PuedeReiniciarPorSonda(g.ReinicioEn, now.Add(time.Hour)) {
		t.Fatal("el servicio no se reinicia otra vez por el eco dentro de las 6 h")
	}
}

func TestDoctorMovil_FixConElCanalRechazadoComparaElDirectorio_REM20(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayAntiguoPrueba)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	if _, _, err := remoto.ActualizarRelay(context.Background(), directorioPrueba(new(int), relayAntiguoPrueba), relayAntiguoPrueba); err != nil {
		t.Fatal(err)
	}
	llamadas := sondaPrueba(t, remoto.ResultadoSonda{OK: true})
	guardarSondaPrueba(t, remoto.ResultadoSonda{Tramo: remoto.TramoCanal, Motivo: remoto.MotivoDeTramo(remoto.TramoCanal), En: now.Add(-time.Minute)})
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	if consultas == 0 {
		t.Fatal("con el canal rechazado, compara el relay con su directorio firmado")
	}
	if *llamadas != 0 {
		t.Fatalf("la única red de --fix es el directorio: %d", *llamadas)
	}
}

func TestDoctorMovil_FixConLaSondaBuenaNoTocaNada_REM20(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	servicioSanoPrueba(t, now)
	*m.corridos = nil
	guardarSondaPrueba(t, remoto.ResultadoSonda{OK: true, Eco: true, En: now.Add(-10 * time.Minute)})
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	if strings.Contains(strings.Join(*m.corridos, ";"), "restart") {
		t.Fatalf("el canal va y vuelve: no reinicia nada: %v", *m.corridos)
	}
}

func TestDoctorMovil_UnEstadoCercaDelTopeAvisaYNoSeRepara_REM21_FND209(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	if err := remoto.GuardarPeso(249075, now); err != nil {
		t.Fatal(err)
	}
	g := buildMovilChecks(m.raiz, now)
	item := itemMovil(t, g, itemMovilPeso)
	if item.Success || !strings.Contains(item.Error, "243 KB") {
		t.Fatalf("a 249 KB del tope de 256 KB avisa: %+v", item)
	}
	repairMovil(m.raiz, g, now)
	if len(*m.corridos) != 0 {
		t.Fatalf("reducir el estado es una decisión: --fix no lo toca: %v", *m.corridos)
	}
	if err := remoto.GuardarPeso(90*1024, now); err != nil {
		t.Fatal(err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilPeso); !item.Success {
		t.Fatalf("lejos del tope, en verde: %+v", item)
	}
}

func TestRemotoSonda_DiceElTramoSinRepararNiAnotar(t *testing.T) {
	movilPrueba(t)
	sondaPrueba(t, remoto.ResultadoSonda{Tramo: remoto.TramoCanal, Motivo: remoto.MotivoDeTramo(remoto.TramoCanal), MS: 12})
	var salida strings.Builder
	if codigo := remotoSonda(&salida, false); codigo != 1 || !strings.Contains(salida.String(), remoto.TramoCanal) {
		t.Fatalf("%d %q", codigo, salida.String())
	}
	if _, err := remoto.LeerSonda(); err == nil {
		t.Fatal("qdd remoto sonda solo mira: no anota")
	}
	sondaPrueba(t, remoto.ResultadoSonda{OK: true, Eco: true, Telefonos: 1, MS: 80})
	salida.Reset()
	if codigo := remotoSonda(&salida, true); codigo != 0 || !strings.Contains(salida.String(), `"eco":true`) {
		t.Fatalf("%d %q", codigo, salida.String())
	}
}
