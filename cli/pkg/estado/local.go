package estado

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'

var noSlug = regexp.MustCompile(`[^a-z0-9]+`)

// RutaLocal es un archivo local del proyecto, fuera del repositorio: <QDD_HOME>/claude/<carpeta>/<slug>-<hash>.json,
// con el nombre de la carpeta del proyecto y un hash de su ruta. Es el único lugar que define esa clave: la usan la
// actividad (CarpetaActividad) y la cola de pedidos de la consola.
func RutaLocal(root, carpeta string) (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(abs))
	slug := strings.Trim(noSlug.ReplaceAllString(strings.ToLower(filepath.Base(abs)), "-"), "-")
	return filepath.Join(home, "claude", carpeta, slug+"-"+hex.EncodeToString(sum[:])[:12]+".json"), nil
}
