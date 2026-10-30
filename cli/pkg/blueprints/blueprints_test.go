package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

const authBlueprint = `id: auth
nombre: Usuarios y acceso
version: 1.0.0
organizacion: ACME
vocabulario:
  - termino: rol
    definicion: Conjunto de permisos.
preguntas:
  - id: identificador
    pregunta: ¿Con qué inicia sesión el usuario?
    opciones:
      - valor: correo
        invariantes: ["El correo se normaliza a minúsculas"]
      - valor: rut
        invariantes: ["El RUT se valida con módulo 11"]
        conformidad:
          - id: RUT-01
            titulo: RUT con dígito verificador inválido
            cuando: se inicia sesión con 12345678-0
            entonces: se rechaza sin consultar la base de datos
  - id: recuperacion
    pregunta: ¿Por qué canal se recupera la contraseña?
    opciones:
      - valor: correo
      - valor: whatsapp
        depende_de: ["Cuenta de WhatsApp Business API"]
  - id: proveedor_whatsapp
    pregunta: ¿Qué proveedor de WhatsApp?
    si: {recuperacion: whatsapp}
    opciones:
      - valor: meta
      - valor: evolution
ficha:
  nombre: Usuarios, roles y permisos
  rutas: [internal/auth]
  objetivo_operativo:
    problema: Controlar quién entra y qué puede hacer.
    criterio_de_exito: Ningún acceso sin permiso explícito.
  objetivo_tecnologico:
    arquitectura: JWT de acceso corto + refresh rotativo; autorización por permiso.
    invariantes: ["Autorización por permiso, nunca por nombre de rol"]
conformidad:
  - id: AUTH-01
    titulo: Credenciales inválidas
    cuando: se inicia sesión con una contraseña incorrecta
    entonces: la respuesta no revela si la cuenta existe
`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// sourceBlueprint publica un blueprint suelto: un repositorio git cuya raíz es
// el blueprint, con la versión v1.0.0 etiquetada.
func sourceBlueprint(t *testing.T, content string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "auth")
	writeFile(t, filepath.Join(dir, ManifestName), content)
	writeFile(t, filepath.Join(dir, "referencias", "go", "README.md"), "referencia\n")
	publicarTodo(t, dir)
	return dir
}

// publicarTodo commitea todo el repositorio y lo etiqueta v1.0.0.
func publicarTodo(t *testing.T, repo string) {
	t.Helper()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "v1")
	gitIn(t, repo, "tag", "v1.0.0")
}

func importedProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, problems := Add(AddInput{Root: root, Origen: sourceBlueprint(t, authBlueprint)}); len(problems) > 0 {
		t.Fatalf("no se pudo importar: %v", problems)
	}
	return root
}

func mustParse(t *testing.T, content string) Blueprint {
	t.Helper()
	b, err := Parse([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func hasProblem(problems []string, fragment string) bool {
	for _, p := range problems {
		if strings.Contains(p, fragment) {
			return true
		}
	}
	return false
}

func hasRule(problems []Problema, rule string) bool {
	for _, p := range problems {
		if p.Regla == rule {
			return true
		}
	}
	return false
}

func TestValidate_AcceptsCompleteBlueprint(t *testing.T) {
	if problems := mustParse(t, authBlueprint).Validate(); len(problems) > 0 {
		t.Fatalf("un blueprint completo no debe tener problemas: %v", problems)
	}
}

func TestTemplate_IsAValidBlueprint(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".qdd", "core", "templates", ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if problems := mustParse(t, string(data)).Validate(); len(problems) > 0 {
		t.Fatalf("la plantilla pública debe ser un blueprint válido: %v", problems)
	}
}

func TestValidate_RejectsUnresolvableQuestions(t *testing.T) {
	b := mustParse(t, authBlueprint)
	b.Version = "v1"
	b.Preguntas[0].PorDefecto = "telefono"
	b.Preguntas[1].Si = map[string]Valores{"proveedor_whatsapp": {"meta"}}
	b.Preguntas[2].Opciones = b.Preguntas[2].Opciones[:1]

	problems := b.Validate()
	for _, want := range []string{"semver", "por_defecto 'telefono'", "no es una pregunta anterior", "al menos dos opciones"} {
		if !hasProblem(problems, want) {
			t.Errorf("falta el problema %q en %v", want, problems)
		}
	}
}

func TestValidate_SameScenarioInAlternativeOptionsButNotOverBase(t *testing.T) {
	b := mustParse(t, authBlueprint)
	rut := b.Preguntas[0].Opciones[1].Conformidad[0]
	b.Preguntas[0].Opciones[0].Conformidad = []Escenario{rut}
	if problems := b.Validate(); len(problems) > 0 {
		t.Fatalf("un escenario puede repetirse en opciones alternativas: %v", problems)
	}
	b.Preguntas[1].Opciones[0].Conformidad = []Escenario{b.Conformidad[0]}
	if !hasProblem(b.Validate(), "'AUTH-01' está repetido") {
		t.Fatal("un escenario de opción no puede pisar uno de la base")
	}
}

func TestResolver_ConditionalQuestionOnlyWhenItApplies(t *testing.T) {
	b := mustParse(t, authBlueprint)

	_, pendientes, _ := b.Resolver(map[string]string{"identificador": "rut", "recuperacion": "correo", "proveedor_whatsapp": "meta"})
	if len(pendientes) != 0 {
		t.Fatalf("con recuperación por correo no se pregunta el proveedor de WhatsApp: %v", pendientes)
	}
	respuestas, pendientes, _ := b.Resolver(map[string]string{"identificador": "rut", "recuperacion": "whatsapp"})
	if len(pendientes) != 1 || pendientes[0].ID != "proveedor_whatsapp" {
		t.Fatalf("con WhatsApp debe quedar pendiente el proveedor: %v", pendientes)
	}
	if _, ok := respuestas["proveedor_whatsapp"]; ok {
		t.Fatal("una respuesta que no aplica no debe registrarse")
	}
}

func TestResolver_RejectsInvalidAndUnknownAnswers(t *testing.T) {
	_, _, problems := mustParse(t, authBlueprint).Resolver(map[string]string{"identificador": "telefono", "color": "azul"})
	if !hasProblem(problems, "'identificador=telefono' no es válido") || !hasProblem(problems, "no tiene la pregunta 'color'") {
		t.Fatalf("se esperaban ambos rechazos: %v", problems)
	}
}

func TestAdd_LocksVersionCommitAndFingerprint(t *testing.T) {
	root := importedProject(t)
	lock, err := LoadLock(root)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := lock.Entry("auth")
	if !ok || entry.Version != "1.0.0" || entry.Ref != "v1.0.0" || len(entry.Commit) != 40 || len(entry.SHA256) != 64 {
		t.Fatalf("el lockfile debe fijar versión, tag, commit y huella: %+v", lock)
	}
}

func TestAdd_OnlyPublishedVersions(t *testing.T) {
	repo := orgRepo(t, nil)
	for _, ref := range []string{"main", "HEAD~0"} {
		_, problems := Add(AddInput{Root: t.TempDir(), Origen: repo, Ref: ref, Blueprints: []string{"auth"}})
		if !hasProblem(problems, "versión publicada (tag vX.Y.Z)") {
			t.Errorf("--ref %s no es una versión publicada y debe rechazarse: %v", ref, problems)
		}
	}
	loose := t.TempDir()
	writeFile(t, filepath.Join(loose, ManifestName), authBlueprint)
	if _, problems := Add(AddInput{Root: t.TempDir(), Origen: loose}); !hasProblem(problems, "no es un repositorio git") {
		t.Fatalf("una carpeta suelta no es un origen: %v", problems)
	}
	sinTags := t.TempDir()
	writeFile(t, filepath.Join(sinTags, "auth", ManifestName), authBlueprint)
	gitIn(t, sinTags, "init", "-q")
	gitIn(t, sinTags, "add", ".")
	gitIn(t, sinTags, "commit", "-q", "-m", "x")
	if _, problems := Add(AddInput{Root: t.TempDir(), Origen: sinTags, Blueprints: []string{"auth"}}); !hasProblem(problems, "no tiene versiones publicadas") {
		t.Fatalf("un repositorio sin tags no tiene versiones que importar: %v", problems)
	}
}

func TestAdd_WithoutRefTakesTheLatestPublishedVersion(t *testing.T) {
	repo := orgRepo(t, nil)
	publicar(t, repo, "1.1.0", "v1.1.0")
	writeFile(t, filepath.Join(repo, "auth", ManifestName), strings.Replace(authBlueprint, "version: 1.0.0", "version: 9.9.9", 1))
	gitIn(t, repo, "commit", "-q", "-am", "sin publicar")

	entries, problems := Add(AddInput{Root: t.TempDir(), Origen: repo, Blueprints: []string{"auth"}})
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if entries[0].Version != "1.1.0" || entries[0].Ref != "v1.1.0" {
		t.Fatalf("sin --ref se importa la última versión publicada, nunca lo no etiquetado: %+v", entries[0])
	}
}

func TestAdd_RejectsInvalidBlueprintWithoutTouchingProject(t *testing.T) {
	root := t.TempDir()
	_, problems := Add(AddInput{Root: root, Origen: sourceBlueprint(t, "id: auth\nnombre: X\nversion: 1.0.0\n")})
	if !hasProblem(problems, "ficha.objetivo_operativo.problema") {
		t.Fatalf("se esperaba el detalle de lo que falta: %v", problems)
	}
	if _, err := os.Stat(filepath.Join(root, Dir)); !os.IsNotExist(err) {
		t.Fatal("un blueprint inválido no debe copiarse al proyecto")
	}
}

func TestAdd_FromGitRepositorySubfolder(t *testing.T) {
	repo := orgRepo(t, nil)
	root := t.TempDir()
	entries, problems := Add(AddInput{Root: root, Origen: "file://" + repo + "//auth", Ref: "v1.0.0"})
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if len(entries) != 1 || entries[0].Subdir != "auth" || entries[0].Ref != "v1.0.0" || len(entries[0].Commit) != 40 {
		t.Fatalf("el lockfile debe fijar subcarpeta, ref y commit: %+v", entries)
	}
}

// orgRepo publica (v1.0.0) un repositorio de organización con varios
// blueprints y un catálogo.
func orgRepo(t *testing.T, extra map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "auth", ManifestName), authBlueprint)
	auditoria := strings.NewReplacer("id: auth", "id: auditoria", "nombre: Usuarios y acceso", "nombre: Auditoría").Replace(authBlueprint)
	writeFile(t, filepath.Join(repo, "auditoria", ManifestName), auditoria)
	writeFile(t, filepath.Join(repo, "estandares", "pagos-webpay", ManifestName), strings.Replace(authBlueprint, "id: auth", "id: pagos", 1))
	writeFile(t, filepath.Join(repo, CatalogName), "organizacion: ACME\nblueprints:\n  pagos: estandares/pagos-webpay\npaquetes:\n  saas: [auth, auditoria]\n")
	for rel, content := range extra {
		writeFile(t, filepath.Join(repo, rel), content)
	}
	publicarTodo(t, repo)
	return repo
}

func lockIDs(t *testing.T, root string) []string {
	t.Helper()
	lock, err := LoadLock(root)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range lock.Blueprints {
		ids = append(ids, e.ID+":"+e.Subdir)
	}
	return ids
}

func TestAdd_SeveralFromOneRepository(t *testing.T) {
	root := t.TempDir()
	if _, problems := Add(AddInput{Root: root, Origen: orgRepo(t, nil), Blueprints: []string{"auth", "pagos"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if got := strings.Join(lockIDs(t, root), ","); got != "auth:auth,pagos:estandares/pagos-webpay" {
		t.Fatalf("se esperaban auth y pagos (con su carpeta del catálogo): %s", got)
	}
}

func TestAdd_PackageAndAll(t *testing.T) {
	repo := orgRepo(t, nil)
	root := t.TempDir()
	if _, problems := Add(AddInput{Root: root, Origen: repo, Paquete: "saas"}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if got := strings.Join(lockIDs(t, root), ","); got != "auditoria:auditoria,auth:auth" {
		t.Fatalf("el paquete saas trae auth y auditoria: %s", got)
	}
	if _, problems := Add(AddInput{Root: root, Origen: repo, Todos: true}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if got := len(lockIDs(t, root)); got != 3 {
		t.Fatalf("--todos trae los tres: %d", got)
	}
}

func TestAdd_RepositoryWithoutSelectionOffersWhatItHas(t *testing.T) {
	_, problems := Add(AddInput{Root: t.TempDir(), Origen: orgRepo(t, nil)})
	if !hasProblem(problems, "auditoria, auth, pagos") || !hasProblem(problems, "saas (auth, auditoria)") {
		t.Fatalf("debe ofrecer los blueprints y paquetes disponibles: %v", problems)
	}
	_, problems = Add(AddInput{Root: t.TempDir(), Origen: orgRepo(t, nil), Blueprints: []string{"crm"}})
	if !hasProblem(problems, "no tiene el blueprint 'crm'") {
		t.Fatalf("un blueprint inexistente se rechaza: %v", problems)
	}
}

func TestAdd_AllOrNothing(t *testing.T) {
	repo := orgRepo(t, map[string]string{"auditoria/" + ManifestName: "id: auditoria\nnombre: X\nversion: 1.0.0\n"})
	root := t.TempDir()
	_, problems := Add(AddInput{Root: root, Origen: repo, Paquete: "saas"})
	if !hasProblem(problems, "blueprint 'auditoria' inválido") {
		t.Fatalf("se esperaba el detalle del inválido: %v", problems)
	}
	if _, err := os.Stat(filepath.Join(root, Dir)); !os.IsNotExist(err) {
		t.Fatal("si uno es inválido no se importa ninguno")
	}
}

func TestSplitSource(t *testing.T) {
	cases := map[string][2]string{
		"git@github.com:acme/bp.git//auth":         {"git@github.com:acme/bp.git", "auth"},
		"https://github.com/acme/bp.git//auth/v2/": {"https://github.com/acme/bp.git", "auth/v2"},
		"https://github.com/acme/auth.git":         {"https://github.com/acme/auth.git", ""},
	}
	for in, want := range cases {
		repo, sub := splitSource(in)
		if repo != want[0] || sub != want[1] {
			t.Errorf("splitSource(%q) = %q, %q; se esperaba %q", in, repo, sub, want)
		}
	}
}

func TestApply_AsksBeforeWritingAnything(t *testing.T) {
	root := importedProject(t)
	res, problems := Apply(ApplyInput{Root: root, Blueprint: "auth", Respuestas: map[string]string{"identificador": "rut"}})
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if len(res.Pendientes) == 0 || res.Pendientes[0].ID != "recuperacion" {
		t.Fatalf("lo primero que debe pedir es la recuperación: %+v", res.Pendientes)
	}
	if _, err := os.Stat(filepath.Join(root, modules.Dir)); !os.IsNotExist(err) {
		t.Fatal("con decisiones pendientes no se escribe nada")
	}
}

func TestApply_GeneratesDraftRecordSkeletonAndConformance(t *testing.T) {
	root := importedProject(t)
	res, problems := Apply(ApplyInput{Root: root, Blueprint: "auth",
		Respuestas: map[string]string{"identificador": "rut", "recuperacion": "correo"},
		Pruebas:    []string{"go test ./internal/auth/..."}})
	if len(problems) > 0 || len(res.Pendientes) > 0 {
		t.Fatalf("problemas=%v pendientes=%v", problems, res.Pendientes)
	}

	m, ok := modules.Find(root, "auth")
	if !ok || m.Estado != modules.EstadoBorrador || m.Blueprint != "auth@1.0.0" {
		t.Fatalf("la ficha debe quedar en borrador y citar el blueprint: %+v", m)
	}
	if invalid := m.Validate(root); len(invalid) > 0 {
		t.Fatalf("la ficha generada debe ser válida: %v", invalid)
	}
	inv := strings.Join(m.ObjetivoTecnologico.Invariantes, "\n")
	if !strings.Contains(inv, "módulo 11") || strings.Contains(inv, "minúsculas") {
		t.Fatalf("solo deben entrar las invariantes de la opción elegida: %v", m.ObjetivoTecnologico.Invariantes)
	}
	conformidad, _ := os.ReadFile(filepath.Join(root, res.Conformidad))
	if !strings.Contains(string(conformidad), "AUTH-01") || !strings.Contains(string(conformidad), "RUT-01") {
		t.Fatalf("faltan escenarios de conformidad:\n%s", conformidad)
	}
	readme, err := os.ReadFile(filepath.Join(root, "internal", "auth", "README.md"))
	if err != nil || !strings.Contains(string(readme), "módulo 11") {
		t.Fatalf("la ruta nueva debe nacer con el esqueleto conceptual: %v", err)
	}
	if problems := Check(root); len(problems) != 0 {
		t.Fatalf("recién aplicado, todo está vigente: %v", problems)
	}
}

func TestApply_NeverTouchesExistingCode(t *testing.T) {
	root := importedProject(t)
	writeFile(t, filepath.Join(root, "internal", "auth", "README.md"), "mío\n")
	res, problems := Apply(ApplyInput{Root: root, Blueprint: "auth", Respuestas: map[string]string{"identificador": "correo", "recuperacion": "correo"}})
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	data, _ := os.ReadFile(filepath.Join(root, "internal", "auth", "README.md"))
	if string(data) != "mío\n" || len(res.Esqueletos) != 0 {
		t.Fatal("una carpeta existente no se toca")
	}
}

func TestApply_RejectsRoutesInsideQDD(t *testing.T) {
	root := importedProject(t)
	_, problems := Apply(ApplyInput{Root: root, Blueprint: "auth", Rutas: []string{".qdd/auth"},
		Respuestas: map[string]string{"identificador": "correo", "recuperacion": "correo"}})
	if !hasProblem(problems, "dentro de .qdd") {
		t.Fatalf("el código nunca vive en .qdd: %v", problems)
	}
}

func TestApply_ReapplyKeepsProjectAdditionsAndPreviousAnswers(t *testing.T) {
	root := importedProject(t)
	answers := map[string]string{"identificador": "correo", "recuperacion": "correo"}
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "auth", Respuestas: answers}); len(problems) > 0 {
		t.Fatal(problems)
	}
	m, _ := modules.Find(root, "auth")
	m.ObjetivoTecnologico.Invariantes = append(m.ObjetivoTecnologico.Invariantes, "Invariante propia del proyecto")
	m.ADRs = []string{"docs/adr/001.md"}
	if _, problems := modules.Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}

	// Cambia solo el identificador: la recuperación se recuerda de la vez anterior.
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "auth", Respuestas: map[string]string{"identificador": "rut"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	m, _ = modules.Find(root, "auth")
	inv := strings.Join(m.ObjetivoTecnologico.Invariantes, "\n")
	if !strings.Contains(inv, "propia del proyecto") || !strings.Contains(inv, "módulo 11") || strings.Contains(inv, "minúsculas") {
		t.Fatalf("debe conservar lo propio y reemplazar lo generado: %v", m.ObjetivoTecnologico.Invariantes)
	}
	if len(m.ADRs) != 1 {
		t.Fatal("los ADRs del proyecto se conservan")
	}
}

func TestCheck_DetectsOutdatedApplication(t *testing.T) {
	root := t.TempDir()
	src := sourceBlueprint(t, authBlueprint)
	if _, problems := Add(AddInput{Root: root, Origen: src}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "auth", Respuestas: map[string]string{"identificador": "correo", "recuperacion": "correo"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	writeFile(t, filepath.Join(src, ManifestName), strings.Replace(authBlueprint, "version: 1.0.0", "version: 1.1.0", 1))
	gitIn(t, src, "commit", "-q", "-am", "1.1.0")
	gitIn(t, src, "tag", "v1.1.0")
	if _, problems := Add(AddInput{Root: root, Origen: src}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if !hasRule(Check(root), "BP-03-OUTDATED") {
		t.Fatal("un módulo generado con una versión anterior debe marcarse")
	}
}
