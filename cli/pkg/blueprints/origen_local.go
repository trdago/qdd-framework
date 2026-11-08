package blueprints

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'

var unidadWindows = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

// esCarpetaLocal: una ruta del disco (también la de Windows, vista desde otro sistema), no un repositorio.
func esCarpetaLocal(origen string) bool {
	if strings.Contains(origen, "://") || strings.HasPrefix(origen, "git@") {
		return false
	}
	return filepath.IsAbs(origen) || unidadWindows.MatchString(origen) || strings.HasPrefix(origen, "~")
}

// AvisosDeOrigen avisa de los blueprints importados desde una carpeta local: el lockfile guarda esa ruta y en otro PC no
// existe, así que no se puede reconstruir (MEJ-038).
func AvisosDeOrigen(entries []LockEntry) []string {
	var avisos []string
	for _, e := range entries {
		if !esCarpetaLocal(e.Origen) {
			continue
		}
		avisos = append(avisos, fmt.Sprintf("%s salió de una carpeta local (%s): en otro PC esa ruta no existe y no se podrá reconstruir. Si va a otros equipos, impórtalo desde la URL de su repositorio.", e.ID, e.Origen))
	}
	return avisos
}
