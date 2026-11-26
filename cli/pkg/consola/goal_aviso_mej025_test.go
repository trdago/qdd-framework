package consola

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// FND-328 (MEJ-025 parte 2, issue #147): la persona encendió el modo goal a mitad de un trabajo y la IA no lo supo. Con el
// mensaje siguiente, cada sesión lee una vez que el modo se encendió o se apagó, y qué cambia para ella.
// contrato: goal#bf58381d
func TestGoal_LaSesionSabeQueSeEncendioYSeApago_MEJ025(t *testing.T) {
	root := proyectoConTareas(t)
	escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoGoal)
	if got := ContextoCiclo(root, "s1", enCola); strings.Contains(got, "modo goal") {
		t.Fatalf("sin cambios del modo no dice nada del modo: %q", got)
	}
	if err := ActivarGoal(root, "panel", enCola); err != nil {
		t.Fatal(err)
	}
	encendido := ContextoCiclo(root, "s1", enCola.Add(time.Minute))
	for _, quiere := range []string{"encendió el modo goal", "no la persona"} {
		if !strings.Contains(encendido, quiere) {
			t.Fatalf("al encenderse la sesión lo sabe (%q):\n%s", quiere, encendido)
		}
	}
	if otra := ContextoCiclo(root, "s2", enCola.Add(time.Minute)); !strings.Contains(otra, "encendió el modo goal") {
		t.Fatalf("cada sesión lo lee: %q", otra)
	}
	if again := ContextoCiclo(root, "s1", enCola.Add(2*time.Minute)); strings.Contains(again, "modo goal") {
		t.Fatalf("una sola vez por sesión: %q", again)
	}
	if err := DesactivarGoal(root, enCola.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	apagado := ContextoCiclo(root, "s1", enCola.Add(4*time.Minute))
	if !strings.Contains(apagado, "se apagó el modo goal") || !strings.Contains(apagado, "esperan a la persona") {
		t.Fatalf("al apagarse la sesión sabe que vuelve a esperar a la persona:\n%s", apagado)
	}
}

// Un proyecto que ya tenía el modo encendido antes de esta versión no recibe un aviso de un cambio que no vio.
func TestGoal_SinCambioRegistradoNoAvisa_MEJ025(t *testing.T) {
	root := conGoal(t)
	if err := conGoalGuardado(root, func(g *goalGuardado) error { g.Cambio = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if got := ContextoCiclo(root, "s1", enCola); strings.Contains(got, "modo goal") {
		t.Fatalf("sin la hora del cambio no hay cambio que avisar: %q", got)
	}
}
