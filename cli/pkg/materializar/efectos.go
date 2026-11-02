package materializar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// EnvEfectos=off apaga los efectos del panel sin tocar el archivo.
const EnvEfectos = "QDD_EFECTOS"

// Efectos es lo que el panel necesita saber: si materializa y con qué estilo.
type Efectos struct {
	Activo bool `json:"activo"`
	Estilo
	CuadroMs int `json:"cuadro_ms"`
}

type archivo struct {
	Activo bool `json:"activo"`
}

// RutaEfectos es el interruptor del usuario, uno para todos sus proyectos: <QDD_HOME>/claude/efectos.json.
func RutaEfectos(qddHome string) string {
	return filepath.Join(qddHome, "claude", "efectos.json")
}

// Leer junta el interruptor (en qddHome) y el estilo (de la configuración del usuario en home).
func Leer(qddHome, home string) Efectos {
	return Efectos{Activo: Activos(qddHome), Estilo: EstiloDe(home), CuadroMs: CuadroMs}
}

// Activos dice si el panel materializa: sí por defecto; no con QDD_EFECTOS=off o 'qdd consola efectos off'. Un archivo
// ilegible no los apaga en silencio: doctor lo dice y lo repara.
func Activos(qddHome string) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv(EnvEfectos)), "off") {
		return false
	}
	a, err := leer(qddHome)
	if err != nil {
		return true
	}
	return a.Activo
}

// Guardar deja el interruptor encendido o apagado.
func Guardar(qddHome string, activo bool) error {
	ruta := RutaEfectos(qddHome)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(archivo{Activo: activo})
	if err != nil {
		return err
	}
	return os.WriteFile(ruta, append(b, '\n'), 0o644)
}

// Revisar es lo que mira doctor: el interruptor se puede leer (o no existe, que es encendido).
func Revisar(qddHome string) error {
	_, err := leer(qddHome)
	return err
}

// Reparar deja legible un interruptor roto, encendido (el valor por defecto).
func Reparar(qddHome string) error {
	if Revisar(qddHome) == nil {
		return nil
	}
	return Guardar(qddHome, true)
}

func leer(qddHome string) (archivo, error) {
	b, err := os.ReadFile(RutaEfectos(qddHome))
	if errors.Is(err, fs.ErrNotExist) {
		return archivo{Activo: true}, nil
	}
	if err != nil {
		return archivo{}, err
	}
	var a archivo
	if err := json.Unmarshal(b, &a); err != nil {
		return archivo{}, fmt.Errorf("%s no se puede leer: %w", RutaEfectos(qddHome), err)
	}
	return a, nil
}
