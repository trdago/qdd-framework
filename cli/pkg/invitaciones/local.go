package invitaciones

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/qdd-framework/qdd/pkg/candado"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'

// Lo que este PC sabe de sus invitaciones vive fuera del repositorio, uno por proyecto: las invitaciones (sin su token,
// que solo va en el archivo del invitado), hasta qué propuesta leyó y qué pasó con cada una.

const (
	PorPedir = "por-pedir"
	Pedida   = "pedida"
)

// Invitacion es una invitación creada desde este proyecto.
type Invitacion struct {
	ID       string    `json:"id"`
	Flujo    string    `json:"flujo"`
	Rol      string    `json:"rol"`
	Para     string    `json:"para,omitempty"`
	Vence    time.Time `json:"vence"`
	Creada   time.Time `json:"creada"`
	Revocada bool      `json:"revocada,omitempty"`
}

// Viva dice si su token todavía sirve.
func (i Invitacion) Viva(ahora time.Time) bool {
	return !i.Revocada && i.Vence.After(ahora)
}

// Vista es una propuesta recibida y lo que este PC hizo con ella. Situacion es por-pedir, pedida, aprobada o
// rechazada; Razon, el motivo que verá el invitado; Avisada, que el relay ya lo sabe.
type Vista struct {
	Propuesta
	Situacion string `json:"situacion"`
	Pedido    string `json:"pedido,omitempty"`
	Razon     string `json:"razon,omitempty"`
	Avisada   bool   `json:"avisada,omitempty"`
}

// Local es el archivo de invitaciones de un proyecto.
type Local struct {
	Invitaciones  []Invitacion `json:"invitaciones"`
	Desde         string       `json:"desde,omitempty"`
	Propuestas    []Vista      `json:"propuestas,omitempty"`
	FotosPorSubir []string     `json:"fotos_por_subir,omitempty"`
	Ultimo        *time.Time   `json:"ultimo,omitempty"`
}

func rutaLocal(root string) (string, error) {
	return estado.RutaLocal(root, "flujo-invitaciones")
}

// Leer devuelve lo que este PC sabe de las invitaciones del proyecto.
func Leer(root string) (Local, error) {
	ruta, err := rutaLocal(root)
	if err != nil {
		return Local{}, err
	}
	return leerDe(ruta)
}

func leerDe(ruta string) (Local, error) {
	var l Local
	datos, err := os.ReadFile(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return l, err
	}
	return l, json.Unmarshal(datos, &l)
}

// conLocal cambia el archivo bajo candado y lo guarda aunque cambiar falle: lo leído del relay no se pierde.
func conLocal(root string, cambiar func(l *Local) error) error {
	ruta, err := rutaLocal(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	soltar, err := candado.Tomar(ruta+".lock", 10*time.Second, "las invitaciones del flujo")
	if err != nil {
		return err
	}
	defer soltar()
	l, err := leerDe(ruta)
	if err != nil {
		return err
	}
	errCambio := cambiar(&l)
	datos, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, datos, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, ruta); err != nil {
		return err
	}
	return errCambio
}

// Anotar guarda una invitación recién creada.
func Anotar(root string, inv Invitacion) error {
	return conLocal(root, func(l *Local) error {
		l.Invitaciones = append(l.Invitaciones, inv)
		return nil
	})
}

// MarcarRevocada anota que su token ya no sirve.
func MarcarRevocada(root, id string) error {
	return conLocal(root, func(l *Local) error {
		i := slices.IndexFunc(l.Invitaciones, func(inv Invitacion) bool { return inv.ID == id })
		if i < 0 {
			return errors.New("este proyecto no tiene esa invitación: 'qdd flujo invitaciones' dice cuáles hay")
		}
		l.Invitaciones[i].Revocada = true
		return nil
	})
}

func (l *Local) invitacion(id string) (Invitacion, bool) {
	i := slices.IndexFunc(l.Invitaciones, func(inv Invitacion) bool { return inv.ID == id })
	if i < 0 {
		return Invitacion{}, false
	}
	return l.Invitaciones[i], true
}

func (l *Local) vista(id string) *Vista {
	i := slices.IndexFunc(l.Propuestas, func(v Vista) bool { return v.ID == id })
	if i < 0 {
		return nil
	}
	return &l.Propuestas[i]
}

// Vivas dice si alguna invitación del flujo (o de cualquiera, con flujo vacío) todavía sirve.
func (l *Local) Vivas(flujo string, ahora time.Time) bool {
	return slices.ContainsFunc(l.Invitaciones, func(i Invitacion) bool {
		return i.Viva(ahora) && (flujo == "" || i.Flujo == flujo)
	})
}

func (l *Local) porAvisar() bool {
	if len(l.FotosPorSubir) > 0 {
		return true
	}
	return slices.ContainsFunc(l.Propuestas, func(v Vista) bool { return resuelta(v) && !v.Avisada })
}

func resuelta(v Vista) bool {
	return v.Situacion == Aprobada || v.Situacion == Rechazada
}

func (l *Local) fotoPorSubir(flujo string) {
	if !slices.Contains(l.FotosPorSubir, flujo) {
		l.FotosPorSubir = append(l.FotosPorSubir, flujo)
	}
}
