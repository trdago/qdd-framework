package consola

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-297 (qdd-movile, 2026-11-22): el pedido [e] de qdd-movile, confirmado el 15/11, reintentó 18 876 veces en 6 días
// cada 30 s. terminar borraba ArmadoEn antes de rearmarlo: cada «todavía no» empezaba la cuenta de nuevo y TopeArmado
// solo valía para el que espera a otro pedido. El tope cuenta desde que se armó la primera vez.

func TestArmado_ElQueSiempreDiceTodaviaNoVuelveAPendienteTrasElTope_FND297(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	cadasCortos(t)
	TopeArmado = 200 * time.Millisecond
	root := proyectoConCola(t)
	p := mustPedir(t, root, "echo 'falta el permiso'; exit 75", "Desplegar", "")
	if _, err := aceptar(root, p.ID, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	select {
	case codigo := <-correrAparte(root, p.ID):
		if codigo != 0 {
			t.Fatalf("termina sin error al dejar de esperar: %d", codigo)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("pasado el tope deja de reintentar")
	}
	q := esperarEstado(t, root, p.ID, EstadoPendiente)
	if !strings.Contains(q.Aviso, "Dejó de esperar") || q.Intentos < 2 {
		t.Fatalf("vuelve a la persona diciendo por qué, tras reintentar: %+v", q)
	}
}
