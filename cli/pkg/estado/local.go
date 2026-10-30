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
// con el nombre de la carpeta del proyecto y un hash de su ruta (ClaveLocal: un worktree usa la de su checkout
// principal). Es el único lugar que define esa clave: la usan la actividad (CarpetaActividad) y la cola de pedidos.
func RutaLocal(root, carpeta string) (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	return rutaLocalDe(home, ClaveLocal(root), carpeta)
}

// RutaLocalSinRepositorio es la clave de qdd 2.7 y anteriores, por la ruta tal cual (un worktree tenía la suya):
// doctor la usa para encontrar y juntar una cola que quedó suelta.
func RutaLocalSinRepositorio(root, carpeta string) (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return rutaLocalDe(home, abs, carpeta)
}

func rutaLocalDe(home, abs, carpeta string) (string, error) {
	sum := sha256.Sum256([]byte(abs))
	slug := strings.Trim(noSlug.ReplaceAllString(strings.ToLower(filepath.Base(abs)), "-"), "-")
	return filepath.Join(home, "claude", carpeta, slug+"-"+hex.EncodeToString(sum[:])[:12]+".json"), nil
}
