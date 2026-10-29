package dashboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/certificaciones"
)

// proyectoMedido crea un proyecto con un puntaje medido con la escala actual,
// que es cuando BuildState escribía el historial (FND-058).
func proyectoMedido(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	escribirHistorial(t, dir, `[{"timestamp":"2026-10-28T10:00:00Z","score":90,"escala":2}]`)
	os.MkdirAll(filepath.Join(dir, ".qdd", "project", "findings"), 0755)
	old, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(old) })
	return dir
}

func mensajesEn(ch chan []byte, ventana time.Duration) int {
	n := 0
	limite := time.After(ventana)
	for {
		select {
		case <-ch:
			n++
		case <-limite:
			return n
		}
	}
}

// FND-058: el dashboard escribía dentro de lo que observaba y se reconstruía
// cada 500 ms aunque nadie cambiara nada.
func TestVigilar_ProyectoQuietoNoReconstruye(t *testing.T) {
	dir := proyectoMedido(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listo := make(chan struct{})
	go Vigilar(ctx, dir, listo)
	<-listo
	ch := Broker.AddClient()
	defer Broker.RemoveClient(ch)
	mensajesEn(ch, 200*time.Millisecond) // estado inicial

	if n := mensajesEn(ch, 3*time.Second); n != 0 {
		t.Errorf("con el proyecto quieto hubo %d reconstrucciones en 3 s; want 0", n)
	}
}

// Dirigido por eventos: un cambio real en una fuente se difunde una sola vez,
// y lo que escribe el propio dashboard o un editor no cuenta como evento.
func TestVigilar_ReaccionaSoloAEventosReales(t *testing.T) {
	dir := proyectoMedido(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listo := make(chan struct{})
	go Vigilar(ctx, dir, listo)
	<-listo
	ch := Broker.AddClient()
	defer Broker.RemoveClient(ch)
	mensajesEn(ch, 200*time.Millisecond)

	qdd := filepath.Join(dir, ".qdd")
	os.WriteFile(filepath.Join(qdd, "knowledge.db-journal"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(qdd, "project", "metrics", "algo.json.tmp"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(qdd, "project", "findings", ".FND-1.yaml.swp"), []byte("x"), 0644)
	if n := mensajesEn(ch, 2*time.Second); n != 0 {
		t.Errorf("archivos propios o temporales dispararon %d reconstrucciones; want 0", n)
	}

	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(qdd, "project", "findings", "FND-001.yaml"), []byte("status: open\n"), 0644)
	}
	if n := mensajesEn(ch, 2500*time.Millisecond); n != 1 {
		t.Errorf("una ráfaga de cambios reales produjo %d difusiones; want 1", n)
	}
}

// Una carpeta creada después de abrir el dashboard también se observa.
func TestVigilar_ObservaCarpetasNuevas(t *testing.T) {
	dir := proyectoMedido(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listo := make(chan struct{})
	go Vigilar(ctx, dir, listo)
	<-listo
	ch := Broker.AddClient()
	defer Broker.RemoveClient(ch)
	mensajesEn(ch, 200*time.Millisecond)

	modulos := filepath.Join(dir, ".qdd", "project", "modules")
	os.MkdirAll(modulos, 0755)
	mensajesEn(ch, 2500*time.Millisecond) // la creación de la carpeta es un evento
	os.WriteFile(filepath.Join(modulos, "pagos.yaml"), []byte("id: pagos\n"), 0644)
	if n := mensajesEn(ch, 2500*time.Millisecond); n != 1 {
		t.Errorf("un cambio en una carpeta nueva produjo %d difusiones; want 1", n)
	}
}

// FND-058: BuildState solo lee. Antes escribía cognitive_history.json, que el
// watcher observa, en cada reconstrucción.
func TestBuildState_NoEscribeHistoriales(t *testing.T) {
	dir := proyectoMedido(t)
	BuildState()
	if _, err := os.Stat(filepath.Join(dir, ".qdd", "project", "metrics", "cognitive_history.json")); err == nil {
		t.Error("BuildState escribió cognitive_history.json")
	}
}

// La tendencia sale de las certificaciones reales: el último puntaje de cada
// día con la escala actual. cognitive_history.json mezcla eventos del MCP y del
// benchmark, sin puntaje, que aparecían como puntos en 0 sin fecha.
func TestLoadHistoricalTrends_DesdeCertificaciones(t *testing.T) {
	dir := t.TempDir()
	escribirHistorial(t, dir, `[
		{"timestamp":"2026-10-26T09:00:00Z","score":100},
		{"timestamp":"2026-10-27T09:00:00Z","score":60,"escala":2},
		{"timestamp":"2026-10-27T18:00:00Z","score":70,"escala":2},
		{"timestamp":"2026-10-28T10:00:00Z","score":90,"escala":2}
	]`)
	metrics := filepath.Join(dir, ".qdd", "project", "metrics", "cognitive_history.json")
	os.WriteFile(metrics, []byte(`[{"timestamp":"2026-10-28T11:00:00Z","tool":"qdd_audit","status":"ok"}]`), 0644)

	var s QDDState
	loadHistoricalTrends(&s, dir)
	want := []HistoricalTrendPoint{{"2026-10-27", 70}, {"2026-10-28", 90}}
	if len(s.Historical) != len(want) {
		t.Fatalf("tendencia = %v; want %v", s.Historical, want)
	}
	for i := range want {
		if s.Historical[i] != want[i] {
			t.Errorf("punto %d = %v; want %v", i, s.Historical[i], want[i])
		}
	}
}

// Un navegador que se conecta recibe el último estado difundido sin que el
// servidor reconstruya otra vez.
func TestBroker_UltimoEstado(t *testing.T) {
	b := &StateBroker{clients: make(map[chan []byte]bool)}
	if _, ok := b.Ultimo(); ok {
		t.Error("un broker sin difusiones no tiene estado")
	}
	b.Broadcast([]byte(`{"score":90}`))
	if data, ok := b.Ultimo(); !ok || string(data) != `{"score":90}` {
		t.Errorf("Ultimo = %q, %v", data, ok)
	}
}

// FND-055: el dashboard inventaba un historial para cada certificación, horas
// ahorradas con constantes y la misma descripción para todo finding.
func TestBuildState_SinDatosSimulados(t *testing.T) {
	dir := proyectoMedido(t)
	os.WriteFile(filepath.Join(dir, ".qdd", "project", "findings", "FND-001.yaml"),
		[]byte("title: Título real\ndescription: Qué pasó de verdad\nstatus: resolved\n"), 0644)
	cert := filepath.Join(dir, ".qdd", "project", "certification")
	os.MkdirAll(cert, 0755)
	os.WriteFile(filepath.Join(cert, "CERT-900.yaml"), []byte("id: CERT-900\nname: Mi certificación\nstatus: certified\n"), 0644)

	s := BuildState()
	for _, c := range s.Certifications {
		if len(c.History) != 0 {
			t.Errorf("certificación %s con historial sin fuente: %v", c.ID, c.History)
		}
	}
	if len(s.Findings) != 1 || s.Findings[0].Desc != "Qué pasó de verdad" {
		t.Errorf("descripción del finding = %+v; want la del archivo", s.Findings)
	}
	if s.ValueMetrics.DebtReduced != 1 {
		t.Errorf("deuda reducida = %d; want 1 (findings resueltos)", s.ValueMetrics.DebtReduced)
	}
}

// FND-060: el dashboard mostraba en PASS toda certificación que no decía su
// estado, incluido el catálogo core, y les ponía un nombre y un tipo fijos.
func TestBuildState_CertificacionesConSuEstadoReal(t *testing.T) {
	dir := proyectoMedido(t)
	for carpeta, archivos := range map[string]map[string]string{
		"core": {
			"CERT-CAT.yaml": "id: CERT-CAT\ntitle: Estándar del catálogo\nstatus: certified\n",
		},
		"project": {
			"CERT-OK.yaml":  "id: CERT-OK\nname: Adoptada y cumplida\nstatus: PASSED\n",
			"CERT-SIN.yaml": "id: CERT-SIN\ntitle: Adoptada sin estado\n",
		},
	} {
		base := filepath.Join(dir, ".qdd", carpeta, "certification")
		os.MkdirAll(base, 0755)
		for nombre, contenido := range archivos {
			os.WriteFile(filepath.Join(base, nombre), []byte(contenido), 0644)
		}
	}

	s := BuildState()
	por := map[string]DashboardCertification{}
	for _, c := range s.Certifications {
		por[c.ID] = c
	}
	casos := []struct {
		id, estado, nombre, tipo string
		core                     bool
	}{
		{"CERT-CAT", "DISPONIBLE", "Estándar del catálogo", "Catálogo", true},
		{"CERT-OK", "CUMPLE", "Adoptada y cumplida", "Proyecto", false},
		{"CERT-SIN", "SIN ESTADO", "Adoptada sin estado", "Proyecto", false},
	}
	for _, c := range casos {
		got := por[c.id]
		if got.Status != c.estado || got.Name != c.nombre || got.Type != c.tipo || got.IsCore != c.core {
			t.Errorf("%s = %+v; want estado %q, nombre %q, tipo %q, core %v", c.id, got, c.estado, c.nombre, c.tipo, c.core)
		}
	}
	want := certificaciones.Conteo{Cumplen: 1, SinEstado: 1, Disponibles: 1}
	if s.CertificacionesConteo != want {
		t.Errorf("conteo = %+v; want %+v", s.CertificacionesConteo, want)
	}
}
