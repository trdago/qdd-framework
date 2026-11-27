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
	return filepath.Join(home, "claude", carpeta, claveDe(abs, MaxClave)+".json"), nil
}

// rutaLocalLarga es la ruta con la clave de antes de FND-334, sin tope: doctor la busca para renombrarla.
func rutaLocalLarga(home, abs, carpeta string) (string, error) {
	return filepath.Join(home, "claude", carpeta, claveDe(abs, 0)+".json"), nil
}

// MaxClave es el largo que el relay acepta en la clave de un proyecto (^[a-z0-9-]{1,80}$, FND-334).
const MaxClave = 80

// claveDe es <carpeta>-<hash de 12> con la carpeta acortada para que el total no pase de tope (0: sin tope). El hash
// queda entero: es lo que distingue a dos proyectos.
func claveDe(abs string, tope int) string {
	sum := sha256.Sum256([]byte(abs))
	hash := hex.EncodeToString(sum[:])[:12]
	slug := strings.Trim(noSlug.ReplaceAllString(strings.ToLower(filepath.Base(abs)), "-"), "-")
	if tope > 0 && len(slug)+1+len(hash) > tope {
		slug = strings.TrimRight(slug[:tope-1-len(hash)], "-")
	}
	return slug + "-" + hash
}
