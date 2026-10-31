package consola

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-089 (captura del usuario, 2026-10-31): un pedido quedó «ejecutando…» con el panel bloqueado. /reload-plugins
// mató el proceso que lo corría después del merge y antes del git push, y nadie anotó que murió: el panel lo mostró
// ejecutándose hasta el tope de 15 min, sin botón para reintentarlo.

// procesoMuerto devuelve el PID de un proceso que ya terminó.
func procesoMuerto(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("true")
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "exit 0")
	}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return cmd.Process.Pid
}

func colaConUnoEjecutando(t *testing.T, root string, pids ...int) {
	t.Helper()
	inicio := time.Now().Add(-time.Minute)
	p := Pedido{ID: "1", Comando: "git push origin v1", Motivo: "Publicar", Carpeta: root, Creado: inicio, Tecla: "a", Estado: EstadoEjecutando, Inicio: &inicio}
	if len(pids) > 0 {
		p.PID = pids[0]
	}
	if len(pids) > 1 {
		p.Hijo = pids[1]
	}
	ruta, _ := colaPath(root)
	os.MkdirAll(filepath.Dir(ruta), 0o700)
	if err := guardarCola(ruta, []Pedido{p}); err != nil {
		t.Fatal(err)
	}
}

func TestPedidos_SiMurioQuienLoEjecutabaQuedaParaReintentar_FND089(t *testing.T) {
	root := proyectoConCola(t)
	colaConUnoEjecutando(t, root, procesoMuerto(t), procesoMuerto(t))
	cola := Pedidos(root)
	if len(cola) != 1 || cola[0].Estado != EstadoFallo || !strings.Contains(cola[0].Salida, "se cortó") {
		t.Fatalf("sin proceso vivo, es una falla al instante: %+v", cola)
	}
	if TeclaDe(root, "1") != "a" {
		t.Fatal("con su tecla para ↻ Reintentar")
	}
}

func TestPedidos_MientrasSigueVivoSigueEjecutando_FND089(t *testing.T) {
	root := proyectoConCola(t)
	colaConUnoEjecutando(t, root, procesoMuerto(t), os.Getpid())
	if cola := Pedidos(root); cola[0].Estado != EstadoEjecutando {
		t.Fatalf("el comando sigue corriendo aunque qdd haya muerto: no se ofrece de nuevo (FND-070): %+v", cola)
	}
	colaConUnoEjecutando(t, root)
	if cola := Pedidos(root); cola[0].Estado != EstadoEjecutando {
		t.Fatalf("una cola sin PID (qdd anterior) espera el tope de siempre: %+v", cola)
	}
}

func TestPedidos_AlEjecutarseAnotaQuienLoCorre_FND089(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	root := proyectoConCola(t)
	ruta, _ := colaPath(root)
	p := mustPedir(t, root, "sleep 0.5; cat '"+ruta+"'", "Ver la cola", "")
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 0 {
		t.Fatalf("corre: %d %s", codigo, salida.String())
	}
	var vista []Pedido
	if err := json.Unmarshal([]byte(salida.String()), &vista); err != nil {
		t.Fatalf("la cola mientras corre: %v\n%s", err, salida.String())
	}
	if vista[0].PID != os.Getpid() || vista[0].Hijo == 0 {
		t.Fatalf("mientras corre, la cola dice qué proceso lo ejecuta y cuál es el comando: %+v", vista[0])
	}
}
