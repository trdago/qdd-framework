package consola

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.7: las acciones del panel las decide qdd. 'qdd consola panel --json' las ofrece y 'qdd consola accion <id>'
// las vuelve a calcular antes de ejecutar una; el panel solo pasa ids (blueprint consola-claude-code v3.0.0).

const (
	TipoAprobar = "aprobar"
	TipoComando = "comando"

	// MaxFichas es el tope de fichas en borrador que se ofrecen: una tecla 1…9 por ficha.
	MaxFichas = 9

	prefijoAprobar = "aprobar:"
)

// Accion es un botón del panel. Args son los argumentos de qdd para un comando; no salen en el JSON.
type Accion struct {
	ID        string   `json:"id"`
	Tipo      string   `json:"tipo"`
	Etiqueta  string   `json:"etiqueta"`
	Detalle   string   `json:"detalle"`
	Confirmar bool     `json:"confirmar"`
	Args      []string `json:"-"`
	ficha     string
}

// DatosConsola es lo que 'qdd consola panel --json' entrega al panel en una sola lectura.
type DatosConsola struct {
	Formato  int                `json:"formato"`
	Estado   *estado.DatosPanel `json:"estado"`
	Motivo   string             `json:"motivo,omitempty"`
	Acciones []Accion           `json:"acciones"`
}

// Panel arma los datos del panel: el estado por frente (o el motivo por el que no hay) y las acciones de ahora.
func Panel(root string, now time.Time) DatosConsola {
	d := DatosConsola{Formato: 1}
	e, err := estado.Load(root)
	if errors.Is(err, os.ErrNotExist) {
		d.Motivo = estado.ErrSinArchivo.Error()
		d.Acciones = Acciones(root, false)
		return d
	}
	if err != nil {
		d.Motivo = err.Error()
		d.Acciones = Acciones(root, false)
		return d
	}
	datos := estado.Datos(e, now)
	d.Estado = &datos
	d.Acciones = Acciones(root, true)
	return d
}

// Acciones lista lo que el panel puede pedir: aprobar cada ficha en borrador (hasta MaxFichas, por id), revisar el
// estado si hay estado.yaml, y auditar si la carpeta es un proyecto QDD (tiene .qdd/).
func Acciones(root string, hayEstado bool) []Accion {
	acciones := []Accion{}
	for _, m := range borradores(root) {
		acciones = append(acciones, Accion{
			ID: prefijoAprobar + m.ID, Tipo: TipoAprobar, Etiqueta: "Aprobar «" + m.Nombre + "»",
			Detalle: "qdd modules approve " + m.ID, Confirmar: true, ficha: m.ID,
		})
	}
	if hayEstado {
		acciones = append(acciones, Accion{
			ID: "revisar-estado", Tipo: TipoComando, Etiqueta: "Revisar estado", Detalle: "qdd estado check",
			Args: []string{"estado", "check"},
		})
	}
	if _, err := os.Stat(filepath.Join(root, ".qdd")); err != nil {
		return acciones
	}
	return append(acciones, Accion{
		ID: "auditar", Tipo: TipoComando, Etiqueta: "Auditar", Detalle: "qdd audit", Confirmar: true,
		Args: []string{"audit"},
	})
}

// HayEstado dice si el proyecto tiene .qdd/project/estado.yaml.
func HayEstado(root string) bool {
	_, err := os.Stat(filepath.Join(root, estado.Ruta))
	return err == nil
}

func borradores(root string) []modules.Module {
	mods, _ := modules.Load(root)
	var out []modules.Module
	for _, m := range mods {
		if m.Estado == modules.EstadoBorrador {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > MaxFichas {
		return out[:MaxFichas]
	}
	return out
}

// BuscarAccion devuelve la acción con ese id; un id que qdd no ofrece ahora es un error.
func BuscarAccion(acciones []Accion, id string) (Accion, error) {
	for _, a := range acciones {
		if a.ID == id {
			return a, nil
		}
	}
	return Accion{}, fmt.Errorf("la acción '%s' no está entre las que ofrece 'qdd consola panel --json'", id)
}

// Autorizar exige el segundo paso de la persona para lo que escribe.
func Autorizar(a Accion, confirmado bool) error {
	if a.Confirmar && !confirmado {
		return fmt.Errorf("'%s' escribe en el proyecto: se ejecuta solo con --confirmado (el segundo paso del panel)", a.ID)
	}
	return nil
}

// Ejecutar corre la acción en la raíz del proyecto y devuelve su código de salida. Aprobar usa el mismo código que
// 'qdd modules approve'; un comando se ejecuta con bin (el binario de qdd en uso).
func (a Accion) Ejecutar(root, bin string, salida io.Writer) int {
	if a.Tipo == TipoAprobar {
		return aprobar(root, a.ficha, salida)
	}
	c := exec.Command(bin, a.Args...)
	c.Dir = root
	c.Stdout = salida
	c.Stderr = salida
	err := c.Run()
	var salio *exec.ExitError
	if errors.As(err, &salio) {
		return salio.ExitCode()
	}
	if err != nil {
		fmt.Fprintf(salida, "[!] no se pudo ejecutar %s: %v\n", a.Detalle, err)
		return 1
	}
	return 0
}

func aprobar(root, id string, salida io.Writer) int {
	m, problemas := modules.Approve(root, id)
	if len(problemas) > 0 {
		fmt.Fprintf(salida, "[!] No se pudo aprobar '%s':\n  - %s\n", id, strings.Join(problemas, "\n  - "))
		return 1
	}
	fmt.Fprintf(salida, "[+] Ficha '%s' aprobada (revisada en %s).\n", m.ID, m.RevisadoEnCommit)
	return 0
}
