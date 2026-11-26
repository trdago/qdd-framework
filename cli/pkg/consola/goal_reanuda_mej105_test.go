package consola

import (
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// FND-331 (MEJ-105): en un proyecto real el modo se detuvo a las 03:08 tras 3 recordatorios sin avance y siguió
// detenido aunque desde las 10:00 la IA avanzaba; a las 10:28 esperó un 🎯 que no podía llegar y nadie le había dicho
// que el modo estaba detenido. Ahora la sesión lo sabe, y con avance real el modo vuelve solo (lo eligió el modo goal con
// su opción recomendada, no la persona, 2026-11-26).
// contrato: goal#c7a87407

func detenerPorSinAvance(t *testing.T, root string) time.Time {
	t.Helper()
	var now time.Time
	for i := 0; i <= MaxRecordatoriosGoal; i++ {
		now = enCola.Add(time.Duration(i+1) * 3 * time.Minute)
		cicloGoal(t, root, silencio(SilencioGoal), now)
	}
	if g := VistaGoal(root, now); g.Motivo != MotivoSinAvance {
		t.Fatalf("detenido sin avance: %+v", g)
	}
	return now
}

func TestGoal_LaSesionSabeQueSeDetuvo_FND331(t *testing.T) {
	root := conGoal(t)
	ContextoCiclo(root, "s1", enCola)
	detenido := detenerPorSinAvance(t, root)
	aviso := ContextoCiclo(root, "s1", detenido.Add(time.Minute))
	if !strings.Contains(aviso, "se detuvo") || !strings.Contains(aviso, "sin avance") {
		t.Fatalf("la sesión sabe que no llegarán más 🎯 y por qué:\n%s", aviso)
	}
}

func TestGoal_ConAvanceRealVuelveATrabajar_FND331(t *testing.T) {
	root := conGoal(t)
	detenido := detenerPorSinAvance(t, root)
	if r := cicloGoal(t, root, silencio(SilencioGoal), detenido.Add(time.Hour)); r.Mensaje != "" {
		t.Fatalf("sin avance sigue detenido: %+v", r)
	}
	ContextoCiclo(root, "s1", detenido.Add(time.Hour))
	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "PR", Estado: "✅", Evidencia: "PR #9"}, detenido); err != nil {
		t.Fatal(err)
	}
	r := cicloGoal(t, root, silencio(SilencioGoal), detenido.Add(2*time.Hour))
	if r.Mensaje == "" || !strings.Contains(r.Aviso, "sigue") {
		t.Fatalf("con avance real vuelve a recordar y se lo dice a la persona: %+v", r)
	}
	if g := VistaGoal(root, detenido.Add(2*time.Hour)); g.Estado != GoalTrabajando || g.Recordatorios != 1 {
		t.Fatalf("trabajando, con la cuenta desde cero: %+v", g)
	}
	if aviso := ContextoCiclo(root, "s1", detenido.Add(2*time.Hour)); !strings.Contains(aviso, "volvió a trabajar") {
		t.Fatalf("la sesión sabe que el modo volvió:\n%s", aviso)
	}
}

// Detenido, no revisa git en cada vuelta: solo cuando la consola calla, y como mucho una vez por SilencioGoal.
func TestGoal_DetenidoNoMiraElAvanceMientrasLaSesionTrabaja_FND331(t *testing.T) {
	root := conGoal(t)
	detenido := detenerPorSinAvance(t, root)
	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "PR", Estado: "✅", Evidencia: "PR #9"}, detenido); err != nil {
		t.Fatal(err)
	}
	if r := cicloGoal(t, root, EntradaGoal{Sesion: "s1", Ocupada: true}, detenido.Add(time.Hour)); r.Mensaje != "" || r.Aviso != "" {
		t.Fatalf("con la sesión trabajando no hace falta recordarle nada: %+v", r)
	}
}
