package puertas

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// archivoMarca vive en la carpeta de git de cada worktree (fuera del árbol): la huella del último qdd audit aprobado.
const archivoMarca = "qdd-audit-marca"

// Huella es el árbol de git del árbol de trabajo tal como está: lo seguido y lo nuevo que git no ignora. Se arma en un
// índice aparte, así que no toca el índice real; da lo mismo lo que esté en stage.
func Huella(dir string) (string, error) {
	indice, err := gitSalida(dir, nil, "rev-parse", "--path-format=absolute", "--git-path", "index")
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(indice), "qdd-huella-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if err := copiarIndice(indice, tmp); err != nil {
		return "", err
	}
	env := []string{"GIT_INDEX_FILE=" + tmp.Name()}
	if _, err := gitSalida(dir, env, "add", "-A"); err != nil {
		return "", err
	}
	return gitSalida(dir, env, "write-tree")
}

// copiarIndice parte del índice real para que git solo vuelva a leer lo que cambió; un repositorio nuevo no tiene.
func copiarIndice(indice string, destino *os.File) error {
	defer destino.Close()
	origen, err := os.Open(indice)
	if errors.Is(err, os.ErrNotExist) {
		return os.Remove(destino.Name())
	}
	if err != nil {
		return err
	}
	defer origen.Close()
	_, err = io.Copy(destino, origen)
	return err
}

// GuardarMarca deja la huella que aprobó qdd audit.
func GuardarMarca(dir, huella string) error {
	ruta, err := gitSalida(dir, nil, "rev-parse", "--path-format=absolute", "--git-path", archivoMarca)
	if err != nil {
		return err
	}
	return os.WriteFile(ruta, []byte(huella+"\n"), 0o600)
}

// LeerMarca es la huella del último qdd audit aprobado; "" si no hay.
func LeerMarca(dir string) (string, error) {
	ruta, err := gitSalida(dir, nil, "rev-parse", "--path-format=absolute", "--git-path", archivoMarca)
	if err != nil {
		return "", err
	}
	datos, err := os.ReadFile(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return strings.TrimSpace(string(datos)), err
}

// Raiz es la carpeta raíz del repositorio de dir.
func Raiz(dir string) (string, error) {
	return gitSalida(dir, nil, "rev-parse", "--show-toplevel")
}

// Proyecto es el nombre del repositorio, el mismo desde cualquiera de sus worktrees.
func Proyecto(dir string) string {
	comun, err := gitSalida(dir, nil, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return filepath.Base(dir)
	}
	if filepath.Base(comun) == ".git" {
		return filepath.Base(filepath.Dir(comun))
	}
	return strings.TrimSuffix(filepath.Base(comun), ".git")
}

// EsTag dice si lo que empuja un refspec es un tag de ese repositorio.
func EsTag(dir, refspec string) bool {
	ref := RefDeTag(refspec)
	if strings.HasPrefix(ref, "refs/tags/") {
		return true
	}
	_, err := gitSalida(dir, nil, "rev-parse", "-q", "--verify", "refs/tags/"+ref)
	return err == nil
}

func gitSalida(dir string, env []string, args ...string) (string, error) {
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(os.Environ(), env...)
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Registro es una decisión anotada: sin contenido de archivos ni rutas. Largo son los caracteres que leyó la IA.
type Registro struct {
	ID        string    `json:"id,omitempty"`
	Fecha     time.Time `json:"fecha"`
	Puerta    string    `json:"puerta"`
	Resultado string    `json:"resultado"`
	MS        int64     `json:"ms"`
	Sesion    string    `json:"sesion,omitempty"`
	Largo     int       `json:"largo"`
}

// Anotar agrega la decisión a <home>/puertas/<proyecto>.jsonl, fuera del proyecto, con su id.
func Anotar(home, proyecto string, r Registro) error {
	if r.ID == "" {
		r.ID = idDe(r)
	}
	linea, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return agregarLinea(home, proyecto, linea)
}

func agregarLinea(home, proyecto string, linea []byte) error {
	dir := filepath.Join(home, "puertas")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(rutaRegistro(home, proyecto), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(linea, '\n'))
	return err
}
