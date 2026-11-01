package incidentes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var ctx = Contexto{Raiz: "/home/ana/clientes/acme-pagos", Home: "/home/ana", Usuario: "ana"}

func TestSanitizar_NoDejaNadaDelProyecto(t *testing.T) {
	casos := []string{
		"open /home/ana/clientes/acme-pagos/src/cobros/tarjeta.go: permission denied",
		"la tarea 'cobro-tarjeta' no existe en «Pagos ACME»",
		`token=ghp_abcdefghijklmnopqrstuvwxyz0123 y clave: hunter2`,
		"escribe a ana.perez@acme.cl desde ~/notas/privado.txt",
		"falló src/cobros/tarjeta.go al leer facturas.csv",
		"remoto https://gitlab.acme.cl/pagos/core.git",
	}
	prohibido := []string{"acme", "ana", "cobros", "tarjeta", "cobro-tarjeta", "Pagos", "ghp_", "hunter2", "facturas", "notas", "privado", "gitlab"}
	for _, c := range casos {
		got := Sanitizar(c, ctx)
		for _, p := range prohibido {
			if strings.Contains(strings.ToLower(got), strings.ToLower(p)) {
				t.Errorf("Sanitizar(%q) = %q: deja %q", c, got, p)
			}
		}
	}
}

func TestSanitizar_ConservaLoQueEsDeQDD(t *testing.T) {
	got := Sanitizar("yaml: line 3: mapping values are not allowed in .qdd/project/estado.yaml (formato 2)", ctx)
	for _, quiere := range []string{"yaml: line 3: mapping values are not allowed", ".qdd/project/estado.yaml", "formato 2"} {
		if !strings.Contains(got, quiere) {
			t.Errorf("Sanitizar quitó %q: %q", quiere, got)
		}
	}
	if got := Sanitizar("runtime error: index out of range [3] with length 3", ctx); got != "runtime error: index out of range [3] with length 3" {
		t.Errorf("un mensaje de Go no se toca: %q", got)
	}
}

const pila = `goroutine 1 [running]:
runtime/debug.Stack()
	/usr/lib/go/src/runtime/debug/stack.go:26 +0x5e
github.com/qdd-framework/qdd/pkg/incidentes.Panico({0x1, 0x2}, {0x3, 0x4})
	/home/ana/src/qdd/cli/pkg/incidentes/incidentes.go:40 +0x1f
github.com/qdd-framework/qdd/cmd.Execute.func1()
	/home/ana/src/qdd/cli/cmd/root.go:174 +0x2a
panic({0x10, 0x20})
	/usr/lib/go/src/runtime/panic.go:785 +0x132
github.com/qdd-framework/qdd/pkg/estado.(*Estado).Set(0xc000, {0x5, 0x6})
	/home/ana/src/qdd/cli/pkg/estado/estado.go:123 +0x1f
github.com/qdd-framework/qdd/cmd.runEstadoSet(0xc0001, {0xc0002, 0x3, 0x3})
	/home/ana/go/pkg/mod/github.com/qdd-framework/qdd@v2.8.0/cmd/estado.go:88 +0x99
github.com/spf13/cobra.(*Command).execute(0xc0003)
	/home/ana/go/pkg/mod/github.com/spf13/cobra@v1.8.0/command.go:989 +0xab
`

func TestTraza_SoloElCodigoDeQDDSinRutasLocales(t *testing.T) {
	got := Traza([]byte(pila))
	quiere := []string{
		"pkg/estado.(*Estado).Set (pkg/estado/estado.go:123)",
		"cmd.runEstadoSet (cmd/estado.go:88)",
	}
	if strings.Join(got, "\n") != strings.Join(quiere, "\n") {
		t.Fatalf("Traza =\n%s\nquiere\n%s", strings.Join(got, "\n"), strings.Join(quiere, "\n"))
	}
}

func TestFirma_EstableEntreVersionesYRutas(t *testing.T) {
	a := Nuevo(TipoPanico, "qdd estado set", "index out of range [3] with length 3", []string{"pkg/estado.(*Estado).Set (pkg/estado/estado.go:123)"}, "2.8.0")
	b := Nuevo(TipoPanico, "qdd estado set", "index out of range [7] with length 7", []string{"pkg/estado.(*Estado).Set (pkg/estado/estado.go:131)"}, "2.9.0")
	if a.Firma == "" || a.Firma != b.Firma {
		t.Fatalf("el mismo defecto con otros números o líneas es la misma firma: %q vs %q", a.Firma, b.Firma)
	}
	c := Nuevo(TipoPanico, "qdd estado ver", "index out of range [3] with length 3", []string{"pkg/estado.(*Estado).Set (pkg/estado/estado.go:123)"}, "2.8.0")
	if c.Firma == a.Firma {
		t.Fatal("otro comando es otra firma")
	}
}

func TestComando_SoloElCaminoYLosNombresDeLasBanderas(t *testing.T) {
	got := Comando("qdd estado set", []string{"cobro-tarjeta", "--etapa", "Pruebas", "--nota=pagos ok", "-v"})
	if got != "qdd estado set <arg> --etapa <valor> --nota=<valor> -v" {
		t.Fatalf("Comando = %q", got)
	}
}

func TestRegistrar_LaMismaFirmaSumaUnaOcurrencia(t *testing.T) {
	dir := t.TempDir()
	ahora := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	i := Nuevo(TipoPanico, "qdd estado set", "boom", nil, "2.8.0")
	if _, err := Registrar(dir, i, ahora); err != nil {
		t.Fatal(err)
	}
	r, err := Registrar(dir, i, ahora.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if r.Ocurrencias != 2 || r.Primera != "2026-11-01T10:00:00Z" || r.Ultima != "2026-11-01T11:00:00Z" {
		t.Fatalf("registro = %+v", r)
	}
	todos, err := Leer(dir)
	if err != nil || len(todos) != 1 {
		t.Fatalf("Leer = %d, %v; quiere 1 archivo por firma", len(todos), err)
	}
}

type ghFalso struct {
	existentes map[string]string
	creados    []string
	comentados []string
	falla      error
}

func (g *ghFalso) Buscar(firma string) (string, error) { return g.existentes[firma], g.falla }
func (g *ghFalso) Crear(titulo, cuerpo string) (string, error) {
	if g.falla != nil {
		return "", g.falla
	}
	g.creados = append(g.creados, cuerpo)
	return "https://github.com/x/y/issues/" + string(rune('0'+len(g.creados))), nil
}
func (g *ghFalso) Comentar(url, cuerpo string) error {
	g.comentados = append(g.comentados, url)
	return g.falla
}

func TestPublicar_CreaUnaVezYDespuesComenta(t *testing.T) {
	dir := t.TempDir()
	ahora := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	i := Nuevo(TipoPanico, "qdd estado set", "boom", nil, "2.8.0")
	_, _ = Registrar(dir, i, ahora)
	gh := &ghFalso{}
	if r := Publicar(dir, gh, ahora, 5); r.Creados != 1 || r.Err != nil {
		t.Fatalf("primera publicación: %+v", r)
	}
	if r := Publicar(dir, gh, ahora, 5); r.Creados+r.Comentados != 0 {
		t.Fatalf("sin ocurrencias nuevas no publica nada: %+v", r)
	}
	_, _ = Registrar(dir, i, ahora.Add(time.Minute))
	if r := Publicar(dir, gh, ahora, 5); r.Comentados != 1 || len(gh.creados) != 1 {
		t.Fatalf("una ocurrencia nueva es un comentario, no otro issue: %+v, creados %d", r, len(gh.creados))
	}
}

func TestPublicar_UnIssueQueYaExisteSeReusa(t *testing.T) {
	dir := t.TempDir()
	ahora := time.Now()
	i := Nuevo(TipoPanico, "qdd estado set", "boom", nil, "2.8.0")
	_, _ = Registrar(dir, i, ahora)
	gh := &ghFalso{existentes: map[string]string{i.Firma: "https://github.com/x/y/issues/9"}}
	r := Publicar(dir, gh, ahora, 5)
	if r.Comentados != 1 || len(gh.creados) != 0 || gh.comentados[0] != "https://github.com/x/y/issues/9" {
		t.Fatalf("otro equipo ya lo publicó: se comenta ahí. %+v", r)
	}
}

func TestPublicar_RespetaElTopeDiario(t *testing.T) {
	dir := t.TempDir()
	ahora := time.Now()
	for _, m := range []string{"uno", "dos", "tres"} {
		_, _ = Registrar(dir, Nuevo(TipoPanico, "qdd "+m, m, nil, "2.8.0"), ahora)
	}
	gh := &ghFalso{}
	if r := Publicar(dir, gh, ahora, 2); r.Creados != 2 || r.Pendientes != 1 {
		t.Fatalf("tope 2: %+v", r)
	}
	if r := Publicar(dir, gh, ahora, 2); r.Creados != 0 || r.Pendientes != 1 {
		t.Fatalf("el mismo día ya no publica más: %+v", r)
	}
	if r := Publicar(dir, gh, ahora.Add(24*time.Hour), 2); r.Creados != 1 {
		t.Fatalf("al día siguiente sigue: %+v", r)
	}
}

func TestPublicar_SinGHQuedaPendienteYLoDice(t *testing.T) {
	dir := t.TempDir()
	_, _ = Registrar(dir, Nuevo(TipoPanico, "qdd x", "boom", nil, "2.8.0"), time.Now())
	r := Publicar(dir, &ghFalso{falla: errors.New("gh: not logged in")}, time.Now(), 5)
	if r.Err == nil || r.Pendientes != 1 {
		t.Fatalf("sin acceso: %+v", r)
	}
}

func TestCuerpo_FormatoEstandarParaLaIA(t *testing.T) {
	i := Nuevo(TipoPanico, "qdd estado set <arg>", "index out of range [3] with length 3", []string{"pkg/estado.(*Estado).Set (pkg/estado/estado.go:123)"}, "2.8.0")
	i.Ocurrencias, i.Primera = 3, "2026-11-01T10:00:00Z"
	c := Cuerpo(i)
	for _, quiere := range []string{
		"<!-- qdd-incidente firma=" + i.Firma,
		"## Qué pasó", "## Dónde (código de QDD)", "## Cómo reproducir", "## Para la IA que lo repara",
		"formato: " + Formato, "firma: " + i.Firma, "qdd estado set <arg>", "pkg/estado/estado.go:123", "ocurrencias: 3",
	} {
		if !strings.Contains(c, quiere) {
			t.Errorf("el cuerpo no trae %q:\n%s", quiere, c)
		}
	}
	if !strings.HasPrefix(Titulo(i), "[incidente] pánico en qdd estado set") {
		t.Errorf("Titulo = %q", Titulo(i))
	}
}

func TestLeer_UnRegistroIlegibleSeApartaSinPerderse(t *testing.T) {
	dir := t.TempDir()
	_, _ = Registrar(dir, Nuevo(TipoPanico, "qdd x", "boom", nil, "2.8.0"), time.Now())
	_ = os.WriteFile(filepath.Join(dir, "roto.yaml"), []byte("::: no es yaml"), 0o600)
	todos, err := Leer(dir)
	if len(todos) != 1 || err == nil {
		t.Fatalf("Leer = %d, %v: lee los buenos y avisa del ilegible", len(todos), err)
	}
	hecho, err := ApartarIlegibles(dir)
	if err != nil || len(hecho) != 1 {
		t.Fatalf("ApartarIlegibles = %v, %v", hecho, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "roto.yaml.ilegible")); err != nil {
		t.Fatal("el ilegible queda apartado, no borrado")
	}
	if _, err := Leer(dir); err != nil {
		t.Fatalf("después de apartar se lee limpio: %v", err)
	}
}

func TestActivo_ApagadoPorVariableYEnPruebasSinCarpeta(t *testing.T) {
	t.Setenv(HomeEnv, "")
	if Activo() {
		t.Fatal("en pruebas, sin QDD_HOME explícito, nunca toca la carpeta real")
	}
	t.Setenv(HomeEnv, t.TempDir())
	if !Activo() {
		t.Fatal("con QDD_HOME de la prueba, sí")
	}
	t.Setenv(ApagadoEnv, "off")
	if Activo() {
		t.Fatal("QDD_INCIDENTES=off lo apaga")
	}
}
