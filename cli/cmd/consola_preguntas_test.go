package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Preguntas desde el teléfono»: el mod registra lo que Claude Code le pregunta a la persona y espera la
// respuesta de un teléfono con 'qdd consola pregunta'; sin QDD Móvil activado no registra nada.

const askPrueba = `{"sesion":"s1","herramienta":"AskUserQuestion","entrada":{"questions":[{"question":"¿Seguimos?","header":"Seguir","multiSelect":false,"options":[{"label":"Sí","description":""},{"label":"No","description":""}]}]}}`

func emparejado(t *testing.T) {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	if err := remoto.GuardarCredenciales(remoto.Credenciales{Maquina: "m", Topico: "qdd:pc:m:x", Firma: strings.Repeat("ab", 32)}); err != nil {
		t.Fatal(err)
	}
}

func TestConsolaPregunta_SinQDDMovilNoRegistraNada(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	var salida, errores bytes.Buffer
	if codigo := preguntaRegistrar(root, strings.NewReader(askPrueba), &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	var r struct{ ID, Motivo string }
	json.Unmarshal(salida.Bytes(), &r)
	if r.ID != "" || r.Motivo == "" || len(consola.PreguntasA(root, time.Now())) != 0 {
		t.Fatalf("sin teléfono emparejado no hay a quién preguntarle: %s", salida.String())
	}
}

func TestConsolaPregunta_RegistrarEsperarYCerrar(t *testing.T) {
	emparejado(t)
	root := t.TempDir()
	var salida, errores bytes.Buffer
	if codigo := preguntaRegistrar(root, strings.NewReader(askPrueba), &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	var r struct{ ID, Accion string }
	json.Unmarshal(salida.Bytes(), &r)
	if r.ID == "" || r.Accion != "pregunta:"+r.ID {
		t.Fatalf("devuelve el id y el de la acción: %s", salida.String())
	}

	salida.Reset()
	inicio := time.Now()
	if codigo := preguntaEsperar(root, r.ID, 300*time.Millisecond, &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	if e := entregaDe(t, salida.Bytes()); e.Estado != consola.PreguntaPendiente || time.Since(inicio) < 250*time.Millisecond {
		t.Fatalf("sin respuesta espera hasta el plazo y dice pendiente: %+v", e)
	}

	go func() {
		time.Sleep(150 * time.Millisecond)
		consola.ResponderPregunta(root, r.ID, `{"q1":{"opciones":["b"]}}`, "Pixel", time.Now())
	}()
	salida.Reset()
	inicio = time.Now()
	if codigo := preguntaEsperar(root, r.ID, 5*time.Second, &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	e := entregaDe(t, salida.Bytes())
	if e.Estado != consola.PreguntaRespondida || e.RespondidaEn != consola.EnTelefono || e.Respuestas["¿Seguimos?"] != "No" || time.Since(inicio) > 2*time.Second {
		t.Fatalf("vuelve en cuanto responde el teléfono, con la respuesta para Claude: %+v", e)
	}

	salida.Reset()
	if codigo := preguntaCerrar(root, r.ID, false, "", &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	if e := entregaDe(t, salida.Bytes()); e.RespondidaEn != consola.EnPC {
		t.Fatalf("cerrar dice que el terminal ganó: %+v", e)
	}
	if codigo := preguntaEsperar(root, "no-existe", time.Millisecond, &salida, &errores); codigo != 1 {
		t.Fatalf("una pregunta que no está es un error: %d", codigo)
	}
}

func TestConsolaPregunta_VencerLaSesion(t *testing.T) {
	emparejado(t)
	root := t.TempDir()
	var salida, errores bytes.Buffer
	preguntaRegistrar(root, strings.NewReader(askPrueba), &salida, &errores)
	salida.Reset()
	if codigo := preguntaVencer(root, "s1", "", &salida, &errores); codigo != 0 || !strings.Contains(salida.String(), "1") {
		t.Fatalf("%d %s %s", codigo, salida.String(), errores.String())
	}
	if p := consola.PreguntasA(root, time.Now()); len(p) != 1 || p[0].Estado != consola.PreguntaVencida || p[0].Motivo == "" {
		t.Fatalf("vence con un motivo por defecto: %+v", p)
	}
	if codigo := preguntaRegistrar(root, strings.NewReader(`{"herramienta":""}`), &salida, &errores); codigo != 1 {
		t.Fatal("una pregunta sin herramienta ni origen qdd se rechaza")
	}
}

func entregaDe(t *testing.T, b []byte) consola.Entrega {
	t.Helper()
	var e consola.Entrega
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	return e
}

// Issue #149: el panel responde un ⏳ con «qdd consola pregunta responder»; la respuesta entra por stdin porque es texto
// de la persona, y lo que devuelve es lo que el panel le envía a la IA.
func TestConsolaPregunta_ResponderUnaEspera(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	ruta := filepath.Join(root, estado.Ruta)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "formato: 1\netapas: [PR]\nfrentes:\n  - {id: barra, nombre: Barra, estado: abierto, celdas: {PR: {estado: \"⏳\", bloqueo: \"¿Mergeamos?\", de_quien: la persona}}}\n"
	if err := os.WriteFile(ruta, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	esperas := consola.EsperasDelPanel(root, time.Now())
	if len(esperas) != 1 {
		t.Fatalf("%+v", esperas)
	}
	var salida, errores bytes.Buffer
	if codigo := preguntaResponder(root, esperas[0].ID, strings.NewReader("sí"), &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	var r struct{ Mensaje string }
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil || !strings.Contains(r.Mensaje, "¿Mergeamos?") || !strings.Contains(r.Mensaje, "sí") {
		t.Fatalf("%v %s", err, salida.String())
	}
	errores.Reset()
	if codigo := preguntaResponder(root, esperas[0].ID, strings.NewReader("no"), &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "ya respondida") {
		t.Fatalf("la segunda no cuenta: %d %s", codigo, errores.String())
	}
}
