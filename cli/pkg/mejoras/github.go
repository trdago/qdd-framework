package mejoras

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'
// La fuente GitHub lee los issues abiertos del repositorio del remote origin con 'gh api' (gh issue list no trae
// authorAssociation, que decide quién puede entrar a la cola). Sin terminal, con el entorno de git limpio y un tope.

// TopeGitHub es lo más que espera una lectura de GitHub.
const TopeGitHub = 20 * time.Second

var remoteGitHub = regexp.MustCompile(`github\.com[:/]([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+?)(?:\.git)?/?$`)

// FuentesReales son las fuentes de verdad: GitHub con gh.
func FuentesReales() Fuentes {
	return Fuentes{GitHub: issuesDeGitHub}
}

func issuesDeGitHub(root string) (string, []Issue, error) {
	ctx, cancelar := context.WithTimeout(context.Background(), TopeGitHub)
	defer cancelar()
	remote, err := correr(ctx, root, "git", "remote", "get-url", "origin")
	if err != nil {
		return "", nil, ErrSinGitHub
	}
	repo, ok := RepoDeRemote(remote)
	if !ok {
		return "", nil, ErrSinGitHub
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return repo, nil, errors.New("el remote es de GitHub pero falta gh (https://cli.github.com) para leer sus issues")
	}
	salida, err := correr(ctx, root, "gh", "api", "repos/"+repo+"/issues?state=open&per_page=100")
	if err != nil {
		return repo, nil, fmt.Errorf("gh api: %v", err)
	}
	issues, err := IssuesDeAPI([]byte(salida))
	return repo, issues, err
}

// RepoDeProyecto es el owner/repo de GitHub del remote origin del proyecto, si lo tiene.
func RepoDeProyecto(root string) (string, bool) {
	ctx, cancelar := context.WithTimeout(context.Background(), TopeGitHub)
	defer cancelar()
	remote, err := correr(ctx, root, "git", "remote", "get-url", "origin")
	if err != nil {
		return "", false
	}
	return RepoDeRemote(remote)
}

// RepoDeRemote saca owner/repo de un remote de GitHub (SSH o HTTPS).
func RepoDeRemote(remote string) (string, bool) {
	m := remoteGitHub.FindStringSubmatch(strings.TrimSpace(remote))
	if m == nil {
		return "", false
	}
	return m[1] + "/" + m[2], true
}

// IssuesDeAPI traduce la respuesta de GET /repos/{owner}/{repo}/issues; los pull requests (que la API también
// devuelve) no son issues.
func IssuesDeAPI(b []byte) ([]Issue, error) {
	var crudos []struct {
		Number            int             `json:"number"`
		Title             string          `json:"title"`
		Body              string          `json:"body"`
		HTMLURL           string          `json:"html_url"`
		CreatedAt         time.Time       `json:"created_at"`
		AuthorAssociation string          `json:"author_association"`
		PullRequest       json.RawMessage `json:"pull_request"`
		User              struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(b, &crudos); err != nil {
		return nil, fmt.Errorf("la respuesta de GitHub no se entiende: %v", err)
	}
	var out []Issue
	for _, c := range crudos {
		if len(c.PullRequest) > 0 {
			continue
		}
		is := Issue{Numero: c.Number, Titulo: c.Title, Cuerpo: c.Body, Autor: c.User.Login, Asociacion: c.AuthorAssociation, URL: c.HTMLURL, Creado: c.CreatedAt}
		for _, l := range c.Labels {
			is.Etiquetas = append(is.Etiquetas, l.Name)
		}
		out = append(out, is)
	}
	return out, nil
}

func correr(ctx context.Context, dir, nombre string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, nombre, args...)
	cmd.Dir = dir
	cmd.Env = append(gitenv.Limpio(os.Environ()), "GIT_TERMINAL_PROMPT=0", "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
