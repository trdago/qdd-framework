package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/invitaciones"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 6: qdd doctor tiene el grupo «Flujos compartidos» solo si el proyecto compartió algo. FLC-01 y FLC-03
// se reparan con --fix (revocar en el relay, subir la foto); FLC-02 solo avisa, porque --fix no toca propuestas.

type relayDeDoctorFalso struct {
	revocadas []string
	fotos     int
}

func (r *relayDeDoctorFalso) Propuestas(context.Context, string, time.Time) ([]invitaciones.Propuesta, error) {
	return nil, nil
}
func (r *relayDeDoctorFalso) Resolver(context.Context, string, string, string, time.Time) error {
	return nil
}
func (r *relayDeDoctorFalso) SubirFoto(context.Context, string, json.RawMessage, time.Time) (int, error) {
	r.fotos++
	return 1, nil
}
func (r *relayDeDoctorFalso) Revocar(_ context.Context, id string, _ time.Time) error {
	r.revocadas = append(r.revocadas, id)
	return nil
}

func TestDoctor_FlujosCompartidos_SinCompartirNoHayGrupo(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if g := buildFlujosCompartidosChecks(proyectoConFlujo(t), time.Now()); len(g.Items) != 0 {
		t.Fatalf("un proyecto que nunca compartió no muestra el grupo: %+v", g.Items)
	}
}

func TestDoctor_FlujosCompartidos_FLC01a03(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := proyectoConFlujo(t)
	ahora := time.Now()
	if err := invitaciones.Anotar(root, invitaciones.Invitacion{ID: "inv-borrado", Flujo: "borrado", Vence: ahora.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := invitaciones.Anotar(root, invitaciones.Invitacion{ID: "inv-pago", Flujo: "pago", Vence: ahora.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	g := buildFlujosCompartidosChecks(root, ahora)
	if g.Name != grupoFlujosCompartidos || len(g.Items) != 3 {
		t.Fatalf("el grupo trae FLC-01, FLC-02 y FLC-03: %+v", g)
	}
	huerfana := itemDeGrupo(g, itemFlujosHuerfanas)
	if huerfana.Success || !strings.Contains(huerfana.Error, "borrado") || strings.Contains(huerfana.Error, "inv-pago") {
		t.Fatalf("FLC-01 nombra la invitación del flujo que ya no existe: %+v", huerfana)
	}
	if !itemDeGrupo(g, itemFlujosPropuestasViejas).Success || !itemDeGrupo(g, itemFlujosFotos).Success {
		t.Fatal("sin propuestas viejas ni fotos pendientes, FLC-02 y FLC-03 están bien")
	}
	falso := &relayDeDoctorFalso{}
	antes := relayDeDoctorFlujos
	relayDeDoctorFlujos = func(context.Context) (relayDoctorFlujos, error) { return falso, nil }
	t.Cleanup(func() { relayDeDoctorFlujos = antes })
	repairFlujosCompartidos(root, g, ahora)
	if len(falso.revocadas) != 1 || falso.revocadas[0] != "inv-borrado" {
		t.Fatalf("--fix revoca solo la huérfana: %v", falso.revocadas)
	}
	if g := buildFlujosCompartidosChecks(root, ahora); !itemDeGrupo(g, itemFlujosHuerfanas).Success {
		t.Fatalf("después de --fix, FLC-01 está bien: %+v", g)
	}
}
