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
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9 (qdd 2.8.0): una cola por repositorio y «todavía no» no es una falla (FND-081, FND-082).

func gitEn(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// dosRepos crea el repositorio de la sesión (con .qdd), otro repositorio con .qdd y un worktree del segundo.
func dosRepos(t *testing.T) (sesion, otro, worktree string) {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	base, _ := filepath.EvalSymlinks(t.TempDir())
	for _, r := range []string{"sesion", "otro"} {
		dir := filepath.Join(base, r)
		if err := os.MkdirAll(filepath.Join(dir, ".qdd"), 0755); err != nil {
			t.Fatal(err)
		}
		gitEn(t, dir, "init", "-q")
		os.WriteFile(filepath.Join(dir, ".qdd", "x"), []byte("x"), 0644)
		gitEn(t, dir, "add", ".")
		gitEn(t, dir, "commit", "-q", "-m", "x")
	}
	worktree = filepath.Join(base, "otro-wt")
	gitEn(t, filepath.Join(base, "otro"), "worktree", "add", "-q", worktree)
	return filepath.Join(base, "sesion"), filepath.Join(base, "otro"), worktree
}

// FND-081: un pedido sobre qdd que dejó otra sesión (con su carpeta en un worktree de qdd) quedaba en la cola del
// proyecto de esa sesión, en otro panel: la persona no veía qué iba primero.
func TestPedir_VaALaColaDelRepositorioDondeCorre_FND081(t *testing.T) {
	sesion, otro, wt := dosRepos(t)
	p, err := PedirEnOrden(sesion, "exit 0", "Mergear #33", wt, Orden{}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	if len(Pedidos(sesion)) != 0 {
		t.Fatalf("no queda en la cola de la sesión: %+v", Pedidos(sesion))
	}
	if cola := Pedidos(otro); len(cola) != 1 || cola[0].ID != p.ID || cola[0].Carpeta != wt {
		t.Fatalf("queda en la cola del repositorio donde corre, con su carpeta: %+v", cola)
	}
	if ColaDe(sesion, wt) != otro {
		t.Fatalf("ColaDe dice en qué panel quedó: %s", ColaDe(sesion, wt))
	}
	if cola := Pedidos(wt); len(cola) != 1 {
		t.Fatalf("una sesión en el worktree ve la misma cola: %+v", cola)
	}
	if _, err := PedirEnOrden(sesion, "exit 0 # despues", "Tag", wt, Orden{Despues: []string{"a"}}, enCola); err != nil {
		t.Fatalf("y se puede ordenar contra lo que ya hay en esa cola: %v", err)
	}
}

// FND-082: un comando cuya condición todavía no se cumple (falta un tag, un merge) salía con error y el panel lo
// mostraba en rojo, igual que una falla de verdad.
func TestEjecutar_TodaviaNoVuelveAPendienteConSuAviso_FND082(t *testing.T) {
	root := proyectoConCola(t)
	comando := "echo Falta el tag v2.7.0; exit 75"
	if runtime.GOOS == "windows" {
		comando = "echo Falta el tag v2.7.0& exit /b 75"
	}
	p := mustPedir(t, root, comando, "Mergear la cadena", "")
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != CodigoTodaviaNo {
		t.Fatalf("el código se respeta: %d %q", codigo, salida.String())
	}
	cola := Pedidos(root)
	if len(cola) != 1 || cola[0].Estado != EstadoPendiente || !strings.Contains(cola[0].Aviso, "Falta el tag v2.7.0") || cola[0].Tecla != "a" {
		t.Fatalf("vuelve a pendiente, con su tecla y el aviso: %+v", cola)
	}
	b, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	if b.Estado != EstadoPendiente || b.Tecla != "a" || !strings.Contains(b.Aviso, "Falta el tag") || b.AvisoEn == nil {
		t.Fatalf("el panel lo muestra pendiente con el aviso y su hora, no en rojo: %+v", b)
	}
}

func TestDiagnosticarCola_JuntaLaColaSueltaDeUnWorktreeYLosPedidosDeOtroRepositorio_FND081(t *testing.T) {
	sesion, otro, wt := dosRepos(t)
	mustPedir(t, otro, "exit 0 # ya estaba", "m", "")
	vieja, _ := RutaLocalSinRepositorio(wt, "pedidos")
	escribirCola(t, vieja, []Pedido{{ID: "1", Comando: "exit 0 # del worktree", Motivo: "m", Carpeta: wt, Tecla: "a", Estado: EstadoPendiente}})
	ajena, _ := colaPath(sesion)
	escribirCola(t, ajena, []Pedido{
		{ID: "1", Comando: "exit 0 # de otro repo", Motivo: "m", Carpeta: wt, Tecla: "a", Estado: EstadoPendiente},
		{ID: "2", Comando: "exit 0 # propio", Motivo: "m", Carpeta: sesion, Tecla: "b", Estado: EstadoPendiente},
	})
	if d := DiagnosticarCola(wt); d.OK || !strings.Contains(d.Detalle, "worktree") {
		t.Fatalf("doctor ve la cola suelta del worktree: %+v", d)
	}
	if d := DiagnosticarCola(sesion); d.OK || !strings.Contains(d.Detalle, "otro repositorio") {
		t.Fatalf("doctor ve el pedido de otro repositorio: %+v", d)
	}
	for _, r := range []string{wt, sesion} {
		if _, err := RepararCola(r); err != nil {
			t.Fatal(err)
		}
	}
	got := comandosDe(Pedidos(otro))
	for _, c := range []string{"ya estaba", "del worktree", "de otro repo"} {
		if !strings.Contains(got, c) {
			t.Fatalf("todo lo de ese repositorio queda en su cola: %s", got)
		}
	}
	if cola := Pedidos(otro); cola[0].Tecla == cola[1].Tecla || cola[1].Tecla == cola[2].Tecla || cola[0].ID == cola[2].ID {
		t.Fatalf("al juntarlas no se repiten ids ni teclas: %+v", cola)
	}
	if got := comandosDe(Pedidos(sesion)); got != "exit 0 # propio" {
		t.Fatalf("lo propio se queda: %s", got)
	}
	if _, err := os.Stat(vieja); !os.IsNotExist(err) {
		t.Fatal("la cola suelta se borra")
	}
	if !DiagnosticarCola(wt).OK || !DiagnosticarCola(sesion).OK {
		t.Fatal("reparado, doctor queda en verde")
	}
}

func escribirCola(t *testing.T, ruta string, cola []Pedido) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0700); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(cola)
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
}
