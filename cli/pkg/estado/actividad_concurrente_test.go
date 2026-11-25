package estado

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// Los pulsos corren en segundo plano, uno por herramienta, a la vez que el agente activa su tarea: un pulso que
// leyó la actividad antes no puede borrar al escribir la activación que otro proceso dejó entre medio.
func TestActividad_PulsosALaVezNoBorranUnaActivacion(t *testing.T) {
	var ids []string
	for i := 0; i < 24; i++ {
		ids = append(ids, fmt.Sprintf("t%02d", i))
	}
	root := proyectoConTareas(t, ids...)
	if err := Activar(root, ids[0], "claude-code", ahora); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i, id := range ids[1:] {
		wg.Add(2)
		go func(id string) { defer wg.Done(); activarConTurno(t, root, id) }(id)
		go func(n int) {
			defer wg.Done()
			_, _ = Pulsar(root, "", FuenteAgente, ahora.Add(time.Duration(n)*time.Millisecond))
		}(i)
	}
	wg.Wait()
	if en := EnCurso(LeerActividad(root), ahora.Add(time.Second)); len(en) != len(ids) {
		t.Fatalf("cada activación sobrevive a los pulsos simultáneos: %d de %d en curso", len(en), len(ids))
	}
}

// activarConTurno activa la tarea y, si el candado estuvo ocupado hasta su plazo, pide el turno otra vez: Activar lo dice
// con un error, que no es perder una escritura. Un equipo lento del CI pasa los 2 s del candado con 47 a la vez.
func activarConTurno(t *testing.T, root, id string) {
	var err error
	for range 10 {
		if err = Activar(root, id, "claude-code", ahora); err == nil {
			return
		}
	}
	t.Errorf("no se pudo activar %s: %v", id, err)
}
