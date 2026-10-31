package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/modules"
)

const cmdTestBlueprint = `id: auth
nombre: Auth
version: 1.0.0
preguntas:
  - id: identificador
    pregunta: ¿Con qué inicia sesión?
    por_defecto: correo
    opciones: [{valor: correo}, {valor: rut}]
  - id: recuperacion
    pregunta: ¿Canal de recuperación?
    opciones: [{valor: correo}, {valor: whatsapp}]
ficha:
  nombre: Auth
  rutas: [internal/auth]
  objetivo_operativo: {problema: p, criterio_de_exito: c}
  objetivo_tecnologico: {arquitectura: a}
`

// TestMain aísla la carpeta central de QDD: ninguna prueba toca
// ~/.local/share/qdd del desarrollador.
func TestMain(m *testing.M) {
	// Un pedido lanzado desde una prueba (FND-089) relanza este binario como «qdd consola correr».
	if root, id, ok := consola.EsCorrer(os.Args[1:]); ok {
		os.Exit(consola.CorrerLanzado(root, id, os.Stdout))
	}
	home, err := os.MkdirTemp("", "qdd-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "no se pudo crear la carpeta central de prueba:", err)
		os.Exit(1)
	}
	os.Setenv(blueprints.HomeEnv, home)
	for k, v := range map[string]string{"GIT_AUTHOR_NAME": "qdd", "GIT_AUTHOR_EMAIL": "qdd@test", "GIT_COMMITTER_NAME": "qdd", "GIT_COMMITTER_EMAIL": "qdd@test"} {
		os.Setenv(k, v)
	}
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

// publishedRepo publica (git, tag v1.0.0) un repositorio con files.
func publishedRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"commit", "-q", "-m", "v1"}, {"tag", "v1.0.0"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return repo
}

func importTestBlueprint(t *testing.T, root string) {
	t.Helper()
	src := publishedRepo(t, map[string]string{blueprints.ManifestName: cmdTestBlueprint})
	if _, problems := blueprints.Add(blueprints.AddInput{Root: root, Origen: src}); len(problems) > 0 {
		t.Fatal(problems)
	}
}

func TestApplyAsking_InteractiveUsesNumbersDefaultsAndRetries(t *testing.T) {
	root := chdirToTempRepo(t)
	importTestBlueprint(t, root)

	// Enter acepta la sugerida (correo); "sms" no es opción y se repregunta; "2" es whatsapp.
	stdin := strings.NewReader("\nsms\n2\n")
	res, problems := applyAsking(blueprints.ApplyInput{Root: root, Blueprint: "auth"}, bufio.NewReader(stdin), io.Discard, true)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	app, err := blueprints.LoadApplication(root, "auth")
	if err != nil || app.Respuestas["identificador"] != "correo" || app.Respuestas["recuperacion"] != "whatsapp" {
		t.Fatalf("respuestas mal registradas: %+v (%v)", app.Respuestas, err)
	}
	if _, ok := modules.Find(root, "auth"); !ok || res.Ficha == "" {
		t.Fatal("debe generarse la ficha")
	}
}

func TestApplyAsking_NonInteractiveListsMissingDecisions(t *testing.T) {
	root := chdirToTempRepo(t)
	importTestBlueprint(t, root)
	_, problems := applyAsking(blueprints.ApplyInput{Root: root, Blueprint: "auth"}, bufio.NewReader(strings.NewReader("")), io.Discard, false)
	if len(problems) != 2 || !strings.Contains(problems[0], "--respuesta identificador=<correo|rut>") {
		t.Fatalf("sin terminal no se decide por defecto; se listan las faltantes: %v", problems)
	}
}

func TestBlueprintApplyTool_ReturnsQuestionsWithoutWriting(t *testing.T) {
	root := chdirToTempRepo(t)
	importTestBlueprint(t, root)
	res, err := handleBlueprintApplyTool(context.Background(), toolRequest(map[string]interface{}{
		"blueprint": "auth", "respuestas": "identificador=rut",
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(t, res)
	if res.IsError || !strings.Contains(text, "no las respondas tú") || !strings.Contains(text, "recuperacion") {
		t.Fatalf("debe devolver la pregunta pendiente para el humano: %q", text)
	}
	if _, ok := modules.Find(root, "auth"); ok {
		t.Fatal("con decisiones pendientes no se escribe la ficha")
	}
}

func TestBlueprintApplyTool_GeneratesDraft(t *testing.T) {
	root := chdirToTempRepo(t)
	importTestBlueprint(t, root)
	res, err := handleBlueprintApplyTool(context.Background(), toolRequest(map[string]interface{}{
		"blueprint": "auth", "respuestas": "identificador=rut, recuperacion=correo", "rutas": "internal/crm",
	}))
	if err != nil {
		t.Fatal(err)
	}
	m, ok := modules.Find(root, "auth")
	if res.IsError || !ok || m.Estado != modules.EstadoBorrador || m.Rutas[0] != "internal/crm" {
		t.Fatalf("debe generar la ficha en borrador con las rutas pedidas: %q %+v", resultText(t, res), m)
	}
}

func TestParseRespuestas_RejectsMalformed(t *testing.T) {
	if _, err := parseRespuestas([]string{"identificador"}); err == nil {
		t.Fatal("una respuesta sin '=' debe rechazarse")
	}
}

func TestBlueprintAddTool_AsksForRepositoryThenUsesRegistry(t *testing.T) {
	root := chdirToTempRepo(t)
	repo := publishedRepo(t, map[string]string{"auth/" + blueprints.ManifestName: cmdTestBlueprint})

	res, _ := handleBlueprintAddTool(context.Background(), toolRequest(map[string]interface{}{"blueprints": "auth"}))
	if !res.IsError || !strings.Contains(resultText(t, res), "todavía no hay repositorio") {
		t.Fatalf("sin registro debe pedir el repositorio: %q", resultText(t, res))
	}
	res, _ = handleBlueprintAddTool(context.Background(), toolRequest(map[string]interface{}{"blueprints": "auth", "origen": repo}))
	if res.IsError || !strings.Contains(resultText(t, res), "registrado en .qdd/config.yaml") {
		t.Fatalf("con origen debe importar y registrar: %q", resultText(t, res))
	}
	res, _ = handleBlueprintAddTool(context.Background(), toolRequest(map[string]interface{}{"todos": true}))
	if res.IsError || !strings.Contains(resultText(t, res), "Importado auth@1.0.0") {
		t.Fatalf("después basta con el registro: %q", resultText(t, res))
	}
	if reg, _ := blueprints.LoadRegistro(root); reg.Origen != repo {
		t.Fatalf("registro = %+v", reg)
	}
}

func TestSplitAddArgs(t *testing.T) {
	if origen, ids := splitAddArgs([]string{"auth", "auditoria"}); origen != "" || len(ids) != 2 {
		t.Fatal("solo ids: usa el registro")
	}
	if origen, ids := splitAddArgs([]string{"git@github.com:acme/bp.git", "auth"}); origen == "" || len(ids) != 1 {
		t.Fatal("el primer argumento con ':' o '/' es el origen")
	}
}

func TestRespuestasDe_PrefixSeparatesBlueprints(t *testing.T) {
	r := map[string]string{"auth.identificador": "rut", "auditoria.retencion": "5a", "canal": "correo"}
	auth := respuestasDe("auth", r, true)
	if len(auth) != 1 || auth["identificador"] != "rut" {
		t.Fatalf("con varios blueprints solo cuentan las respuestas con prefijo: %v", auth)
	}
	solo := respuestasDe("auth", r, false)
	if solo["canal"] != "correo" || solo["identificador"] != "rut" {
		t.Fatalf("con uno solo también vale la forma corta: %v", solo)
	}
}

// legacyCmdProject arma un proyecto del esquema anterior (v1.10–v1.11): copia
// vendorizada en .qdd/blueprints/auth y lockfile sin formato.
func legacyCmdProject(t *testing.T, root string) {
	t.Helper()
	legacy := filepath.Join(root, blueprints.Dir, "auth")
	if err := os.MkdirAll(legacy, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, blueprints.ManifestName), []byte(cmdTestBlueprint), 0644); err != nil {
		t.Fatal(err)
	}
	sum, err := blueprints.HashDir(legacy)
	if err != nil {
		t.Fatal(err)
	}
	lock := "blueprints:\n  - id: auth\n    version: 1.0.0\n    origen: /ya/no/existe\n    sha256: " + sum + "\n    agregado_el: \"2026-10-23\"\n"
	if err := os.WriteFile(filepath.Join(root, blueprints.LockFile), []byte(lock), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectContextTool_MigratesLegacyBlueprintsOnceAndTellsTheAI(t *testing.T) {
	root := chdirToTempRepo(t)
	legacyCmdProject(t, root)

	res, err := handleProjectContextTool(context.Background(), toolRequest(map[string]interface{}{}))
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(t, res)
	if !strings.Contains(text, "Migración de blueprints") || !strings.Contains(text, "commit") {
		t.Fatalf("la IA debe enterarse de la migración y de qué commitear: %q", text)
	}
	if _, err := os.Stat(filepath.Join(root, blueprints.Dir)); !os.IsNotExist(err) {
		t.Fatal("la copia del proyecto se migró a la carpeta central")
	}
	if _, err := blueprints.Find(root, "auth"); err != nil {
		t.Fatalf("el blueprint migrado se lee desde la carpeta central: %v", err)
	}
	again, _ := handleProjectContextTool(context.Background(), toolRequest(map[string]interface{}{}))
	if strings.Contains(resultText(t, again), "Migración de blueprints") {
		t.Fatal("la migración ocurre una sola vez")
	}
}

func TestMigrationNotice_LeftoversAreTheAIsTask(t *testing.T) {
	root := t.TempDir()
	legacyCmdProject(t, root)
	manifest := filepath.Join(root, blueprints.Dir, "auth", blueprints.ManifestName)
	if err := os.WriteFile(manifest, []byte(strings.Replace(cmdTestBlueprint, "nombre: Auth\nversion", "nombre: Auth editado\nversion", 1)), 0644); err != nil {
		t.Fatal(err)
	}
	text := migrationNotice(root)
	if !strings.Contains(text, "BP-05-LEGACY") || !strings.Contains(text, "qdd blueprint contribute auth") {
		t.Fatalf("una copia editada queda como tarea guiada para la IA: %q", text)
	}
}

func TestSellarVersion_OlderQDDStopsInsteadOfMissingTheBlueprints(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, ".qdd", "state.json")
	if err := os.MkdirAll(filepath.Dir(state), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, []byte(`{"status":"initialized","version":"v1.11.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := sellarVersion(root, "v2.0.0"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(state)
	if !strings.Contains(string(data), `"version": "v2.0.0"`) || !strings.Contains(string(data), `"status": "initialized"`) {
		t.Fatalf("el proyecto migrado exige la versión que lo migró y conserva el resto: %s", data)
	}
	if err := checkVersionAge("v1.11.0", "v2.0.0"); err == nil {
		t.Fatal("un QDD anterior debe detenerse ante un proyecto migrado")
	}
	if err := sellarVersion(root, "v1.12.0"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(state); !strings.Contains(string(data), "v2.0.0") {
		t.Fatalf("nunca se baja la versión exigida: %s", data)
	}
	if err := sellarVersion(root, "dev"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(state); !strings.Contains(string(data), "v2.0.0") {
		t.Fatalf("un binario de desarrollo no sella versión: %s", data)
	}
}

func blueprintsGroup(t *testing.T, list *Checklist) *CheckGroup {
	t.Helper()
	for _, g := range list.Groups {
		if strings.Contains(g.Name, "Blueprints") {
			return g
		}
	}
	t.Fatalf("doctor debe revisar los blueprints de la organización: %+v", list.Groups)
	return nil
}

func TestDoctor_DiagnosesAndRepairsBlueprints(t *testing.T) {
	root := chdirToTempRepo(t)
	legacyCmdProject(t, root)

	g := blueprintsGroup(t, runDeterministicChecks(root))
	if len(g.Items) != 5 || g.Items[0].Success {
		t.Fatalf("una copia del esquema anterior es un problema que doctor ve: %+v", g.Items)
	}
	runSelectiveFixes(root, runDeterministicChecks(root))
	for _, item := range blueprintsGroup(t, runDeterministicChecks(root)).Items {
		if !item.Success {
			t.Fatalf("doctor --fix deja los blueprints operativos: %s (%s)", item.Name, item.Error)
		}
	}
	if _, err := os.Stat(filepath.Join(root, blueprints.Dir)); !os.IsNotExist(err) {
		t.Fatal("doctor --fix migró las copias a la carpeta central")
	}
}

// FND-069: la ayuda de 'release --origen' decía «fuente de un blueprint nuevo»;
// en release es la fuente que se etiqueta.
func TestReleaseOrigenFlag_NamesTheSourceToTag(t *testing.T) {
	release := blueprintReleaseCmd.Flags().Lookup("origen").Usage
	if strings.Contains(release, "nuevo") || !strings.Contains(release, "etiqueta") {
		t.Fatalf("release --origen es la fuente que se etiqueta: %q", release)
	}
	for _, c := range []*cobra.Command{blueprintContributeCmd, blueprintProposeCmd} {
		if u := c.Flags().Lookup("origen").Usage; !strings.Contains(u, "nuevo") {
			t.Fatalf("%s --origen sigue siendo la fuente de un blueprint nuevo: %q", c.Name(), u)
		}
	}
}

// FND-073: sin terminal, 'qdd blueprint apply' listaba una pregunta condicional
// como si fuera obligatoria, sin decir de qué respuesta depende.
func TestPendingProblems_ConditionalQuestionSaysWhenItApplies(t *testing.T) {
	canales := blueprints.Pregunta{ID: "canales", Pregunta: "¿Por qué canales?", Opciones: []blueprints.Opcion{{Valor: "correo"}, {Valor: "whatsapp"}}}
	proveedor := blueprints.Pregunta{ID: "proveedor", Pregunta: "¿Qué proveedor?", Si: map[string]blueprints.Valores{"canales": {"whatsapp"}}, Opciones: []blueprints.Opcion{{Valor: "meta"}, {Valor: "otro"}}}
	got := pendingProblems([]blueprints.Pregunta{canales, proveedor})
	if strings.Contains(got[0], "solo si") {
		t.Fatalf("una pregunta sin condición no debe decir 'solo si': %s", got[0])
	}
	if !strings.Contains(got[1], "solo si canales = whatsapp") {
		t.Fatalf("la pregunta condicional debe decir de qué depende: %s", got[1])
	}
}

func TestBlueprintListTool_UnaLineaPorBlueprintYDetalleAPedido(t *testing.T) {
	root := chdirToTempRepo(t)
	importTestBlueprint(t, root)
	res, _ := handleBlueprintListTool(context.Background(), toolRequest(map[string]interface{}{}))
	lista := resultText(t, res)
	if !strings.Contains(lista, "auth@1.0.0 · Auth · ❓2") || !strings.Contains(lista, "seccion") || strings.Contains(lista, "¿Canal de recuperación?") || len(lista) > 600 {
		t.Fatalf("sin id, una línea por blueprint y cómo pedir el detalle (no el blueprint entero): %q", lista)
	}
	res, _ = handleBlueprintListTool(context.Background(), toolRequest(map[string]interface{}{"blueprint": "auth"}))
	indice := resultText(t, res)
	var i blueprints.Indice
	if !strings.Contains(lista, "qdd_blueprint_apply") || !strings.Contains(indice, "receta completa") {
		t.Fatalf("el índice sirve para elegir; para construir, la receta completa con qdd_blueprint_apply: %q", indice)
	}
	if res.IsError || json.Unmarshal([]byte(indice[:strings.LastIndex(indice, "}")+1]), &i) != nil || i.Ref != "auth@1.0.0" || len(i.Preguntas) != 2 || i.Secciones["preguntas"] == 0 {
		t.Fatalf("con id, el índice en JSON con la línea de cada sección: %q", indice)
	}
	res, _ = handleBlueprintListTool(context.Background(), toolRequest(map[string]interface{}{"blueprint": "auth", "seccion": "preguntas"}))
	if res.IsError || !strings.Contains(resultText(t, res), "¿Canal de recuperación?") {
		t.Fatalf("con seccion, solo ese detalle: %q", resultText(t, res))
	}
	res, _ = handleBlueprintListTool(context.Background(), toolRequest(map[string]interface{}{"blueprint": "auth", "seccion": "otra"}))
	if !res.IsError {
		t.Fatal("una sección desconocida es un error que dice cuáles hay")
	}
}

func TestIndicesDelProyecto_DiceDondeSeAplico(t *testing.T) {
	root := chdirToTempRepo(t)
	importTestBlueprint(t, root)
	if _, problems := blueprints.Apply(blueprints.ApplyInput{Root: root, Blueprint: "auth", Modulo: "acceso", Respuestas: map[string]string{"identificador": "rut", "recuperacion": "correo"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	ix, errs := indicesDelProyecto(root)
	if len(errs) > 0 || len(ix) != 1 || len(ix[0].Modulos) != 1 || ix[0].Modulos[0] != "acceso" {
		t.Fatalf("cada índice nombra los módulos generados con él: %+v %v", ix, errs)
	}
}

func TestBlueprintListTool_CalidadDeLaReceta(t *testing.T) {
	root := chdirToTempRepo(t)
	importTestBlueprint(t, root)
	for _, args := range []map[string]interface{}{{"seccion": "calidad"}, {"blueprint": "auth", "seccion": "calidad"}} {
		res, _ := handleBlueprintListTool(context.Background(), toolRequest(args))
		texto := resultText(t, res)
		if res.IsError || !strings.Contains(texto, "auth@1.0.0 · calidad ") || !strings.Contains(texto, "tokens") {
			t.Fatalf("con seccion calidad, la nota de la receta y lo que cuesta (con o sin blueprint) %v: %q", args, texto)
		}
		if strings.Contains(texto, "⚠") && !strings.Contains(texto, "qdd_blueprint_contribute") {
			t.Fatalf("con defectos, dice cómo arreglarlos: %q", texto)
		}
	}
}

func TestInformeDeCalidad_SinHallazgosNoPideNada(t *testing.T) {
	informe := informeDeCalidad([]blueprints.Calidad{{Ref: "x@1.0.0", Nota: 100}}, "importados", "a", "b")
	if !strings.Contains(informe, "✅") || strings.Contains(informe, "Arreglo") {
		t.Fatalf("una receta completa no tiene arreglos: %q", informe)
	}
}

func TestBlueprintContributeTool_CertificarSoloPrepararOVerificar(t *testing.T) {
	chdirToTempRepo(t)
	res, _ := handleBlueprintContributeTool(context.Background(), toolRequest(map[string]interface{}{"blueprint": "auth", "certificar": "ya"}))
	if !res.IsError || !strings.Contains(resultText(t, res), "preparar o verificar") {
		t.Fatalf("certificar es preparar o verificar: %q", resultText(t, res))
	}
}

func TestBlueprintListTool_LeyendaEnLaSalidaNoEnLaDescripcion(t *testing.T) {
	root := chdirToTempRepo(t)
	importTestBlueprint(t, root)
	res, _ := handleBlueprintListTool(context.Background(), toolRequest(map[string]interface{}{}))
	if !strings.Contains(resultText(t, res), "▶pasos ⚙parámetros") {
		t.Fatalf("la leyenda de los símbolos va con la lista (cuesta solo al pedirla): %q", resultText(t, res))
	}
}

func TestTextoDeEntrevista_SoloCuandoHayAlgoQuePreguntar(t *testing.T) {
	if textoDeEntrevista(blueprints.Calidad{}) != "" {
		t.Fatal("sin nada que delimitar no hay entrevista")
	}
	c := blueprints.Calidad{Hallazgos: []blueprints.HallazgoCalidad{{Codigo: blueprints.CalSinAlcance, Donde: "alcance"}}}
	if e := textoDeEntrevista(c); !strings.Contains(e, "no lo respondas tú") || !strings.Contains(e, "incluye") {
		t.Fatalf("la entrevista es de la persona: %q", e)
	}
}
