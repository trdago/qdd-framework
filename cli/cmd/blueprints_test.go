package cmd

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/blueprints"
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

func importTestBlueprint(t *testing.T, root string) {
	t.Helper()
	src := filepath.Join(t.TempDir(), "auth")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, blueprints.ManifestName), []byte(cmdTestBlueprint), 0644); err != nil {
		t.Fatal(err)
	}
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
	repo := filepath.Join(t.TempDir(), "org")
	if err := os.MkdirAll(filepath.Join(repo, "auth"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "auth", blueprints.ManifestName), []byte(cmdTestBlueprint), 0644); err != nil {
		t.Fatal(err)
	}

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
