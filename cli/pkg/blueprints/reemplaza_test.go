package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0009: un blueprint que reemplaza a otro.

// accesoQueReemplazaAAuth es auth con otro id, que declara reemplazarlo y suma una pregunta.
var accesoQueReemplazaAAuth = strings.NewReplacer(
	"id: auth", "id: acceso\nreemplaza: [auth]",
	"nombre: Usuarios y acceso", "nombre: Acceso",
	"version: 1.0.0", "version: 2.0.0",
).Replace(authBlueprint)

// proyectoConAuthAplicado importa auth desde repo y lo aplica como el módulo crm.
func proyectoConAuthAplicado(t *testing.T, repo string) string {
	t.Helper()
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "auth", Modulo: "crm", Respuestas: map[string]string{"identificador": "correo", "recuperacion": "correo"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	return root
}

// publicarReemplazo agrega acceso (que reemplaza a auth) a la fuente y lo etiqueta.
func publicarReemplazo(t *testing.T, repo, tag string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, "acceso", ManifestName), accesoQueReemplazaAAuth)
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "acceso reemplaza a auth")
	gitIn(t, repo, "tag", tag)
}

func TestValidate_ReemplazaConIdsValidosYAjenos(t *testing.T) {
	b := mustParse(t, accesoQueReemplazaAAuth)
	if problems := b.Validate(); len(problems) > 0 {
		t.Fatalf("reemplazar a otro id es válido: %v", problems)
	}
	b.Reemplaza = []string{"acceso"}
	if problems := b.Validate(); !hasProblem(problems, "reemplaza") {
		t.Fatalf("un blueprint no se reemplaza a sí mismo: %v", problems)
	}
	b.Reemplaza = []string{"Con Espacios"}
	if problems := b.Validate(); !hasProblem(problems, "reemplaza") {
		t.Fatalf("reemplaza solo lleva ids válidos: %v", problems)
	}
}

func TestCheck_UnBlueprintReemplazadoPideMigrar_BP07(t *testing.T) {
	repo := gitOrgRepo(t)
	root := proyectoConAuthAplicado(t, repo)
	publicarReemplazo(t, repo, "v2.0.0")
	if hasRule(Check(root), "BP-07-REPLACED") {
		t.Fatal("sin sincronizar, el gate no inventa novedades")
	}
	if _, err := Sincronizar(root); err != nil {
		t.Fatal(err)
	}
	problems := Check(root)
	p, ok := problemaDe(problems, "BP-07-REPLACED")
	if !ok || !strings.Contains(p.Descripcion, "qdd blueprint add acceso") || !strings.Contains(p.Descripcion, "--modulo crm") {
		t.Fatalf("el gate dice quién lo reemplaza y cómo migrar, con el módulo: %v", problems)
	}
	if hasRule(problems, "BP-06-BEHIND") {
		t.Fatalf("un reemplazado no se reporta además como atrasado: %v", problems)
	}
}

func TestAdd_ElReemplazoMigraElLockfileYElRegistroConservandoElModulo(t *testing.T) {
	repo := gitOrgRepo(t)
	root := proyectoConAuthAplicado(t, repo)
	publicarReemplazo(t, repo, "v2.0.0")
	if _, err := Sincronizar(root); err != nil {
		t.Fatal(err)
	}
	if _, problems := AddManaged(AddInput{Root: root, Blueprints: []string{"acceso"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if ids := lockIDs(t, root); len(ids) != 1 || !strings.HasPrefix(ids[0], "acceso") {
		t.Fatalf("el reemplazado sale del lockfile: %v", ids)
	}
	a, err := LoadApplication(root, "crm")
	if err != nil {
		t.Fatal(err)
	}
	if a.Blueprint != "acceso" || a.Modulo != "crm" || a.Respuestas["identificador"] != "correo" || a.Version != "1.0.0" {
		t.Fatalf("el registro pasa al nuevo blueprint con el mismo módulo y sus respuestas: %+v", a)
	}
	problems := Check(root)
	p, ok := problemaDe(problems, "BP-03-OUTDATED")
	if !ok || !strings.Contains(p.Descripcion, "qdd blueprint apply acceso --modulo crm") {
		t.Fatalf("después de migrar, el gate pide volver a aplicarlo: %v", problems)
	}
	if hasRule(problems, "BP-04-MISSING") || hasRule(problems, "BP-07-REPLACED") {
		t.Fatalf("migrado, el módulo no queda huérfano ni reemplazado: %v", problems)
	}
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "acceso", Modulo: "crm"}); len(problems) > 0 {
		t.Fatalf("volver a aplicarlo reutiliza las respuestas: %v", problems)
	}
	if len(Check(root)) != 0 {
		t.Fatalf("aplicado de nuevo, el gate queda limpio: %v", Check(root))
	}
}

func TestAdd_UnReemplazoDeOtraFuenteNoTocaElBlueprint(t *testing.T) {
	repo := gitOrgRepo(t)
	root := proyectoConAuthAplicado(t, repo)
	otra := t.TempDir()
	writeFile(t, filepath.Join(otra, "acceso", ManifestName), accesoQueReemplazaAAuth)
	publicarTodo(t, otra)
	if _, problems := AddManaged(AddInput{Root: root, Origen: otra, Blueprints: []string{"acceso"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	ids := lockIDs(t, root)
	if len(ids) != 2 {
		t.Fatalf("un blueprint de otra fuente no reemplaza al de la organización: %v", ids)
	}
	if a, _ := LoadApplication(root, "crm"); a.Blueprint != "auth" {
		t.Fatalf("el registro sigue en auth: %+v", a)
	}
}

func TestBuscarFichaPorBlueprint(t *testing.T) {
	repo := gitOrgRepo(t)
	root := proyectoConAuthAplicado(t, repo)
	if m, ok := FichaDeBlueprint(root, "acceso", "auth"); !ok || m.ID != "crm" {
		t.Fatalf("la ficha se encuentra por el blueprint que la generó, en orden: %+v %v", m, ok)
	}
	if _, ok := FichaDeBlueprint(root, "acceso"); ok {
		t.Fatal("sin ficha de esos blueprints, no hay ficha")
	}
	if _, err := os.Stat(filepath.Join(root, ".qdd", "project", "modules", "crm.yaml")); err != nil {
		t.Fatal(err)
	}
}

func problemaDe(problems []Problema, regla string) (Problema, bool) {
	for _, p := range problems {
		if p.Regla == regla {
			return p, true
		}
	}
	return Problema{}, false
}
