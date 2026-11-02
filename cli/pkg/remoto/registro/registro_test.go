package registro

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.1): el registro de proyectos de la máquina. Se anota en qdd init y en cada qdd consola ciclo (cada
// mensaje de la persona), así que tiene que ser barato: si ya está y se vio hace menos de una hora, solo se lee.

func homeTemporal(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("QDD_HOME", h)
	return h
}

func carpeta(t *testing.T, nombre string) string {
	t.Helper()
	base, _ := filepath.EvalSymlinks(t.TempDir())
	dir := filepath.Join(base, nombre)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAnotar_CreaElRegistroPrivadoConLaClaveDeQDD(t *testing.T) {
	home := homeTemporal(t)
	root := carpeta(t, "Mi Proyecto")
	ahora := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	if err := Anotar(root, ahora); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(home, "proyectos.json")
	info, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("el registro es privado (0600): %v", info.Mode().Perm())
	}
	reg, err := Leer()
	if err != nil {
		t.Fatal(err)
	}
	if reg.Formato != 1 || len(reg.Proyectos) != 1 {
		t.Fatalf("un proyecto, formato 1: %+v", reg)
	}
	p := reg.Proyectos[0]
	local, _ := estado.RutaLocal(root, "pedidos")
	if p.Ruta != root || p.Clave != strings.TrimSuffix(filepath.Base(local), ".json") || !p.Visto.Equal(ahora) {
		t.Fatalf("ruta, clave de RutaLocal y visto: %+v (clave esperada de %s)", p, local)
	}
	if !strings.HasPrefix(p.Clave, "mi-proyecto-") {
		t.Fatalf("la clave lleva el nombre de la carpeta, no la ruta: %s", p.Clave)
	}
}

func TestAnotar_SiSeVioHaceMenosDeUnaHoraNoEscribe(t *testing.T) {
	home := homeTemporal(t)
	root := carpeta(t, "p")
	t0 := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	if err := Anotar(root, t0); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(home, "proyectos.json")
	antes, _ := os.ReadFile(ruta)
	if err := Anotar(root, t0.Add(59*time.Minute)); err != nil {
		t.Fatal(err)
	}
	despues, _ := os.ReadFile(ruta)
	if string(antes) != string(despues) {
		t.Fatal("dentro de la hora no se reescribe")
	}
	if err := Anotar(root, t0.Add(61*time.Minute)); err != nil {
		t.Fatal(err)
	}
	reg, _ := Leer()
	if len(reg.Proyectos) != 1 || !reg.Proyectos[0].Visto.Equal(t0.Add(61*time.Minute)) {
		t.Fatalf("pasada la hora se actualiza visto, sin duplicar: %+v", reg.Proyectos)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestAnotar_UnWorktreeSeRegistraComoSuCheckoutPrincipal(t *testing.T) {
	homeTemporal(t)
	main := carpeta(t, "repo")
	git(t, main, "init", "-q")
	git(t, main, "commit", "-q", "--allow-empty", "-m", "x")
	wt := filepath.Join(filepath.Dir(main), "wt")
	git(t, main, "worktree", "add", "-q", wt)
	ahora := time.Now()
	if err := Anotar(wt, ahora); err != nil {
		t.Fatal(err)
	}
	if err := Anotar(main, ahora.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	reg, _ := Leer()
	if len(reg.Proyectos) != 1 || reg.Proyectos[0].Ruta != main {
		t.Fatalf("el worktree y el principal son un solo proyecto, con la ruta del principal: %+v", reg.Proyectos)
	}
}

func TestAnotar_VariasSesionesALaVezNoSePierdenProyectos(t *testing.T) {
	homeTemporal(t)
	base, _ := filepath.EvalSymlinks(t.TempDir())
	var wg sync.WaitGroup
	n := 16
	for i := 0; i < n; i++ {
		dir := filepath.Join(base, fmt.Sprintf("p%02d", i))
		os.MkdirAll(dir, 0o755)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Anotar(dir, time.Now()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	reg, err := Leer()
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Proyectos) != n {
		t.Fatalf("con el bloqueo, ninguna anotación pisa a otra: %d de %d", len(reg.Proyectos), n)
	}
}

func TestLeer_OrdenaPorVistoElMasRecientePrimero(t *testing.T) {
	homeTemporal(t)
	a, b, c := carpeta(t, "a"), carpeta(t, "b"), carpeta(t, "c")
	t0 := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	Anotar(a, t0)
	Anotar(b, t0.Add(2*time.Hour))
	Anotar(c, t0.Add(time.Hour))
	reg, _ := Leer()
	got := []string{filepath.Base(reg.Proyectos[0].Ruta), filepath.Base(reg.Proyectos[1].Ruta), filepath.Base(reg.Proyectos[2].Ruta)}
	if strings.Join(got, ",") != "b,c,a" {
		t.Fatalf("por visto, el más reciente primero: %v", got)
	}
}

func TestMuertas_SenalaYQuitaLasRutasQueYaNoExisten(t *testing.T) {
	homeTemporal(t)
	viva, muerta := carpeta(t, "viva"), carpeta(t, "muerta")
	Anotar(viva, time.Now())
	Anotar(muerta, time.Now())
	os.RemoveAll(muerta)
	reg, _ := Leer()
	if m := Muertas(reg); len(m) != 1 || m[0] != muerta {
		t.Fatalf("solo la que no existe: %v", m)
	}
	quitadas, err := QuitarMuertas()
	if err != nil || len(quitadas) != 1 {
		t.Fatalf("quita una: %v %v", quitadas, err)
	}
	reg, _ = Leer()
	if len(reg.Proyectos) != 1 || reg.Proyectos[0].Ruta != viva {
		t.Fatalf("queda la viva: %+v", reg.Proyectos)
	}
}

func TestAnotar_UnRegistroIlegibleNoSePisaYSeAparta(t *testing.T) {
	home := homeTemporal(t)
	ruta := filepath.Join(home, "proyectos.json")
	os.WriteFile(ruta, []byte("{roto"), 0o600)
	if err := Anotar(carpeta(t, "p"), time.Now()); err == nil {
		t.Fatal("un registro ilegible es un error, no se reescribe en silencio")
	}
	if b, _ := os.ReadFile(ruta); string(b) != "{roto" {
		t.Fatal("no se pisa")
	}
	if _, err := Leer(); err == nil {
		t.Fatal("Leer dice que está ilegible")
	}
	apartado, err := Apartar()
	if err != nil || apartado != ruta+".ilegible" {
		t.Fatalf("se aparta, no se borra: %s %v", apartado, err)
	}
	if b, _ := os.ReadFile(ruta + ".ilegible"); string(b) != "{roto" {
		t.Fatal("el apartado conserva lo que había")
	}
	if err := Anotar(carpeta(t, "q"), time.Now()); err != nil {
		t.Fatalf("después de apartarlo se anota de nuevo: %v", err)
	}
}

func TestLeer_SinRegistroEsVacio(t *testing.T) {
	homeTemporal(t)
	reg, err := Leer()
	if err != nil || len(reg.Proyectos) != 0 || reg.Formato != 1 {
		t.Fatalf("sin archivo, registro vacío: %+v %v", reg, err)
	}
	if Existe() {
		t.Fatal("no existe hasta la primera anotación")
	}
}
