package remoto

import (
	"context"
	"time"
)

// Rule: Prohibido usar 'else'
// La sonda dentro del servicio (sonda.go): corre sola mientras el PC está emparejado y el servicio vive, aplica solo la
// reparación segura de su tramo y deja el resultado al teléfono (en el estado) y a doctor (AlSondear → sonda.json).

// relojSonda dispara la primera sonda; sin Sondear devuelve un reloj detenido que nunca suena.
func (b *Bucle) relojSonda() *time.Timer {
	t := time.NewTimer(b.PrimeraSonda)
	if b.Sondear == nil {
		t.Stop()
	}
	return t
}

// correrSonda la corre aparte: su conexión propia no detiene el bucle.
func (b *Bucle) correrSonda(ctx context.Context) {
	go func() {
		r := b.Sondear(ctx)
		select {
		case b.sondas <- r:
		case <-ctx.Done():
		}
	}()
}

// alSonda anota el resultado y repara: con el canal rechazado compara el relay con su directorio (si cambió, el bucle
// vuelve a arrancar con él); si Realtime no responde o no devuelve, se reinicia, como mucho una vez cada EntreReinicios.
func (b *Bucle) alSonda(ctx context.Context, r ResultadoSonda) error {
	if r.En.IsZero() {
		r.En = b.ahora().UTC()
	}
	b.mensajes += r.Mensajes
	if !r.OK {
		b.logf("prueba del canal con el teléfono: falla en %s: %s", r.Tramo, r.Detalle())
	}
	r.Reparado = ReparacionDe(r)
	if r.Reparado == ReparaReiniciar && !b.puedeReiniciarPorSonda() {
		r.Reparado = ""
	}
	antes := diagnosticoDe(b.ultimaSonda, b.ahora())
	b.ultimaSonda = &r
	if b.AlSondear != nil {
		b.AlSondear(r)
	}
	if r.Reparado == ReparaReiniciar {
		b.logf("%v", ErrSondaReinicia)
		return ErrSondaReinicia
	}
	if r.Reparado == ReparaDirectorio {
		b.consultarAhora(ctx)
	}
	b.avisarDiagnostico(ctx, antes)
	return nil
}

// puedeReiniciarPorSonda: una sola política de reinicio por el canal. Con el canal propio caído la caída es de REM-18
// (el canal reintenta solo y doctor --fix reinicia una vez); si no, como mucho un reinicio cada EntreReinicios, contando
// los que hizo doctor (ReinicioEn de sonda.json).
func (b *Bucle) puedeReiniciarPorSonda() bool {
	if !b.caidoDesde.IsZero() {
		return false
	}
	return PuedeReiniciarPorSonda(b.ReinicioSonda, b.ahora())
}

// consultarAhora pregunta al directorio por el relay sin esperar una caída larga, pero no más de una vez por CadaConsulta.
func (b *Bucle) consultarAhora(ctx context.Context) {
	now := b.ahora()
	if b.Consultar == nil || (!b.consultadoEn.IsZero() && now.Sub(b.consultadoEn) < b.CadaConsulta) {
		return
	}
	b.consultadoEn = now
	go func() {
		cambio, err := b.Consultar(ctx)
		select {
		case b.consultas <- resultadoConsulta{cambio: cambio, err: err}:
		case <-ctx.Done():
		}
	}()
}

// avisarDiagnostico manda el estado completo a los teléfonos cuando el diagnóstico cambió: un delta no lo lleva.
func (b *Bucle) avisarDiagnostico(ctx context.Context, antes *DiagnosticoCanal) {
	ahora := diagnosticoDe(b.ultimaSonda, b.ahora())
	if len(b.telefonos) == 0 || mismoDiagnostico(antes, ahora) {
		return
	}
	b.mandarEstado(ctx, b.armar())
}

func mismoDiagnostico(a, b *DiagnosticoCanal) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.OK == b.OK && a.Tramo == b.Tramo && a.Reparado == b.Reparado
}
