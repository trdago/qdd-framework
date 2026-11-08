package mejoras

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/candado"
	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// RFC 0024, fase 3: la fuente relay son los reportes que cualquier instalación envía con 'qdd reportar'. Como un issue
// de un desconocido, no deciden el trabajo de la IA: entran por_admitir y la persona los admite con un pedido 📮 que
// solo ella confirma (DePersona, el modo goal no lo lanza). Admitidos, siguen el camino de cualquier mejora.

const (
	FuenteRelay = "relay"
	// PorAdmitir: un reporte del relay que espera a que la persona lo admita.
	PorAdmitir = "por_admitir"
	// MaxAdmisiones es cuántos pedidos de admisión hay a la vez en el panel: el resto espera su turno.
	MaxAdmisiones = 2
)

// ErrSinRelay: este proyecto no lee el relay de reportes (no es el repositorio de QDD); la fuente no aplica.
var ErrSinRelay = errors.New("este proyecto no lee el relay de reportes")

var formatoRelay = regexp.MustCompile(`^REP-(\d+)$`)

// ReporteRelay es un reporte que el relay entregó a esta instalación lectora.
type ReporteRelay struct {
	ID      string
	Tipo    string
	Titulo  string
	Detalle string
	Version string
	SO      string
	Llego   string
}

// ComandoAdmitir es el comando del pedido que la persona confirma para admitir una mejora del relay.
func ComandoAdmitir(id string) string {
	return "qdd mejoras admitir " + id
}

func numeroRelay(id string) int {
	m := formatoRelay.FindStringSubmatch(id)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// desdeRelay es el último REP-N que ya está en la cola: el relay entrega los que siguen.
func (c *cicloEnCurso) desdeRelay() string {
	mayor := 0
	for _, m := range c.cola {
		ref, ok := strings.CutPrefix(m.Ref, "relay:")
		if !ok {
			continue
		}
		mayor = max(mayor, numeroRelay(ref))
	}
	if mayor == 0 {
		return ""
	}
	return fmt.Sprintf("REP-%d", mayor)
}

func (c *cicloEnCurso) leerRelay(f Fuentes) {
	if f.Relay == nil {
		return
	}
	lote, err := f.Relay(c.root, c.desdeRelay())
	if errors.Is(err, ErrSinRelay) {
		return
	}
	if err != nil {
		c.r.Avisos = append(c.r.Avisos, "el relay de reportes no se leyó: "+err.Error())
		return
	}
	for _, rep := range lote {
		ref := "relay:" + rep.ID
		if numeroRelay(rep.ID) == 0 || c.tiene(ref) {
			continue
		}
		m := Mejora{Titulo: recortar(strings.TrimSpace(rep.Titulo), maxTitulo), Detalle: recortar(strings.TrimSpace(rep.Detalle), maxDetalle), Fuente: FuenteRelay, Ref: ref,
			Autor: "QDD " + rep.Version + " · " + rep.SO, Etiquetas: []string{rep.Tipo}, Recibida: rep.Llego, Estado: PorAdmitir}
		if err := c.agregar(m); err != nil {
			c.r.Avisos = append(c.r.Avisos, err.Error())
		}
	}
}

// pedirAdmisiones deja en el panel, de la más antigua a la más nueva, hasta MaxAdmisiones pedidos para admitir.
func (c *cicloEnCurso) pedirAdmisiones() {
	pedidas := 0
	for _, m := range c.cola {
		if m.Estado != PorAdmitir || pedidas >= MaxAdmisiones {
			continue
		}
		motivo := recortar("📮 Admitir "+m.ID+": "+m.Titulo, consola.MaxMotivo)
		_, err := consola.PedirEnOrden(c.root, ComandoAdmitir(m.ID), motivo, "", consola.Orden{DePersona: true}, c.now)
		var ya *consola.YaHecho
		if errors.As(err, &ya) {
			err = nil
		}
		if err != nil {
			c.r.Avisos = append(c.r.Avisos, "no se pidió admitir "+m.ID+": "+err.Error())
			return
		}
		pedidas++
	}
}

// conCandado corre f con el candado de la cola: el ciclo y Admitir son sus únicos escritores.
func conCandado(root string, f func() error) error {
	ruta, err := estado.RutaLocal(root, "mejoras-ciclo")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	soltar, err := candado.Tomar(ruta+".lock", 5*time.Second, "el ciclo de mejoras")
	if err != nil {
		return err
	}
	defer soltar()
	return f()
}

// Admitir pasa a la cola una mejora del relay que esperaba a la persona. Quién la admite lo comprueba el comando.
func Admitir(root, id string, now time.Time) error {
	return conCandado(root, func() error {
		cola, err := Lista(root)
		if err != nil {
			return err
		}
		for _, m := range cola {
			if m.ID != id {
				continue
			}
			if m.Estado != PorAdmitir {
				return fmt.Errorf("%s no espera admisión (está %s)", id, m.Estado)
			}
			m.Estado = EnCola
			return escribir(m)
		}
		return fmt.Errorf("no hay una mejora %s en la cola", id)
	})
}
