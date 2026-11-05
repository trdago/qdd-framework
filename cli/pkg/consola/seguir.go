package consola

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// pausaSenales junta en un solo aviso la ráfaga de eventos de una escritura atómica (temporal + rename).
const pausaSenales = 50 * time.Millisecond

// Seguir lee el panel una vez y después con cada señal, y emite solo lo que cambió. Si no cambió, o la lectura falló (un
// archivo a medio escribir no corta el flujo), late: sin latido, un flujo sin cambios nunca se entera de que quien lo
// leía se fue. Si no se puede emitir ni latir, termina.
func Seguir(ctx context.Context, leer func() ([]byte, error), senal <-chan struct{}, emitir func([]byte) error, latir func() error) error {
	var ultimo []byte
	for {
		nuevo, err := entregar(leer, ultimo, emitir, latir)
		if err != nil {
			return err
		}
		ultimo = nuevo
		select {
		case <-ctx.Done():
			return nil
		case <-senal:
		}
	}
}

func entregar(leer func() ([]byte, error), ultimo []byte, emitir func([]byte) error, latir func() error) ([]byte, error) {
	actual, err := leer()
	if err != nil || bytes.Equal(actual, ultimo) {
		return ultimo, latir()
	}
	return actual, emitir(actual)
}

// VigiaBinario dice si el archivo de RutaVigilada cambió desde que se empezó a vigilar: después de una actualización, un flujo
// abierto no sigue corriendo el binario viejo. Mientras falta (npm lo está reemplazando, FND-135) todavía no cambió.
func VigiaBinario(bin string) func() bool {
	ruta := RutaVigilada(bin)
	inicial, err := os.Stat(ruta)
	if err != nil {
		return func() bool { return false }
	}
	return func() bool {
		actual, err := os.Stat(ruta)
		if err != nil {
			return false
		}
		return actual.Size() != inicial.Size() || !actual.ModTime().Equal(inicial.ModTime())
	}
}

// SenalesDelPanel avisa cuando se escribe en una de las carpetas y, como respaldo, cada tanto: lo que depende de la
// hora («hace 3 min», un pedido armado) cambia sin que nadie escriba. Una carpeta que no existe no se vigila: la cubre
// el respaldo.
func SenalesDelPanel(ctx context.Context, carpetas []string, respaldo time.Duration) (<-chan struct{}, error) {
	vigia, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	for _, c := range carpetas {
		_ = vigia.Add(c)
	}
	senal := make(chan struct{}, 1)
	go repartirSenales(ctx, vigia, respaldo, senal)
	return senal, nil
}

func repartirSenales(ctx context.Context, vigia *fsnotify.Watcher, respaldo time.Duration, senal chan<- struct{}) {
	defer vigia.Close()
	reloj := time.NewTicker(respaldo)
	defer reloj.Stop()
	pausa := time.NewTimer(pausaSenales)
	pausa.Stop()
	defer pausa.Stop()
	avisar := func() {
		select {
		case senal <- struct{}{}:
		default:
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-vigia.Events:
			pausa.Reset(pausaSenales)
		case <-vigia.Errors:
		case <-pausa.C:
			avisar()
		case <-reloj.C:
			avisar()
		}
	}
}

// CarpetasDelPanel son las carpetas de donde sale lo que dibuja el panel: el tablero y las fichas del proyecto, y lo
// local de esta máquina (pedidos, preguntas, teléfono, goal, ciclo y actividad).
func CarpetasDelPanel(root string) []string {
	carpetas := []string{filepath.Join(root, ".qdd", "project"), filepath.Join(root, ".qdd", "project", "modules")}
	for _, local := range []string{"pedidos", "preguntas", "telefono", "goal", "ciclo", estado.CarpetaActividad, "prs-descartados", "prs-revisados"} {
		ruta, err := RutaLocal(root, local)
		if err != nil {
			continue
		}
		carpetas = append(carpetas, filepath.Dir(ruta))
	}
	return carpetas
}
