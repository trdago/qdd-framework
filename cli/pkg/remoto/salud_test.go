package remoto

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

func TestBinarioInstalado_LeeElDeCadaSistema(t *testing.T) {
	bin := `/home/a b/.local/share/mise/installs/node/26.2.0/lib/node_modules/qdd-framework/bin/qdd`
	for _, so := range []string{"linux", "darwin", "windows"} {
		s := Servicio{SO: so, Bin: bin, Usuario: t.TempDir(), DirQDD: t.TempDir(), Path: "/usr/bin", QDDHome: "/q%d"}
		contenido, err := s.Contenido()
		if err != nil {
			t.Fatal(err)
		}
		if got := binarioDe(so, contenido); got != bin {
			t.Fatalf("%s: %q", so, got)
		}
	}
}

func TestBinarioInstalado_ConComillasYPorcentajeEnSystemd(t *testing.T) {
	bin := `/opt/q"d%d/qdd`
	s := Servicio{SO: "linux", Bin: bin}
	if got := binarioDe("linux", []byte(s.unidadSystemd())); got != bin {
		t.Fatalf("%q", got)
	}
	if got := binarioDe("linux", []byte("[Service]\nExecStart=/usr/bin/qdd remoto correr\n")); got != "/usr/bin/qdd" {
		t.Fatalf("sin comillas, escrito a mano: %q", got)
	}
}

func TestBinarioPerdido_SoloSiElArchivoNoEsta(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	s := Servicio{SO: "linux", Bin: "/no/existe/qdd", Usuario: t.TempDir()}
	if _, perdido := s.BinarioPerdido(); perdido {
		t.Fatal("sin servicio instalado no hay nada perdido")
	}
	if _, err := s.Instalar(func(string, ...string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if bin, perdido := s.BinarioPerdidoDeEsteHome(); !perdido || bin != "/no/existe/qdd" {
		t.Fatalf("%q %v", bin, perdido)
	}
}

func TestQuitadoEn_SoloParaElMismoTopico(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if _, ok := QuitadoEn("qdd:pc:a"); ok {
		t.Fatal("sin marca")
	}
	if err := GuardarQuitado("qdd:pc:a", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, ok := QuitadoEn("qdd:pc:a"); !ok {
		t.Fatal("el mismo tópico")
	}
	if _, ok := QuitadoEn("qdd:pc:b"); ok {
		t.Fatal("emparejar de nuevo da otro tópico")
	}
}

func TestAvisarConexion_UnaCaidaSeAvisaUnaVezConSuPrimeraHora(t *testing.T) {
	inicio := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	ahora := inicio
	var avisos []string
	b := &Bucle{Ahora: func() time.Time { return ahora }}
	b.AlConectar = func(conectado bool, en time.Time) {
		avisos = append(avisos, map[bool]string{true: "unido", false: "caido"}[conectado]+"@"+en.Format("15:04"))
	}
	b.anotarConexion()
	b.avisarConexion(true)
	for i := 0; i < 3; i++ {
		ahora = inicio.Add(time.Duration(i+1) * time.Minute)
		b.anotarCaida(ahora)
		b.avisarConexion(false)
	}
	ahora = inicio.Add(10 * time.Minute)
	b.anotarConexion()
	b.avisarConexion(true)
	want := []string{"unido@10:00", "caido@10:01", "unido@10:10"}
	if len(avisos) != len(want) {
		t.Fatalf("%v", avisos)
	}
	for i := range want {
		if avisos[i] != want[i] {
			t.Fatalf("%v", avisos)
		}
	}
}

func TestLatidoConHora_MideElRelojContraElDateDelRelay_REM19(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", time.Now().Add(-4*time.Minute).UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plan":"gratis","pcs":1,"proyectos":3,"consola_permitida":true}`))
	}))
	defer srv.Close()
	r := NuevoRelay(ConfigRelay{URL: srv.URL, Clave: "sb_publishable_a"})
	_, desfase, conHora, err := r.LatidoConHora(context.Background(), "t", "f", 0, 0, 0)
	if err != nil || !conHora || desfase < 3*time.Minute || desfase > 5*time.Minute {
		t.Fatalf("el relay va 4 min atrás: este PC, 4 min adelantado: %v %v %v", desfase, conHora, err)
	}
	if err := GuardarLatidoConHora(time.Now(), desfase, conHora); err != nil {
		t.Fatal(err)
	}
	if d, ok := DesfaseRelay(); !ok || (d-desfase).Abs() > time.Millisecond {
		t.Fatalf("latido.json lo guarda: %v %v", d, ok)
	}
	if err := GuardarLatido(time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, ok := DesfaseRelay(); ok {
		t.Fatal("sin hora del relay no hay desfase")
	}
}

func TestDesfaseReloj_SinLogNiDescartesEsCero(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if d, n := DesfaseReloj(time.Now()); d != 0 || n != 0 {
		t.Fatalf("%v %d", d, n)
	}
	if err := Anotar(EntradaLog{En: time.Now(), Estado: EstadoHecha}); err != nil {
		t.Fatal(err)
	}
	if _, n := DesfaseReloj(time.Now()); n != 0 {
		t.Fatal("una acción hecha no dice nada del reloj")
	}
}

func TestDesfaseReloj_RespuestaViejaDelTelefonoTambienCuenta(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	now := time.Now().UTC().Truncate(time.Second)
	tel := now.Add(8 * time.Minute).Format(formatoEn)
	if err := Anotar(EntradaLog{En: now, Estado: EstadoDescartada, Resultado: "vieja (" + tel + ")"}); err != nil {
		t.Fatal(err)
	}
	d, n := DesfaseReloj(now)
	if n != 1 || d != -8*time.Minute {
		t.Fatalf("PC atrasado 8 min: %v %d", d, n)
	}
}
