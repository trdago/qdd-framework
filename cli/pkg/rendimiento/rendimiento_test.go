package rendimiento

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-07): QDD trae una prueba de rendimiento estándar, la misma en cada equipo y sistema
// operativo, que no gasta tokens: cuánto tarda, cuánto consume y si deja procesos abiertos.

const varAyudante = "QDD_RENDIMIENTO_AYUDANTE"

// TestMain: con varAyudante, este binario de prueba hace de qdd. «sucio» deja un zombie y un huérfano.
func TestMain(m *testing.M) {
	modo := os.Getenv(varAyudante)
	if modo == "" {
		os.Exit(m.Run())
	}
	ayudante(modo, os.Args[1:])
}

func ayudante(modo string, args []string) {
	if len(args) > 0 && args[0] == "un-rato" {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}
	if len(args) > 0 && args[0] == "dormir" {
		time.Sleep(20 * time.Second)
		os.Exit(0)
	}
	if strings.Join(args, " ") == "consola seguir" || strings.HasPrefix(strings.Join(args, " "), "consola seguir ") {
		seguirFalso(modo)
	}
	fmt.Println("{}")
	os.Exit(0)
}

func seguirFalso(modo string) {
	if modo == "sucio" {
		huerfano := exec.Command(os.Args[0], "dormir")
		_ = huerfano.Start()
		zombie := exec.Command(os.Args[0])
		zombie.Env = append(os.Environ(), varAyudante+"=limpio")
		_ = zombie.Start()
	}
	for i := 0; ; i++ {
		if _, err := fmt.Println(`{"formato":1}`); err != nil {
			os.Exit(0)
		}
		time.Sleep(100 * time.Millisecond)
		if modo == "sucio" && i%3 == 0 {
			hijo := exec.Command(os.Args[0], "un-rato")
			_ = hijo.Start()
			go func() { _ = hijo.Wait() }()
		}
	}
}

func opcionesDePrueba(t *testing.T, modo string) Opciones {
	return Opciones{Bin: os.Args[0], Segundos: 2, Repeticiones: 3, Entorno: []string{varAyudante + "=" + modo}, Version: "v0.0.0-prueba"}
}

func medida(t *testing.T, inf Informe, nombre string) Medida {
	for _, m := range inf.Medidas {
		if m.Nombre == nombre {
			return m
		}
	}
	t.Fatalf("falta la medida %q en %+v", nombre, inf.Medidas)
	return Medida{}
}

func TestMedir_UnQddLimpioPasaYDiceDondeCorrio(t *testing.T) {
	inf, err := Medir(opcionesDePrueba(t, "limpio"))
	if err != nil {
		t.Fatal(err)
	}
	if inf.SO != runtime.GOOS || inf.Nucleos < 1 || inf.Version != "v0.0.0-prueba" {
		t.Fatalf("el informe dice dónde y con qué corrió: %+v", inf)
	}
	for _, nombre := range []string{MedidaZombies, MedidaHuerfanos} {
		m := medida(t, inf, nombre)
		if m.Estado == EstadoFalla {
			t.Fatalf("un qdd limpio no deja procesos: %+v", m)
		}
	}
	if m := medida(t, inf, "latencia_statusline_p95"); m.Valor == nil || *m.Valor <= 0 {
		t.Fatalf("mide la latencia de la barra: %+v", m)
	}
	if m := medida(t, inf, MedidaCPU); m.Valor == nil {
		t.Fatalf("mide la CPU de consola seguir: %+v", m)
	}
}

func TestMedir_DetectaElHuerfanoYLoLimpia(t *testing.T) {
	inf, err := Medir(opcionesDePrueba(t, "sucio"))
	if err != nil {
		t.Fatal(err)
	}
	h := medida(t, inf, MedidaHuerfanos)
	if h.Valor == nil || *h.Valor < 1 || h.Estado != EstadoFalla || !inf.Falla() {
		t.Fatalf("el proceso que quedó vivo es una falla: %+v", h)
	}
	if l := medida(t, inf, MedidaLanzados); l.Valor == nil || *l.Valor <= 0 {
		t.Fatalf("cuenta los procesos que lanzó: %+v", l)
	}
	if len(inf.Restos) == 0 {
		t.Fatalf("el informe nombra lo que quedó vivo: %+v", inf)
	}
	time.Sleep(200 * time.Millisecond)
	for _, p := range listarOFallar(t) {
		for _, r := range inf.Restos {
			if p.PID == r.PID && !p.Zombie && filepath.Base(p.Nombre) == filepath.Base(r.Nombre) {
				t.Fatalf("la medición no deja el huérfano corriendo en la máquina: %+v", p)
			}
		}
	}
}

func TestMedir_DetectaElZombie(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows no tiene procesos zombie")
	}
	inf, err := Medir(opcionesDePrueba(t, "sucio"))
	if err != nil {
		t.Fatal(err)
	}
	if z := medida(t, inf, MedidaZombies); z.Valor == nil || *z.Valor < 1 || z.Estado != EstadoFalla {
		t.Fatalf("el hijo que nadie recogió es una falla: %+v", z)
	}
}

func TestMedir_NoTocaElQddHomeDeLaMaquina(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	if _, err := Medir(opcionesDePrueba(t, "limpio")); err != nil {
		t.Fatal(err)
	}
	if entradas, _ := os.ReadDir(home); len(entradas) != 0 {
		t.Fatalf("medir corre en una carpeta temporal; el QDD_HOME quedó con %v", entradas)
	}
}

func listarOFallar(t *testing.T) []Proceso {
	l, err := Procesos()
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestProcesos_SeVeAsiMismoConSuPadre(t *testing.T) {
	for _, p := range listarOFallar(t) {
		if p.PID == os.Getpid() && p.PPID == os.Getppid() {
			return
		}
	}
	t.Fatal("la lista de procesos no tiene a este proceso con su padre")
}

func TestEvaluar_PresupuestoYSinMedir(t *testing.T) {
	v := func(f float64) *float64 { return &f }
	if m := Evaluar(MedidaCPU, v(0.1), "linux"); m.Estado != EstadoOK || m.Presupuesto <= 0 {
		t.Fatalf("dentro del presupuesto: %+v", m)
	}
	if m := Evaluar(MedidaCPU, v(99), "linux"); m.Estado != EstadoSobre {
		t.Fatalf("sobre el presupuesto: %+v", m)
	}
	if m := Evaluar(MedidaZombies, v(1), "linux"); m.Estado != EstadoFalla {
		t.Fatalf("un zombie es una falla, no un exceso: %+v", m)
	}
	if m := Evaluar(MedidaMemoria, nil, "windows"); m.Estado != EstadoSinMedir {
		t.Fatalf("lo que no se pudo medir queda sin medir, no en cero: %+v", m)
	}
	lin := Evaluar("latencia_statusline_p95", v(1), "linux").Presupuesto
	win := Evaluar("latencia_statusline_p95", v(1), "windows").Presupuesto
	if win <= lin {
		t.Fatalf("Windows lanza procesos más lento: su presupuesto es mayor (%v ≤ %v)", win, lin)
	}
}

func TestGuardar_UnInformePorVersionYSeLee(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if _, err := Ultimo("v1"); err == nil {
		t.Fatal("sin medir no hay informe")
	}
	cpu := 0.5
	inf := Informe{Version: "v1", SO: "linux", Cuando: time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC), Medidas: []Medida{Evaluar(MedidaCPU, &cpu, "linux")}}
	if err := Guardar(inf); err != nil {
		t.Fatal(err)
	}
	leido, err := Ultimo("v1")
	if err != nil || leido.Medidas[0].Estado != EstadoOK {
		t.Fatalf("se lee lo guardado: %+v %v", leido, err)
	}
	b, _ := json.Marshal(leido)
	if !strings.Contains(string(b), `"so":"linux"`) {
		t.Fatalf("JSON: %s", b)
	}
}
