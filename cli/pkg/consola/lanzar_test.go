package consola

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-089: el panel ya no espera el pedido. qdd lo lanza en un proceso aparte, en su propia sesión, y vuelve en el
// acto: recargar el panel o cerrar Claude Code no lo corta, el panel nunca queda bloqueado y el resultado llega por la
// cola.

// TestMain hace del binario de pruebas el «qdd consola correr» que lanza Lanzar.
func TestMain(m *testing.M) {
	if root, id, ok := EsCorrer(os.Args[1:]); ok {
		os.Exit(CorrerLanzado(root, id, os.Stdout))
	}
	os.Exit(m.Run())
}

func esperarEstado(t *testing.T, root, id, estado string) Pedido {
	t.Helper()
	tope := time.Now().Add(10 * time.Second)
	for time.Now().Before(tope) {
		for _, p := range Pedidos(root) {
			if p.ID == id && p.Estado == estado {
				return p
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("el pedido %s no llegó a %s: %+v", id, estado, Pedidos(root))
	return Pedido{}
}

func TestLanzar_VuelveEnElActoYElPedidoTerminaSolo_FND089(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	root := proyectoConCola(t)
	marca := filepath.Join(t.TempDir(), "corrio")
	p := mustPedir(t, root, "sleep 1; echo listo > '"+marca+"'; echo publicado", "Publicar", "")
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	var salida bytes.Buffer
	inicio := time.Now()
	if codigo := a.Lanzar(root, os.Args[0], &salida); codigo != 0 {
		t.Fatalf("lanza: %d %s", codigo, salida.String())
	}
	if time.Since(inicio) > 900*time.Millisecond {
		t.Fatal("Lanzar no espera al comando: el panel queda libre")
	}
	enCurso := esperarEstado(t, root, p.ID, EstadoEjecutando)
	if enCurso.PID == os.Getpid() || enCurso.PID == 0 {
		t.Fatalf("lo ejecuta otro proceso, no el que lo lanzó: %+v", enCurso)
	}
	if !separado(enCurso.PID) {
		t.Fatal("el proceso que lo corre está en su propia sesión: cerrar el panel no lo corta")
	}
	if TeclaDe(root, p.ID) != "" {
		t.Fatal("mientras corre no se ofrece de nuevo (FND-070)")
	}
	hecho := esperarEstado(t, root, p.ID, EstadoHecho)
	if !strings.Contains(hecho.Salida, "publicado") || hecho.PID != 0 {
		t.Fatalf("el resultado llega por la cola: %+v", hecho)
	}
	if _, err := os.Stat(marca); err != nil {
		t.Fatal("el comando corrió completo")
	}
}

func TestLanzar_UnPedidoNoEjecutableNoSeLanza_FND089(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 0", "m", "")
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	if codigo := a.Ejecutar(root, "qdd", &bytes.Buffer{}); codigo != 0 {
		t.Fatal("corre")
	}
	var salida bytes.Buffer
	if codigo := a.Lanzar(root, os.Args[0], &salida); codigo == 0 || !strings.Contains(salida.String(), "ya se ejecutó") {
		t.Fatalf("uno hecho no se lanza de nuevo: %d %q", codigo, salida.String())
	}
}

func TestCorrerLanzado_SoloCorreUnoMarcadoYUnaVez_FND089(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 0", "m", "")
	var salida bytes.Buffer
	if codigo := CorrerLanzado(root, p.ID, &salida); codigo == 0 {
		t.Fatal("un pedido pendiente no se corre sin que alguien lo haya lanzado")
	}
	if err := marcarEjecutando(root, p.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if codigo := CorrerLanzado(root, p.ID, &salida); codigo != 0 {
		t.Fatalf("el lanzado corre: %d %s", codigo, salida.String())
	}
	if codigo := CorrerLanzado(root, p.ID, &salida); codigo == 0 {
		t.Fatal("y no corre dos veces")
	}
}
