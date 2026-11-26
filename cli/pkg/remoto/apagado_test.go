package remoto

import (
	"context"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Criterio de éxito de la ficha remoto: el teléfono ve «apagado» un PC que dejó de correr qdd remoto. Del lado del PC
// eso es: al terminar, la conexión al relay se cierra (Supabase saca su presence, que es lo que la app lee) y no vuelve
// a latir ni a reconectarse. Cómo lo dibuja la app es de qdd-movile.

func TestCanal_AlApagarseElPCCierraLaConexionYNoVuelve(t *testing.T) {
	s := nuevoServidorFalso(t, "k", topicoPrueba)
	c, cancelar := canalDePrueba(t, s, "k")
	esperarEvento(t, c.Eventos(), 2*time.Second, conectado)
	if err := c.Track(context.Background(), PresenciaPC{Rol: "pc", VersionQDD: "v", Desde: "x"}); err != nil {
		t.Fatal(err)
	}
	s.Esperar(t, time.Second, func(m Mensaje) bool { return m.Event == "presence" })
	if s.Cerradas() != 0 {
		t.Fatal("encendido, la conexión sigue abierta")
	}
	cancelar()
	limite := time.Now().Add(2 * time.Second)
	for s.Cerradas() == 0 && time.Now().Before(limite) {
		time.Sleep(10 * time.Millisecond)
	}
	if s.Cerradas() != 1 {
		t.Fatal("al apagarse, el PC cierra su conexión: sin eso el relay lo sigue mostrando encendido")
	}
	latidos, joins := s.Latidos(), len(s.Joins())
	time.Sleep(300 * time.Millisecond) // más de 4 latidos (60 ms) y de la espera máxima para reconectar (80 ms)
	if s.Latidos() != latidos || len(s.Joins()) != joins {
		t.Fatalf("apagado no late ni se vuelve a unir: latidos %d → %d, joins %d → %d", latidos, s.Latidos(), joins, len(s.Joins()))
	}
}

func TestBucle_AlApagarseTerminaYNoDaMasLatidos(t *testing.T) {
	homeTemporal(t)
	tr, rl := nuevoTransporte(), nuevoRelayBucle()
	b := &Bucle{QDD: "v-prueba", Cred: Credenciales{Topico: topicoPrueba, Firma: "f"}, T: tr, R: rl,
		Agrupar: 10 * time.Millisecond, CadaLatido: 20 * time.Millisecond, CadaSondeo: 20 * time.Millisecond}
	ctx, cancelar := context.WithCancel(context.Background())
	fin := make(chan error, 1)
	go func() { fin <- b.Correr(ctx) }()
	limite := time.Now().Add(2 * time.Second)
	for len(rl.Latidos()) < 2 && time.Now().Before(limite) {
		time.Sleep(5 * time.Millisecond)
	}
	if len(rl.Latidos()) < 2 {
		t.Fatal("encendido, el PC late")
	}
	cancelar()
	select {
	case err := <-fin:
		if err != nil {
			t.Fatalf("apagarlo no es una falla: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("al apagarse, el bucle termina")
	}
	time.Sleep(50 * time.Millisecond) // un latido que ya iba en camino
	latidos := len(rl.Latidos())
	time.Sleep(150 * time.Millisecond)
	if len(rl.Latidos()) != latidos {
		t.Fatalf("apagado no da más latidos: %d → %d", latidos, len(rl.Latidos()))
	}
}
