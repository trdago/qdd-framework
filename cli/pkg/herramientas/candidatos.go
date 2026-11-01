package herramientas

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// Rule: Prohibido usar 'else'

// Candidato es un ejecutable del proyecto que se puede agregar como herramienta.
type Candidato struct {
	Comando  []string `json:"comando"`
	Origen   string   `json:"origen"`   // bin, scripts, go, rust, npm o make
	Agregada bool     `json:"agregada"` // ya es una herramienta guardada
}

// Candidatos lista los ejecutables del proyecto: bin/ y scripts/, cmd/* de Go, target/release de Rust, los scripts de
// package.json y los objetivos del Makefile.
func Candidatos(root string) []Candidato {
	var out []Candidato
	for _, dir := range []string{"bin", "scripts"} {
		for _, f := range ejecutablesEn(filepath.Join(root, dir)) {
			out = append(out, Candidato{Comando: []string{"./" + dir + "/" + f}, Origen: dir})
		}
	}
	for _, d := range subcarpetasCon(filepath.Join(root, "cmd"), "main.go") {
		out = append(out, Candidato{Comando: []string{"go", "run", "./cmd/" + d}, Origen: "go"})
	}
	for _, f := range ejecutablesEn(filepath.Join(root, "target", "release")) {
		out = append(out, Candidato{Comando: []string{"./target/release/" + f}, Origen: "rust"})
	}
	for _, s := range scriptsDeNpm(root) {
		out = append(out, Candidato{Comando: []string{"npm", "run", s}, Origen: "npm"})
	}
	for _, o := range objetivosDeMake(root) {
		out = append(out, Candidato{Comando: []string{"make", o}, Origen: "make"})
	}
	guardadas := map[string]bool{}
	hs, _ := Cargar(root)
	for _, h := range hs {
		guardadas[strings.Join(h.Comando, "\x00")] = true
	}
	for i := range out {
		out[i].Agregada = guardadas[strings.Join(out[i].Comando, "\x00")]
	}
	return out
}

func ejecutablesEn(dir string) []string {
	entradas, _ := os.ReadDir(dir)
	var out []string
	for _, e := range entradas {
		info, err := e.Info()
		if err != nil || info.IsDir() || strings.HasPrefix(e.Name(), ".") || !ejecutable(info) {
			continue
		}
		out = append(out, e.Name())
	}
	return out
}

// ejecutable: en Windows por su extensión, en el resto por sus permisos.
func ejecutable(info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(info.Name()))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".ps1"
	}
	return info.Mode()&0111 != 0
}

func subcarpetasCon(dir, archivo string) []string {
	entradas, _ := os.ReadDir(dir)
	var out []string
	for _, e := range entradas {
		if _, err := os.Stat(filepath.Join(dir, e.Name(), archivo)); e.IsDir() && err == nil {
			out = append(out, e.Name())
		}
	}
	return out
}

func scriptsDeNpm(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}
	var out []string
	for s := range pkg.Scripts {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// objetivoDeMake: «nombre:» al comienzo de la línea; no «:=», ni objetivos especiales (.PHONY) ni patrones (%).
var objetivoDeMake = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_.-]*)\s*:([^=]|$)`)

func objetivosDeMake(root string) []string {
	f, err := os.Open(filepath.Join(root, "Makefile"))
	if err != nil {
		return nil
	}
	defer f.Close()
	vistos := map[string]bool{}
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		m := objetivoDeMake.FindStringSubmatch(sc.Text())
		if m == nil || vistos[m[1]] {
			continue
		}
		vistos[m[1]] = true
		out = append(out, m[1])
	}
	return out
}
