package incidentes

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

// Publicador es dónde van los issues: GitHub con gh en el producto, uno en memoria en las pruebas.
type Publicador interface {
	// Buscar devuelve el issue (abierto o cerrado) que ya lleva la firma, o "".
	Buscar(firma string) (string, error)
	Crear(titulo, cuerpo string) (string, error)
	Comentar(url, cuerpo string) error
}

// Resultado de una publicación: cuántos issues nuevos, cuántos comentarios y cuántos incidentes quedan pendientes
// (por el tope del día o porque no hubo acceso, que dice Err).
type Resultado struct {
	Creados, Comentados, Pendientes int
	Err                             error
}

// Publicar sube los incidentes pendientes, del más antiguo al más nuevo, sin pasar el tope del día: uno sin issue busca
// primero el que ya lleve su firma (de esta máquina o de otra) y lo comenta; si no hay, lo crea. Uno ya publicado con
// ocurrencias nuevas comenta las nuevas. Si no hay acceso, para y los deja pendientes.
func Publicar(dir string, p Publicador, now time.Time, tope int) Resultado {
	todos, _ := Leer(dir)
	var r Resultado
	hoy := now.UTC().Format("2006-01-02")
	usados := 0
	for _, i := range todos {
		if dia(i.PublicadoEl) == hoy {
			usados++
		}
	}
	for _, i := range todos {
		if !i.Pendiente() {
			continue
		}
		if usados >= tope || r.Err != nil {
			r.Pendientes++
			continue
		}
		creado, err := publicarUno(p, &i)
		if err != nil {
			r.Err = err
			r.Pendientes++
			continue
		}
		i.Publicadas, i.PublicadoEl = i.Ocurrencias, now.UTC().Format(time.RFC3339)
		if err := guardar(filepath.Join(dir, i.Firma+".yaml"), i); err != nil {
			r.Err = err
		}
		usados++
		if creado {
			r.Creados++
			continue
		}
		r.Comentados++
	}
	return r
}

func publicarUno(p Publicador, i *Incidente) (bool, error) {
	if i.Issue != "" {
		return false, p.Comentar(i.Issue, Ocurrencia(*i))
	}
	url, err := p.Buscar(i.Firma)
	if err != nil {
		return false, err
	}
	if url != "" {
		i.Issue = url
		return false, p.Comentar(url, Ocurrencia(*i))
	}
	url, err = p.Crear(Titulo(*i), Cuerpo(*i))
	if err != nil {
		return false, err
	}
	i.Issue = url
	return true, nil
}

// GH publica con la CLI de GitHub (sus credenciales; un repositorio público acepta issues de cualquiera con cuenta).
type GH struct{ Repo string }

// Repo es el repositorio de QDD, o el de QDD_INCIDENTES_REPO.
func Repo() string {
	if r := strings.TrimSpace(os.Getenv(RepoEnv)); r != "" {
		return r
	}
	return RepoPorDefecto
}

func (g GH) Buscar(firma string) (string, error) {
	out, err := g.gh("", "issue", "list", "-R", g.Repo, "--state", "all", "--search", "qdd-incidente firma="+firma+" in:body", "--json", "url", "--jq", ".[0].url // \"\"")
	return strings.TrimSpace(out), err
}

func (g GH) Crear(titulo, cuerpo string) (string, error) {
	out, err := g.gh(cuerpo, "issue", "create", "-R", g.Repo, "--title", titulo, "--body-file", "-")
	if err != nil {
		return "", err
	}
	lineas := strings.Fields(out)
	if len(lineas) == 0 {
		return "", errors.New("gh issue create no devolvió la URL")
	}
	return lineas[len(lineas)-1], nil
}

func (g GH) Comentar(url, cuerpo string) error {
	_, err := g.gh(cuerpo, "issue", "comment", url, "--body-file", "-")
	return err
}

func (g GH) gh(entrada string, args ...string) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", errors.New("no está instalada la CLI de GitHub (gh): los incidentes quedan pendientes en el registro")
	}
	c := exec.Command("gh", args...)
	c.Stdin = strings.NewReader(entrada)
	c.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1")
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh %s: %s", args[0]+" "+args[1], primeraLinea(string(out)))
	}
	return string(out), nil
}

func primeraLinea(s string) string {
	l, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return l
}

// esperaEntreLanzamientos: una ráfaga de fallas lanza una sola publicación.
const esperaEntreLanzamientos = 2 * time.Minute

// Lanzar deja corriendo aparte «bin incidentes publicar --silencioso» y vuelve en el acto: quien falló no espera a la
// red. Nunca en las pruebas, ni si se lanzó hace menos de esperaEntreLanzamientos.
func Lanzar(dir, bin string, now time.Time) bool {
	if enPruebas() || !Activo() || bin == "" {
		return false
	}
	marca := filepath.Join(dir, ".lanzado")
	if st, err := os.Stat(marca); err == nil && now.Sub(st.ModTime()) < esperaEntreLanzamientos {
		return false
	}
	if err := os.WriteFile(marca, nil, 0o600); err != nil {
		return false
	}
	_ = os.Chtimes(marca, now, now)
	c := exec.Command(bin, "incidentes", "publicar", "--silencioso")
	separar(c)
	if err := c.Start(); err != nil {
		return false
	}
	_ = c.Process.Release()
	return true
}
