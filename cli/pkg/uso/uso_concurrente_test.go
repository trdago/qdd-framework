package uso

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

// contrato: estado#bcb808b7
// Leer, sumar y escribir el registro de uso es uno a la vez: 20 eventos que llegan juntos (20 consolas, 20 proyectos)
// no pierden ninguno. Cada evento sube la última hora o no suma; con todos anotados, lo medido es exactamente el tramo
// entre el primero y el último (20 s), repartido entre los proyectos.
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
	for _, err := range errores {
		if err != nil {
			t.Fatalf("ningún evento choca con otro: %v", err)
		}
	}
	u, err := Leer("2.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if u.Segundos != n || u.Ultimo != t0.Add(n*time.Second).Format(time.RFC3339) {
		t.Fatalf("lo medido es el tramo entero, del primero al último evento: %d s, último %s", u.Segundos, u.Ultimo)
	}
	var suma int64
	for _, s := range u.Proyectos {
		suma += s
	}
	if suma != u.Segundos {
		t.Fatalf("lo de cada proyecto suma el total: %d ≠ %d (%v)", suma, u.Segundos, u.Proyectos)
	}
}
