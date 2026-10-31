package blueprints

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestClasificar_QDDDecidesTheVersionBump(t *testing.T) {
	base := mustParse(t, authBlueprint)
	cases := []struct {
		nombre string
		cambia func(string) string
		want   string
	}{
		{"redacción de una pregunta", func(s string) string {
			return strings.Replace(s, "¿Con qué inicia sesión el usuario?", "¿Con qué dato inicia sesión?", 1)
		}, CambioPatch},
		{"nueva invariante", func(s string) string {
			return strings.Replace(s, `invariantes: ["Autorización por permiso, nunca por nombre de rol"]`, `invariantes: ["Autorización por permiso, nunca por nombre de rol", "Toda sesión expira"]`, 1)
		}, CambioMinor},
		{"nueva opción", func(s string) string {
			return strings.Replace(s, "      - valor: evolution\n", "      - valor: evolution\n      - valor: twilio\n", 1)
		}, CambioMinor},
		{"invariante relajada", func(s string) string {
			return strings.Replace(s, "nunca por nombre de rol", "o por rol", 1)
		}, CambioMajor},
		{"opción eliminada", func(s string) string {
			return strings.Replace(s, "      - valor: evolution\n", "", 1)
		}, CambioMajor},
		{"escenario con otro resultado", func(s string) string {
			return strings.Replace(s, "la respuesta no revela si la cuenta existe", "se informa que la cuenta no existe", 1)
		}, CambioMajor},
	}
	for _, c := range cases {
		got, razones := Clasificar(base, mustParse(t, c.cambia(authBlueprint)))
		if got != c.want {
			t.Errorf("%s: se esperaba %s, salió %q (%v)", c.nombre, c.want, got, razones)
		}
	}
	if got, _ := Clasificar(base, base); got != "" {
		t.Fatalf("sin cambios no hay versión nueva: %q", got)
	}
}

func TestSubirVersion(t *testing.T) {
	for in, want := range map[[2]string]string{
		{"1.2.3", CambioPatch}:  "1.2.4",
		{"1.2.3", CambioMinor}:  "1.3.0",
		{"v1.2.3", CambioMajor}: "2.0.0",
	} {
		if got := subirVersion(in[0], in[1]); got != want {
			t.Errorf("subirVersion(%v) = %s; se esperaba %s", in, got, want)
		}
	}
}

// sinGH evita llamar a la CLI de GitHub en las pruebas y registra el PR pedido.
func sinGH(t *testing.T) *[]string {
	t.Helper()
	var pedidos []string
	previo := abrirPR
	abrirPR = func(espacio, rama, titulo, cuerpo string) (string, error) {
		pedidos = append(pedidos, rama+"|"+titulo+"|"+cuerpo)
		return "https://github.com/acme/qdd-blueprints/pull/1", nil
	}
	t.Cleanup(func() { abrirPR = previo })
	return &pedidos
}

func TestProponer_ValidatesVersionsPushesAndOpensThePR(t *testing.T) {
	pedidos := sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if _, err := Proponer(root, "auth", "sin aporte", ""); err == nil || !strings.Contains(err.Error(), "qdd blueprint contribute auth") {
		t.Fatalf("sin aporte en curso, se pide preparar uno: %v", err)
	}
	ap, err := Aportar(root, "auth", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Proponer(root, "auth", "nada", ""); err == nil || !strings.Contains(err.Error(), "no tiene cambios") {
		t.Fatalf("un aporte sin cambios se rechaza: %v", err)
	}
	if _, err := Proponer(root, "auth", "", ""); err == nil {
		t.Fatal("un aporte sin descripción se rechaza")
	}

	manifest := filepath.Join(ap.Carpeta, ManifestName)
	editado := strings.Replace(authBlueprint, `invariantes: ["Autorización por permiso, nunca por nombre de rol"]`, `invariantes: ["Autorización por permiso, nunca por nombre de rol", "Toda sesión expira"]`, 1)
	writeFile(t, manifest, strings.Replace(editado, "version: 1.0.0", "version: 7.0.0", 1))

	p, err := Proponer(root, "auth", "las sesiones expiran", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Cambio != CambioMinor || p.Version != "1.1.0" || p.Rama != "aporte/auth" || p.PR == "" {
		t.Fatalf("QDD fija la versión según el cambio (minor → 1.1.0), no la que escribió el autor: %+v", p)
	}
	if got := versionEn(repo, "aporte/auth", "auth"); got != "1.1.0" {
		t.Fatalf("la rama publicada en el repositorio lleva la versión fijada por QDD: %q", got)
	}
	if len(*pedidos) != 1 || !strings.Contains((*pedidos)[0], "Cambio minor → v1.1.0") || !strings.Contains((*pedidos)[0], "Toda sesión expira") {
		t.Fatalf("el PR explica el cambio y su versión: %v", *pedidos)
	}
}

func TestProponer_InvalidBlueprintIsNotPublished(t *testing.T) {
	sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	ap, err := Aportar(root, "auth", "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(ap.Carpeta, ManifestName), "id: auth\nnombre: X\nversion: 1.0.0\n")
	if _, err := Proponer(root, "auth", "rompe todo", ""); err == nil || !strings.Contains(err.Error(), "no es válido") {
		t.Fatalf("un blueprint inválido no se propone: %v", err)
	}
	if _, err := git(repo, "rev-parse", "--verify", "--quiet", "refs/heads/aporte/auth"); err == nil {
		t.Fatal("nada se publica en el repositorio de la organización")
	}
}

func TestPublicar_TagsTheRepositoryFromTheChanges(t *testing.T) {
	sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if _, err := Publicar(root, ""); err == nil || !strings.Contains(err.Error(), "no hay cambios") {
		t.Fatalf("sin cambios no hay versión que publicar: %v", err)
	}

	ap, err := Aportar(root, "auth", "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(ap.Carpeta, ManifestName), strings.Replace(authBlueprint, "        invariantes: [\"El correo se normaliza a minúsculas\"]\n", "        descripcion: Inicia sesión con su correo, tal como lo escribe.\n", 1))
	if _, err := Proponer(root, "auth", "el correo ya no se normaliza", ""); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "merge", "-q", "--no-edit", "aporte/auth") // el PR se integra

	pub, err := Publicar(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if pub.Tag != "v2.0.0" || pub.Cambio != CambioMajor {
		t.Fatalf("quitar una invariante es major: el repositorio pasa a v2.0.0: %+v", pub)
	}
	if tags, _ := versionTags(repo); len(tags) == 0 || tags[0] != "v2.0.0" {
		t.Fatalf("el tag queda publicado en el repositorio: %v", tags)
	}
}

func TestPublicar_RejectsChangesThatSkippedPropose(t *testing.T) {
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	// Alguien commitea directo en main, sin subir la versión.
	writeFile(t, filepath.Join(repo, "auth", ManifestName), strings.Replace(authBlueprint, "nunca por nombre de rol", "o por rol", 1))
	gitIn(t, repo, "commit", "-q", "-am", "a mano")

	if _, err := Publicar(root, ""); err == nil || !strings.Contains(err.Error(), "qdd blueprint propose") {
		t.Fatalf("un cambio sin versión dictada por QDD no se publica: %v", err)
	}
}

func TestCheck_ProjectBehindItsOrganization(t *testing.T) {
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	publicar(t, repo, "1.1.0", "v1.1.0")
	if hasRule(Check(root), "BP-06-BEHIND") {
		t.Fatal("sin sincronizar, el gate no inventa novedades")
	}
	if _, err := Sincronizar(root); err != nil {
		t.Fatal(err)
	}
	if !hasRule(Check(root), "BP-06-BEHIND") {
		t.Fatal("un proyecto atrasado respecto de su organización no pasa el gate")
	}
	if _, problems := AddManaged(AddInput{Root: root, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if problems := Check(root); len(problems) != 0 {
		t.Fatalf("al actualizar, el proyecto vuelve a estar al día: %v", problems)
	}
}
