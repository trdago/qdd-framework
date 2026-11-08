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
	antesServicio, antesEjecutor, antesConsultar := remotoServicio, remotoEjecutor, remotoConsultarRelay
	t.Cleanup(func() {
		remotoServicio, remotoEjecutor, remotoConsultarRelay = antesServicio, antesEjecutor, antesConsultar
	})
	remotoConsultarRelay = func(context.Context, int) (remoto.ConfigRelay, error) {
		return remoto.ConfigRelay{}, errors.New("la prueba no da directorio")
	}
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
	confirmado := directorioPrueba(new(int), relayAntiguoPrueba)
	if _, _, err := remoto.ActualizarRelay(context.Background(), confirmado, relayAntiguoPrueba); err != nil {
		t.Fatal(err)
	}
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
	if item := itemDeGrupo(g, itemFlujosValidos); item == nil || item.Success || !strings.Contains(item.Error, "roto") {
		t.Fatalf("%+v", g.Items)
	}
}

// FND-171: un PC con un relay antiguo guardado daba un código que la app no acepta, y doctor no lo veía: REM-11 solo salía
// con el servicio corriendo, daba verde con un relay propio y --fix no lo reparaba.

func guardarRelayPrueba(t *testing.T, c remoto.ConfigRelay) {
	t.Helper()
	if err := remoto.GuardarRelay(c); err != nil {
		t.Fatal(err)
	}
}

func directorioPrueba(consultas *int, c remoto.ConfigRelay) func(context.Context, int) (remoto.ConfigRelay, error) {
	return func(context.Context, int) (remoto.ConfigRelay, error) {
		*consultas++
		return c, nil
	}
}

var (
	relayAntiguoPrueba = remoto.ConfigRelay{URL: "https://antiguo.supabase.co", Clave: "sb_publishable_a"}
	relayNuevoPrueba   = remoto.ConfigRelay{URL: "https://nuevo.supabase.co", Clave: "sb_publishable_n", Serie: 2}
)

func propioPrueba() remoto.ConfigRelay {
	c := relayAntiguoPrueba
	c.Propio = true
	return c
}

func TestDoctorMovil_SinRelayGuardadoNoDiceNada_FND171(t *testing.T) {
	m := movilPrueba(t)
	if g := buildMovilChecks(m.raiz, time.Now()); len(g.Items) != 0 {
		t.Fatalf("un PC que nunca activó QDD Móvil no tiene nada que revisar: %+v", g.Items)
	}
}

func TestDoctorMovil_UnRelayPropioSinEmparejarSeVeYNoUsaLaRed_FND171(t *testing.T) {
	m := movilPrueba(t)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	guardarRelayPrueba(t, propioPrueba())
	item := itemMovil(t, buildMovilChecks(m.raiz, time.Now()), itemMovilDirectorio)
	if item.Success || !strings.Contains(item.Error, "propio") || !strings.Contains(item.Error, "qdd doctor --fix") {
		t.Fatalf("debe decirlo y cómo se repara: %+v", item)
	}
	if strings.Contains(item.Error, "antiguo.supabase.co") {
		t.Fatalf("el reporte no lleva el relay: %+v", item)
	}
	if consultas != 0 {
		t.Fatalf("sin --fix no usa la red: %d consultas", consultas)
	}
}

func TestDoctorMovil_UnRelayNuncaConfirmadoSeVe_FND171(t *testing.T) {
	m := movilPrueba(t)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	item := itemMovil(t, buildMovilChecks(m.raiz, time.Now()), itemMovilDirectorio)
	if item.Success || !strings.Contains(item.Error, "qdd doctor --fix") {
		t.Fatalf("nada lo confirmó: %+v", item)
	}
}

func TestDoctorMovil_ElDirectorioSeRevisaConElServicioApagado_FND171(t *testing.T) {
	m := movilPrueba(t)
	emparejarPrueba(t)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	item := itemMovil(t, buildMovilChecks(m.raiz, time.Now()), itemMovilDirectorio)
	if item.Success {
		t.Fatalf("con el servicio apagado tampoco se confirmó: %+v", item)
	}
}

func TestDoctorMovil_FixReemplazaUnRelayPropioSinEmparejar_FND171(t *testing.T) {
	m := movilPrueba(t)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	guardarRelayPrueba(t, propioPrueba())
	now := time.Now()
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	c, err := remoto.LeerRelay()
	if err != nil || c.URL != relayNuevoPrueba.URL || c.Propio {
		t.Fatalf("queda el del directorio: %+v %v", c, err)
	}
	if consultas != 1 {
		t.Fatalf("--fix consulta una vez: %d", consultas)
	}
	if item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilDirectorio); !item.Success {
		t.Fatalf("después de repararlo queda en verde: %+v", item)
	}
}

func TestDoctorMovil_FixNuncaTocaUnRelayPropioEmparejado_FND171(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	guardarRelayPrueba(t, propioPrueba())
	g := buildMovilChecks(m.raiz, now)
	if item := itemMovil(t, g, itemMovilDirectorio); !item.Success {
		t.Fatalf("un PC emparejado en su relay propio está bien: %+v", item)
	}
	repairMovil(m.raiz, g, now)
	c, err := remoto.LeerRelay()
	if err != nil || c != propioPrueba() || consultas != 0 {
		t.Fatalf("no se toca ni se consulta: %+v %v, %d consultas", c, err, consultas)
	}
}

func TestDoctorMovil_FixSiNoPuedeConsultarDejaTodoYSigueEnRojo_FND171(t *testing.T) {
	m := movilPrueba(t)
	guardarRelayPrueba(t, propioPrueba())
	now := time.Now()
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	c, err := remoto.LeerRelay()
	if err != nil || c != propioPrueba() {
		t.Fatalf("no se pierde lo guardado: %+v %v", c, err)
	}
	item := itemMovil(t, buildMovilChecks(m.raiz, now), itemMovilDirectorio)
	if item.Success {
		t.Fatalf("sigue sin confirmarse: %+v", item)
	}
}

func TestDoctorMovil_FixConElServicioCorriendoLoReiniciaSiCambiaElRelay_FND171(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	s, _ := remotoServicio()
	if _, err := s.Instalar(remotoEjecutor); err != nil {
		t.Fatal(err)
	}
	*m.corridos = nil
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	if !strings.Contains(strings.Join(*m.corridos, "\n"), "restart") {
		t.Fatalf("el servicio sigue conectado al relay anterior: hay que reiniciarlo: %v", *m.corridos)
	}
}

func TestDoctorMovil_FixConQddRemotoAManoDiceQueHayQueReiniciarlo_FND171(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	hecho, err := repararRelayMovil()
	if len(hecho) == 0 || err == nil || !strings.Contains(err.Error(), "qdd remoto correr") {
		t.Fatalf("no instala un servicio que nadie pidió, y lo dice: %v %v", hecho, err)
	}
	if len(*m.corridos) != 0 {
		t.Fatalf("no corre nada: %v", *m.corridos)
	}
}

// El servicio solo lee el relay al arrancar: si --fix lo arranca primero, queda en el relay anterior.
func TestDoctorMovil_FixCambiaElRelayAntesDeArrancarElServicio_FND171(t *testing.T) {
	m := movilPrueba(t)
	emparejarPrueba(t)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	alArrancar := ""
	remotoEjecutor = func(string, ...string) error {
		c, err := remoto.LeerRelay()
		if err != nil {
			t.Fatal(err)
		}
		alArrancar = c.URL
		return nil
	}
	now := time.Now()
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	if alArrancar != relayNuevoPrueba.URL {
		t.Fatalf("el servicio arrancó con %q", alArrancar)
	}
}

func TestDoctorMovil_FixDiceLoQueHizo_FND171(t *testing.T) {
	movilPrueba(t)
	consultas := 0
	guardarRelayPrueba(t, propioPrueba())
	hecho, err := repararRelayMovil()
	if err == nil || !strings.Contains(err.Error(), "la prueba no da directorio") || !strings.Contains(err.Error(), "otra") || len(hecho) != 0 {
		t.Fatalf("si no puede consultar, muestra el motivo: %v %v", hecho, err)
	}
	remotoConsultarRelay = directorioPrueba(&consultas, relayNuevoPrueba)
	hecho, err = repararRelayMovil()
	if err != nil || len(hecho) != 1 || !strings.Contains(hecho[0], "serie 2") {
		t.Fatalf("si lo reemplaza, lo dice: %v %v", hecho, err)
	}
	hecho, err = repararRelayMovil()
	if err != nil || len(hecho) != 1 || !strings.Contains(hecho[0], "coincide") {
		t.Fatalf("si coincide, lo dice: %v %v", hecho, err)
	}
}

func TestDoctorMovil_FixConElRelayIgualNoReiniciaElServicio_FND171(t *testing.T) {
	m := movilPrueba(t)
	now := time.Now()
	movilSano(t, m, now)
	consultas := 0
	remotoConsultarRelay = directorioPrueba(&consultas, relayAntiguoPrueba)
	guardarRelayPrueba(t, relayAntiguoPrueba)
	antes := len(*m.corridos)
	repairMovil(m.raiz, buildMovilChecks(m.raiz, now), now)
	if len(*m.corridos) != antes {
		t.Fatalf("nada cambió, nada se reinicia: %v", *m.corridos)
	}
}
