package consola

import (
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-294: sin QDD Móvil ni «consola seguir», el mod late la sesión cada 60 s (TELEFONO_PAUSA) y una sesión cuenta
// como abierta solo VidaSesion (30 s): la mitad del tiempo, la vista del modo goal decía «pausado · sin consola» con
// la sesión trabajando. El ciclo lo pide esa misma sesión cada 5 s, así que también la late.

func TestGoal_ElCicloDeUnaSesionLaLateYLaVistaNoParpadea_FND294(t *testing.T) {
	root := conGoal(t)
	despues := enCola.Add(VidaSesion + 10*time.Second)
	if v := VistaGoal(root, despues); v.Estado != GoalPausado {
		t.Fatalf("sin latidos la sesión se da por cerrada: %+v", v)
	}
	cicloGoal(t, root, silencio(0), despues)
	if v := VistaGoal(root, despues.Add(2*time.Second)); v.Estado != GoalTrabajando {
		t.Fatalf("la sesión que pide el ciclo está abierta: %+v", v)
	}
}

func TestGoal_UnCicloSinSesionNoLaInventa_FND294(t *testing.T) {
	root := conGoal(t)
	despues := enCola.Add(VidaSesion + 10*time.Second)
	cicloGoal(t, root, EntradaGoal{}, despues)
	if HaySesion(root, despues) {
		t.Fatal("sin --sesion no hay a quién latir")
	}
}

// Sin ninguna sesión abierta nadie puede avanzar (la vista dice «pausado · sin consola»): un ciclo así no recuerda ni
// cuenta recordatorios, y nunca termina en «sin avance» (observado en las pruebas de contrato de la oleada 2).
func TestGoal_SinSesionAbiertaElCicloNoCuentaRecordatorios_FND294(t *testing.T) {
	root := conGoal(t)
	despues := enCola.Add(VidaSesion + 10*time.Second)
	for i := 0; i <= MaxRecordatoriosGoal; i++ {
		r := cicloGoal(t, root, EntradaGoal{Silencio: SilencioGoal}, despues.Add(time.Duration(i)*3*time.Minute))
		if r.Mensaje != "" {
			t.Fatalf("sin sesión no hay a quién recordarle: %+v", r)
		}
	}
	g, _ := leerGoal(root)
	if g.Recordatorios != 0 || g.Motivo == MotivoSinAvance {
		t.Fatalf("no cuenta recordatorios ni se detiene: %+v", g.Goal)
	}
}
