package remoto

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// El servicio corre la sonda solo (al arrancar y cada CadaSonda) y aplica la reparación segura de su tramo: comparar el
// relay con el directorio, o reiniciarse como mucho una vez cada 6 h. Lo que sigue roto lo ve el teléfono en el estado.

type sondasFalsas struct {
	mu        sync.Mutex
	resultado ResultadoSonda
	vistas    []ResultadoSonda
	anotadas  chan ResultadoSonda
}

func nuevasSondas(r ResultadoSonda) *sondasFalsas {
	return &sondasFalsas{resultado: r, anotadas: make(chan ResultadoSonda, 16)}
}

func (s *sondasFalsas) sondear(context.Context) ResultadoSonda {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resultado
}

func (s *sondasFalsas) anotar(r ResultadoSonda) { s.anotadas <- r }

func (s *sondasFalsas) esperar(t *testing.T) ResultadoSonda {
	t.Helper()
	select {
	case r := <-s.anotadas:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("el servicio no corrió la sonda")
		return ResultadoSonda{}
	}
}

func conSondas(s *sondasFalsas, ajustes ...func(*Bucle)) func(*Bucle) {
	return func(b *Bucle) {
		b.Sondear, b.AlSondear = s.sondear, s.anotar
		b.PrimeraSonda, b.CadaSonda = 10*time.Millisecond, time.Hour
		for _, a := range ajustes {
			a(b)
		}
	}
}

func TestBucle_LaSondaCorreSolaAlArrancar(t *testing.T) {
	homeTemporal(t)
	s := nuevasSondas(ResultadoSonda{OK: true})
	bucleDePrueba(t, conSondas(s))
	if r := s.esperar(t); !r.OK || r.Reparado != "" {
		t.Fatalf("una sonda buena no repara nada: %+v", r)
	}
}

func TestBucle_SinSondearNoHaySonda(t *testing.T) {
	homeTemporal(t)
	_, _, _, fin := bucleDePrueba(t)
	select {
	case err := <-fin:
		t.Fatalf("sin Sondear el bucle sigue igual: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestBucle_UnCanalRechazadoComparaElRelayConElDirectorio(t *testing.T) {
	homeTemporal(t)
	s := nuevasSondas(ResultadoSonda{Tramo: TramoCanal, Motivo: MotivoDeTramo(TramoCanal)})
	consultas := make(chan bool, 4)
	_, _, _, fin := bucleDePrueba(t, conSondas(s, func(b *Bucle) {
		b.Consultar = func(context.Context) (bool, error) { consultas <- true; return true, nil }
	}))
	if r := s.esperar(t); r.Reparado != ReparaDirectorio {
		t.Fatalf("anota qué reparó: %+v", r)
	}
	select {
	case err := <-fin:
		fin <- err // la limpieza de bucleDePrueba también lo espera
		if !errors.Is(err, ErrRelayCambio) {
			t.Fatalf("el directorio dio otro relay: vuelve a arrancar con él: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("el servicio no consultó el directorio")
	}
}

func TestBucle_UnEcoPerdidoReiniciaElServicioUnaSolaVez(t *testing.T) {
	homeTemporal(t)
	s := nuevasSondas(ResultadoSonda{Tramo: TramoEco, Motivo: MotivoDeTramo(TramoEco)})
	_, _, _, fin := bucleDePrueba(t, conSondas(s))
	select {
	case err := <-fin:
		fin <- err
		if !errors.Is(err, ErrSondaReinicia) {
			t.Fatalf("se reinicia para volver a conectar: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("el servicio no se reinició")
	}
	if r := s.esperar(t); r.Reparado != ReparaReiniciar {
		t.Fatalf("anota el reinicio antes de salir: %+v", r)
	}
}

func TestBucle_TrasUnReinicioRecienteNoVuelveAReiniciarYLoMuestraAlTelefono(t *testing.T) {
	homeTemporal(t)
	s := nuevasSondas(ResultadoSonda{Tramo: TramoEco, Motivo: MotivoDeTramo(TramoEco), En: time.Now().UTC()})
	_, tr, _, fin := bucleDePrueba(t, conSondas(s, func(b *Bucle) { b.ReinicioSonda = time.Now().Add(-time.Hour) }))
	if r := s.esperar(t); r.Reparado != "" {
		t.Fatalf("reinició hace una hora: no repite: %+v", r)
	}
	select {
	case err := <-fin:
		t.Fatalf("sigue corriendo: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	tr.eventos <- telefono("d1", "")
	e := tr.esperar(t, time.Second, "estado")
	var est Estado
	json.Unmarshal(e.Payload, &est)
	if est.Diagnostico == nil || est.Diagnostico.OK || est.Diagnostico.Tramo != TramoEco || est.Diagnostico.Motivo != "eco_perdido" {
		t.Fatalf("el teléfono ve que el canal falló, en qué tramo y con un código que la app traduce: %+v", est.Diagnostico)
	}
	var crudo struct {
		Diagnostico map[string]any `json:"diagnostico"`
	}
	json.Unmarshal(e.Payload, &crudo)
	en, _ := crudo.Diagnostico["en"].(string)
	if _, err := time.Parse(time.RFC3339, en); err != nil || !strings.HasSuffix(en, "Z") {
		t.Fatalf("en va en RFC3339 UTC: %q", en)
	}
}

// El motivo que viaja al teléfono es una lista cerrada y estable (pedido de qdd-movile): un código por tramo, nunca el
// texto en español, que queda para doctor y la CLI.
func TestDiagnostico_ElMotivoEsUnCodigoDeLaListaCerradaYLaHoraEsUTC(t *testing.T) {
	cerrada := map[string]string{TramoRelay: "sin_relay", TramoConexion: "sin_red", TramoCanal: "canal_rechazado", TramoRealtime: "realtime_sin_respuesta", TramoEco: "eco_perdido"}
	if len(CodigosMotivo) != len(cerrada) {
		t.Fatalf("la lista cambió: documenta el código nuevo en docs/qdd-movil-comunicacion.md y en la ficha remoto: %v", CodigosMotivo)
	}
	santiago := time.FixedZone("CLT", -3*3600)
	for tramo, codigo := range cerrada {
		r := ResultadoSonda{Tramo: tramo, Motivo: MotivoDeTramo(tramo), En: time.Now().In(santiago)}
		d := diagnosticoDe(&r, time.Now())
		if d == nil || d.Motivo != codigo || d.En.Location() != time.UTC {
			t.Fatalf("%s: %+v", tramo, d)
		}
	}
}

// Una sola política de reinicio por el canal: con el canal propio caído, la caída es de REM-18 (el canal reintenta solo
// y doctor --fix reinicia una vez); la sonda no reinicia otra vez por la misma caída.
func TestBucle_ConElCanalCaidoLaSondaNoReiniciaPorLaMismaCaida(t *testing.T) {
	homeTemporal(t)
	s := nuevasSondas(ResultadoSonda{Tramo: TramoEco, Motivo: MotivoDeTramo(TramoEco)})
	_, tr, _, fin := bucleDePrueba(t, conSondas(s, func(b *Bucle) { b.PrimeraSonda = 200 * time.Millisecond }))
	tr.eventos <- Evento{Tipo: TipoDesconectado, Motivo: "prueba"}
	if r := s.esperar(t); r.Reparado != "" {
		t.Fatalf("el canal está caído: la sonda no se suma a reiniciar: %+v", r)
	}
	select {
	case err := <-fin:
		t.Fatalf("sigue corriendo: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBucle_ConLaSondaBuenaElEstadoNoTraeDiagnostico(t *testing.T) {
	homeTemporal(t)
	s := nuevasSondas(ResultadoSonda{OK: true})
	_, tr, _, _ := bucleDePrueba(t, conSondas(s))
	s.esperar(t)
	tr.eventos <- telefono("d1", "")
	e := tr.esperar(t, time.Second, "estado")
	var crudo map[string]any
	json.Unmarshal(e.Payload, &crudo)
	if _, hay := crudo["diagnostico"]; hay {
		t.Fatalf("sin falla, el estado no gasta bytes en el diagnóstico: %s", e.Payload)
	}
}
