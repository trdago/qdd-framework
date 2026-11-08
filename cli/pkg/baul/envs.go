package baul

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// doctor avisa de los .env con valores reales para pasarlos al baúl. Lista solo los nombres de las claves; nunca lee un
// valor más allá de saber si está vacío.

// EnvConValores es un .env con claves de secretos que tienen valor.
type EnvConValores struct {
	Archivo string
	Claves  []string
}

var (
	lineaEnv     = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$`)
	claveSecreto = regexp.MustCompile(`(?i)(TOKEN|SECRET|PASSWORD|PASSWD|API_?KEY|_KEY$|PRIVATE)`)
	deEjemplo    = regexp.MustCompile(`(?i)\.(example|sample|template|dist|defaults?)$`)
)

// EnvsConValores revisa los .env de la raíz del proyecto.
func EnvsConValores(root string) []EnvConValores {
	archivos, _ := filepath.Glob(filepath.Join(root, ".env*"))
	var out []EnvConValores
	for _, ruta := range archivos {
		if deEjemplo.MatchString(ruta) {
			continue
		}
		if info, err := os.Stat(ruta); err != nil || info.IsDir() {
			continue
		}
		claves := clavesConValor(ruta)
		if len(claves) == 0 {
			continue
		}
		out = append(out, EnvConValores{Archivo: filepath.Base(ruta), Claves: claves})
	}
	return out
}

func clavesConValor(ruta string) []string {
	f, err := os.Open(ruta)
	if err != nil {
		return nil
	}
	defer f.Close()
	var claves []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		m := lineaEnv.FindStringSubmatch(sc.Text())
		if m == nil || !claveSecreto.MatchString(m[1]) {
			continue
		}
		if strings.Trim(strings.TrimSpace(m[2]), `"'`) == "" {
			continue
		}
		claves = append(claves, m[1])
	}
	return claves
}
