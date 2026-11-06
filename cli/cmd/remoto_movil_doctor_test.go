package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'

// El 2026-11-06, qdd doctor dio 100 % mientras la persona decía que QDD Móvil no veía el proyecto: no miraba si el PC
// estaba emparejado, si el servicio corría, si llegaba al relay ni si el proyecto se publicaba.

type movilDePrueba struct {
	raiz     string
	dirHome  string
	corridos *[]string
}

func movilPrueba(t *testing.T) movilDePrueba {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	dir := t.TempDir()
	antesServicio, antesEjecutor := remotoServicio, remotoEjecutor
	t.Cleanup(func() { remotoServicio, remotoEjecutor = antesServicio, antesEjecutor })
	remotoServicio = func() (remoto.Servicio, error) {
		return remoto.Servicio{SO: "linux", Bin: "/opt/qdd", Usuario: dir, DirQDD: dir}, nil
	}
	corridos := []string{}
	remotoEjecutor = func(n string, a ...string) error {
		corridos = append(corridos, n+" "+strings.Join(a, " "))
		return nil
	}
	return movilDePrueba{raiz: t.TempDir(), dirHome: dir, corridos: &corridos}
}

func emparejarPrueba(t *testing.T) {
	t.Helper()
	if err := remoto.GuardarCredenciales(remoto.Credenciales{Formato: 1, Maquina: "m", Nombre: "n", Topico: "qdd:pc:t", Firma: "f"}); err != nil {
		t.Fatal(err)
	}
}

func servicioCorriendo(t *testing.T) {
	t.Helper()
	dir, err := remoto.DirRemoto()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "correr.pid"), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
}

func itemMovil(t *testing.T, g *CheckGroup, nombre string) *CheckItem {
	t.Helper()
	for _, i := range g.Items {
		if i.Name == nombre {
			return i
		}
	}
	t.Fatalf("falta %q en %+v", nombre, g.Items)
	return nil
}

func movilSano(t *testing.T, m movilDePrueba, now time.Time) {
	t.Helper()
	emparejarPrueba(t)
	servicioCorriendo(t)
	if err := remoto.GuardarPlan(remoto.PlanGratis(), now); err != nil {
		t.Fatal(err)
	}
	if err := registro.Anotar(m.raiz, now); err != nil {
		t.Fatal(err)
	}
}

func TestDoctorMovil_SinQDDMovilNoReportaNada(t *testing.T) {
	m := movilPrueba(t)
	if g := buildMovilChecks(m.raiz, time.Now()); len(g.Items) != 0 {
		t.Fatalf("sin activar no opina: %+v", g.Items)
	}
}

func TestDoctorMovil_TodoSanoEnVerde(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	g := buildMovilChecks(m.raiz, now)
	if len(g.Items) != 5 {
		t.Fatalf("emparejado, servicio, directorio, relay y proyecto: %+v", g.Items)
	}
	for _, i := range g.Items {
		if !i.Success {
			t.Fatalf("todo sano: %+v", i)
		}
	}
}

func TestDoctorMovil_ServicioInstaladoSinEmparejarPideLoginYNoLoIntenta(t *testing.T) {
	m := movilPrueba(t)
	s, _ := remotoServicio()
	if _, err := s.Instalar(remotoEjecutor); err != nil {
		t.Fatal(err)
	}
	*m.corridos = nil
	g := buildMovilChecks(m.raiz, time.Now())
	item := itemMovil(t, g, itemMovilEmparejado)
	if item.Success || !strings.Contains(item.Error, "qdd login") {
		t.Fatalf("%+v", item)
	}
	repairMovil(m.raiz, g, time.Now())
	if len(*m.corridos) != 0 {
		t.Fatalf("emparejar pide el teléfono de la persona: %v", *m.corridos)
	}
}

func TestDoctorMovil_ServicioQueNoCorreSeArranca(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	emparejarPrueba(t)
	if err := registro.Anotar(m.raiz, now); err != nil {
		t.Fatal(err)
	}
	s, _ := remotoServicio()
	if _, err := s.Instalar(remotoEjecutor); err != nil {
		t.Fatal(err)
	}
	*m.corridos = nil
	g := buildMovilChecks(m.raiz, now)
	if item := itemMovil(t, g, itemMovilCorre); item.Success {
		t.Fatalf("no corre: %+v", item)
	}
	repairMovil(m.raiz, g, now)
	if len(*m.corridos) == 0 || !strings.Contains(strings.Join(*m.corridos, "\n"), "qdd-remoto") {
		t.Fatalf("lo arranca: %v", *m.corridos)
	}
}

func TestDoctorMovil_EmparejadoSinServicioLoInstala(t *testing.T) {
	m := movilPrueba(t)
	emparejarPrueba(t)
	g := buildMovilChecks(m.raiz, time.Now())
	repairMovil(m.raiz, g, time.Now())
	s, _ := remotoServicio()
	if instalado, _, err := s.DejaVivoLoLanzado(); err != nil || !instalado {
		t.Fatalf("lo deja instalado: %v %v", instalado, err)
	}
}

func TestDoctorMovil_ConElServicioCorriendoNoLoToca(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	if len(*m.corridos) != 0 {
		t.Fatalf("no reinicia lo que funciona: %v", *m.corridos)
	}
}

func TestDoctorMovil_UnDirectorioQueNoRespondeSeDiceSinUsarLaRed_REM11(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	c := remoto.ConfigRelay{URL: "https://guardado.supabase.co", Clave: "sb_publishable_g", Serie: 1}
	if err := remoto.GuardarRelay(c); err != nil {
		t.Fatal(err)
	}
	if _, _, err := remoto.ActualizarRelay(context.Background(), func(context.Context, int) (remoto.ConfigRelay, error) {
		return remoto.ConfigRelay{}, errors.New("la firma del directorio no es válida")
	}, c); err == nil {
		t.Fatal("la consulta debía fallar")
	}
	item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilDirectorio)
	if item.Success || !strings.Contains(item.Error, "sigue con el relay guardado") || !strings.Contains(item.Error, "firma") {
		t.Fatalf("dice que sigue y por qué: %+v", item)
	}
}

func TestDoctorMovil_UnRelayPropioNoMiraElDirectorio_REM11(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	if err := remoto.GuardarRelay(remoto.ConfigRelay{URL: "https://mio.supabase.co", Clave: "sb_publishable_mio", Propio: true}); err != nil {
		t.Fatal(err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilDirectorio); !item.Success {
		t.Fatalf("un relay propio no consulta: %+v", item)
	}
}

func TestDoctorMovil_LatidoViejoFalla(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now.Add(-10*time.Minute))
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilRelay); item.Success || !strings.Contains(item.Error, "10 min") {
		t.Fatalf("dice hace cuánto: %+v", item)
	}
}

// FND-169: plan.json se queda quieto si el plan no cambia, y eso no es falta de respuesta del relay.
func TestDoctorMovil_PlanViejoConLatidoRecienteNoAlarma_FND169(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now.Add(-10*time.Minute))
	if err := remoto.GuardarLatido(now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilRelay); !item.Success {
		t.Fatalf("el PC late bien: %+v", item)
	}
}

func TestDoctorMovil_ProyectoSinAnotarSeAnota(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	emparejarPrueba(t)
	servicioCorriendo(t)
	if err := remoto.GuardarPlan(remoto.PlanGratis(), now); err != nil {
		t.Fatal(err)
	}
	g := buildMovilChecks(m.raiz, now)
	if item := itemMovil(t, g, itemMovilProyecto); item.Success {
		t.Fatalf("no se publica: %+v", item)
	}
	repairMovil(m.raiz, g, now)
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilProyecto); !item.Success {
		t.Fatalf("quedó anotado: %+v", item)
	}
}

func TestDoctorMovil_ProyectoFueraDelPlanLoDiceYNoLoRepara(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now.Add(-time.Minute))
	limite := 1
	if err := remoto.GuardarPlan(remoto.Plan{Proyectos: &limite}, now); err != nil {
		t.Fatal(err)
	}
	if err := registro.Anotar(t.TempDir(), now); err != nil {
		t.Fatal(err)
	}
	g := buildMovilChecks(m.raiz, now)
	item := itemMovil(t, g, itemMovilProyecto)
	if item.Success || !strings.Contains(item.Error, "plan") {
		t.Fatalf("el más viejo queda fuera del plan: %+v", item)
	}
}

func TestDoctorMovil_UnEnvioFallidoFallaYSeQuitaAlVolverAMandar(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	if err := remoto.GuardarPublicacion(false, now); err != nil {
		t.Fatal(err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilEnvio); item.Success {
		t.Fatalf("%+v", item)
	}
	if err := remoto.GuardarPublicacion(true, now); err != nil {
		t.Fatal(err)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilEnvio); !item.Success {
		t.Fatalf("%+v", item)
	}
}

func TestDoctorMovil_UnaColaApartadaFallaYNoSeTocaSola(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	ruta, err := consola.RutaLocal(m.raiz, "telefono")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	apartada := ruta + ".ilegible"
	if err := os.WriteFile(apartada, []byte("{roto"), 0o600); err != nil {
		t.Fatal(err)
	}
	g := buildMovilChecks(m.raiz, now)
	item := itemMovil(t, g, itemMovilColas)
	if item.Success || !strings.Contains(item.Error, apartada) {
		t.Fatalf("dice cuál: %+v", item)
	}
	repairMovil(m.raiz, g, now)
	if _, err := os.Stat(apartada); err != nil {
		t.Fatalf("puede traer un mensaje de la persona: no se borra: %v", err)
	}
}

func TestDoctorFlujos_SinFlujosNoReportaYUnoRotoSeNombra(t *testing.T) {
	root := t.TempDir()
	if g := buildFlujosChecks(root); len(g.Items) != 0 {
		t.Fatalf("sin flujos no opina: %+v", g.Items)
	}
	dir := filepath.Join(root, ".qdd", "project", "flujos")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "roto.yaml"), []byte("id: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := buildFlujosChecks(root)
	if len(g.Items) != 1 || g.Items[0].Success || !strings.Contains(g.Items[0].Error, "roto") {
		t.Fatalf("%+v", g.Items)
	}
}
