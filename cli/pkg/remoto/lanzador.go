package remoto

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
)

// Rule: Prohibido usar 'else'
// Sprint 9 (FND-238): la tarea «QDD remoto» de Windows lanzaba qdd.exe, un programa de consola, y Windows abría una
// ventana que la persona tenía que dejar abierta. El release de Windows compila antes qddw.exe (cli/qddw, programa de
// ventanas) en lanzador/, y qdd.exe lo lleva dentro: lo deja junto a sí al registrar o renovar la tarea, que lo lanza
// a él. Así el zip y npm/install.js no cambian, y 'qdd doctor --fix' lo repara sin descargar nada. Una compilación sin
// él (desarrollo, otros sistemas) deja la tarea como antes.

//go:embed all:lanzador
var lanzadorFS embed.FS

const (
	// NombreLanzador es qddw.exe, junto al qdd.exe de su versión.
	NombreLanzador = "qddw.exe"
	// EnvSinConsola lo pone qddw.exe: qdd no tiene consola y escribe su registro en un archivo (cli/qddw).
	EnvSinConsola = "QDD_SIN_CONSOLA"
)

// LanzadorEmbebido es qddw.exe tal como viene en este qdd; vacío si la compilación no lo trae. Las pruebas lo cambian.
var LanzadorEmbebido = leerLanzador()

func leerLanzador() []byte {
	b, _ := lanzadorFS.ReadFile("lanzador/" + NombreLanzador)
	return b
}

// RutaLanzador es dónde va qddw.exe para el qdd de bin, o vacío si este qdd no lo trae.
func RutaLanzador(bin string) string {
	if len(LanzadorEmbebido) == 0 {
		return ""
	}
	return filepath.Join(filepath.Dir(bin), NombreLanzador)
}

// lanzadorAlDia: no hace falta lanzador, o el que está es el de este qdd.
func (s Servicio) lanzadorAlDia() bool {
	if s.Lanzador == "" {
		return true
	}
	b, err := os.ReadFile(s.Lanzador)
	return err == nil && bytes.Equal(b, LanzadorEmbebido)
}

// dejarLanzador escribe qddw.exe si falta o es otro. Uno igual no se toca: puede estar corriendo.
func (s Servicio) dejarLanzador() error {
	if s.lanzadorAlDia() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.Lanzador), 0o755); err != nil {
		return err
	}
	nuevo := s.Lanzador + ".nuevo"
	if err := os.WriteFile(nuevo, LanzadorEmbebido, 0o755); err != nil {
		return err
	}
	return os.Rename(nuevo, s.Lanzador)
}
