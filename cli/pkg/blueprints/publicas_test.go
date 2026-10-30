package blueprints

import (
	"path/filepath"
	"strings"
	"testing"
)

// Un repositorio de blueprints público lo usa cualquiera: por HTTPS, sin llave
// SSH, o desde un repositorio bare en disco.

// FND-067: una ruta local a un repositorio bare se rechazaba como «no es un
// repositorio git».
func TestAdd_FromALocalBareRepository(t *testing.T) {
	bare := filepath.Join(t.TempDir(), "blueprints.git")
	gitIn(t, "", "clone", "-q", "--bare", gitOrgRepo(t), bare)
	root := t.TempDir()

	if _, problems := Add(AddInput{Root: root, Origen: bare, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatalf("un repositorio bare en disco es una fuente válida: %v", problems)
	}
	if e := lockEntry(t, root, "auth"); e.Origen != bare || e.Subdir != "auth" {
		t.Fatalf("el lockfile nombra el repositorio bare tal como se indicó: %+v", e)
	}
}

func TestHTTPSEquivalente(t *testing.T) {
	for in, want := range map[string]string{
		"git@github.com:trdago/qdd-blueprints.git":       "https://github.com/trdago/qdd-blueprints.git",
		"git@gitlab.example.com:grupo/sub/bp.git":        "https://gitlab.example.com/grupo/sub/bp.git",
		"ssh://git@github.com/trdago/qdd-blueprints.git": "https://github.com/trdago/qdd-blueprints.git",
		"ssh://git@git.example.com:2222/org/bp.git":      "https://git.example.com/org/bp.git",
		"https://github.com/trdago/qdd-blueprints.git":   "",
		"file:///srv/blueprints.git":                     "",
		"/srv/blueprints.git":                            "",
		"C:/repos/blueprints":                            "",
	} {
		got, ok := httpsEquivalente(in)
		if got != want || ok != (want != "") {
			t.Errorf("httpsEquivalente(%q) = %q, %v; se esperaba %q", in, got, ok, want)
		}
	}
}

// FND-068: sin llave SSH, el error de git llegaba solo; no decía que un
// repositorio público se importa por HTTPS sin credenciales.
func TestAdd_WithoutSSHAccessSuggestsHTTPS(t *testing.T) {
	freshHome(t)
	t.Setenv("GIT_SSH_COMMAND", "false")
	_, problems := Add(AddInput{Root: t.TempDir(), Origen: "git@github.com:acme/qdd-blueprints.git", Blueprints: []string{"auth"}})
	if !hasProblem(problems, "https://github.com/acme/qdd-blueprints.git") {
		t.Fatalf("el error sugiere la URL HTTPS equivalente: %v", problems)
	}
}

func TestDoctor_WithoutSSHAccessSuggestsHTTPS(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "false")
	lock := Lock{Blueprints: []LockEntry{{ID: "auth", Origen: "git@github.com:acme/qdd-blueprints.git"}}}
	d := diagAcceso(lock, []string{"auth"})
	if d.OK || !strings.Contains(d.Detalle, "https://github.com/acme/qdd-blueprints.git") {
		t.Fatalf("doctor sugiere la URL HTTPS equivalente: %+v", d)
	}
}

// sinPushSSH hace que el origin del clon de aportes rechace el push (como una
// deploy key de solo lectura) y que su «URL HTTPS» sea el repositorio real.
func sinPushSSH(t *testing.T, repo string) {
	t.Helper()
	espacio, err := espacioPath(repo)
	if err != nil {
		t.Fatal(err)
	}
	gitIn(t, espacio, "remote", "set-url", "--push", "origin", filepath.Join(t.TempDir(), "sin-permiso.git"))
	previo := httpsEquivalente
	httpsEquivalente = func(r string) (string, bool) { return repo, r == repo }
	t.Cleanup(func() { httpsEquivalente = previo })
}

// FND-063: propose y release solo publicaban por SSH; con una deploy key sin
// escritura fallaban aunque git tuviera credenciales HTTPS (p. ej. las de gh).
func TestProponer_PublishesOverHTTPSWhenSSHCannotPush(t *testing.T) {
	sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}})
	ap, err := Aportar(root, "auth", "")
	if err != nil {
		t.Fatal(err)
	}
	sinPushSSH(t, repo)
	writeFile(t, filepath.Join(ap.Carpeta, ManifestName), strings.Replace(authBlueprint, "¿Con qué inicia sesión el usuario?", "¿Con qué dato inicia sesión?", 1))

	p, err := Proponer(root, "auth", "redacción", "")
	if err != nil {
		t.Fatalf("si el push por SSH no tiene permiso, se publica por HTTPS: %v", err)
	}
	if got := versionEn(repo, "aporte/auth", "auth"); got != "1.0.1" {
		t.Fatalf("la rama llegó al repositorio: %q", got)
	}
	if !strings.Contains(p.Via, "HTTPS") || p.Aviso != "" {
		t.Fatalf("la salida dice que se publicó por HTTPS, y el PR se abrió igual: %+v", p)
	}
}

func TestPublicar_TagsOverHTTPSWhenSSHCannotPush(t *testing.T) {
	sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}})
	ap, err := Aportar(root, "auth", "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(ap.Carpeta, ManifestName), strings.Replace(authBlueprint, "¿Con qué inicia sesión el usuario?", "¿Con qué dato inicia sesión?", 1))
	if _, err := Proponer(root, "auth", "redacción", ""); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "merge", "-q", "--no-edit", "aporte/auth") // el PR se integra
	sinPushSSH(t, repo)

	pub, err := Publicar(root, "")
	if err != nil {
		t.Fatalf("si el push por SSH no tiene permiso, el tag se publica por HTTPS: %v", err)
	}
	if _, err := git(repo, "rev-parse", "--verify", "--quiet", "refs/tags/"+pub.Tag); err != nil {
		t.Fatalf("el tag %s llegó al repositorio", pub.Tag)
	}
	if !strings.Contains(pub.Via, "HTTPS") {
		t.Fatalf("la salida dice que se publicó por HTTPS: %+v", pub)
	}
}
