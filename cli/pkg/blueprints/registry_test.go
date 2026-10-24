package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const baseConfig = "# configuración del equipo\nproject: demo\nlanguages: [go]\ntest_commands:\n    - go test ./...\n"

func projectWithConfig(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, configFile), baseConfig)
	return root
}

func TestSaveRegistro_KeepsTheRestOfConfig(t *testing.T) {
	root := projectWithConfig(t)
	if err := SaveRegistro(root, Registro{Origen: "git@github.com:acme/bp.git", Ref: "v1.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveRegistro(root, Registro{Origen: "git@github.com:acme/bp.git", Ref: "v1.1.0"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, configFile))
	text := string(data)
	for _, want := range []string{"# configuración del equipo", "project: demo", "go test ./...", "ref: v1.1.0"} {
		if !strings.Contains(text, want) {
			t.Errorf("config.yaml perdió %q:\n%s", want, text)
		}
	}
	if strings.Count(text, "blueprints:") != 1 {
		t.Fatalf("la clave blueprints debe reemplazarse, no duplicarse:\n%s", text)
	}
	reg, err := LoadRegistro(root)
	if err != nil || reg.Ref != "v1.1.0" {
		t.Fatalf("registro = %+v, %v", reg, err)
	}
}

func TestAddManaged_RemembersRepositoryAndThenOnlyNamesAreNeeded(t *testing.T) {
	root := projectWithConfig(t)
	repo := orgRepo(t, nil)

	first, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}})
	if len(problems) > 0 || !first.Guardado || first.Registro.Origen != repo {
		t.Fatalf("la primera vez debe registrar el repositorio: %+v %v", first, problems)
	}
	second, problems := AddManaged(AddInput{Root: root, Blueprints: []string{"auditoria", "pagos"}})
	if len(problems) > 0 || len(second.Entries) != 2 || second.Guardado {
		t.Fatalf("después basta con nombrarlos: %+v %v", second, problems)
	}
	if got := len(lockIDs(t, root)); got != 3 {
		t.Fatalf("se esperaban 3 blueprints importados: %d", got)
	}
}

func TestAddManaged_WithoutRegistryAsksForRepository(t *testing.T) {
	_, problems := AddManaged(AddInput{Root: projectWithConfig(t), Blueprints: []string{"auth"}})
	if !hasProblem(problems, "todavía no hay repositorio") {
		t.Fatalf("sin registro debe pedir el repositorio una vez: %v", problems)
	}
}

func TestAddManaged_SingleBlueprintFolderIsNotARegistry(t *testing.T) {
	root := projectWithConfig(t)
	res, problems := AddManaged(AddInput{Root: root, Origen: sourceBlueprint(t, authBlueprint)})
	if len(problems) > 0 || res.Guardado {
		t.Fatalf("una carpeta que es un blueprint no es un repositorio de la organización: %+v %v", res, problems)
	}
}

func TestAddManaged_OtherRepositoryDoesNotReplaceRegistry(t *testing.T) {
	root := projectWithConfig(t)
	repo := orgRepo(t, nil)
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if _, problems := AddManaged(AddInput{Root: root, Origen: orgRepo(t, nil), Blueprints: []string{"pagos"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if reg, _ := LoadRegistro(root); reg.Origen != repo {
		t.Fatalf("importar algo puntual de otro repositorio no cambia el registro: %+v", reg)
	}
}

func TestEsOrigen(t *testing.T) {
	for _, origen := range []string{"git@github.com:acme/bp.git", "https://x/y.git", "./bp", "../bp", "~/bp", "/abs/bp"} {
		if !EsOrigen(origen) {
			t.Errorf("%q es un origen", origen)
		}
	}
	for _, id := range []string{"auth", "pagos-webpay", "auditoria_v2"} {
		if EsOrigen(id) {
			t.Errorf("%q es un id", id)
		}
	}
}

func TestEstandares_ReportsAppliedModules(t *testing.T) {
	root := importedProject(t)
	if e := Estandares(root); len(e) != 1 || len(e[0].Modulos) != 0 || e[0].Rutas[0] != "internal/auth" {
		t.Fatalf("recién importado, auth está sin aplicar: %+v", e)
	}
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "auth", Respuestas: map[string]string{"identificador": "correo", "recuperacion": "correo"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if e := Estandares(root); e[0].Modulos[0] != "auth" {
		t.Fatalf("debe indicar en qué ficha se aplicó: %+v", e)
	}
}
