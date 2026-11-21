package puertas

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Fila es el informe de una puerta (criterio de éxito de la ficha): Corregidos/ConReintento es el % corregido al
// primer reintento; un rechazo sin otro intento de la misma puerta en la misma sesión no cuenta para ese %.
type Fila struct {
	Puerta           string  `json:"puerta"`
	Decisiones       int     `json:"decisiones"`
	Rechazos         int     `json:"rechazos"`
	Errores          int     `json:"errores"`
	ConReintento     int     `json:"con_reintento"`
	Corregidos       int     `json:"corregidos"`
	Falsos           int     `json:"falsos"`
	MSPromedio       float64 `json:"ms_promedio"`
	MSMax            int64   `json:"ms_max"`
	TokensPorRechazo float64 `json:"tokens_por_rechazo"`
}

// marca es una línea del registro que marca un rechazo como falso positivo: no es una decisión.
type marca struct {
	Fecha  time.Time `json:"fecha"`
	Falso  string    `json:"falso"`
	Motivo string    `json:"motivo,omitempty"`
}

var ordenPuertas = []string{PuertaFicha, PuertaAudit, PuertaFinding, PuertaPublicar, PuertaSalto}

// Resumir arma una fila por puerta, en el orden de la ficha. falsos: id del rechazo → motivo.
func Resumir(regs []Registro, falsos map[string]string) []Fila {
	regs = slices.Clone(regs)
	slices.SortStableFunc(regs, func(a, b Registro) int { return a.Fecha.Compare(b.Fecha) })
	filas := map[string]*Fila{}
	for i, r := range regs {
		f := filas[r.Puerta]
		if f == nil {
			f = &Fila{Puerta: r.Puerta}
			filas[r.Puerta] = f
		}
		sumar(f, r, falsos)
		contarReintento(f, r, regs[i+1:])
	}
	var out []Fila
	for _, p := range ordenPuertas {
		if f := filas[p]; f != nil {
			out = append(out, cerrar(*f))
		}
	}
	return out
}

func sumar(f *Fila, r Registro, falsos map[string]string) {
	f.Decisiones++
	f.MSPromedio += float64(r.MS)
	f.MSMax = max(f.MSMax, r.MS)
	if r.Resultado == Error {
		f.Errores++
	}
	if r.Resultado != Rechaza {
		return
	}
	f.Rechazos++
	f.TokensPorRechazo += float64(r.Largo) / 4
	if _, ok := falsos[r.ID]; ok {
		f.Falsos++
	}
}

// contarReintento: el siguiente intento de la misma puerta en la misma sesión dice si el rechazo se corrigió.
func contarReintento(f *Fila, r Registro, despues []Registro) {
	if r.Resultado != Rechaza || r.Sesion == "" {
		return
	}
	i := slices.IndexFunc(despues, func(s Registro) bool { return s.Puerta == r.Puerta && s.Sesion == r.Sesion })
	if i < 0 {
		return
	}
	f.ConReintento++
	if despues[i].Resultado == Pasa {
		f.Corregidos++
	}
}

func cerrar(f Fila) Fila {
	f.MSPromedio /= float64(f.Decisiones)
	if f.Rechazos > 0 {
		f.TokensPorRechazo /= float64(f.Rechazos)
	}
	return f
}

// LeerRegistro lee las decisiones y las marcas de falso positivo (id → motivo) de un proyecto. Sin registro, vacío.
func LeerRegistro(home, proyecto string) ([]Registro, map[string]string, error) {
	falsos := map[string]string{}
	f, err := os.Open(rutaRegistro(home, proyecto))
	if errors.Is(err, os.ErrNotExist) {
		return nil, falsos, nil
	}
	if err != nil {
		return nil, falsos, err
	}
	defer f.Close()
	var regs []Registro
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var m marca
		if json.Unmarshal(sc.Bytes(), &m) == nil && m.Falso != "" {
			falsos[m.Falso] = m.Motivo
			continue
		}
		var r Registro
		if json.Unmarshal(sc.Bytes(), &r) == nil && r.Puerta != "" {
			regs = append(regs, r)
		}
	}
	return regs, falsos, sc.Err()
}

// MarcarFalso anota que el rechazo id fue un falso positivo.
func MarcarFalso(home, proyecto, id, motivo string) error {
	linea, err := json.Marshal(marca{Fecha: time.Now(), Falso: id, Motivo: motivo})
	if err != nil {
		return err
	}
	return agregarLinea(home, proyecto, linea)
}

// UltimoRechazo es el último rechazo de la sesión ("" es de cualquiera).
func UltimoRechazo(regs []Registro, sesion string) (Registro, bool) {
	for i := len(regs) - 1; i >= 0; i-- {
		r := regs[i]
		if r.Resultado == Rechaza && (sesion == "" || r.Sesion == sesion) {
			return r, true
		}
	}
	return Registro{}, false
}

func rutaRegistro(home, proyecto string) string {
	return filepath.Join(home, "puertas", filepath.Base(proyecto)+".jsonl")
}

// idDe es un id corto y estable de una decisión.
func idDe(r Registro) string {
	h := fnv.New32a()
	fmt.Fprintf(h, "%d|%s|%s|%s", r.Fecha.UnixNano(), r.Puerta, r.Sesion, r.Resultado)
	return fmt.Sprintf("%08x", h.Sum32())
}
