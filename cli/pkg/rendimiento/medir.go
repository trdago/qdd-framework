package rendimiento

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Opciones de una medición. Entorno se suma al de la caja (las pruebas lo usan para su qdd falso).
type Opciones struct {
	Bin          string
	Version      string
	Segundos     int // lo que corre qdd consola seguir
	Repeticiones int // de cada comando, sin contar la primera, que calienta
	Entorno      []string
	Extra        []Medida // las que mide quien llama, como el contexto del MCP
}

const (
	cadaMuestra    = 200 * time.Millisecond
	cadaCambio     = 5 * time.Second
	maxArranque    = 5 * time.Second // lo que lanza al arrancar no cuenta para el ritmo por hora
	esperaAlCerrar = 3 * time.Second
	esperaRestos   = time.Second
)

// conDelPanel son los tipos de acción que pide el panel de Claude Code, sin teléfono: medir no habla con el relay.
const conDelPanel = "atender,herramientas,mejoras,flujos,esperas_persona,pregunta"

// comando es uno de los que el panel llama a cada rato.
type comando struct {
	nombre  string
	args    []string
	entrada string
}

var comandos = []comando{
	{"latencia_statusline_p95", []string{"consola", "statusline"}, "{}"},
	{"latencia_panel_p95", []string{"consola", "panel", "--json", "--con", conDelPanel}, ""},
	{"latencia_estado_p95", []string{"estado"}, ""},
	{"latencia_flujo_p95", []string{"flujo", "ver", "grande", "--ancho", "80", "--json"}, ""},
}

// Medir corre qdd en una caja (proyecto, HOME y QDD_HOME temporales, los mismos en cada equipo) y mide latencias,
// consumo y procesos. No lanza Claude ni una IA: no gasta tokens. No guarda nada: eso lo decide quien llama.
func Medir(op Opciones) (Informe, error) {
	op = conDefectos(op)
	c, err := prepararCaja(op.Entorno)
	if err != nil {
		return Informe{}, err
	}
	defer os.RemoveAll(c.dir)
	inf := Informe{Version: op.Version, SO: runtime.GOOS, Arquitectura: runtime.GOARCH, Nucleos: runtime.NumCPU(), Cuando: time.Now().UTC(), Segundos: op.Segundos}
	for _, cmd := range comandos {
		inf.Medidas = append(inf.Medidas, medirLatencia(op, c, cmd))
	}
	medidas, restos := medirSeguir(op, c)
	inf.Medidas = append(append(inf.Medidas, medidas...), op.Extra...)
	inf.Restos = restos
	return inf, nil
}

func conDefectos(op Opciones) Opciones {
	if op.Segundos <= 0 {
		op.Segundos = 30
	}
	if op.Repeticiones <= 0 {
		op.Repeticiones = 10
	}
	return op
}

type caja struct {
	dir      string
	proyecto string
	entorno  []string
}

// variablesDeLaCaja apuntan a la caja todo lo que qdd, git, gh o npm leerían de la carpeta del usuario.
var variablesDeLaCaja = []string{"HOME", "USERPROFILE", "QDD_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME", "APPDATA", "LOCALAPPDATA"}

func prepararCaja(extra []string) (caja, error) {
	dir, err := os.MkdirTemp("", "qdd-rendimiento-")
	if err != nil {
		return caja{}, err
	}
	c := caja{dir: dir, proyecto: filepath.Join(dir, "proyecto")}
	entorno := sinVariables(os.Environ(), variablesDeLaCaja)
	for _, v := range variablesDeLaCaja {
		ruta := filepath.Join(dir, "usuario", strings.ToLower(v))
		if err := os.MkdirAll(ruta, 0o700); err != nil {
			return caja{}, err
		}
		entorno = append(entorno, v+"="+ruta)
	}
	c.entorno = append(entorno, extra...)
	if err := escribirProyecto(c.proyecto, 0); err != nil {
		return caja{}, err
	}
	c.conRepositorio()
	return c, nil
}

// conRepositorio hace del proyecto un repositorio con un remote de GitHub que no existe, como el de cualquier persona:
// así consola seguir revisa PRs cada minuto, el camino que dejaba zombies (FND-179). Sin git queda sin repositorio.
func (c caja) conRepositorio() {
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "https://github.com/qdd-medicion/proyecto.git"}} {
		g := exec.Command("git", args...)
		g.Dir, g.Env = c.proyecto, c.entorno
		if g.Run() != nil {
			return
		}
	}
}

func sinVariables(entorno, quitar []string) []string {
	var salida []string
	for _, kv := range entorno {
		nombre, _, _ := strings.Cut(kv, "=")
		if contiene(quitar, strings.ToUpper(nombre)) {
			continue
		}
		salida = append(salida, kv)
	}
	return salida
}

func contiene(lista []string, x string) bool {
	for _, y := range lista {
		if y == x {
			return true
		}
	}
	return false
}

// escribirProyecto deja el mismo proyecto en cada equipo: un tablero de 12 tareas y un flujo de 25 nodos. vuelta cambia
// un nombre, para que consola seguir tenga algo que leer.
func escribirProyecto(raiz string, vuelta int) error {
	estado := "formato: 1\netapas: [Código, PR, Pruebas]\nfrentes:\n"
	for i := 0; i < 12; i++ {
		estado += fmt.Sprintf("  - {id: t%d, nombre: Tarea %d vuelta %d, estado: abierto, celdas: {Código: {estado: ⏳, bloqueo: '¿Sigo?', de_quien: la persona}}}\n", i, i, vuelta)
	}
	flujo := "formato: 1\nid: grande\nnombre: Flujo de medición\nnodos:\n"
	for i := 0; i < 25; i++ {
		siguen := fmt.Sprintf("[n%d]", i+1)
		if i%4 == 1 {
			siguen = fmt.Sprintf("[r%d, n%d]", i, i+1)
		}
		if i == 24 {
			siguen = "[]"
		}
		flujo += fmt.Sprintf("  - {id: n%d, tipo: paso, nombre: Paso %d, descripcion: Un paso con una descripción de un par de líneas para que el dibujo tenga que partir el texto., siguen: %s}\n", i, i, siguen)
		if i%4 == 1 {
			flujo += fmt.Sprintf("  - {id: r%d, tipo: resultado, nombre: Rechazo %d}\n", i, i)
		}
	}
	archivos := map[string]string{
		filepath.Join(raiz, ".qdd", "project", "estado.yaml"):           estado,
		filepath.Join(raiz, ".qdd", "project", "flujos", "grande.yaml"): flujo,
	}
	for ruta, texto := range archivos {
		if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(ruta, []byte(texto), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (c caja) comando(bin string, args ...string) *exec.Cmd {
	cmd := exec.Command(bin, args...)
	cmd.Dir = c.proyecto
	cmd.Env = c.entorno
	return cmd
}

// medirLatencia corre el comando una vez para calentar y luego Repeticiones veces; da el p95 en milisegundos.
func medirLatencia(op Opciones, c caja, cmd comando) Medida {
	var tiempos []float64
	for i := 0; i <= op.Repeticiones; i++ {
		x := c.comando(op.Bin, cmd.args...)
		x.Stdin = strings.NewReader(cmd.entrada)
		x.Stdout, x.Stderr = io.Discard, io.Discard
		inicio := time.Now()
		if err := x.Run(); err != nil {
			m := Evaluar(cmd.nombre, nil, runtime.GOOS)
			m.Que += " — no corrió: " + err.Error()
			return m
		}
		if i == 0 {
			continue
		}
		tiempos = append(tiempos, float64(time.Since(inicio).Microseconds())/1000)
	}
	p95 := percentil(tiempos, 0.95)
	return Evaluar(cmd.nombre, &p95, runtime.GOOS)
}

func percentil(valores []float64, p float64) float64 {
	sort.Float64s(valores)
	i := int(float64(len(valores))*p+0.999999) - 1
	return valores[max(0, min(i, len(valores)-1))]
}

// medirSeguir corre qdd consola seguir como lo hace el panel, cambia el tablero cada 5 s, y mira la tabla de procesos
// cada 200 ms: qué lanza, cuántos zombies hay y qué sigue vivo al cerrarlo.
func medirSeguir(op Opciones, c caja) ([]Medida, []Proceso) {
	so := runtime.GOOS
	sinMedir := func(motivo string) ([]Medida, []Proceso) {
		var ms []Medida
		for _, n := range []string{MedidaCPU, MedidaMemoria, MedidaLanzados, MedidaZombies, MedidaHuerfanos} {
			m := Evaluar(n, nil, so)
			m.Que += " — " + motivo
			ms = append(ms, m)
		}
		return ms, nil
	}
	if _, err := Procesos(); err != nil {
		return sinMedir("no se pudo leer la tabla de procesos: " + err.Error())
	}
	lectura, escritura, err := os.Pipe()
	if err != nil {
		return sinMedir(err.Error())
	}
	x := c.comando(op.Bin, "consola", "seguir", "--con", conDelPanel)
	x.Stdout, x.Stderr = escritura, io.Discard
	inicio := time.Now()
	if err := x.Start(); err != nil {
		lectura.Close()
		escritura.Close()
		return sinMedir("no arrancó: " + err.Error())
	}
	escritura.Close()
	go func() { _, _ = io.Copy(io.Discard, lectura) }()
	total := time.Duration(op.Segundos) * time.Second
	arranque := min(maxArranque, total/4)
	v := vigilar(x.Process.Pid, inicio, total, arranque, func(vuelta int) { _ = escribirProyecto(c.proyecto, vuelta) })
	terminado := make(chan struct{})
	go func() {
		_ = x.Wait()
		close(terminado)
	}()
	lectura.Close() // como un panel que se cierra: la próxima línea que escriba falla
	select {
	case <-terminado:
	case <-time.After(esperaAlCerrar):
		_ = x.Process.Kill()
		<-terminado
	}
	fin := time.Since(inicio).Seconds()
	time.Sleep(esperaRestos)
	restos := v.vivos()
	cerrar(restos)

	cpu := (x.ProcessState.UserTime() + x.ProcessState.SystemTime()).Seconds() / fin * 100
	memoria := v.memoria
	if memoria == nil {
		memoria = memoriaAlTerminar(x.ProcessState)
	}
	ritmo := float64(v.lanzadosTrasArranque) / (total - arranque).Seconds() * 3600
	zombies, huerfanos := float64(v.zombiesMax), float64(len(restos))
	return []Medida{
		Evaluar(MedidaCPU, &cpu, so),
		Evaluar(MedidaMemoria, memoria, so),
		Evaluar(MedidaLanzados, &ritmo, so),
		Evaluar(MedidaZombies, &zombies, so),
		Evaluar(MedidaHuerfanos, &huerfanos, so),
	}, restos
}

// vigia guarda todo lo que vio bajar del proceso medido.
type vigia struct {
	vistos               map[int]Proceso
	arranque             time.Duration
	lanzadosTrasArranque int
	zombiesMax           int
	memoria              *float64
}

// vigilar mira la tabla durante segundos y llama a cambiar cada 5 s. Bloquea hasta terminar.
func vigilar(raiz int, inicio time.Time, total, arranque time.Duration, cambiar func(vuelta int)) *vigia {
	v := &vigia{vistos: map[int]Proceso{}, arranque: arranque}
	hasta := inicio.Add(total)
	proximoCambio := inicio.Add(cadaCambio)
	for vuelta := 1; time.Now().Before(hasta); time.Sleep(cadaMuestra) {
		v.mirar(raiz, inicio)
		if time.Now().Before(proximoCambio) {
			continue
		}
		proximoCambio = proximoCambio.Add(cadaCambio)
		vuelta++
		cambiar(vuelta)
	}
	v.mirar(raiz, inicio)
	return v
}

func (v *vigia) mirar(raiz int, inicio time.Time) {
	lista, err := Procesos()
	if err != nil {
		return
	}
	zombies := 0
	for _, p := range descendientes(lista, raiz) {
		if p.Zombie {
			zombies++
		}
		if _, ya := v.vistos[p.PID]; ya {
			continue
		}
		v.vistos[p.PID] = p
		if time.Since(inicio) >= v.arranque {
			v.lanzadosTrasArranque++
		}
	}
	v.zombiesMax = max(v.zombiesMax, zombies)
	if m := memoriaDurante(raiz); m != nil && (v.memoria == nil || *m > *v.memoria) {
		v.memoria = m
	}
}

// vivos son los que vio y siguen corriendo, con el mismo nombre (un PID puede reusarse).
func (v *vigia) vivos() []Proceso {
	lista, err := Procesos()
	if err != nil {
		return nil
	}
	var restos []Proceso
	for _, p := range lista {
		visto, ok := v.vistos[p.PID]
		if !ok || p.Zombie || visto.Nombre != p.Nombre {
			continue
		}
		restos = append(restos, p)
	}
	return restos
}

// cerrar termina lo que quedó vivo: medir no deja procesos en la máquina.
func cerrar(restos []Proceso) {
	for _, p := range restos {
		if pr, err := os.FindProcess(p.PID); err == nil {
			_ = pr.Kill()
		}
	}
}
