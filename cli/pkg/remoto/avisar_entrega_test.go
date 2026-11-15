package remoto

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-027 (FND-217): avisar responde 200 con {enviados} aunque el push no llegue a ningún teléfono, y el PC descartaba el
// cuerpo. Contrato con qdd-movile (2026-11-15): enviados, y aditivos destinos y motivo (sin_destinos, sin_fcm,
// fcm_rechazo) solo con enviados 0. Sin enviados (relay anterior), como antes.

func relayAvisar(t *testing.T, cuerpo string) *Relay {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cuerpo))
	}))
	t.Cleanup(srv.Close)
	return &Relay{URL: srv.URL, Clave: "sb_publishable_x", HTTP: srv.Client()}
}

func avisarCon(t *testing.T, cuerpo string) error {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	return relayAvisar(t, cuerpo).Avisar(ctx, topicoPrueba, "f", AvisoPedido, "qdd", "qdd-0123")
}

func TestAvisar_ConEntregaOSinCamposEsComoAntes_FND217(t *testing.T) {
	// Las dos primeras son las de resultadoEnvio en avisos_test.ts de qdd-movile: con enviados > 0 no va motivo, aunque
	// alguno falle.
	for _, cuerpo := range []string{`{"enviados":2,"destinos":2}`, `{"enviados":1,"destinos":2}`, `{"enviados":2}`, `{}`, ``, `ok`, `{"enviados":"x"}`} {
		if err := avisarCon(t, cuerpo); err != nil {
			t.Errorf("%q: un aviso entregado o de un relay anterior no es una falla: %v", cuerpo, err)
		}
	}
}

func TestAvisar_SinEntregaDiceLaCausa_FND217(t *testing.T) {
	// Las tres con motivo son las de resultadoEnvio en avisos_test.ts de qdd-movile (relay a1fbc9a); sin motivo o sin
	// destinos son las de un relay intermedio o anterior.
	casos := map[string]string{
		`{"enviados":0,"destinos":0,"motivo":"sin_destinos"}`: "ningún teléfono tiene los avisos activados (sin token de push)",
		`{"enviados":0,"destinos":3,"motivo":"sin_fcm"}`:      "falta la configuración de FCM en el relay",
		`{"enviados":0,"destinos":3,"motivo":"fcm_rechazo"}`:  "FCM no entregó: tokens vencidos o FCM caído",
		`{"enviados":0}`:                                        "el último aviso no llegó a ningún teléfono",
		`{"enviados":0,"destinos":0}`:                           "ningún teléfono tiene los avisos activados (sin token de push)",
		`{"enviados":0,"destinos":2}`:                           "FCM no entregó o falta su configuración en el relay",
		`{"enviados":0,"destinos":1,"motivo":"otro_que_crece"}`: "FCM no entregó o falta su configuración en el relay",
	}
	for cuerpo, texto := range casos {
		err := avisarCon(t, cuerpo)
		var sin ErrSinEntrega
		if !errors.As(err, &sin) {
			t.Fatalf("%s: enviados 0 es un aviso que no llegó: %v", cuerpo, err)
		}
		if sin.Tipo != AvisoPedido || sin.Clave != "qdd-0123" {
			t.Errorf("%s: el error dice qué aviso no llegó: %+v", cuerpo, sin)
		}
		if got := TextoSinEntrega(sin.Motivo, sin.Destinos); got != texto {
			t.Errorf("%s: texto %q, want %q", cuerpo, got, texto)
		}
	}
}

func TestAvisar_UnCuerpoGrandeSeCortaYNoEsFalla_FND217(t *testing.T) {
	cuerpo := `{"relleno":"` + strings.Repeat("x", 8<<10) + `","enviados":0}`
	if err := avisarCon(t, cuerpo); err != nil {
		t.Fatalf("un cuerpo de más de %d bytes se corta y se trata como un relay anterior: %v", topeCuerpoAvisar, err)
	}
}

func TestBucle_UnAvisoSinEntregaQuedaParaDoctorYUnoEntregadoLoLimpia_FND217(t *testing.T) {
	homeTemporal(t)
	destinos := 0
	b := &Bucle{}
	b.alAviso(ErrSinEntrega{Tipo: AvisoPregunta, Clave: "qdd-0123", Destinos: &destinos, Motivo: MotivoSinDestinos})
	s, err := LeerSinEntrega()
	if err != nil || s.Tipo != AvisoPregunta || s.Motivo != MotivoSinDestinos || s.Destinos == nil || *s.Destinos != 0 || s.En.IsZero() {
		t.Fatalf("el aviso sin entrega queda anotado para doctor: %+v %v", s, err)
	}
	ruta, _ := rutaRemoto(archivoSinEntrega)
	datos, _ := os.ReadFile(ruta)
	if len(datos) == 0 || strings.Contains(string(datos), "qdd-0123") {
		t.Fatal("lo anotado no trae el proyecto (solo tipo, motivo, destinos y hora)")
	}
	b.alAviso(nil)
	if _, err := LeerSinEntrega(); err == nil {
		t.Fatal("un aviso que sí llegó limpia lo anotado")
	}
}

func TestBucle_ElRegistroNoRepiteElMismoAvisoSinEntregaEnUnaHora_FND217(t *testing.T) {
	homeTemporal(t)
	var registro strings.Builder
	lineas := func() int { return strings.Count(registro.String(), "\n") }
	ahora := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	b := &Bucle{Log: &registro, Ahora: func() time.Time { return ahora }}
	sin := ErrSinEntrega{Tipo: AvisoPedido, Clave: "qdd-0123"}
	b.alAviso(sin)
	b.alAviso(sin)
	b.alAviso(ErrSinEntrega{Tipo: AvisoPedido, Clave: "otro-0456"})
	if lineas() != 2 {
		t.Fatalf("el mismo tipo y proyecto se anota una vez por hora; otro proyecto sí: %d líneas", lineas())
	}
	ahora = ahora.Add(61 * time.Minute)
	b.alAviso(sin)
	if lineas() != 3 {
		t.Fatalf("pasada la hora se vuelve a anotar: %d líneas", lineas())
	}
}
