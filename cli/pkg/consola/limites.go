package consola

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'
// Pedido de qdd-movile (2026-11-04): la persona quiere ver en la app cuánto queda de sus límites de Claude sin gastar
// tokens. Claude Code los entrega solo a la barra, así que la barra deja el último valor en <QDD_HOME>/claude/limites.json
// y qdd remoto lo publica. Es un dato de la máquina (una cuenta de Claude por PC), no de cada proyecto: solo lleva
// porcentajes y fechas, nunca rutas ni el modelo. Lo que no vino se omite, nunca un 0 inventado. La barra solo corre con
// una sesión abierta, así que el dato puede quedar viejo: por eso va con Actualizado.
// Cada barra sabe lo último que le dijeron a su sesión, y una inactiva tiene datos viejos (FND-130): el archivo no
// retrocede. En la misma ventana lo usado solo sube, una ventana más nueva manda y una que ya pasó no se guarda. Con dos
// cuentas en el mismo PC y la misma hora de reinicio se vería la mayor: la entrada de la barra no dice de qué cuenta es.

// CadaLimites es cada cuánto la barra reescribe los límites que no cambiaron, para decir que siguen vigentes: la barra
// corre cada 2 s y un cambio se escribe en el acto.
const CadaLimites = time.Minute

// holguraVentana: dos fechas de reinicio así de cerca son la misma ventana.
const holguraVentana = time.Minute

// TramoLimite es una ventana de uso: el used_percentage tal cual y cuándo se reinicia, si vino.
type TramoLimite struct {
	Usado    float64    `json:"usado"`
	Reinicia *time.Time `json:"reinicia,omitempty"`
}

// Limites son los límites de uso de Claude de la máquina, como los leyó la barra.
type Limites struct {
	CincoHoras  *TramoLimite `json:"cinco_horas,omitempty"`
	Semana      *TramoLimite `json:"semana,omitempty"`
	Actualizado time.Time    `json:"actualizado"`
}

// LimitesDe saca los límites de la entrada de la barra; nil si no trae ningún tramo o es ilegible.
func LimitesDe(datos []byte, now time.Time) *Limites {
	var in entrada
	if json.Unmarshal(datos, &in) != nil {
		return nil
	}
	cinco, semana := tramoLimite(in.RateLimits.FiveHour, now), tramoLimite(in.RateLimits.SevenDay, now)
	if cinco == nil && semana == nil {
		return nil
	}
	return &Limites{CincoHoras: cinco, Semana: semana, Actualizado: now.Truncate(time.Second)}
}

func tramoLimite(t tramoJSON, now time.Time) *TramoLimite {
	if t.UsedPercentage == nil {
		return nil
	}
	out := &TramoLimite{Usado: *t.UsedPercentage}
	if cuando, ok := renovacion(t.ResetsAt, now.Location()); ok {
		out.Reinicia = &cuando
	}
	return out
}

// Huella es lo que cuenta como cambio: lo usado y cuándo se reinicia, no Actualizado.
func (l *Limites) Huella() string {
	if l == nil {
		return ""
	}
	return huellaTramo(l.CincoHoras) + "|" + huellaTramo(l.Semana)
}

func huellaTramo(t *TramoLimite) string {
	if t == nil {
		return ""
	}
	if t.Reinicia == nil {
		return fmt.Sprint(t.Usado)
	}
	return fmt.Sprint(t.Usado, "@", t.Reinicia.Unix())
}

func rutaLimites() (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "claude", "limites.json"), nil
}

// GuardarLimites es lo que hace la barra en cada vuelta: guarda si cambió o si pasó CadaLimites. Una entrada sin límites
// no borra los últimos. Nunca falla: la barra no depende de esto.
func GuardarLimites(datos []byte, now time.Time) {
	l := LimitesDe(datos, now)
	if l == nil {
		return
	}
	ruta, err := rutaLimites()
	if err != nil {
		return
	}
	antes := LeerLimites()
	l = fundirLimites(antes, l, now)
	if l.CincoHoras == nil && l.Semana == nil {
		return
	}
	if antes != nil && antes.Huella() == l.Huella() && now.Sub(antes.Actualizado) < CadaLimites {
		return
	}
	if os.MkdirAll(filepath.Dir(ruta), 0o700) != nil {
		return
	}
	_ = guardarJSON(ruta, l)
}

func fundirLimites(antes, nuevo *Limites, now time.Time) *Limites {
	if antes == nil {
		antes = &Limites{}
	}
	return &Limites{
		CincoHoras:  fundirTramo(antes.CincoHoras, nuevo.CincoHoras, now),
		Semana:      fundirTramo(antes.Semana, nuevo.Semana, now),
		Actualizado: nuevo.Actualizado,
	}
}

func fundirTramo(antes, nuevo *TramoLimite, now time.Time) *TramoLimite {
	antes, nuevo = tramoVigente(antes, now), tramoVigente(nuevo, now)
	if antes == nil {
		return nuevo
	}
	if nuevo == nil {
		return antes
	}
	if nuevo.Reinicia == nil && antes.Reinicia != nil {
		return antes
	}
	if nuevo.Reinicia == nil || antes.Reinicia == nil {
		return nuevo
	}
	distancia := nuevo.Reinicia.Sub(*antes.Reinicia)
	if distancia > holguraVentana {
		return nuevo
	}
	if distancia < -holguraVentana || nuevo.Usado <= antes.Usado {
		return antes
	}
	return &TramoLimite{Usado: nuevo.Usado, Reinicia: antes.Reinicia}
}

// tramoVigente: una ventana que ya se reinició no dice nada de la actual. Sin fecha no se sabe, y se conserva.
func tramoVigente(t *TramoLimite, now time.Time) *TramoLimite {
	if t == nil || t.Reinicia == nil || t.Reinicia.After(now) {
		return t
	}
	return nil
}

// LeerLimites son los últimos límites que guardó la barra; nil si no hay o el archivo es ilegible.
func LeerLimites() *Limites {
	ruta, err := rutaLimites()
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil
	}
	var l Limites
	if json.Unmarshal(b, &l) != nil {
		return nil
	}
	return &l
}
