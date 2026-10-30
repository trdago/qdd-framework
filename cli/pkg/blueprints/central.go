package blueprints

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// HomeEnv mueve la carpeta central de QDD. Existe para el CI y las pruebas;
// en una máquina de desarrollo QDD usa siempre ~/.local/share/qdd.
const HomeEnv = "QDD_HOME"

// La carpeta central (RFC 0004) la administra QDD, una por usuario:
//
//	blueprints/organizaciones.yaml  qué repositorio usa cada organización
//	blueprints/mirrors/<repo>.git   un clon espejo por repositorio
//	blueprints/store/<sha256>/      cada versión importada, inmutable
//	blueprints/aportes/<repo>/      el clon editable para proponer cambios
//
// Un proyecto solo versiona su lockfile; nunca una copia ni un clon.

var (
	shaPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)
	tagPattern    = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	slugPattern   = regexp.MustCompile(`[^a-z0-9]+`)
)

// errHuella indica que lo publicado en el origen no coincide con el lockfile;
// errSinHuella, que el lockfile no fija huella o commit válidos.
var (
	errHuella    = errors.New("no coincide con la huella del lockfile")
	errSinHuella = errors.New("el lockfile no fija una huella y un commit válidos")
)

// Home es la carpeta central de QDD del usuario.
func Home() (string, error) {
	if h := strings.TrimSpace(os.Getenv(HomeEnv)); h != "" {
		return filepath.Abs(h)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("no se pudo ubicar la carpeta del usuario para los blueprints: %w", err)
	}
	return filepath.Join(home, ".local", "share", "qdd"), nil
}

// centralDir devuelve (y crea, privada) una carpeta dentro de la central.
func centralDir(parts ...string) (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(append([]string{home, "blueprints"}, parts...)...)
	return dir, os.MkdirAll(dir, 0700)
}

// repoKey nombra la carpeta de un repositorio: legible y sin colisiones.
func repoKey(repo string) string {
	name := strings.TrimSuffix(path.Base(filepath.ToSlash(strings.TrimRight(repo, "/"))), ".git")
	slug := strings.Trim(slugPattern.ReplaceAllString(strings.ToLower(name), "-"), "-")
	sum := sha256.Sum256([]byte(repo))
	return firstNonEmpty(slug, "repo") + "-" + hex.EncodeToString(sum[:])[:12]
}

func gitRaw(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-c", "protocol.ext.allow=never"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		// git escribe algunos errores en la salida («nothing to commit»): sin ellos solo queda «exit status 1» (FND-083).
		return nil, fmt.Errorf("%s", firstNonEmpty(strings.TrimSpace(stderr.String()), strings.TrimSpace(string(out)), err.Error()))
	}
	return out, nil
}

func git(dir string, args ...string) (string, error) {
	out, err := gitRaw(dir, args...)
	return strings.TrimSpace(string(out)), err
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func mirrorPath(repo string) (string, error) {
	dir, err := centralDir("mirrors")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, repoKey(repo)+".git"), nil
}

// ensureMirror deja el espejo del repositorio en la carpeta central: lo clona
// la primera vez y, si se pide, lo actualiza con un fetch incremental.
func ensureMirror(repo string, actualizar bool) (string, error) {
	mirror, err := mirrorPath(repo)
	if err != nil {
		return "", err
	}
	if exists(mirror) && !actualizar {
		return mirror, nil
	}
	if exists(mirror) {
		if _, err := git(mirror, "fetch", "--quiet", "--prune", "origin"); err != nil {
			return "", sinAcceso("actualizar", repo, err)
		}
		return mirror, nil
	}
	return mirror, clonarAtomico(repo, mirror, "--mirror")
}

// clonarAtomico clona en una carpeta temporal y la renombra: dos procesos
// que clonan a la vez nunca dejan un clon a medias.
func clonarAtomico(repo, dest string, flags ...string) error {
	tmp, err := os.MkdirTemp(filepath.Dir(dest), ".clon-")
	if err != nil {
		return err
	}
	args := append(append([]string{"clone", "--quiet"}, flags...), "--", repo, tmp)
	if _, err := git("", args...); err != nil {
		os.RemoveAll(tmp)
		return sinAcceso("clonar", repo, err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.RemoveAll(tmp)
		if exists(dest) {
			return nil
		}
		return err
	}
	return nil
}

func tieneCommit(repoDir, commit string) bool {
	_, err := git(repoDir, "cat-file", "-e", commit+"^{commit}")
	return err == nil
}

// extraer escribe en dest los archivos regulares de ref (sin enlaces
// simbólicos ni rutas que escapen de la carpeta).
func extraer(repoDir, ref, dest string) error {
	data, err := gitRaw(repoDir, "archive", "--format=tar", ref)
	if err != nil {
		return fmt.Errorf("no se pudo leer %s: %v", ref, err)
	}
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.FromSlash(path.Clean(h.Name))
		if !isInside(name) {
			return fmt.Errorf("ruta inválida en el repositorio: %s", h.Name)
		}
		if err := writeFrom(tr, filepath.Join(dest, name)); err != nil {
			return err
		}
	}
}

func writeFrom(r io.Reader, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

func storePath(sum string) (string, error) {
	if !shaPattern.MatchString(sum) {
		return "", fmt.Errorf("huella '%s': %w", sum, errSinHuella)
	}
	dir, err := centralDir("store")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sum), nil
}

// intacto indica si la carpeta existe y coincide con la huella.
func intacto(dir, sum string) bool {
	got, err := HashDir(dir)
	return err == nil && got == sum
}

// storePut guarda una copia inmutable (de solo lectura) del blueprint en el
// store, direccionada por su huella; si ya está íntegra, no la toca.
func storePut(src string) (string, string, error) {
	sum, err := HashDir(src)
	if err != nil {
		return "", "", err
	}
	dest, err := storePath(sum)
	if err != nil {
		return "", "", err
	}
	if intacto(dest, sum) {
		return dest, sum, nil
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dest), ".nuevo-")
	if err != nil {
		return "", "", err
	}
	if err := copyTree(src, tmp, 0444); err != nil {
		os.RemoveAll(tmp)
		return "", "", err
	}
	os.RemoveAll(dest)
	if err := os.Rename(tmp, dest); err != nil {
		os.RemoveAll(tmp)
		if intacto(dest, sum) {
			return dest, sum, nil
		}
		return "", "", err
	}
	return dest, sum, nil
}

// materialize devuelve la carpeta central de un blueprint del lockfile. Si
// no está (otra máquina, el CI) o se alteró, la reconstruye desde el commit
// fijado en el lockfile, con las credenciales git del usuario.
func materialize(e LockEntry) (string, error) {
	dest, err := storePath(e.SHA256)
	if err != nil {
		return "", err
	}
	if intacto(dest, e.SHA256) {
		return dest, nil
	}
	if !commitPattern.MatchString(e.Commit) {
		return "", fmt.Errorf("'%s': %w; vuelve a importarlo con 'qdd blueprint add %s' o 'qdd doctor --fix'", e.ID, errSinHuella, e.ID)
	}
	mirror, err := ensureMirror(e.Origen, false)
	if err != nil {
		return "", err
	}
	if !tieneCommit(mirror, e.Commit) {
		if mirror, err = ensureMirror(e.Origen, true); err != nil {
			return "", err
		}
	}
	if !tieneCommit(mirror, e.Commit) {
		return "", fmt.Errorf("el commit %.12s ya no existe en %s", e.Commit, e.Origen)
	}
	tmp, err := os.MkdirTemp("", "qdd-blueprint-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	if err := extraer(mirror, e.Commit, tmp); err != nil {
		return "", err
	}
	src := filepath.Join(tmp, filepath.FromSlash(e.Subdir))
	if !intacto(src, e.SHA256) {
		return "", fmt.Errorf("'%s' en %s@%.12s %w", e.ID, e.Origen, e.Commit, errHuella)
	}
	_, _, err = storePut(src)
	return dest, err
}

// load lee un blueprint del lockfile desde la carpeta central.
func load(e LockEntry) (Blueprint, error) {
	dir, err := materialize(e)
	if err != nil {
		return Blueprint{}, err
	}
	return ParseDir(dir)
}
