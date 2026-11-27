package estado

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'
// FND-334: la clave local de un proyecto se acorta a MaxClave para que el relay la acepte. Lo que un qdd anterior dejó
// con la clave larga (en cualquier carpeta de <QDD_HOME>/claude: cola, goal, teléfono, verificaciones…) doctor lo ve y
// --fix lo renombra a la clave nueva. Si ya hay un archivo con la clave nueva, no se pisa: el viejo queda a la vista.

// ClavesLargas son los archivos locales de root que siguen con la clave larga.
func ClavesLargas(root string) []string {
	home, larga, corta, err := clavesDe(root)
	if err != nil || larga == corta {
		return nil
	}
	carpetas, err := os.ReadDir(filepath.Join(home, "claude"))
	if err != nil {
		return nil
	}
	var out []string
	for _, c := range carpetas {
		if !c.IsDir() {
			continue
		}
		dir := filepath.Join(home, "claude", c.Name())
		entradas, _ := os.ReadDir(dir)
		for _, e := range entradas {
			if strings.HasPrefix(e.Name(), larga+".") {
				out = append(out, filepath.Join(dir, e.Name()))
			}
		}
	}
	sort.Strings(out)
	return out
}

// MigrarClavesLargas renombra a la clave nueva cada archivo de ClavesLargas que no choca con uno existente. Devuelve
// lo que hizo.
func MigrarClavesLargas(root string) ([]string, error) {
	_, larga, corta, err := clavesDe(root)
	if err != nil {
		return nil, err
	}
	var hecho []string
	var errs []error
	for _, vieja := range ClavesLargas(root) {
		nueva := filepath.Join(filepath.Dir(vieja), corta+strings.TrimPrefix(filepath.Base(vieja), larga))
		if _, err := os.Stat(nueva); err == nil {
			hecho = append(hecho, fmt.Sprintf("%s queda: ya existe %s", vieja, filepath.Base(nueva)))
			continue
		}
		if err := os.Rename(vieja, nueva); err != nil {
			errs = append(errs, err)
			continue
		}
		hecho = append(hecho, fmt.Sprintf("%s/%s → %s", filepath.Base(filepath.Dir(vieja)), filepath.Base(vieja), filepath.Base(nueva)))
	}
	return hecho, errors.Join(errs...)
}

func clavesDe(root string) (string, string, string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", "", "", err
	}
	abs := ClaveLocal(root)
	return home, claveDe(abs, 0), claveDe(abs, MaxClave), nil
}
