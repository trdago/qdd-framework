package audit

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Rule: Prohibido usar 'else'
// RunFormatoCheck revisa el formato con el formateador que usa el proyecto, como lo hace su CI (MEJ-061, FND-203): un commit
// que pasa el gate no puede caer en el CI por formato. Go se revisa siempre con gofmt; los demás, solo si el proyecto los
// declara (su configuración, o el CI, el Makefile o pre-commit que los corren). Sin el formateador en esta máquina queda
// «sin medir»: nunca se informa un formato como bueno ni como malo sin haberlo revisado.

const (
	ReglaFormato         = "FMT-01-FORMATO"
	ReglaFormatoSinMedir = "FMT-02-FORMATO-SIN-MEDIR"
)

// binGofmt y binDart son los programas del PATH; las pruebas los cambian.
var (
	binGofmt = "gofmt"
	binDart  = "dart"
)

// archivosPorMensaje acota la lista de archivos de una violación: el resto se cuenta.
const archivosPorMensaje = 10

// formateador es uno de los que el proyecto puede declarar, sin contar gofmt.
type formateador struct {
	nombre   string
	aplica   func(dir string) bool
	declara  func(dir, ci string) bool
	programa func(dir string) string
	args     []string
	arreglo  string
	archivos func(salida string) []string
}

var formateadores = []formateador{
	{
		nombre:   "prettier",
		aplica:   func(dir string) bool { return fileExists(filepath.Join(dir, "package.json")) },
		declara:  prettierDeclarado,
		programa: func(dir string) string { return deNodeModules(dir, "prettier") },
		args:     []string{"--check", "."},
		arreglo:  "npx prettier --write .",
		archivos: lineasConPrefijo(`^\[warn\] (.+)$`),
	},
	{
		nombre:   "biome",
		aplica:   func(dir string) bool { return fileExists(filepath.Join(dir, "package.json")) },
		declara:  func(dir, _ string) bool { return algunoExiste(dir, "biome.json", "biome.jsonc") },
		programa: func(dir string) string { return deNodeModules(dir, "biome") },
		args:     []string{"format", "."},
		arreglo:  "npx biome format --write .",
		archivos: lineasConPrefijo(`^(\S+\.\w+) format\b`),
	},
	{
		nombre:   "ruff format",
		aplica:   detectaPython,
		declara:  ruffFormatDeclarado,
		programa: func(dir string) string { return dePython(dir, "ruff") },
		args:     []string{"format", "--check", "--no-cache", "."}, // sin .ruff_cache en el proyecto
		arreglo:  "ruff format .",
		archivos: lineasConPrefijo(`^Would reformat: (.+)$`),
	},
	{
		nombre:   "black",
		aplica:   detectaPython,
		declara:  blackDeclarado,
		programa: func(dir string) string { return dePython(dir, "black") },
		args:     []string{"--check", "."},
		arreglo:  "black .",
		archivos: lineasConPrefijo(`^would reformat (.+)$`),
	},
	{
		nombre:   "dart format",
		aplica:   func(dir string) bool { return fileExists(filepath.Join(dir, "pubspec.yaml")) },
		declara:  func(_, ci string) bool { return strings.Contains(ci, "dart format") },
		programa: func(string) string { return binDart },
		args:     []string{"format", "--output=none", "--set-exit-if-changed", "."},
		arreglo:  "dart format .",
		archivos: lineasConPrefijo(`^Changed (.+)$`),
	},
}

func RunFormatoCheck(cwd string) []Violation {
	violations := revisarGofmt(cwd)
	ci := textoDelCI(cwd)
	for _, f := range formateadores {
		for _, dir := range carpetasQueAplican(cwd, f.aplica) {
			if !f.declara(dir, ci+textoDelCI(dir)) {
				continue
			}
			violations = append(violations, correrFormateador(cwd, dir, f)...)
		}
	}
	return violations
}

// revisarGofmt revisa todos los .go del proyecto, salvo los que Go mismo no compila como parte de él (testdata, vendor) y
// lo que walkProject salta (lo que git ignora y los repos anidados).
func revisarGofmt(cwd string) []Violation {
	goes := archivosGo(cwd)
	if len(goes) == 0 {
		return nil
	}
	if _, err := exec.LookPath(binGofmt); err != nil {
		return []Violation{sinMedirFormato("gofmt", "no está en el PATH (viene con Go)")}
	}
	var marcados []string
	for lote := range slices.Chunk(goes, 200) {
		cmd := exec.Command(binGofmt, append([]string{"-l"}, lote...)...)
		cmd.Dir = cwd
		// gofmt sale con 2 si un archivo no compila: eso lo ven las pruebas; aquí cuenta lo que lista.
		salida, _ := cmd.Output()
		marcados = append(marcados, lineasNoVacias(string(salida))...)
	}
	if len(marcados) == 0 {
		return nil
	}
	for i, m := range marcados {
		marcados[i] = filepath.ToSlash(m)
	}
	arreglo := "gofmt -w <archivos>"
	if len(marcados) <= archivosPorMensaje {
		arreglo = "gofmt -w " + strings.Join(marcados, " ")
	}
	return []Violation{formatoMalo("gofmt", arreglo, marcados)}
}

var carpetasQueGoNoCompila = map[string]bool{"testdata": true, "vendor": true, "node_modules": true}

func archivosGo(cwd string) []string {
	var goes []string
	_ = walkProject(cwd, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && carpetasQueGoNoCompila[d.Name()] {
			return filepath.SkipDir
		}
		if d.IsDir() || filepath.Ext(ruta) != ".go" {
			return nil
		}
		relativa, err := filepath.Rel(cwd, ruta)
		if err != nil {
			return nil
		}
		goes = append(goes, relativa)
		return nil
	})
	return goes
}

// carpetasQueAplican es la raíz si aplica; si no, las subcarpetas de primer nivel que aplican (frontend/, backend/…), como
// las suites de pruebas.
func carpetasQueAplican(cwd string, aplica func(string) bool) []string {
	if aplica(cwd) {
		return []string{cwd}
	}
	var dirs []string
	for _, sub := range subcarpetasDeCodigo(cwd) {
		if aplica(filepath.Join(cwd, sub)) {
			dirs = append(dirs, filepath.Join(cwd, sub))
		}
	}
	return dirs
}

func correrFormateador(cwd, dir string, f formateador) []Violation {
	programa := f.programa(dir)
	if programa == "" {
		return []Violation{sinMedirFormato(f.nombre, "el proyecto lo usa pero no está instalado en "+relativaA(cwd, dir)+" (instala las dependencias)")}
	}
	cmd := exec.Command(programa, f.args...)
	cmd.Dir = dir
	var salida bytes.Buffer
	cmd.Stdout, cmd.Stderr = &salida, &salida
	err := cmd.Run()
	if err == nil {
		return nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		return []Violation{sinMedirFormato(f.nombre, "el proyecto lo usa pero no está en el PATH")}
	}
	marcados := f.archivos(salida.String())
	if len(marcados) == 0 {
		return []Violation{sinMedirFormato(f.nombre, "no pudo revisar: "+primeraLinea(salida.String()))}
	}
	for i, m := range marcados {
		marcados[i] = relativaA(cwd, filepath.Join(dir, m))
		if filepath.IsAbs(m) {
			marcados[i] = relativaA(cwd, m)
		}
	}
	arreglo := f.arreglo
	if dir != cwd {
		arreglo = "cd " + relativaA(cwd, dir) + " && " + arreglo
	}
	return []Violation{formatoMalo(f.nombre, arreglo, marcados)}
}

func formatoMalo(nombre, arreglo string, archivos []string) Violation {
	slices.Sort(archivos)
	lista := archivos
	resto := ""
	if len(lista) > archivosPorMensaje {
		lista = lista[:archivosPorMensaje]
		resto = fmt.Sprintf(" y %d más", len(archivos)-archivosPorMensaje)
	}
	return Violation{
		Category:    "FORMATO",
		RuleID:      ReglaFormato,
		Description: fmt.Sprintf("%s marca %d archivo(s) sin formato: %s%s. El CI los rechazaría; arréglalo con %s.", nombre, len(archivos), strings.Join(lista, ", "), resto, arreglo),
	}
}

func sinMedirFormato(nombre, motivo string) Violation {
	return Violation{
		Category:      "FORMATO",
		RuleID:        ReglaFormatoSinMedir,
		Description:   fmt.Sprintf("Formato sin medir: %s %s. El CI puede rechazar lo que aquí no se revisó.", nombre, motivo),
		Recomendacion: true,
	}
}

// textoDelCI junta lo que corre el proyecto fuera del gate (CI, Makefile, pre-commit) para saber qué formateador usa.
func textoDelCI(dir string) string {
	archivos, _ := filepath.Glob(filepath.Join(dir, ".github", "workflows", "*.y*ml"))
	archivos = append(archivos, filepath.Join(dir, ".gitlab-ci.yml"), filepath.Join(dir, "Makefile"), filepath.Join(dir, ".pre-commit-config.yaml"))
	var texto strings.Builder
	for _, a := range archivos {
		b, err := os.ReadFile(a)
		if err != nil {
			continue
		}
		texto.Write(b)
		texto.WriteByte('\n')
	}
	return texto.String()
}

var archivosDePrettier = []string{".prettierrc", ".prettierrc.json", ".prettierrc.yml", ".prettierrc.yaml", ".prettierrc.json5", ".prettierrc.js", ".prettierrc.cjs", ".prettierrc.mjs", ".prettierrc.toml", "prettier.config.js", "prettier.config.cjs", "prettier.config.mjs", "prettier.config.ts"}

// prettierDeclarado: su configuración, prettier en package.json (dependencia o clave «prettier») o el CI que lo corre.
func prettierDeclarado(dir, ci string) bool {
	if algunoExiste(dir, archivosDePrettier...) || strings.Contains(ci, "prettier") {
		return true
	}
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	return err == nil && strings.Contains(string(b), `"prettier"`)
}

// ruffFormatDeclarado: ruff también es solo un linter; cuenta como formateador si se configura su formato o algo lo corre.
func ruffFormatDeclarado(dir, ci string) bool {
	if strings.Contains(ci, "ruff format") || strings.Contains(ci, "ruff-format") {
		return true
	}
	if archivoContiene(filepath.Join(dir, "pyproject.toml"), "[tool.ruff.format]") {
		return true
	}
	return archivoContiene(filepath.Join(dir, "ruff.toml"), "[format]") || archivoContiene(filepath.Join(dir, ".ruff.toml"), "[format]")
}

var blackEnElCI = regexp.MustCompile(`(?m)(psf/black|\bblack\s+(--check|\.))`)

func blackDeclarado(dir, ci string) bool {
	return archivoContiene(filepath.Join(dir, "pyproject.toml"), "[tool.black]") || blackEnElCI.MatchString(ci)
}

// deNodeModules busca el programa en node_modules/.bin de la carpeta o de la raíz de un workspace; nunca usa npx, que
// podría bajarlo de la red.
func deNodeModules(dir, programa string) string {
	for _, d := range []string{dir, filepath.Dir(dir)} {
		ruta := filepath.Join(d, "node_modules", ".bin", programa)
		if fileExists(ruta) {
			return ruta
		}
	}
	return ""
}

// dePython prefiere el entorno virtual del proyecto y si no, el PATH.
func dePython(dir, programa string) string {
	for _, venv := range []string{".venv", "venv"} {
		ruta := filepath.Join(dir, venv, "bin", programa)
		if fileExists(ruta) {
			return ruta
		}
	}
	ruta, err := exec.LookPath(programa)
	if err != nil {
		return ""
	}
	return ruta
}

func lineasConPrefijo(patron string) func(string) []string {
	re := regexp.MustCompile(patron)
	return func(salida string) []string {
		var archivos []string
		for _, l := range lineasNoVacias(salida) {
			m := re.FindStringSubmatch(l)
			if m == nil || strings.Contains(m[1], " ") {
				continue
			}
			archivos = append(archivos, m[1])
		}
		return archivos
	}
}

func lineasNoVacias(texto string) []string {
	var lineas []string
	for _, l := range strings.Split(texto, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		lineas = append(lineas, l)
	}
	return lineas
}

func primeraLinea(texto string) string {
	lineas := lineasNoVacias(texto)
	if len(lineas) == 0 {
		return "terminó con error y sin salida"
	}
	return lineas[0]
}

func relativaA(cwd, ruta string) string {
	relativa, err := filepath.Rel(cwd, ruta)
	if err != nil {
		return ruta
	}
	return filepath.ToSlash(relativa)
}

func algunoExiste(dir string, nombres ...string) bool {
	for _, n := range nombres {
		if fileExists(filepath.Join(dir, n)) {
			return true
		}
	}
	return false
}

func archivoContiene(archivo, texto string) bool {
	b, err := os.ReadFile(archivo)
	return err == nil && strings.Contains(string(b), texto)
}
