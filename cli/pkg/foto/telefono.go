package foto

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Fotos de QDD Móvil (ADR-0003 de qdd-movile, acordado el 2026-11-23): qdd remoto arma la foto que la persona adjunta a
// un mensaje para la IA y la deja aquí, junto a las de qdd foto, para que la sesión la abra por su ruta. La persona
// eligió que se borren a los 7 días: el original sigue en su teléfono.

const (
	// PrefijoTelefono distingue las fotos del teléfono de las de qdd foto: solo esas se borran solas.
	PrefijoTelefono = "movil-"
	// VidaTelefono es cuánto queda una foto del teléfono en el PC.
	VidaTelefono = 7 * 24 * time.Hour
)

// IDTelefono es el id que da el teléfono a una foto: es parte del nombre del archivo, así que nunca trae / ni ..
var IDTelefono = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// GuardarDelTelefono valida la imagen, la guarda 0600 en tmp/fotos/movil-<id>.<ext> (la extensión sale de sus bytes) y
// borra las del teléfono de más de VidaTelefono. Devuelve la ruta absoluta.
func GuardarDelTelefono(root, id string, data []byte, now time.Time) (string, error) {
	if !IDTelefono.MatchString(id) {
		return "", errors.New("el id de la foto solo lleva letras, números, - y _ (de 1 a 64)")
	}
	tipo, err := Validar(data)
	if err != nil {
		return "", err
	}
	dir, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(Carpeta)))
	if err != nil {
		return "", err
	}
	if err := prepararCarpeta(dir); err != nil {
		return "", err
	}
	ruta := filepath.Join(dir, PrefijoTelefono+id+tipos[tipo])
	temporal := ruta + ".parcial"
	if err := os.WriteFile(temporal, data, 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(temporal, ruta); err != nil {
		os.Remove(temporal)
		return "", err
	}
	PodarDelTelefono(root, now)
	return ruta, nil
}

// PodarDelTelefono borra de tmp/fotos del proyecto las fotos del teléfono de más de VidaTelefono; las de qdd foto no las
// toca. La llaman GuardarDelTelefono y qdd remoto, cada hora (FND-307).
func PodarDelTelefono(root string, now time.Time) {
	dir := filepath.Join(root, filepath.FromSlash(Carpeta))
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entradas {
		if e.IsDir() || !strings.HasPrefix(e.Name(), PrefijoTelefono) {
			continue
		}
		info, err := e.Info()
		if err != nil || now.Sub(info.ModTime()) <= VidaTelefono {
			continue
		}
		os.Remove(filepath.Join(dir, e.Name()))
	}
}
