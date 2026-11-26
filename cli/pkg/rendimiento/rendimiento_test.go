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

	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-07): QDD trae una prueba de rendimiento estándar, la misma en cada equipo y sistema
// operativo, que no gasta tokens: cuánto tarda, cuánto consume y si deja procesos abiertos.

const varAyudante = "QDD_RENDIMIENTO_AYUDANTE"

// TestMain: con varAyudante, este binario de prueba hace de qdd. «sucio» deja un zombie y un huérfano.
func TestMain(m *testing.M) {
	gitenv.LimpiarProceso()
	modo := os.Getenv(varAyudante)
	if modo == "" {
		os.Exit(m.Run())
	}
	ayudante(modo, os.Args[1:])
}

func ayudante(modo string, args []string) {
	anotarLlamada(args)
	if modo == "escribe" {
		dejarHuellas()
	}
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

// varRegistro: si está, el qdd falso anota ahí cada llamada, una por línea.
const varRegistro = "QDD_RENDIMIENTO_REGISTRO"

func anotarLlamada(args []string) {
	ruta := os.Getenv(varRegistro)
	if ruta == "" {
		return
	}
	f, err := os.OpenFile(ruta, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, strings.Join(args, " "))
}

// dejarHuellas escribe en el HOME, el QDD_HOME y la carpeta en que corre: un qdd de verdad lo haría al leer y guardar.
func dejarHuellas() {
	for _, dir := range []string{os.Getenv("HOME"), os.Getenv("USERPROFILE"), os.Getenv("QDD_HOME"), "."} {
		if dir == "" {
			continue
		}
		_ = os.WriteFile(filepath.Join(dir, "huella-de-qdd"), []byte("x"), 0o600)
	}
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

// contrato: rendimiento#0e1fa88d
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

// contrato: gitenv#11ed527d
// FND-263: Medir hacía git init y git remote add en la caja con el entorno de quien lo corría: con GIT_DIR exportado (un
// hook de git), el remote origin de prueba quedaba en ese repositorio.
func TestMedir_ConGitDirExportadoNoTocaEseRepositorio_FND263(t *testing.T) {
	real := t.TempDir()
	gitLimpio(t, real, "init", "-q")
	t.Setenv("GIT_DIR", filepath.Join(real, ".git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(real, ".git", "index"))
	if _, err := Medir(opcionesDePrueba(t, "limpio")); err != nil {
		t.Fatal(err)
	}
	if remotes := gitLimpio(t, real, "remote"); remotes != "" {
		t.Fatalf("el repositorio del hook quedó intacto, sin remotes: %q", remotes)
	}
}

func gitLimpio(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir, c.Env = dir, gitenv.Limpio(os.Environ())
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
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

func vacia(t *testing.T, que, dir string) {
	t.Helper()
	if entradas, _ := os.ReadDir(dir); len(entradas) != 0 {
		t.Fatalf("%s quedó con %v", que, entradas)
	}
}

// contrato: rendimiento#b11d4f5c
// Un qdd que escribe en su HOME, su QDD_HOME y su carpeta lo hace dentro de la caja: los de la persona quedan como
// estaban. Solo Guardar escribe, y en <QDD_HOME>/rendimiento.
func TestMedir_NoTocaHomeQddHomeNiProyectoYSoloGuardarEscribe(t *testing.T) {
	home, qddHome, proyecto := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("QDD_HOME", qddHome)
	t.Chdir(proyecto)
	inf, err := Medir(opcionesDePrueba(t, "escribe"))
	if err != nil {
		t.Fatal(err)
	}
	vacia(t, "el HOME", home)
	vacia(t, "el QDD_HOME", qddHome)
	vacia(t, "el proyecto en que se corrió", proyecto)
	if err := Guardar(inf); err != nil {
		t.Fatal(err)
	}
	vacia(t, "el HOME tras guardar", home)
	vacia(t, "el proyecto tras guardar", proyecto)
	guardado, _ := filepath.Glob(filepath.Join(qddHome, "*", "*"))
	if len(guardado) != 1 || guardado[0] != filepath.Join(qddHome, "rendimiento", inf.Version+".json") {
		t.Fatalf("Guardar deja solo el informe en <QDD_HOME>/rendimiento: %v", guardado)
	}
}

// llamadasPermitidas son las lecturas del panel: ninguna lanza una IA ni gasta tokens.
var llamadasPermitidas = []string{"consola statusline", "consola panel --json", "estado", "flujo ver", "consola seguir"}

// contrato: rendimiento#d5f6ecce
// Medir solo corre qdd con las lecturas del panel (y git en la caja): ni claude ni otra IA. Un claude en el PATH lo
// anotaría si alguien lo llamara.
func TestMedir_NoLlamaAUnaIA(t *testing.T) {
	registro := filepath.Join(t.TempDir(), "llamadas")
	op := opcionesDePrueba(t, "limpio")
	op.Entorno = append(op.Entorno, varRegistro+"="+registro)
	if runtime.GOOS != "windows" {
		bin := t.TempDir()
		ia := filepath.Join(t.TempDir(), "ia")
		for _, nombre := range []string{"claude", "ollama", "codex", "gemini"} {
			script := "#!/bin/sh\necho " + nombre + " >> " + ia + "\n"
			if err := os.WriteFile(filepath.Join(bin, nombre), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		defer func() {
			if b, err := os.ReadFile(ia); err == nil {
				t.Fatalf("medir llamó a una IA: %s", b)
			}
		}()
	}
	if _, err := Medir(op); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(registro)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if !empiezaConAlguna(l, llamadasPermitidas) {
			t.Fatalf("medir corrió «qdd %s», que no es una lectura del panel", l)
		}
	}
}

func empiezaConAlguna(l string, prefijos []string) bool {
	for _, p := range prefijos {
		if l == p || strings.HasPrefix(l, p+" ") {
			return true
		}
	}
	return false
}

// contrato: rendimiento#f9f9dc18
// Sin un qdd que corra, ninguna medida se inventa: todas quedan sin medir, con valor null en el JSON y nunca 0.
func TestMedir_LoQueNoSeMidioEsNullNuncaCero(t *testing.T) {
	op := opcionesDePrueba(t, "limpio")
	op.Bin = filepath.Join(t.TempDir(), "no-existe")
	inf, err := Medir(op)
	if err != nil {
		t.Fatal(err)
	}
	if len(inf.Medidas) == 0 {
		t.Fatal("el informe nombra cada medida aunque no se haya podido medir")
	}
	for _, m := range inf.Medidas {
		if m.Valor != nil || m.Estado != EstadoSinMedir {
			t.Fatalf("%s no se midió y quedó con valor: %+v", m.Nombre, m)
		}
		b, _ := json.Marshal(m)
		if !strings.Contains(string(b), `"valor":null`) {
			t.Fatalf("sin medir es null en el JSON: %s", b)
		}
	}
	if inf.Falla() {
		t.Fatal("sin medir no es una falla ni un exceso")
	}
}

// arbol lee una carpeta: ruta relativa → contenido.
func arbol(t *testing.T, raiz string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(raiz, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		rel, _ := filepath.Rel(raiz, p)
		out[filepath.ToSlash(rel)] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// contrato: rendimiento#9e8ab609
// La caja no depende de quién ni dónde mide: con otro HOME, otro QDD_HOME y otra carpeta, el proyecto es byte a byte el
// mismo y todo lo de la persona apunta dentro de la caja.
func TestPrepararCaja_LaMismaEnCadaEquipo(t *testing.T) {
	var proyectos []map[string]string
	for _, equipo := range []string{"uno", "dos"} {
		t.Setenv("HOME", filepath.Join(t.TempDir(), equipo))
		t.Setenv("QDD_HOME", filepath.Join(t.TempDir(), equipo, "qdd"))
		t.Chdir(t.TempDir())
		c, err := prepararCaja(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(c.dir)
		proyectos = append(proyectos, arbol(t, filepath.Join(c.proyecto, ".qdd")))
		for _, v := range variablesDeLaCaja {
			if !contieneVariableEn(c.entorno, v, c.dir) {
				t.Fatalf("%s no apunta dentro de la caja: %v", v, c.entorno)
			}
		}
	}
	if len(proyectos[0]) == 0 || fmt.Sprint(proyectos[0]) != fmt.Sprint(proyectos[1]) {
		t.Fatalf("el proyecto de la caja cambia de un equipo a otro:\n%v\n%v", proyectos[0], proyectos[1])
	}
}

// contieneVariableEn: la última definición de v (la que vale) está dentro de dir.
func contieneVariableEn(entorno []string, v, dir string) bool {
	valor := ""
	for _, kv := range entorno {
		nombre, x, _ := strings.Cut(kv, "=")
		if nombre == v {
			valor = x
		}
	}
	return valor != "" && strings.HasPrefix(valor, dir+string(filepath.Separator))
}
