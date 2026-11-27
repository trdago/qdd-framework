package consola

import (
	"fmt"
	"slices"
	"time"
)

// Rule: Prohibido usar 'else'
// Aviso del modo goal a la sesión (MEJ-025 parte 2, issue #147): la persona lo enciende o lo apaga a mitad de un trabajo
// y la IA tiene que saberlo, porque cambia de quién son las respuestas y quién corre sus pedidos. Cada sesión lo lee una
// vez, con su mensaje siguiente (el 🎯 del modo también lo es).

const (
	cambioEncendido = "encendido"
	cambioApagado   = "apagado"
	cambioDetenido  = "detenido"
	cambioReanudado = "reanudado"
)

// marcarCambio anota qué cambió y cuándo: cada sesión lo lee de nuevo una vez.
func marcarCambio(g *goalGuardado, fue string, now time.Time) {
	g.Cambio, g.CambioFue, g.Enteradas = &now, fue, nil
}

// maxEnteradas es el tope de sesiones que recuerda por cambio: una sesión más vieja que esas ya terminó.
const maxEnteradas = 20

// tomarAvisoGoal devuelve, una sola vez por sesión, que el modo se encendió o se apagó. Sin la hora del cambio (un
// archivo anterior a este aviso) no hay cambio que avisar.
func tomarAvisoGoal(root, sesion string) string {
	sesion = sesionValida(sesion)
	g, err := leerGoal(root)
	if err != nil || g.Cambio == nil || slices.Contains(g.Enteradas, sesion) {
		return ""
	}
	_ = conGoalGuardado(root, func(g *goalGuardado) error {
		g.Enteradas = append(g.Enteradas, sesion)
		if len(g.Enteradas) > maxEnteradas {
			g.Enteradas = g.Enteradas[len(g.Enteradas)-maxEnteradas:]
		}
		return nil
	})
	return mensajeCambioGoal(g, *g.Cambio)
}

func mensajeCambioGoal(g goalGuardado, cuando time.Time) string {
	hora := cuando.Local().Format("15:04")
	if g.CambioFue == cambioDetenido {
		return fmt.Sprintf("[qdd · panel] A las %s el modo goal se detuvo: %d recordatorios sin avance. No llegan más 🎯 hasta que haya avance real "+
			"(un cambio en el tablero o en git) o la persona lo active de nuevo: sigue con lo que puedas mover, sin esperarlo.", hora, MaxRecordatoriosGoal)
	}
	if g.CambioFue == cambioReanudado {
		return "[qdd · panel] A las " + hora + " el modo goal volvió a trabajar: hubo avance desde que se detuvo. Siguen llegando los 🎯, " +
			"y sus respuestas las elige el modo, no la persona."
	}
	if !g.Activo {
		return "[qdd · panel] A las " + hora + " se apagó el modo goal: desde ahora tus preguntas, pedidos y fichas esperan a la persona, " +
			"y no llegan más recordatorios 🎯. Lo que eligió el modo antes sigue en qdd goal estado."
	}
	return "[qdd · panel] A las " + hora + " la persona encendió el modo goal: qdd elige la opción recomendada de tus preguntas, " +
		"corre tus pedidos y aprueba tus fichas. Esas respuestas las eligió el modo, no la persona: escríbelo así. " +
		"Una pregunta de aprobación, una sin opción recomendada y los permisos siguen esperándola."
}
