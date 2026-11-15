package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Preguntas desde el teléfono»: el mod registra lo que Claude Code le pregunta a la persona y espera la
// respuesta de un teléfono con 'qdd consola pregunta'; sin QDD Móvil activado registra igual, para «▶ Para ti» (MEJ-057).

const askPrueba = `{"sesion":"s1","herramienta":"AskUserQuestion","entrada":{"questions":[{"question":"¿Seguimos?","header":"Seguir","multiSelect":false,"options":[{"label":"Sí","description":""},{"label":"No","description":""}]}]}}`

func emparejado(t *testing.T) {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	if err := remoto.GuardarCredenciales(remoto.Credenciales{Maquina: "m", Topico: "qdd:pc:m:x", Firma: strings.Repeat("ab", 32)}); err != nil {
		t.Fatal(err)
	}
}

// MEJ-057 (sprint 8): toda pregunta de la terminal aparece en «▶ Para ti», con QDD Móvil o sin él. Sin teléfono se
// registra igual y lo dice, para que el panel no ofrezca responderla desde QDD Móvil.
func TestConsolaPregunta_SinQDDMovilIgualRegistraParaElPanel_MEJ057(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	var salida, errores bytes.Buffer
	if codigo := preguntaRegistrar(root, strings.NewReader(askPrueba), &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	var r struct {
		ID       string
		Telefono bool
	}
	json.Unmarshal(salida.Bytes(), &r)
	if r.ID == "" || r.Telefono || len(consola.PreguntasA(root, time.Now())) != 1 {
		t.Fatalf("sin teléfono se registra igual para «Para ti» y dice que no hay teléfono: %s", salida.String())
	}
}

func TestConsolaPregunta_ConQDDMovilDiceQueHayTelefono(t *testing.T) {
	emparejado(t)
	var salida, errores bytes.Buffer
	if codigo := preguntaRegistrar(t.TempDir(), strings.NewReader(askPrueba), &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	var r struct{ Telefono bool }
	json.Unmarshal(salida.Bytes(), &r)
	if !r.Telefono {
		t.Fatalf("con QDD Móvil activado lo dice: %s", salida.String())
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

// MEJ-057: el botón de una opción en «▶ Para ti» responde la pregunta de la terminal con el mismo comando que un ⏳.
func TestConsolaPregunta_ResponderDesdeElPanelUnaPreguntaDeLaTerminal_MEJ057(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	var salida, errores bytes.Buffer
	if codigo := preguntaRegistrar(root, strings.NewReader(askPrueba), &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	var r struct{ ID string }
	json.Unmarshal(salida.Bytes(), &r)
	salida.Reset()
	if codigo := preguntaResponder(root, "pregunta:"+r.ID, "b", &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	if !strings.Contains(salida.String(), "No") {
		t.Fatalf("dice qué quedó respondido: %s", salida.String())
	}
	p, _ := consola.PreguntaDe(root, r.ID, time.Now())
	if e := consola.EntregaDe(p); e.Respuestas["¿Seguimos?"] != "No" || e.Lugar != consola.LugarPanel {
		t.Fatalf("la sesión recibe la opción del panel: %+v", e)
	}
	if codigo := preguntaResponder(root, "pregunta:"+r.ID, "a", &salida, &errores); codigo != 1 {
		t.Fatalf("ya respondida es un error: %d", codigo)
	}
}
