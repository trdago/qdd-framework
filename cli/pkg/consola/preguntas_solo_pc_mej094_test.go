package consola

import (
	"encoding/json"
	"errors"
	"testing"
)

// Rule: Prohibido usar 'else'
// MEJ-094 (REP-510): sin el hook PermissionRequest, nadie puede llevarle a Claude Code una respuesta que no sea la de
// la terminal. El mod la registra igual con responder_en "pc" (acordado con qdd-movile): sale en el panel y en el
// teléfono, suena el push, pero solo se responde en el PC; ni el teléfono, ni el panel, ni el modo goal la responden.

func registrarSoloPC(t *testing.T, root string) Pregunta {
	t.Helper()
	p, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: json.RawMessage(entradaAskUser), ResponderEn: ResponderEnPC}, enPregunta)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPregunta_SoloEnElPCLlegaAlPanelConResponderEn_MEJ094(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarSoloPC(t, root)
	if p.ResponderEn != ResponderEnPC {
		t.Fatalf("queda marcada: %+v", p)
	}
	for _, a := range accionesDePreguntas(root, enPregunta) {
		if a.ID == prefijoPregunta+p.ID && a.ResponderEn == ResponderEnPC && a.Estado == PreguntaPendiente {
			return
		}
	}
	t.Fatal("la acción del panel dice responder_en pc")
}

func TestPregunta_SoloEnElPCNoSeRespondeDesdeOtroLado_MEJ094(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarSoloPC(t, root)
	_, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["a"]},"q2":{"opciones":["a"]}}`, "Pixel", enPregunta)
	if !errors.Is(err, ErrSoloEnPC) {
		t.Fatalf("el teléfono no la responde: %v", err)
	}
	if q, _ := PreguntaDe(root, p.ID, enPregunta); q.Estado != PreguntaPendiente {
		t.Fatalf("sigue pendiente: %+v", q)
	}
	if _, err := CerrarPregunta(root, p.ID, EnPC, "", enPregunta); err != nil {
		t.Fatalf("la terminal sí la cierra: %v", err)
	}
}

func TestPregunta_SinElCampoOConOtroValorTodoComoHoy_MEJ094(t *testing.T) {
	root := proyectoConCola(t)
	p, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: json.RawMessage(entradaAskUser), ResponderEn: "marte"}, enPregunta)
	if err != nil {
		t.Fatal(err)
	}
	if p.ResponderEn != "" {
		t.Fatalf("un valor desconocido no se guarda: %+v", p)
	}
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["a"]},"q2":{"opciones":["a"]}}`, "Pixel", enPregunta); err != nil {
		t.Fatalf("sin la marca, el teléfono responde como hoy: %v", err)
	}
}

// contrato: goal#49d86e1f
func TestGoal_UnaPreguntaSoloEnElPCEsperaALaPersona_MEJ094(t *testing.T) {
	root := conGoal(t)
	p, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, ResponderEn: ResponderEnPC,
		Entrada: []byte(`{"questions":[{"question":"¿Cuál va?","header":"H","multiSelect":false,"options":[{"label":"Sí (Recommended)","description":"d"},{"label":"No","description":"d"}]}]}`)}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	r := cicloGoal(t, root, silencio(0), enCola)
	if len(r.Elegidas) != 0 {
		t.Fatalf("el modo goal no puede llevarle la respuesta a Claude Code: %+v", r.Elegidas)
	}
	if q, _ := PreguntaDe(root, p.ID, enCola); q.Estado != PreguntaPendiente {
		t.Fatalf("sigue pendiente: %+v", q)
	}
}
