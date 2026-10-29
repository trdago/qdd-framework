package findings

import (
	"os"
	"path/filepath"
	"strings"
)

// PorDecidir devuelve los findings sin un estado reconocido. doctor los
// informa: el estado lo decide una persona, QDD no lo infiere (FND-061).
func PorDecidir(cwd string) []Finding {
	fs, _ := Leer(cwd)
	var out []Finding
	for _, f := range fs {
		if f.Estado == SinEstado {
			out = append(out, f)
		}
	}
	return out
}

// NoCanonicos devuelve los findings cuyo estado se reconoce pero no está
// escrito en su forma canónica (por ejemplo RESOLVED o closed).
func NoCanonicos(cwd string) []Finding {
	fs, _ := Leer(cwd)
	var out []Finding
	for _, f := range fs {
		if f.Estado != SinEstado && f.EstadoOriginal != string(f.Estado) {
			out = append(out, f)
		}
	}
	return out
}

// EscribirCanonicos reescribe la línea status de los findings no canónicos
// con la forma canónica y no toca el resto del archivo. Devuelve los archivos
// que cambió.
func EscribirCanonicos(cwd string) ([]string, error) {
	var cambiados []string
	for _, f := range NoCanonicos(cwd) {
		path := filepath.Join(Dir(cwd), f.Archivo)
		data, err := os.ReadFile(path)
		if err != nil {
			return cambiados, err
		}
		nuevo, ok := conEstado(string(data), f.Estado)
		if !ok {
			continue
		}
		if err := os.WriteFile(path, []byte(nuevo), 0644); err != nil {
			return cambiados, err
		}
		cambiados = append(cambiados, f.Archivo)
	}
	return cambiados, nil
}

// conEstado reemplaza la primera línea status de primer nivel. Las líneas
// indentadas pertenecen a otro campo (un bloque de texto, por ejemplo).
func conEstado(contenido string, e Estado) (string, bool) {
	lineas := strings.Split(contenido, "\n")
	for i, l := range lineas {
		if strings.HasPrefix(l, "status:") {
			lineas[i] = "status: " + string(e)
			return strings.Join(lineas, "\n"), true
		}
	}
	return contenido, false
}
