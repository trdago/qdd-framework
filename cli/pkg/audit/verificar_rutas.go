package audit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// RFC 0020: un build de Docker solo ve lo que se le copió, así que no se le olvida una dependencia. VerificarRutas hace lo
// mismo: corre cada suite que declara rutas en una copia que solo tiene esos archivos. Si pasa, la huella no se pierde nada.

// EstadoRutas es lo que se sabe de las rutas de una suite.
type EstadoRutas string

const (
	RutasCompletas    EstadoRutas = "completas"
	RutasIncompletas  EstadoRutas = "incompletas"
	RutasSinComprobar EstadoRutas = "sin comprobar"
	SuiteFalla        EstadoRutas = "la suite falla"
)

// ResultadoRutas es el resultado de una suite.
type ResultadoRutas struct {
	Suite   string
	Estado  EstadoRutas
	Detalle string
}

// VerificarRutas comprueba cada suite con rutas. Las demás no se tocan.
func VerificarRutas(cwd string) []ResultadoRutas {
	resultados := []ResultadoRutas{}
	for _, tc := range resolveTestCommands(cwd) {
		if len(tc.Rutas) == 0 {
			continue
		}
		resultados = append(resultados, verificarSuite(tc))
	}
	return resultados
}

func verificarSuite(tc testCommand) ResultadoRutas {
	r := ResultadoRutas{Suite: tc.Label}
	if falta := programasQueFaltan(tc.Requiere); len(falta) > 0 {
		r.Estado, r.Detalle = RutasSinComprobar, "falta "+strings.Join(falta, ", ")+" en esta máquina"
		return r
	}
	copia, err := os.MkdirTemp("", "qdd-rutas-")
	if err != nil {
		r.Estado, r.Detalle = RutasSinComprobar, err.Error()
		return r
	}
	defer os.RemoveAll(copia)
	for _, ruta := range tc.Rutas {
		archivos := archivosDeRuta(tc.Dir, ruta)
		if len(archivos) == 0 {
			r.Estado, r.Detalle = RutasIncompletas, "la ruta "+ruta+" no tiene archivos"
			return r
		}
		if err := copiarArchivos(tc.Dir, copia, archivos); err != nil {
			r.Estado, r.Detalle = RutasSinComprobar, err.Error()
			return r
		}
	}
	aislada := tc
	aislada.Dir = copia
	salida, err := salidaDeSuite(aislada)
	if err == nil {
		r.Estado = RutasCompletas
		return r
	}
	// Solo si falla aislada vale la pena repetirla en el árbol: una suite rota no dice nada de sus rutas.
	if _, err := salidaDeSuite(tc); err != nil {
		r.Estado, r.Detalle = SuiteFalla, "falla también en el árbol: arréglala primero"
		return r
	}
	return clasificarFalloAislado(r, salida)
}

var moduloQueFalta = regexp.MustCompile(`Cannot find (?:module|package) ['"]([^'"]+)['"]`)

// clasificarFalloAislado separa lo que falta declarar de lo que falta instalar. Un módulo relativo o absoluto es un archivo
// del proyecto; uno con nombre viene de lo instalado, que la copia no trae.
func clasificarFalloAislado(r ResultadoRutas, salida string) ResultadoRutas {
	resumen := resumenDeSalida(salida)
	for _, m := range moduloQueFalta.FindAllStringSubmatch(salida, -1) {
		if strings.HasPrefix(m[1], ".") || strings.HasPrefix(m[1], "/") {
			r.Estado, r.Detalle = RutasIncompletas, resumen
			return r
		}
	}
	for _, senal := range señalesDeDependencias {
		if strings.Contains(salida, senal) {
			r.Estado, r.Detalle = RutasSinComprobar, "necesita lo instalado, que la copia no trae: "+resumen
			return r
		}
	}
	r.Estado, r.Detalle = RutasIncompletas, resumen
	return r
}

func resumenDeSalida(salida string) string {
	lineas := strings.Split(strings.TrimSpace(salida), "\n")
	if len(lineas) > 3 {
		lineas = lineas[len(lineas)-3:]
	}
	return strings.Join(lineas, " | ")
}

// copiarArchivos copia conservando la ruta, los permisos y los enlaces. Un archivo borrado del árbol no se copia.
func copiarArchivos(origen, destino string, archivos []string) error {
	for _, archivo := range archivos {
		desde := filepath.Join(origen, archivo)
		hacia := filepath.Join(destino, archivo)
		info, err := os.Lstat(desde)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(hacia), 0o755); err != nil {
			return err
		}
		if err := copiarUno(desde, hacia, info); err != nil {
			return err
		}
	}
	return nil
}

func copiarUno(desde, hacia string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		enlace, err := os.Readlink(desde)
		if err != nil {
			return err
		}
		return os.Symlink(enlace, hacia)
	}
	contenido, err := os.ReadFile(desde)
	if err != nil {
		return err
	}
	return os.WriteFile(hacia, contenido, info.Mode().Perm())
}

// InformeDeRutas escribe el resultado y dice si hay algo que arreglar. «Sin comprobar» no falla: no se sabe.
func InformeDeRutas(salida io.Writer, resultados []ResultadoRutas) bool {
	if len(resultados) == 0 {
		fmt.Fprintln(salida, "[i] Ninguna suite declara rutas en test_commands.")
		return false
	}
	falla := false
	for _, r := range resultados {
		marca := "[✓]"
		if r.Estado == RutasSinComprobar {
			marca = "[i]"
		}
		if r.Estado == RutasIncompletas || r.Estado == SuiteFalla {
			marca = "[✗]"
			falla = true
		}
		fmt.Fprintf(salida, "%s %s: %s\n", marca, r.Suite, r.Estado)
		if r.Detalle != "" {
			fmt.Fprintf(salida, "    %s\n", r.Detalle)
		}
	}
	return falla
}
