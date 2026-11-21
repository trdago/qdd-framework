package invitaciones

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 6: qdd doctor revisa los flujos compartidos sin red. FLC-01 una invitación viva cuyo flujo ya no
// existe (--fix la revoca), FLC-02 propuestas sin resolver con más de 7 días (solo avisa: --fix no toca propuestas) y
// FLC-03 fotos que el relay todavía no tiene (--fix las sube).

type revocadorDePrueba struct {
	revocadas []string
	caido     bool
}

func (r *revocadorDePrueba) Revocar(_ context.Context, id string, _ time.Time) error {
	if r.caido {
		return errors.New("sin red")
	}
	r.revocadas = append(r.revocadas, id)
	return nil
}

func existeSolo(ids ...string) func(string) bool {
	return func(id string) bool {
		for _, x := range ids {
			if x == id {
				return true
			}
		}
		return false
	}
}

func TestRevisar_SinNadaCompartidoNoHayNadaQueRevisar(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if r := Revisar(t.TempDir(), existeSolo(), ahoraDePrueba); r.Compartido {
		t.Fatalf("un proyecto que nunca compartió no muestra el grupo: %+v", r)
	}
}

func TestRevisar_FLC01_UnaInvitacionVivaDeUnFlujoQueYaNoExiste(t *testing.T) {
	e, _, _ := entornoDePrueba(t)
	vencida := Invitacion{ID: "vencida", Flujo: "borrado", Vence: ahoraDePrueba.Add(-time.Hour)}
	viva := Invitacion{ID: "viva", Flujo: "borrado", Vence: ahoraDePrueba.Add(time.Hour)}
	for _, i := range []Invitacion{vencida, viva} {
		if err := Anotar(e.Root, i); err != nil {
			t.Fatal(err)
		}
	}
	r := Revisar(e.Root, existeSolo("pago"), ahoraDePrueba)
	if !r.Compartido || len(r.Huerfanas) != 1 || r.Huerfanas[0].ID != "viva" {
		t.Fatalf("solo la viva de un flujo que no existe: %+v", r.Huerfanas)
	}
	rev := &revocadorDePrueba{}
	hechas, err := RevocarHuerfanas(context.Background(), e.Root, rev, existeSolo("pago"), ahoraDePrueba)
	if err != nil || len(hechas) != 1 || len(rev.revocadas) != 1 || rev.revocadas[0] != "viva" {
		t.Fatalf("--fix la revoca en el relay: %v %v %v", hechas, rev.revocadas, err)
	}
	if r := Revisar(e.Root, existeSolo("pago"), ahoraDePrueba); len(r.Huerfanas) != 0 {
		t.Fatalf("revocada queda anotada y deja de contar: %+v", r.Huerfanas)
	}
}

func TestRevisar_FLC01_SinRedNoLaDaPorRevocada(t *testing.T) {
	e, _, _ := entornoDePrueba(t)
	if err := Anotar(e.Root, Invitacion{ID: "viva", Flujo: "borrado", Vence: ahoraDePrueba.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := RevocarHuerfanas(context.Background(), e.Root, &revocadorDePrueba{caido: true}, existeSolo("pago"), ahoraDePrueba); err == nil {
		t.Fatal("sin red lo dice")
	}
	if r := Revisar(e.Root, existeSolo("pago"), ahoraDePrueba); len(r.Huerfanas) != 1 {
		t.Fatal("la invitación sigue viva en el relay: no se marca revocada aquí")
	}
}

func TestRevisar_FLC02_PropuestasSinResolverConMasDe7Dias(t *testing.T) {
	e, _, _ := entornoDePrueba(t)
	vieja := ahoraDePrueba.Add(-PropuestaVieja - time.Hour).Format(time.RFC3339)
	nueva := ahoraDePrueba.Add(-time.Hour).Format(time.RFC3339)
	err := conLocal(e.Root, func(l *Local) error {
		l.Propuestas = []Vista{
			{Propuesta: Propuesta{ID: "PRO-1", Flujo: "pago", Llego: vieja}, Situacion: Pedida},
			{Propuesta: Propuesta{ID: "PRO-2", Flujo: "pago", Llego: nueva}, Situacion: Pedida},
			{Propuesta: Propuesta{ID: "PRO-3", Flujo: "pago", Llego: vieja}, Situacion: Aprobada},
			{Propuesta: Propuesta{ID: "PRO-4", Flujo: "pago", Llego: "ilegible"}, Situacion: PorPedir},
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	r := Revisar(e.Root, existeSolo("pago"), ahoraDePrueba)
	if len(r.Viejas) != 1 || r.Viejas[0].ID != "PRO-1" {
		t.Fatalf("solo la sin resolver de más de %v: %+v", PropuestaVieja, r.Viejas)
	}
}

func TestRevisar_FLC03_FotosQueElRelayNoTieneSeSuben(t *testing.T) {
	e, relay, _ := entornoDePrueba(t)
	if err := conLocal(e.Root, func(l *Local) error { l.FotosPorSubir = []string{"pago"}; return nil }); err != nil {
		t.Fatal(err)
	}
	if r := Revisar(e.Root, existeSolo("pago"), ahoraDePrueba); len(r.FotosPorSubir) != 1 {
		t.Fatalf("la foto pendiente se nota: %+v", r)
	}
	if err := SubirFotosPendientes(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if relay.fotos != 1 {
		t.Fatalf("--fix sube la foto: %d", relay.fotos)
	}
	if r := Revisar(e.Root, existeSolo("pago"), ahoraDePrueba); len(r.FotosPorSubir) != 0 {
		t.Fatalf("subida, deja de contar: %+v", r)
	}
}
