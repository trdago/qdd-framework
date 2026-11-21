package consola

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// El panel se lee en el mismo proceso y solo se emite lo que cambió, en vez de lanzar un proceso cada 2 s (MEJ-020).
func sinLatido() error { return nil }

func TestSeguir_EmiteLaPrimeraYSoloLoQueCambia(t *testing.T) {
	lecturas := [][]byte{[]byte("a"), []byte("a"), []byte("b"), []byte("b"), []byte("a")}
	senal := make(chan struct{})
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	n := 0
	leer := func() ([]byte, error) {
		l := lecturas[n]
		n++
		return l, nil
	}
	var emitidas []string
	termino := make(chan error, 1)
	go func() {
		termino <- Seguir(ctx, leer, senal, func(l []byte) error {
			emitidas = append(emitidas, string(l))
			return nil
		}, sinLatido)
	}()
	for range lecturas[1:] {
		senal <- struct{}{}
	}
	cancelar()
	if err := <-termino; err != nil {
		t.Fatalf("con el contexto cancelado termina sin error: %v", err)
	}
	if got, want := emitidas, []string{"a", "b", "a"}; !iguales(got, want) {
		t.Fatalf("emitidas = %v, quiero %v", got, want)
	}
}

func TestSeguir_UnaLecturaFallidaNoEmiteNiCorta(t *testing.T) {
	senal := make(chan struct{})
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	n := 0
	leer := func() ([]byte, error) {
		n++
		if n == 2 {
			return nil, errors.New("archivo a medio escribir")
		}
		return []byte{byte('0' + n)}, nil
	}
	var emitidas []string
	termino := make(chan error, 1)
	go func() {
		termino <- Seguir(ctx, leer, senal, func(l []byte) error {
			emitidas = append(emitidas, string(l))
			return nil
		}, sinLatido)
	}()
	senal <- struct{}{}
	senal <- struct{}{}
	cancelar()
	<-termino
	if got, want := emitidas, []string{"1", "3"}; !iguales(got, want) {
		t.Fatalf("emitidas = %v, quiero %v", got, want)
	}
}

// Si quien lee se fue (la sesión terminó), el flujo termina: nunca queda un proceso huérfano escribiendo.
func TestSeguir_TerminaSiNoSePuedeEmitir(t *testing.T) {
	roto := errors.New("tubería cerrada")
	err := Seguir(context.Background(), func() ([]byte, error) { return []byte("a"), nil }, make(chan struct{}),
		func([]byte) error { return roto }, sinLatido)
	if !errors.Is(err, roto) {
		t.Fatalf("err = %v, quiero %v", err, roto)
	}
}

// Nada cambia durante horas y la sesión murió: el latido es lo que permite enterarse, también en Windows.
func TestSeguir_SinCambiosLateYTerminaSiElLatidoFalla(t *testing.T) {
	roto := errors.New("tubería cerrada")
	senal := make(chan struct{}, 1)
	senal <- struct{}{}
	emitidas := 0
	err := Seguir(context.Background(), func() ([]byte, error) { return []byte("a"), nil }, senal,
		func([]byte) error { emitidas++; return nil }, func() error { return roto })
	if !errors.Is(err, roto) {
		t.Fatalf("err = %v, quiero %v", err, roto)
	}
	if emitidas != 1 {
		t.Fatalf("emitidas = %d, quiero 1", emitidas)
	}
}

// El ayudante es el proceso real de TestSeguir_UnProcesoRealTerminaAlCerrarseSuSalida.
func TestSeguirAyudante(t *testing.T) {
	if os.Getenv("QDD_SEGUIR_AYUDANTE") == "" {
		t.Skip("solo como proceso hijo")
	}
	senal := make(chan struct{}, 1)
	go func() {
		for range time.Tick(20 * time.Millisecond) {
			select {
			case senal <- struct{}{}:
			default:
			}
		}
	}()
	err := Seguir(context.Background(), func() ([]byte, error) { return []byte("a"), nil }, senal,
		func(l []byte) error { _, err := fmt.Fprintf(os.Stdout, "%s\n", l); return err },
		func() error { _, err := fmt.Fprintln(os.Stdout); return err })
	if err != nil {
		os.Exit(0)
	}
}

// Con procesos de verdad, para que el CI lo pruebe también en Windows: cerrar la salida equivale a que muera la sesión.
func TestSeguir_UnProcesoRealTerminaAlCerrarseSuSalida(t *testing.T) {
	c := exec.Command(os.Args[0], "-test.run=^TestSeguirAyudante$")
	c.Env = append(os.Environ(), "QDD_SEGUIR_AYUDANTE=1")
	salida, err := c.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	if !bufio.NewScanner(salida).Scan() {
		t.Fatal("el proceso no entregó su primera línea")
	}
	salida.Close()
	termino := make(chan error, 1)
	go func() { termino <- c.Wait() }()
	select {
	case <-termino:
	case <-time.After(10 * time.Second):
		c.Process.Kill()
		t.Fatal("el proceso sigue vivo sin nadie que lo lea")
	}
}

// Tras una actualización, un flujo abierto no sigue corriendo el binario viejo.
func TestBinarioCambio(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "qdd")
	if err := os.WriteFile(ruta, []byte("v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	cambio := VigiaBinario(ruta)
	if cambio() {
		t.Fatal("sin tocarlo no cambió")
	}
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}
	if cambio() {
		t.Fatal("mientras npm lo reemplaza falta unos segundos (FND-135): eso todavía no es un cambio")
	}
	if err := os.WriteFile(ruta, []byte("version 2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !cambio() {
		t.Fatal("el binario nuevo es un cambio")
	}
}

func TestSenalesDelPanel_AvisaAlEscribirEnUnaCarpetaVigilada(t *testing.T) {
	dir := t.TempDir()
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	senal, err := SenalesDelPanel(ctx, []string{dir, filepath.Join(dir, "no-existe")}, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "estado.yaml"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-senal:
	case <-time.After(3 * time.Second):
		t.Fatal("escribir en la carpeta vigilada no avisó")
	}
}

// El reloj de respaldo cubre lo que depende de la hora («hace 3 min», un pedido armado) aunque nada se escriba.
func TestSenalesDelPanel_ElRespaldoAvisaSinEscrituras(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	senal, err := SenalesDelPanel(ctx, nil, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-senal:
	case <-time.After(3 * time.Second):
		t.Fatal("el reloj de respaldo no avisó")
	}
}

func iguales(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Una carpeta que nace después de empezar a vigilar se vigila: la primera vez que se guarda un flujo, por ejemplo.
func TestSenalesDelPanel_UnaCarpetaQueNaceSeVigila(t *testing.T) {
	root := t.TempDir()
	nueva := filepath.Join(root, "flujos")
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	senal, err := SenalesDelPanel(ctx, []string{root, nueva}, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(nueva, 0o755); err != nil {
		t.Fatal(err)
	}
	esperarSenal(t, senal)
	if err := os.WriteFile(filepath.Join(nueva, "pago.yaml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	esperarSenal(t, senal)
}

func esperarSenal(t *testing.T, senal <-chan struct{}) {
	t.Helper()
	select {
	case <-senal:
	case <-time.After(5 * time.Second):
		t.Fatal("no llegó la señal")
	}
}

// FND-190: el candado de un latido (<archivo>.json.lock, creado y borrado cada 5 s por cada panel abierto) despertaba a
// todos los paneles de la máquina, y cada uno releía todo el panel: ~50 % de un núcleo entre 8 paneles sin trabajo.
func TestSenalesDelPanel_UnCandadoOUnTemporalNoAvisan(t *testing.T) {
	dir := t.TempDir()
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	senal, err := SenalesDelPanel(ctx, []string{dir}, time.Hour, EventoDelPanel(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	for _, nombre := range []string{"qdd-abc.json.lock", "qdd-abc.json.tmp"} {
		ruta := filepath.Join(dir, nombre)
		if err := os.WriteFile(ruta, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(ruta); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-senal:
		t.Fatal("un candado o un temporal despertó al panel")
	case <-time.After(300 * time.Millisecond):
	}
	if err := os.WriteFile(filepath.Join(dir, "estado.yaml"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	esperarSenal(t, senal)
}

// Las carpetas locales (<QDD_HOME>/claude/<carpeta>) son de todos los proyectos: al panel solo le importa su archivo.
func TestEventoDelPanel_EnLoCompartidoSoloElArchivoPropio(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	importa := EventoDelPanel(root)
	propio, err := RutaLocal(root, "telefono")
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]bool{
		propio: true,
		filepath.Join(filepath.Dir(propio), "otro-proyecto-0123456789ab.json"): false,
		propio + ".lock": false,
		propio + ".tmp":  false,
		filepath.Join(root, ".qdd", "project", "estado.yaml"):            true,
		filepath.Join(root, ".qdd", "project", "flujos", "pago.yaml"):    true,
		filepath.Join(root, ".qdd", "project", "modules", "estado.yaml"): true,
	}
	for nombre, quiere := range casos {
		if got := importa(nombre); got != quiere {
			t.Errorf("EventoDelPanel(%s) = %v, quiere %v", nombre, got, quiere)
		}
	}
}
