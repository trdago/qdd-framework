package baul

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// registro.jsonl solo agrega. Cada línea lleva la huella de la anterior: si alguien borra o cambia una, doctor lo ve.
// Nunca un valor.

// Linea es un uso del baúl: quién, dónde, qué token, para qué, qué comando y cómo terminó.
type Linea struct {
	En       time.Time `json:"en"`
	Quien    string    `json:"quien,omitempty"` // «persona» (su terminal) o la sesión de la IA
	Proyecto string    `json:"proyecto,omitempty"`
	Accion   string    `json:"accion"` // guardar, verificar, pedir, permitir, usar
	Token    string    `json:"token"`
	Fin      string    `json:"fin,omitempty"`
	Comando  string    `json:"comando,omitempty"`
	Codigo   *int      `json:"codigo,omitempty"`
	Previa   string    `json:"previa"`
	Huella   string    `json:"huella"`
}

func huellaDe(l Linea) string {
	l.Huella = ""
	b, _ := json.Marshal(l)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// Anotar agrega una línea encadenada a la última.
func Anotar(l Linea, ahora time.Time) error {
	return conBloqueo(func() error {
		lineas, err := leerRegistro()
		if err != nil {
			return err
		}
		l.En, l.Previa = ahora.UTC(), ""
		if len(lineas) > 0 {
			l.Previa = lineas[len(lineas)-1].Huella
		}
		l.Huella = huellaDe(l)
		b, err := json.Marshal(l)
		if err != nil {
			return err
		}
		ruta, err := rutaEn("registro.jsonl")
		if err != nil {
			return err
		}
		f, err := os.OpenFile(ruta, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(append(b, '\n'))
		return err
	})
}

// Registro devuelve las líneas del registro.
func Registro() ([]Linea, error) {
	return leerRegistro()
}

func leerRegistro() ([]Linea, error) {
	ruta, err := rutaEn("registro.jsonl")
	if err != nil {
		return nil, err
	}
	f, err := os.Open(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lineas []Linea
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var l Linea
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			return nil, fmt.Errorf("registro del baúl, línea %d ilegible: %v", n, err)
		}
		lineas = append(lineas, l)
	}
	return lineas, sc.Err()
}

// VerificarRegistro recorre la cadena y dice dónde se rompe.
func VerificarRegistro() error {
	lineas, err := leerRegistro()
	if err != nil {
		return err
	}
	previa := ""
	for i, l := range lineas {
		if l.Previa != previa || huellaDe(l) != l.Huella {
			return fmt.Errorf("el registro del baúl se rompe en la línea %d: falta una línea o alguien la cambió", i+1)
		}
		previa = l.Huella
	}
	return nil
}
