package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Un proyecto puede usar blueprints de varias fuentes: el repositorio de su
// organización y otros (p. ej. el repositorio público de una persona). El
// lockfile dice de qué fuente salió cada uno.

// miBlueprint es el manifiesto de mi-bp, un blueprint que solo existe en la
// segunda fuente, en la versión indicada.
func miBlueprint(version string) string {
	return strings.NewReplacer("id: auth", "id: mi-bp", "nombre: Usuarios y acceso", "nombre: Mi estándar", "version: 1.0.0", "version: "+version).Replace(authBlueprint)
}

// personalRepo es una segunda fuente, publicada como v1.0.0: tiene su propio
// auth (mismo id que el de la organización, otro contenido) y mi-bp.
func personalRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "auth", ManifestName), strings.Replace(authBlueprint, "nombre: Usuarios y acceso", "nombre: Mi auth", 1))
	writeFile(t, filepath.Join(repo, "mi-bp", ManifestName), miBlueprint("1.0.0"))
	publicarTodo(t, repo)
	return repo
}

// publicarMiBP sube la versión de mi-bp en la segunda fuente y la etiqueta.
func publicarMiBP(t *testing.T, repo, version, tag string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, "mi-bp", ManifestName), miBlueprint(version))
	gitIn(t, repo, "commit", "-q", "-am", version)
	gitIn(t, repo, "tag", tag)
}

func mustAddManaged(t *testing.T, in AddInput) {
	t.Helper()
	if _, problems := AddManaged(in); len(problems) > 0 {
		t.Fatalf("no se pudo importar %v desde %q: %v", in.Blueprints, in.Origen, problems)
	}
}

func lockEntry(t *testing.T, root, id string) LockEntry {
	t.Helper()
	lock, err := LoadLock(root)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := lock.Entry(id)
	if !ok {
		t.Fatalf("'%s' no está en el lockfile", id)
	}
	return e
}

// fuente es el repositorio tal como lo guarda el lockfile.
func fuente(t *testing.T, origen string) string {
	t.Helper()
	repo, _, err := resolverOrigen(origen)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

// sincronizacionDe busca el resultado de una fuente.
func sincronizacionDe(t *testing.T, res []Sincronizacion, repo string) Sincronizacion {
	t.Helper()
	for _, s := range res {
		if s.Origen == repo {
			return s
		}
	}
	t.Fatalf("no se sincronizó %s: %+v", repo, res)
	return Sincronizacion{}
}

// FND-064: un blueprint con el mismo id desde otra fuente reemplazaba al
// importado sin avisar.
func TestAdd_SameIDFromAnotherSourceIsNotSilentlyReplaced(t *testing.T) {
	org, personal := gitOrgRepo(t), personalRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: org, Blueprints: []string{"auth"}})

	_, problems := Add(AddInput{Root: root, Origen: personal, Blueprints: []string{"auth", "mi-bp"}})
	if !hasProblem(problems, "'auth' ya está importado desde "+fuente(t, org)) || !hasProblem(problems, "--reemplazar") {
		t.Fatalf("el choque de ids se rechaza y se dice cómo reemplazarlo: %v", problems)
	}
	if e := lockEntry(t, root, "auth"); e.Origen != fuente(t, org) {
		t.Fatalf("el auth de la organización sigue en el lockfile: %+v", e)
	}
	if hasProblem(lockIDs(t, root), "mi-bp") {
		t.Fatal("si alguno se rechaza no se importa ninguno")
	}

	if _, problems := Add(AddInput{Root: root, Origen: personal, Blueprints: []string{"auth"}, Reemplazar: true}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if e := lockEntry(t, root, "auth"); e.Origen != fuente(t, personal) {
		t.Fatalf("con --reemplazar, auth pasa a la otra fuente: %+v", e)
	}
}

// FND-065: 'qdd blueprint add <id>' (lo que pide BP-06-BEHIND) buscaba el id
// en el repositorio de la organización aunque se hubiera importado de otra
// fuente.
func TestAddManaged_ImportedBlueprintUpdatesFromItsOwnSource(t *testing.T) {
	freshHome(t)
	org, personal := gitOrgRepo(t), personalRepo(t)
	root := projectWithRemote(t, "git@github.com:acme/tienda.git")
	mustAddManaged(t, AddInput{Root: root, Origen: org, Blueprints: []string{"auth"}})
	mustAddManaged(t, AddInput{Root: root, Origen: personal, Blueprints: []string{"mi-bp"}})
	if reg, _ := LoadRegistro(root); reg.Origen != fuente(t, org) {
		t.Fatalf("importar de otra fuente no cambia el repositorio de la organización: %+v", reg)
	}
	publicarMiBP(t, personal, "1.1.0", "v1.1.0")

	mustAddManaged(t, AddInput{Root: root, Blueprints: []string{"mi-bp", "auth"}})
	if e := lockEntry(t, root, "mi-bp"); e.Origen != fuente(t, personal) || e.Version != "1.1.0" {
		t.Fatalf("mi-bp se actualiza desde su propia fuente: %+v", e)
	}
	if e := lockEntry(t, root, "auth"); e.Origen != fuente(t, org) {
		t.Fatalf("auth sigue viniendo de la organización: %+v", e)
	}
}

// FND-066: sync solo miraba el repositorio de la organización.
func TestSincronizar_EverySourceOfTheProject(t *testing.T) {
	org, personal := gitOrgRepo(t), personalRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: org, Blueprints: []string{"auth"}})
	mustAddManaged(t, AddInput{Root: root, Origen: personal, Blueprints: []string{"mi-bp"}})
	publicar(t, org, "1.1.0", "v1.1.0")
	publicarMiBP(t, personal, "2.0.0", "v2.0.0")

	res, err := Sincronizar(root)
	if err != nil {
		t.Fatal(err)
	}
	if s := sincronizacionDe(t, res, fuente(t, org)); len(s.Novedades) != 1 || s.Novedades[0].ID != "auth" || s.Novedades[0].Disponible != "1.1.0" {
		t.Fatalf("auth 1.0.0 → 1.1.0 en la organización: %+v", s)
	}
	if s := sincronizacionDe(t, res, fuente(t, personal)); s.Ref != "v2.0.0" || len(s.Novedades) != 1 || s.Novedades[0].ID != "mi-bp" {
		t.Fatalf("mi-bp 1.0.0 → 2.0.0 en la segunda fuente: %+v", s)
	}

	var behind []string
	for _, p := range Check(root) {
		if p.Regla == "BP-06-BEHIND" {
			behind = append(behind, p.Descripcion)
		}
	}
	if !hasProblem(behind, "qdd blueprint add mi-bp") || !hasProblem(behind, fuente(t, personal)) {
		t.Fatalf("el gate dice de qué fuente es la versión nueva: %v", behind)
	}
	mustAddManaged(t, AddInput{Root: root, Blueprints: []string{"auth", "mi-bp"}})
	if problems := Check(root); len(problems) != 0 {
		t.Fatalf("siguiendo lo que pide el gate, el proyecto queda al día: %v", problems)
	}
}

func TestSincronizar_AnUnreachableSourceDoesNotHideTheOthers(t *testing.T) {
	org, personal := gitOrgRepo(t), personalRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: org, Blueprints: []string{"auth"}})
	mustAddManaged(t, AddInput{Root: root, Origen: personal, Blueprints: []string{"mi-bp"}})
	repoPersonal := fuente(t, personal)
	if err := os.RemoveAll(personal); err != nil {
		t.Fatal(err)
	}
	publicar(t, org, "1.1.0", "v1.1.0")

	res, err := Sincronizar(root)
	if err != nil {
		t.Fatal(err)
	}
	if s := sincronizacionDe(t, res, fuente(t, org)); s.Problema != "" || len(s.Novedades) != 1 {
		t.Fatalf("la organización se sincroniza igual: %+v", s)
	}
	if s := sincronizacionDe(t, res, repoPersonal); s.Problema == "" {
		t.Fatalf("la fuente inaccesible se informa: %+v", s)
	}
}

func TestAportar_NewBlueprintGoesToTheChosenSource(t *testing.T) {
	sinGH(t)
	org, personal := gitOrgRepo(t), personalRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: org, Blueprints: []string{"auth"}})

	ap, err := Aportar(root, "nuevo", personal)
	if err != nil {
		t.Fatal(err)
	}
	if ap.Repo != fuente(t, personal) || filepath.Base(ap.Carpeta) != "nuevo" {
		t.Fatalf("el blueprint nuevo se prepara en la fuente elegida: %+v", ap)
	}
	writeFile(t, filepath.Join(ap.Carpeta, ManifestName), strings.Replace(recetaLimpia, "id: panel", "id: nuevo", 1)) // un blueprint nuevo nace sin defectos
	if _, err := Proponer(root, "nuevo", "mi estándar", personal); err != nil {
		t.Fatal(err)
	}
	if _, err := git(personal, "rev-parse", "--verify", "--quiet", "refs/heads/aporte/nuevo"); err != nil {
		t.Fatal("la rama se publica en la fuente elegida")
	}
	if _, err := git(org, "rev-parse", "--verify", "--quiet", "refs/heads/aporte/nuevo"); err == nil {
		t.Fatal("nada se publica en el repositorio de la organización")
	}
	gitIn(t, personal, "merge", "-q", "--no-edit", "aporte/nuevo") // el PR se integra

	pub, err := Publicar(root, personal)
	if err != nil {
		t.Fatal(err)
	}
	if tags, _ := versionTags(personal); len(tags) == 0 || tags[0] != pub.Tag || pub.Tag == "v1.0.0" {
		t.Fatalf("la versión nueva se etiqueta en la fuente elegida: %+v %v", pub, tags)
	}
	if tags, _ := versionTags(org); tags[0] != "v1.0.0" {
		t.Fatalf("la organización no se toca: %v", tags)
	}
}

func TestAportar_AnImportedBlueprintKeepsItsSource(t *testing.T) {
	org, personal := gitOrgRepo(t), personalRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: org, Blueprints: []string{"auth"}})
	mustAddManaged(t, AddInput{Root: root, Origen: personal, Blueprints: []string{"mi-bp"}})

	ap, err := Aportar(root, "mi-bp", "")
	if err != nil || ap.Repo != fuente(t, personal) {
		t.Fatalf("sin indicar fuente, el aporte va a la fuente del lockfile: %+v (%v)", ap, err)
	}
	if _, err := Aportar(root, "auth", personal); err == nil || !strings.Contains(err.Error(), "está importado desde") {
		t.Fatalf("un blueprint importado no se aporta a otra fuente: %v", err)
	}
}

func TestAddManaged_AFailingSourceLeavesTheLockfileAsItWas(t *testing.T) {
	org, personal := gitOrgRepo(t), personalRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: org, Blueprints: []string{"auth"}})
	mustAddManaged(t, AddInput{Root: root, Origen: personal, Blueprints: []string{"mi-bp"}})
	antes, err := os.ReadFile(filepath.Join(root, LockFile))
	if err != nil {
		t.Fatal(err)
	}
	publicar(t, org, "1.1.0", "v1.1.0")
	freshHome(t) // otra máquina: sin espejo de la segunda fuente…
	if err := os.RemoveAll(personal); err != nil {
		t.Fatal(err)
	} // …y la segunda fuente ya no está

	if _, problems := AddManaged(AddInput{Root: root, Blueprints: []string{"auth", "mi-bp"}}); len(problems) == 0 {
		t.Fatal("una fuente inaccesible se informa")
	}
	if despues, _ := os.ReadFile(filepath.Join(root, LockFile)); string(despues) != string(antes) {
		t.Fatalf("si una fuente falla no se actualiza ninguna:\n%s", despues)
	}
}

func TestAddManaged_RefBelongsToASingleSource(t *testing.T) {
	org, personal := gitOrgRepo(t), personalRepo(t)
	root := t.TempDir()
	mustAddManaged(t, AddInput{Root: root, Origen: org, Blueprints: []string{"auth"}})
	mustAddManaged(t, AddInput{Root: root, Origen: personal, Blueprints: []string{"mi-bp"}})
	if _, problems := AddManaged(AddInput{Root: root, Ref: "v1.0.0", Blueprints: []string{"auth", "mi-bp"}}); !hasProblem(problems, "--ref") {
		t.Fatalf("una versión es de una sola fuente: %v", problems)
	}
	mustAddManaged(t, AddInput{Root: root, Ref: "v1.0.0", Blueprints: []string{"mi-bp"}})
}
