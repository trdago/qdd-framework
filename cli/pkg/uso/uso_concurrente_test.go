package uso

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

// contrato: estado#bcb808b7
// Leer, sumar y escribir el registro de uso es uno a la vez: de 20 eventos que llegan juntos (20 consolas, 20 proyectos)
// no se pierde ninguno de los que se anotaron. Cada evento sube la última hora o no suma, así que lo medido llega hasta
// el último anotado, repartido entre los proyectos. Un evento que esperó el bloqueo más de EsperaBloqueo no se anota
// (FND-280): con la máquina cargada pasa (CI de #311, 2026-11-21), y no es una pérdida.
func TestRegistrar_VeinteEventosALaVezNoPierdenNinguno(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if err := Registrar("2.9.0", "base", t0); err != nil {
		t.Fatal(err)
	}
	const n = 20
	errores := make([]error, n)
	var listos, fin sync.WaitGroup
	partida := make(chan struct{})
	for i := 1; i <= n; i++ {
		listos.Add(1)
		fin.Add(1)
		go func(i int) {
			defer fin.Done()
			listos.Done()
			<-partida
			errores[i-1] = Registrar("2.9.0", fmt.Sprintf("p%02d", i), t0.Add(time.Duration(i)*time.Second))
		}(i)
	}
	listos.Wait()
	close(partida)
	fin.Wait()
	ultimo := 0
	for i, err := range errores {
		if err != nil && !strings.Contains(err.Error(), "lo usa otro proceso") {
			t.Fatalf("un evento solo puede quedar fuera por esperar el bloqueo: %v", err)
		}
		if err == nil {
			ultimo = i + 1
		}
	}
	if ultimo == 0 {
		t.Fatal("alguno se anota")
	}
	u, err := Leer("2.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if u.Segundos != int64(ultimo) || u.Ultimo != t0.Add(time.Duration(ultimo)*time.Second).Format(time.RFC3339) {
		t.Fatalf("lo medido llega hasta el último evento anotado (%d s): %d s, último %s", ultimo, u.Segundos, u.Ultimo)
	}
	var suma int64
	for _, s := range u.Proyectos {
		suma += s
	}
	if suma != u.Segundos {
		t.Fatalf("lo de cada proyecto suma el total: %d ≠ %d (%v)", suma, u.Segundos, u.Proyectos)
	}
}
