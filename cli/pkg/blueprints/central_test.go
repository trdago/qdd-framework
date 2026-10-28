package blueprints

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain aísla la carpeta central: ninguna prueba toca ~/.local/share/qdd.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "qdd-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "no se pudo crear la carpeta central de prueba:", err)
		os.Exit(1)
	}
	os.Setenv(HomeEnv, home)
	// Identidad git para los commits que QDD hace en el espacio de aportes.
	for k, v := range map[string]string{"GIT_AUTHOR_NAME": "qdd", "GIT_AUTHOR_EMAIL": "qdd@test", "GIT_COMMITTER_NAME": "qdd", "GIT_COMMITTER_EMAIL": "qdd@test", "GIT_CONFIG_NOSYSTEM": "1"} {
		os.Setenv(k, v)
	}
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

// freshHome simula otra máquina (o el CI): una carpeta central vacía.
func freshHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(HomeEnv, home)
	return home
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "init.defaultBranch=main"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// gitOrgRepo es orgRepo (ya publicado como v1.0.0).
func gitOrgRepo(t *testing.T) string {
	t.Helper()
	return orgRepo(t, nil)
}

// publicar sube la versión de auth en el repositorio y la etiqueta.
func publicar(t *testing.T, repo, version, tag string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, "auth", ManifestName), strings.Replace(authBlueprint, "version: 1.0.0", "version: "+version, 1))
	gitIn(t, repo, "commit", "-q", "-am", version)
	gitIn(t, repo, "tag", tag)
}

// projectWithRemote crea un proyecto git cuyo remote 'origin' es remote.
func projectWithRemote(t *testing.T, remote string) string {
	t.Helper()
	root := t.TempDir()
	gitIn(t, root, "init", "-q")
	gitIn(t, root, "remote", "add", "origin", remote)
	return root
}

func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && !strings.HasPrefix(rel, "..")
}

func TestAdd_KeepsOnlyTheLockfileInTheProject(t *testing.T) {
	root := importedProject(t)
	if _, err := os.Stat(filepath.Join(root, Dir)); !os.IsNotExist(err) {
		t.Fatal("el proyecto no debe guardar copias de los blueprints")
	}
	lock, err := LoadLock(root)
	if err != nil || lock.Formato != FormatoLock {
		t.Fatalf("el lockfile debe declarar el formato %d: %+v (%v)", FormatoLock, lock, err)
	}
	b, err := Find(root, "auth")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := Home()
	if !isUnder(b.Dir, home) {
		t.Fatalf("el blueprint debe leerse desde la carpeta central %s, no desde %s", home, b.Dir)
	}
	if _, err := os.Stat(filepath.Join(b.Dir, "referencias", "go", "README.md")); err != nil {
		t.Fatal("las implementaciones de referencia viajan con el blueprint")
	}
	if problems := Check(root); len(problems) != 0 {
		t.Fatalf("un blueprint recién importado está íntegro: %v", problems)
	}
}

func TestAdd_ProjectsShareOneCentralCopy(t *testing.T) {
	src := sourceBlueprint(t, authBlueprint)
	a, b := t.TempDir(), t.TempDir()
	for _, root := range []string{a, b} {
		if _, problems := Add(AddInput{Root: root, Origen: src}); len(problems) > 0 {
			t.Fatal(problems)
		}
	}
	ba, _ := Find(a, "auth")
	bb, _ := Find(b, "auth")
	if ba.Dir == "" || ba.Dir != bb.Dir {
		t.Fatalf("dos proyectos con la misma versión comparten la copia central: %s vs %s", ba.Dir, bb.Dir)
	}
}

func TestAdd_GitRepositoryIsClonedOnceAndFetchedAfter(t *testing.T) {
	repo := gitOrgRepo(t)
	first := t.TempDir()
	if _, problems := Add(AddInput{Root: first, Origen: repo, Ref: "v1.0.0", Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	mirror, err := mirrorPath(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mirror); err != nil {
		t.Fatal("el repositorio se clona una vez en la carpeta central")
	}

	publicar(t, repo, "1.1.0", "v1.1.0")
	second := t.TempDir()
	entries, problems := Add(AddInput{Root: second, Origen: repo, Ref: "v1.1.0", Blueprints: []string{"auth"}})
	if len(problems) > 0 {
		t.Fatalf("el espejo existente se actualiza con fetch: %v", problems)
	}
	if entries[0].Version != "1.1.0" || len(entries[0].Commit) != 40 {
		t.Fatalf("se esperaba auth@1.1.0 con su commit: %+v", entries[0])
	}
}

func TestFind_RebuildsFromTheLockfileOnAnotherMachine(t *testing.T) {
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := Add(AddInput{Root: root, Origen: repo, Ref: "v1.0.0", Blueprints: []string{"auth", "pagos"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	publicar(t, repo, "2.0.0", "v2.0.0")

	freshHome(t) // el CI: sin espejo ni store
	b, err := Find(root, "auth")
	if err != nil {
		t.Fatalf("el CI reconstruye el blueprint desde el commit del lockfile: %v", err)
	}
	if b.Version != "1.0.0" {
		t.Fatalf("debe usar el commit fijado, no el último: %s", b.Version)
	}
	problems := Check(root)
	if hasRule(problems, "BP-01-INVALID") || hasRule(problems, "BP-02-ALTERED") {
		t.Fatalf("lo reconstruido coincide con la huella: %v", problems)
	}
	if !hasRule(problems, "BP-06-BEHIND") {
		t.Fatalf("el CI también ve que la organización publicó v2.0.0: %v", problems)
	}
}

func TestFind_RepairsATamperedCentralCopy(t *testing.T) {
	root := importedProject(t)
	b, _ := Find(root, "auth")
	manifest := filepath.Join(b.Dir, ManifestName)
	os.Chmod(manifest, 0644)
	writeFile(t, manifest, strings.Replace(authBlueprint, "nunca por nombre de rol", "o por rol", 1))

	again, err := Find(root, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(again.Ficha.ObjetivoTecnologico.Invariantes, " "), "nunca por nombre de rol") {
		t.Fatal("una copia central alterada se reemplaza por la del origen")
	}
}

func TestCheck_LockfileThatDoesNotMatchTheOrigin(t *testing.T) {
	root := importedProject(t)
	lock, _ := LoadLock(root)
	lock.Blueprints[0].SHA256 = strings.Repeat("0", 64)
	if err := saveLock(root, lock); err != nil {
		t.Fatal(err)
	}
	if !hasRule(Check(root), "BP-02-ALTERED") {
		t.Fatal("un lockfile cuya huella no coincide con el origen debe reportarse")
	}
}

// legacyProject arma un proyecto del esquema anterior (v1.10/v1.11): copia
// vendorizada en .qdd/blueprints/<id> y lockfile sin formato.
func legacyProject(t *testing.T) (string, string) {
	t.Helper()
	src := sourceBlueprint(t, authBlueprint)
	root := t.TempDir()
	legacy := filepath.Join(root, Dir, "auth")
	if err := copyTree(src, legacy, 0644); err != nil {
		t.Fatal(err)
	}
	sum, err := HashDir(legacy)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, LockFile), "blueprints:\n  - id: auth\n    version: 1.0.0\n    origen: "+src+"\n    sha256: "+sum+"\n    agregado_el: \"2026-10-23\"\n")
	return root, src
}

func TestMigrar_MovesIntactLegacyCopiesOnlyOnce(t *testing.T) {
	root, _ := legacyProject(t)
	m, err := Migrar(root)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(m.Migrados, ",") != "auth" || len(m.Pendientes) != 0 {
		t.Fatalf("la copia íntegra se migra: %+v", m)
	}
	if _, err := os.Stat(filepath.Join(root, Dir)); !os.IsNotExist(err) {
		t.Fatal("tras migrar, el proyecto ya no guarda copias")
	}
	if lock, _ := LoadLock(root); lock.Formato != FormatoLock {
		t.Fatalf("el lockfile queda en el formato nuevo: %+v", lock)
	}
	if problems := Check(root); len(problems) != 0 {
		t.Fatalf("un proyecto migrado queda íntegro: %v", problems)
	}
	again, err := Migrar(root)
	if err != nil || again.Hizo() {
		t.Fatalf("la migración ocurre una sola vez: %+v (%v)", again, err)
	}
}

func TestMigrar_WorksEvenIfTheOriginIsGone(t *testing.T) {
	root, src := legacyProject(t)
	os.RemoveAll(src)
	if m, err := Migrar(root); err != nil || len(m.Migrados) != 1 {
		t.Fatalf("la copia legada íntegra basta para migrar: %+v (%v)", m, err)
	}
	if _, err := Find(root, "auth"); err != nil {
		t.Fatalf("lo migrado queda en la carpeta central: %v", err)
	}
}

func TestMigrar_NeverDiscardsEditedOrUnlockedCopies(t *testing.T) {
	root, _ := legacyProject(t)
	edited := filepath.Join(root, Dir, "auth", ManifestName)
	writeFile(t, edited, strings.Replace(authBlueprint, "nunca por nombre de rol", "o por rol", 1))
	writeFile(t, filepath.Join(root, Dir, "crm", ManifestName), strings.Replace(authBlueprint, "id: auth", "id: crm", 1))

	m, err := Migrar(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Migrados) != 0 || len(m.Pendientes) != 2 {
		t.Fatalf("las copias editadas o sin lockfile quedan pendientes: %+v", m)
	}
	if _, err := os.Stat(edited); err != nil {
		t.Fatal("una copia editada no se borra: sus cambios deben llegar al repositorio de la organización")
	}
	problems := Check(root)
	if !hasRule(problems, "BP-05-LEGACY") {
		t.Fatalf("el gate pide terminar la migración: %v", problems)
	}
	for _, p := range problems {
		if p.Regla == "BP-05-LEGACY" && strings.Contains(p.Descripcion, "auth") && !strings.Contains(p.Descripcion, "qdd blueprint contribute auth") {
			t.Fatalf("la descripción guía a la IA a aportar los cambios: %s", p.Descripcion)
		}
	}
}

func TestNormalizarRemote(t *testing.T) {
	cases := map[string]string{
		"git@github.com:Acme/qdd.git":            "github.com/Acme/qdd",
		"https://github.com/acme/qdd.git":        "github.com/acme/qdd",
		"https://user@GitHub.com/acme/qdd/":      "github.com/acme/qdd",
		"ssh://git@gitlab.acme.cl:22/eq/app.git": "gitlab.acme.cl/eq/app",
		"/home/yo/repos/app":                     "",
	}
	for in, want := range cases {
		if got := normalizarRemote(in); got != want {
			t.Errorf("normalizarRemote(%q) = %q; se esperaba %q", in, got, want)
		}
	}
}

func TestRegistro_ProjectsOfTheSameOrganizationSeeItWithoutConfiguration(t *testing.T) {
	repo := gitOrgRepo(t)
	a := projectWithRemote(t, "git@github.com:acme/tienda.git")
	if _, problems := AddManaged(AddInput{Root: a, Origen: repo, Ref: "v1.0.0", Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}

	b := projectWithRemote(t, "https://github.com/acme/reservas.git")
	reg, err := LoadRegistro(b)
	if err != nil || reg.Origen != repo || !reg.Central {
		t.Fatalf("un proyecto de la misma organización hereda el registro central: %+v (%v)", reg, err)
	}
	res, problems := AddManaged(AddInput{Root: b, Blueprints: []string{"auditoria"}})
	if len(problems) > 0 || len(res.Entries) != 1 {
		t.Fatalf("basta con nombrar el blueprint: %+v %v", res, problems)
	}

	other := projectWithRemote(t, "git@github.com:otra-empresa/app.git")
	if reg, _ := LoadRegistro(other); reg.Origen != "" {
		t.Fatalf("un proyecto de otra organización no ve el registro: %+v", reg)
	}
}

func TestSincronizar_ReportsNewerVersions(t *testing.T) {
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Ref: "v1.0.0", Blueprints: []string{"auth", "auditoria"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	publicar(t, repo, "1.1.0", "v1.1.0")

	s, err := Sincronizar(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Ref != "v1.1.0" || len(s.Novedades) != 1 || s.Novedades[0].ID != "auth" || s.Novedades[0].Disponible != "1.1.0" {
		t.Fatalf("se esperaba auth 1.0.0 → 1.1.0 en v1.1.0: %+v", s)
	}
}

func TestAportar_PreparesABranchInTheCentralWorkspace(t *testing.T) {
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Ref: "v1.0.0", Blueprints: []string{"pagos"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	ap, err := Aportar(root, "pagos")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := Home()
	if !isUnder(ap.Carpeta, home) || !strings.HasSuffix(filepath.ToSlash(ap.Carpeta), "estandares/pagos-webpay") || ap.Rama != "aporte/pagos" {
		t.Fatalf("el aporte se prepara en la carpeta central, en la subcarpeta del blueprint y en su rama: %+v", ap)
	}
	if isUnder(ap.Carpeta, root) {
		t.Fatal("el repositorio nunca se clona dentro del proyecto")
	}

	// Un trabajo en curso no se pisa al volver a pedir el aporte.
	writeFile(t, filepath.Join(ap.Carpeta, "NOTAS.md"), "en curso\n")
	again, err := Aportar(root, "pagos")
	if err != nil || again.Carpeta != ap.Carpeta {
		t.Fatalf("se reutiliza el mismo espacio de trabajo: %+v (%v)", again, err)
	}
	if _, err := os.Stat(filepath.Join(ap.Carpeta, "NOTAS.md")); err != nil {
		t.Fatal("los cambios sin commitear se conservan")
	}
}
