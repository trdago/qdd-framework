package consola

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Propuesta de qdd-movile aprobada por la persona (2026-11-07): cada ⏳ suyo trae «Ya lo hice», «Todavía no» y «Ya no
// va», en el teléfono y en el panel del PC, y funciona sin una sesión abierta: la respuesta queda en el estado.

func celdaPR(t *testing.T, root string) estado.Celda {
	t.Helper()
	e, err := estado.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range e.Frentes {
		if f.ID == "barra" {
			return f.Celdas["PR"]
		}
	}
	t.Fatal("no está la tarea barra")
	return estado.Celda{}
}

func pendientesDelEstado(root string, now time.Time) []Pregunta {
	var out []Pregunta
	for _, p := range PreguntasA(root, now) {
		if p.Origen == OrigenEstado && p.Estado == PreguntaPendiente {
			out = append(out, p)
		}
	}
	return out
}

func TestEspera_TraeLosTresBotonesYSeRespondeSinSesion(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	p := esperaDe(t, root, enCola)
	q := p.Preguntas[0]
	if !q.Otra || q.Multiple || len(q.Opciones) != 3 {
		t.Fatalf("tres opciones y «Otra»: %+v", q)
	}
	esperadas := [][2]string{{EsperaHecho, "Ya lo hice"}, {EsperaTodaviaNo, "Todavía no"}, {EsperaYaNoVa, "Ya no va"}}
	for i, o := range q.Opciones {
		if o.Valor != esperadas[i][0] || o.Etiqueta != esperadas[i][1] {
			t.Fatalf("opción %d: %+v", i, o)
		}
	}
	if !p.SinSesion {
		t.Fatalf("la pregunta dice que no necesita una sesión: %+v", p)
	}
	a, err := BuscarAccion(panel(root, enCola).Acciones, prefijoPregunta+p.ID)
	if err != nil || !a.SinSesion {
		t.Fatalf("la acción que ve el teléfono también: %+v %v", a, err)
	}
}

func TestEspera_YaLoHiceLaPasaALaIAParaVerificar(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	p := esperaDe(t, root, enCola)
	r, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["hecho"]}}`, "Pixel", enCola)
	if err != nil {
		t.Fatalf("sin sesión se responde igual: %v", err)
	}
	c := celdaPR(t, root)
	if c.Estado != estado.Espera || estado.DeLaPersona(c.DeQuien) || !strings.Contains(c.Bloqueo, "ya lo hizo") || !strings.Contains(c.Bloqueo, "Pixel") || !strings.Contains(c.Bloqueo, "¿mergeamos el #8?") {
		t.Fatalf("sigue ⏳, ahora de la IA, con quién dijo qué y lo que esperaba: %+v", c)
	}
	if r.Aplicado == "" || r.Instruccion != "" {
		t.Fatalf("dice qué quedó, sin una instrucción que seguir: %+v", r)
	}
	if ps := pendientesDelEstado(root, enCola); len(ps) != 0 {
		t.Fatalf("sale del teléfono: %+v", ps)
	}
	if e := EjecucionesA(root, enCola); len(e) != 0 {
		t.Fatalf("sin sesión no queda una instrucción que vaya a vencer: %+v", e)
	}
	var cerrada *PreguntaCerrada
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["ya_no_va"]}}`, "Otro", enCola.Add(time.Minute)); !errors.As(err, &cerrada) || LugarDe(cerrada.Pregunta) != "Pixel" {
		t.Fatalf("otro teléfono ve que ya se respondió y dónde: %v", err)
	}
}

func TestEspera_YaNoVaDejaLaEtapaEnNoAplica(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	p := esperaDe(t, root, enCola)
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["ya_no_va"]}}`, "Pixel", enCola); err != nil {
		t.Fatal(err)
	}
	c := celdaPR(t, root)
	if c.Estado != estado.NoAplica || c.Bloqueo != "" || !strings.Contains(c.Evidencia, "ya no va") || !strings.Contains(c.Evidencia, "¿mergeamos el #8?") {
		t.Fatalf("la etapa queda en —, con por qué: %+v", c)
	}
}

func TestEspera_TodaviaNoLaVuelveAPreguntarAlDia(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	antes := celdaPR(t, root)
	p := esperaDe(t, root, enCola)
	r, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["todavia_no"]}}`, "Pixel", enCola)
	if err != nil {
		t.Fatal(err)
	}
	if celdaPR(t, root) != antes || r.Aplicado == "" {
		t.Fatalf("el estado no cambia y la respuesta dice qué pasa: %+v", r)
	}
	if ps := pendientesDelEstado(root, enCola.Add(time.Hour)); len(ps) != 0 {
		t.Fatalf("no se pregunta mientras tanto: %+v", ps)
	}
	if got := EsperasDelPanel(root, enCola.Add(time.Hour)); len(got) != 0 {
		t.Fatalf("tampoco en «Para ti»: %+v", got)
	}
	if ps := pendientesDelEstado(root, enCola.Add(VidaEsperaRespondida+time.Minute)); len(ps) != 1 {
		t.Fatalf("al día vuelve: %+v", ps)
	}
}

func TestEspera_OtraQuedaEnElEstadoParaLaIA(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	p := esperaDe(t, root, enCola)
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"mejor el lunes"}}`, "Pixel", enCola); err != nil {
		t.Fatal(err)
	}
	c := celdaPR(t, root)
	if c.Estado != estado.Espera || estado.DeLaPersona(c.DeQuien) || !strings.Contains(c.Bloqueo, "mejor el lunes") || !strings.Contains(c.Bloqueo, "¿mergeamos el #8?") {
		t.Fatalf("pasa a la IA con lo que escribió la persona: %+v", c)
	}
}

func TestEspera_ConSesionLaIATambienSeEntera(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["hecho"]}}`, "Pixel", enCola); err != nil {
		t.Fatal(err)
	}
	e := EjecucionesA(root, enCola)
	if len(e) != 1 || e[0].Tipo != EjecutarIA || e[0].Nombre != "Pixel" || !strings.Contains(e[0].Texto, "Ya lo hice") || !strings.Contains(e[0].Texto, "verif") {
		t.Fatalf("le llega a la IA para que lo verifique: %+v", e)
	}
}

func TestEspera_TodaviaNoNoMolestaALaIA(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["todavia_no"]}}`, "Pixel", enCola); err != nil {
		t.Fatal(err)
	}
	if e := EjecucionesA(root, enCola); len(e) != 0 {
		t.Fatalf("no hay nada que hacer: %+v", e)
	}
}

func TestEspera_ElPanelDelPCRespondeConLosMismosBotones(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	if _, _, err := ResponderEsperaEnPC(root, prefijoPregunta+p.ID, "quizas", enCola); !errors.Is(err, ErrRespuestaInvalida) {
		t.Fatalf("solo las tres opciones: %v", err)
	}
	r, mensaje, err := ResponderEsperaEnPC(root, prefijoPregunta+p.ID, EsperaHecho, enCola)
	if err != nil {
		t.Fatal(err)
	}
	if r.RespondidaEn != EnPC || LugarDe(r) != "PC" || !strings.Contains(celdaPR(t, root).Bloqueo, "desde el PC") {
		t.Fatalf("respondida en el PC: %+v", r)
	}
	if !strings.Contains(mensaje, "Ya lo hice") {
		t.Fatalf("el panel le envía a la IA el mensaje: %q", mensaje)
	}
	if e := EjecucionesA(root, enCola); len(e) != 0 {
		t.Fatalf("desde el PC no pasa por la cola del teléfono: %+v", e)
	}
	if _, _, err := ResponderEsperaEnPC(root, "pregunta:otra-cosa", EsperaHecho, enCola); !errors.Is(err, ErrPreguntaDesconocida) {
		t.Fatalf("solo responde un ⏳: %v", err)
	}
}
