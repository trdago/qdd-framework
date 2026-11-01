package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/incidentes"
)

// conProyectoPrivado deja la prueba en un proyecto «acme-pagos» con un QDD_HOME propio y devuelve la carpeta del
// registro de incidentes.
func conProyectoPrivado(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	raiz := filepath.Join(t.TempDir(), "acme-pagos")
	if err := os.MkdirAll(raiz, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(raiz)
	return raiz, filepath.Join(home, "incidentes")
}

func soloIncidente(t *testing.T, dir string) incidentes.Incidente {
	t.Helper()
	todos, err := incidentes.Leer(dir)
	if err != nil || len(todos) != 1 {
		t.Fatalf("registro = %d incidentes, %v; quiere 1", len(todos), err)
	}
	return todos[0]
}

func sinRastrosDelProyecto(t *testing.T, dir, raiz string) {
	t.Helper()
	rutas, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	for _, r := range rutas {
		data, _ := os.ReadFile(r)
		for _, prohibido := range []string{raiz, "acme", "cobros"} {
			if strings.Contains(string(data), prohibido) {
				t.Errorf("el registro lleva %q:\n%s", prohibido, data)
			}
		}
	}
}

func TestIncidentes_UnPanicoDeLaCLIQuedaAnonimizadoConSuTraza(t *testing.T) {
	raiz, dir := conProyectoPrivado(t)
	func() {
		defer func() {
			if r := recover(); r != nil {
				origen, comando := origenCLI([]string{"estado", "set", "cobros", "--etapa", "Pruebas"})
				registrarPanico(origen, comando, r)
			}
		}()
		// qdd:ignore CERT-020-ZERO-PANIC
		panic("open " + raiz + "/src/cobros/tarjeta.go: boom")
	}()
	i := soloIncidente(t, dir)
	if i.Tipo != incidentes.TipoPanico || i.Origen != "qdd estado set" || i.Comando != "qdd estado set <arg> --etapa <valor>" {
		t.Fatalf("incidente = %+v", i)
	}
	if len(i.Traza) == 0 || !strings.HasPrefix(i.Traza[0], "cmd.TestIncidentes_UnPanicoDeLaCLIQuedaAnonimizadoConSuTraza") {
		t.Fatalf("la traza empieza donde se produjo el pánico: %v", i.Traza)
	}
	sinRastrosDelProyecto(t, dir, raiz)
}

func TestIncidentes_UnPanicoEnUnaToolMCPQuedaRegistrado(t *testing.T) {
	raiz, dir := conProyectoPrivado(t)
	s := server.NewMCPServer("TEST-MCP", "1.0.0")
	RegisterToolWithTelemetry(s, mcp.NewTool("qdd_prueba"), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// qdd:ignore CERT-020-ZERO-PANIC
		panic("no se pudo leer " + raiz + "/.env")
	})
	res, err := s.GetTool("qdd_prueba").Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil || res == nil || !res.IsError {
		t.Fatalf("la tool devuelve un error, no cae: %+v, %v", res, err)
	}
	if i := soloIncidente(t, dir); i.Origen != "mcp:qdd_prueba" {
		t.Fatalf("origen = %q", i.Origen)
	}
	sinRastrosDelProyecto(t, dir, raiz)
}

func TestIncidentes_LaMismaFallaDosVecesEsUnIncidenteConDosOcurrencias(t *testing.T) {
	_, dir := conProyectoPrivado(t)
	for range 2 {
		registrarPanico("qdd estado ver", "qdd estado ver", "boom")
	}
	if i := soloIncidente(t, dir); i.Ocurrencias != 2 {
		t.Fatalf("ocurrencias = %d", i.Ocurrencias)
	}
}

func TestIncidentes_ApagadosNoSeRegistran(t *testing.T) {
	_, dir := conProyectoPrivado(t)
	t.Setenv(incidentes.ApagadoEnv, "off")
	registrarPanico("qdd x", "qdd x", "boom")
	if todos, _ := incidentes.Leer(dir); len(todos) != 0 {
		t.Fatalf("QDD_INCIDENTES=off: %d registrados", len(todos))
	}
}

func TestIncidentes_VerMuestraElIssueExactoYPublicarNoDuplica(t *testing.T) {
	_, dir := conProyectoPrivado(t)
	registrarPanico("qdd estado ver", "qdd estado ver", "boom")
	i := soloIncidente(t, dir)
	var out bytes.Buffer
	if verIncidente(i.Firma, &out) != 0 || !strings.Contains(out.String(), "## Cómo reproducir") || !strings.Contains(out.String(), "firma: "+i.Firma) {
		t.Fatalf("ver:\n%s", out.String())
	}
	gh := &ghEnMemoria{}
	for range 2 {
		out.Reset()
		if publicarIncidentes(gh, time.Now(), &out) != 0 {
			t.Fatalf("publicar: %s", out.String())
		}
	}
	if gh.creados != 1 {
		t.Fatalf("dos publicaciones sin ocurrencias nuevas abren un solo issue: %d", gh.creados)
	}
	out.Reset()
	listarIncidentes(&out)
	if !strings.Contains(out.String(), "https://github.com/qdd/issues/1") {
		t.Fatalf("la lista dice dónde se publicó:\n%s", out.String())
	}
}

func TestIncidentes_DoctorRegistraLaAutorreparacionDeUnRegistroIlegible(t *testing.T) {
	_, dir := conProyectoPrivado(t)
	registrarAutorreparacion("doctor: registro de uso", "un registro de uso estaba ilegible", nil)
	if todos, _ := incidentes.Leer(dir); len(todos) != 0 {
		t.Fatal("sin nada reparado no hay incidente")
	}
	registrarAutorreparacion("doctor: registro de uso", "un registro de uso estaba ilegible", []string{"apartado 2.8.0.json.ilegible"})
	i := soloIncidente(t, dir)
	if i.Tipo != incidentes.TipoAutorreparacion || !strings.Contains(i.Reparacion, ".ilegible") {
		t.Fatalf("incidente = %+v", i)
	}
}

type ghEnMemoria struct{ creados int }

func (g *ghEnMemoria) Buscar(string) (string, error) { return "", nil }
func (g *ghEnMemoria) Crear(string, string) (string, error) {
	g.creados++
	return "https://github.com/qdd/issues/1", nil
}
func (g *ghEnMemoria) Comentar(string, string) error { return nil }

func TestDoctor_ElRegistroDeIncidentesIlegibleSeApartaYLosPendientesSeVen(t *testing.T) {
	_, dir := conProyectoPrivado(t)
	if g := buildIncidentesChecks(); len(g.Items) != 0 {
		t.Fatalf("sin registro, el grupo no aparece: %+v", g.Items)
	}
	registrarPanico("qdd x", "qdd x", "boom")
	_ = os.WriteFile(filepath.Join(dir, "roto.yaml"), []byte("::: no"), 0o600)
	g := buildIncidentesChecks()
	if len(g.Items) != 1 || g.Items[0].Success || !strings.Contains(g.Items[0].Error, "ilegible") {
		t.Fatalf("doctor ve el ilegible: %+v", g.Items)
	}
	repairIncidentes()
	g = buildIncidentesChecks()
	if !g.Items[0].Success || !strings.Contains(g.Items[0].Error, "1 pendiente") {
		t.Fatalf("tras --fix queda sano y dice lo pendiente: %+v", g.Items[0])
	}
}

func TestDoctor_ApartarUnRegistroDeUsoEsUnaAutorreparacionQueSeInforma(t *testing.T) {
	_, dir := conProyectoPrivado(t)
	root := t.TempDir()
	usoDir := filepath.Join(os.Getenv("QDD_HOME"), "uso")
	_ = os.MkdirAll(usoDir, 0o700)
	_ = os.WriteFile(filepath.Join(usoDir, "2.8.0-rc.1.json"), []byte("{roto"), 0o600)
	repairConsola(root, buildConsolaChecks(root))
	i := soloIncidente(t, dir)
	if i.Tipo != incidentes.TipoAutorreparacion || i.Origen != "doctor: "+nombreUso {
		t.Fatalf("incidente = %+v", i)
	}
}
