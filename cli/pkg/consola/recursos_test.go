package consola

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func f(v float64) *float64 { return &v }

func TestLineaRecursos_Completa_CCC_C14(t *testing.T) {
	m := Medidas{Disco: f(69.6), GPUs: []*float64{f(7), nil}, RAM: f(38.2), CPU: f(12.4)}
	if got := LineaRecursos(m); got != "D 70% · G0 7% · G1 — · R 38% · C 12%" {
		t.Fatalf("línea de recursos: %q", got)
	}
}

func TestLineaRecursos_LoQueFaltaSeOmite_CCC_C15(t *testing.T) {
	if got := LineaRecursos(Medidas{Disco: f(70), RAM: f(38)}); got != "D 70% · R 38%" {
		t.Fatalf("sin GPU ni CPU se omiten, sin ceros inventados: %q", got)
	}
	if got := LineaRecursos(Medidas{}); got != "" {
		t.Fatalf("sin medidas no hay línea: %q", got)
	}
}

func TestParseMeminfo(t *testing.T) {
	in := "MemTotal:       40000000 kB\nMemFree:         2600000 kB\nMemAvailable:   25000000 kB\n"
	p, ok := parseMeminfo(in)
	if !ok || p < 37.49 || p > 37.51 {
		t.Fatalf("RAM usada = (total − disponible) / total = 37,5 %%: %v %v", p, ok)
	}
	if _, ok := parseMeminfo("MemTotal: 0 kB\n"); ok {
		t.Fatal("sin MemAvailable no se inventa la RAM")
	}
}

func TestParseProcStatYUsoCPU(t *testing.T) {
	a, ok := parseProcStat("cpu  100 0 100 800 0 0 0 0 0 0\ncpu0 1 2 3\n")
	if !ok || a.Total != 1000 || a.Ocioso != 800 {
		t.Fatalf("muestra: %+v %v", a, ok)
	}
	b, _ := parseProcStat("cpu  150 0 150 900 0 0 0 0 0 0\n")
	p, ok := usoCPU(a, b)
	if !ok || p < 49.999 || p > 50.001 {
		t.Fatalf("CPU usada entre dos lecturas = 1 − Δocioso/Δtotal = 50 %%: %v %v", p, ok)
	}
	if _, ok := usoCPU(b, b); ok {
		t.Fatal("sin tiempo transcurrido no hay lectura")
	}
}

func TestParseNvidia(t *testing.T) {
	got := parseNvidia("7\n 63 \n[N/A]\n")
	if len(got) != 3 || *got[0] != 7 || *got[1] != 63 || got[2] != nil {
		t.Fatalf("una GPU por línea; [N/A] queda sin lectura: %v", got)
	}
}

func TestMedirGPUs_CacheCadaCincoSegundos_CCC_C16(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "recursos.json")
	llamadas := 0
	nvidia := func() (string, error) {
		llamadas++
		return "7\n", nil
	}
	t0 := time.Date(2026, 10, 29, 20, 0, 0, 0, time.UTC)
	s := sensores{nvidia: nvidia, otras: func() []*float64 { return []*float64{nil} }}
	for i, ahora := range []time.Time{t0, t0.Add(2 * time.Second), t0.Add(4 * time.Second), t0.Add(6 * time.Second)} {
		m := s.medir(t.TempDir(), cache, ahora)
		if len(m.GPUs) != 2 || m.GPUs[0] == nil || *m.GPUs[0] != 7 || m.GPUs[1] != nil {
			t.Fatalf("lectura %d: NVIDIA primero y la integrada sin lectura: %v", i, m.GPUs)
		}
	}
	if llamadas != 2 {
		t.Fatalf("nvidia-smi se consulta como mucho cada 5 s: %d llamadas en 6 s", llamadas)
	}
}

func TestMedirGPUs_SinNvidiaSmiSeOmite_CCC_C15(t *testing.T) {
	s := sensores{nvidia: func() (string, error) { return "", errors.New("no existe") }, otras: func() []*float64 { return nil }}
	m := s.medir(t.TempDir(), filepath.Join(t.TempDir(), "r.json"), time.Now())
	if len(m.GPUs) != 0 {
		t.Fatalf("sin nvidia-smi ni otras GPU no hay G: %v", m.GPUs)
	}
}

func TestMedirCPU_PrimeraLecturaSeOmiteYLaSegundaCompara_CCC_C15(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "recursos.json")
	muestras := []muestraCPU{{Total: 1000, Ocioso: 800}, {Total: 1100, Ocioso: 870}}
	i := 0
	s := sensores{cpu: func() (muestraCPU, bool) { m := muestras[i]; i++; return m, true }}
	t0 := time.Now()
	if m := s.medir(t.TempDir(), cache, t0); m.CPU != nil {
		t.Fatalf("la primera lectura no tiene con qué comparar: %v", *m.CPU)
	}
	m := s.medir(t.TempDir(), cache, t0.Add(2*time.Second))
	if m.CPU == nil || *m.CPU < 29.999 || *m.CPU > 30.001 {
		t.Fatalf("segunda lectura: 1 − 70/100 = 30 %%: %v", m.CPU)
	}
}

func TestMedirCPU_MuestraViejaNoSeUsa(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "recursos.json")
	muestras := []muestraCPU{{Total: 1000, Ocioso: 800}, {Total: 2000, Ocioso: 1000}}
	i := 0
	s := sensores{cpu: func() (muestraCPU, bool) { m := muestras[i]; i++; return m, true }}
	t0 := time.Now()
	s.medir(t.TempDir(), cache, t0)
	if m := s.medir(t.TempDir(), cache, t0.Add(10*time.Minute)); m.CPU != nil {
		t.Fatal("con una muestra de hace 10 min el promedio no es el uso de ahora: se omite")
	}
}

func TestBarraConRecursos_SegundaLinea(t *testing.T) {
	dir := t.TempDir()
	in := `{"workspace":{"project_dir":"` + filepath.ToSlash(dir) + `"},"model":{"display_name":"Opus 5.5"}}`
	s := sensores{disco: func(string) (float64, bool) { return 70, true }, ram: func() (float64, bool) { return 38, true }}
	got := barraCon([]byte(in), time.Now(), false, &s, filepath.Join(dir, "r.json"))
	lineas := strings.Split(got, "\n")
	if len(lineas) != 2 || lineas[1] != "D 70% · R 38%" || !strings.HasPrefix(lineas[0], "📁 ") {
		t.Fatalf("barra con recursos: %q", got)
	}
	if got := barraCon([]byte(in), time.Now(), false, nil, ""); strings.Contains(got, "\n") {
		t.Fatalf("sin recursos la barra es de una línea: %q", got)
	}
}

func TestDiscoDelProyecto(t *testing.T) {
	p, ok := usoDisco(os.TempDir())
	if !ok {
		t.Skip("este sistema no expone el uso de disco")
	}
	if p < 0 || p > 100 {
		t.Fatalf("uso de disco fuera de rango: %v", p)
	}
}

func TestActivar_RecursosSeRecuerdaYReparaLosConserva(t *testing.T) {
	c := nuevaConsola(t, "2.4.0", &claudeFalso{})
	si := true
	c.Recursos = &si
	if _, err := c.Activar(true); err != nil {
		t.Fatal(err)
	}
	reg, ok := c.LeerRegistro()
	if !ok || !reg.Recursos {
		t.Fatalf("--recursos queda en activo.json: %+v", reg)
	}
	c.Recursos = nil
	if _, err := c.Reparar(); err != nil {
		t.Fatal(err)
	}
	if reg, _ := c.LeerRegistro(); !reg.Recursos {
		t.Fatal("reparar (doctor --fix, actualizar) conserva la elección")
	}
	no := false
	c.Recursos = &no
	if _, err := c.Activar(true); err != nil {
		t.Fatal(err)
	}
	if reg, _ := c.LeerRegistro(); reg.Recursos {
		t.Fatal("--sin-recursos la quita")
	}
}
