package baul

import (
	"bytes"
	"io"
	"sort"
	"sync"
)

// Rule: Prohibido usar 'else'

// Tapado reemplaza un valor del baúl en la salida.
const Tapado = "••••"

// Ocultador tapa los valores del baúl en lo que escribe un comando. Retiene lo justo para ver un valor que llega
// partido entre dos escrituras; Close entrega lo que quedó.
type Ocultador struct {
	mu      sync.Mutex
	destino io.Writer
	valores [][]byte
	resto   []byte
	maximo  int
}

// NuevoOcultador tapa esos valores; los de menos de 6 caracteres no se buscan (taparían texto común).
func NuevoOcultador(destino io.Writer, valores []string) *Ocultador {
	o := &Ocultador{destino: destino}
	for _, v := range valores {
		if len(v) < 6 {
			continue
		}
		o.valores = append(o.valores, []byte(v))
		o.maximo = max(o.maximo, len(v))
	}
	sort.Slice(o.valores, func(i, j int) bool { return len(o.valores[i]) > len(o.valores[j]) })
	return o
}

func (o *Ocultador) tapar(b []byte) []byte {
	for _, v := range o.valores {
		b = bytes.ReplaceAll(b, v, []byte(Tapado))
	}
	return b
}

// Write tapa y entrega todo salvo la cola que podría ser el comienzo de un valor.
func (o *Ocultador) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	b := o.tapar(append(o.resto, p...))
	corte := len(b)
	if o.maximo > 0 {
		corte = max(0, len(b)-(o.maximo-1))
	}
	o.resto = append([]byte(nil), b[corte:]...)
	if _, err := o.destino.Write(b[:corte]); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close entrega lo retenido, tapado.
func (o *Ocultador) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	b := o.tapar(o.resto)
	o.resto = nil
	_, err := o.destino.Write(b)
	return err
}
