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
	sinHTTPS(t)
	_, problems := Add(AddInput{Root: t.TempDir(), Origen: "git@github.com:acme/qdd-blueprints.git", Blueprints: []string{"auth"}})
	if !hasProblem(problems, "https://github.com/acme/qdd-blueprints.git") {
		t.Fatalf("el error sugiere la URL HTTPS equivalente: %v", problems)
	}
}

func TestDoctor_WithoutSSHAccessSuggestsHTTPS(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "false")
	sinHTTPS(t)
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

// publicoSinLlave: el origen es SSH y esta máquina (o el CI) no tiene llave, pero el repositorio es público: su «URL
// HTTPS» es el repositorio real.
func publicoSinLlave(t *testing.T, ssh, repo string) {
	t.Helper()
	t.Setenv("GIT_SSH_COMMAND", "false")
	previo := httpsEquivalente
	httpsEquivalente = func(r string) (string, bool) { return repo, r == ssh }
	t.Cleanup(func() { httpsEquivalente = previo })
}

// sinHTTPS: tampoco hay acceso por HTTPS (repositorio privado o inexistente). git lleva github.com a una carpeta que no
// existe: la prueba no sale a la red y la sugerencia sigue nombrando la URL real.
func sinHTTPS(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url."+filepath.Join(t.TempDir(), "no-existe")+"/.insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://github.com/")
}

// FND-104: la lectura (add, sync y el gate) solo probaba SSH. En el CI, sin llave, un repositorio público bloqueaba el
// gate con BP-01-INVALID aunque se leyera por HTTPS; propose y release ya reintentaban (FND-063).
func TestAdd_ReadsAPublicRepositoryOverHTTPSWithoutSSHKey_FND104(t *testing.T) {
	freshHome(t)
	ssh := "git@github.com:acme/qdd-blueprints.git"
	publicoSinLlave(t, ssh, gitOrgRepo(t))
	root := t.TempDir()

	if _, problems := Add(AddInput{Root: root, Origen: ssh, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatalf("sin llave SSH, un repositorio público se lee por HTTPS: %v", problems)
	}
	if e := lockEntry(t, root, "auth"); e.Origen != ssh {
		t.Fatalf("el lockfile guarda el origen tal como se indicó, igual en todas las máquinas: %+v", e)
	}
}

func TestEnsureMirror_UpdatesOverHTTPSWhenSSHFails_FND104(t *testing.T) {
	freshHome(t)
	ssh := "git@github.com:acme/qdd-blueprints.git"
	repo := gitOrgRepo(t)
	mirror, err := mirrorPath(ssh)
	if err != nil {
		t.Fatal(err)
	}
	gitIn(t, "", "clone", "-q", "--mirror", repo, mirror)
	gitIn(t, mirror, "remote", "set-url", "origin", ssh) // el espejo se clonó por SSH en otra sesión
	publicar(t, repo, "1.1.0", "v1.1.0")
	publicoSinLlave(t, ssh, repo)

	if _, err := ensureMirror(ssh, true); err != nil {
		t.Fatalf("sin llave SSH, el espejo se actualiza por HTTPS: %v", err)
	}
	if _, err := git(mirror, "rev-parse", "--verify", "--quiet", "refs/tags/v1.1.0"); err != nil {
		t.Fatal("el espejo trae la versión nueva")
	}
}

func TestDoctor_PublicRepositoryWithoutSSHKeyHasAccess_FND104(t *testing.T) {
	ssh := "git@github.com:acme/qdd-blueprints.git"
	publicoSinLlave(t, ssh, gitOrgRepo(t))
	lock := Lock{Blueprints: []LockEntry{{ID: "auth", Origen: ssh}}}
	d := diagAcceso(lock, []string{"auth"})
	if !d.OK || !strings.Contains(d.Detalle, "HTTPS") {
		t.Fatalf("doctor dice que hay acceso, por HTTPS: %+v", d)
	}
}
