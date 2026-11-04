package consola

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Pedido de sfrc (2026-11-03): la IA dejó decisiones como ⏳ «de la persona» en el estado y nunca llegaron como
// preguntas al teléfono, porque solo AskUserQuestion las registra. Cada ⏳ de la persona es una pregunta de respuesta
// libre, sacada del estado cada vez que se lee: no hay una copia que se desincronice, y sale sola cuando la celda
// cambia. La respuesta del teléfono le llega a la IA como un mensaje de la persona.

// conEspera deja el ⏳ de la persona de Barra · PR con este bloqueo.
func conEspera(t *testing.T, root, bloqueo string) {
	t.Helper()
	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "PR", Estado: "⏳", Bloqueo: bloqueo, DeQuien: "la persona"}, enCola); err != nil {
		t.Fatal(err)
	}
}

// conSesionAbierta: una sesión de Claude Code late en el proyecto y puede recibir la respuesta.
func conSesionAbierta(t *testing.T, root string) {
	t.Helper()
	if err := LatirSesion(root, "s1", enCola); err != nil {
		t.Fatal(err)
	}
}

func delEstado(root string) []Pregunta {
	var out []Pregunta
	for _, p := range PreguntasA(root, enCola) {
		if p.Origen == OrigenEstado {
			out = append(out, p)
		}
	}
	return out
}

func esperaDe(t *testing.T, root string, now time.Time) Pregunta {
	t.Helper()
	for _, p := range PreguntasA(root, now) {
		if p.Origen == OrigenEstado {
			return p
		}
	}
	t.Fatalf("no hay pregunta del estado: %+v", PreguntasA(root, now))
	return Pregunta{}
}

func TestEspera_UnEsperaDeLaPersonaEsUnaPreguntaDeRespuestaLibre(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	var delEstado []Pregunta
	for _, p := range PreguntasA(root, enCola) {
		if p.Origen == OrigenEstado {
			delEstado = append(delEstado, p)
		}
	}
	if len(delEstado) != 1 {
		t.Fatalf("solo el ⏳ de la persona (Barra · PR), no el de otro: %+v", delEstado)
	}
	p := delEstado[0]
	if !strings.HasPrefix(p.ID, prefijoEspera) || p.Estado != PreguntaPendiente || !strings.Contains(p.Etiqueta, "Barra · PR") {
		t.Fatalf("pendiente, con un id propio y la tarea y etapa a la vista: %+v", p)
	}
	q := p.Preguntas[0]
	if len(p.Preguntas) != 1 || q.ID != "q1" || q.Texto != "¿mergeamos el #8?" || q.Encabezado != "Barra · PR" || !q.Otra || len(q.Opciones) != 0 {
		t.Fatalf("una pregunta: lo que espera, de respuesta libre: %+v", p.Preguntas)
	}
	if again := esperaDe(t, root, enCola.Add(time.Hour)); again.ID != p.ID {
		t.Fatalf("la misma espera tiene el mismo id en cada lectura: %q != %q", again.ID, p.ID)
	}
	a, err := BuscarAccion(panel(root, enCola).Acciones, prefijoPregunta+p.ID)
	if err != nil || a.Tipo != TipoPregunta || a.Origen != OrigenEstado {
		t.Fatalf("el panel la ofrece como pregunta (la ve qdd remoto): %+v", a)
	}
}

func TestEspera_LaRespuestaDelTelefonoLlegaALaIAComoMensaje(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	if _, err := ResponderPregunta(root, prefijoPregunta+p.ID, `{"q1":{"opciones":["a"]}}`, "Pixel", enCola); !errors.Is(err, ErrRespuestaInvalida) {
		t.Fatalf("no tiene opciones: solo texto: %v", err)
	}
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"ya lo mergeé"}}`, "Pixel", enCola); err != nil {
		t.Fatal(err)
	}
	r := esperaDe(t, root, enCola)
	if r.ID != p.ID || r.Estado != PreguntaRespondida || r.RespondidaEn != EnTelefono || r.Lugar != "Pixel" {
		t.Fatalf("queda respondida desde el teléfono: %+v", r)
	}
	ejecuciones := EjecucionesA(root, enCola)
	if len(ejecuciones) != 1 || ejecuciones[0].Tipo != EjecutarIA || ejecuciones[0].Nombre != "Pixel" {
		t.Fatalf("una instrucción para la IA, del teléfono que respondió: %+v", ejecuciones)
	}
	if texto := ejecuciones[0].Texto; !strings.Contains(texto, "Barra · PR") || !strings.Contains(texto, "merge") || !strings.Contains(texto, "ya lo mergeé") || !strings.Contains(texto, "qdd_estado") {
		t.Fatalf("dice a qué espera responde, la respuesta y cómo reflejarla: %q", texto)
	}
	var cerrada *PreguntaCerrada
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"otra vez"}}`, "Otro", enCola.Add(time.Minute)); !errors.As(err, &cerrada) || LugarDe(cerrada.Pregunta) != "Pixel" {
		t.Fatalf("gana el primero: %v", err)
	}
	if n := len(EjecucionesA(root, enCola.Add(time.Minute))); n != 1 {
		t.Fatalf("la segunda respuesta no llega a la IA: %d", n)
	}
}

func TestEspera_RespondidaNoSeVuelveAOfrecerMientrasLaCeldaNoCambie(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"ya lo mergeé"}}`, "Pixel", enCola); err != nil {
		t.Fatal(err)
	}
	if r := esperaDe(t, root, enCola.Add(2*time.Hour)); r.Estado != PreguntaRespondida {
		t.Fatalf("pasado VidaCerrada sigue respondida, no vuelve a preguntar lo mismo: %+v", r)
	}
	if r := esperaDe(t, root, enCola.Add(VidaEsperaRespondida+time.Minute)); r.Estado != PreguntaPendiente {
		t.Fatalf("si la IA nunca la reflejó, al día vuelve a preguntar: %+v", r)
	}
}

func TestEspera_CuandoLaCeldaCambiaLaPreguntaSale(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "PR", Estado: "⏳", Bloqueo: "¿mergeamos el #9?", DeQuien: "la persona"}, enCola); err != nil {
		t.Fatal(err)
	}
	if nueva := esperaDe(t, root, enCola); nueva.ID == p.ID || nueva.Preguntas[0].Texto != "¿mergeamos el #9?" {
		t.Fatalf("otra espera es otra pregunta: %+v", nueva)
	}
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"tarde"}}`, "Pixel", enCola); !errors.Is(err, ErrPreguntaDesconocida) {
		t.Fatalf("la vieja ya no está: %v", err)
	}
	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "PR", Estado: "✅", Evidencia: "PR #9 mergeado"}, enCola); err != nil {
		t.Fatal(err)
	}
	for _, q := range PreguntasA(root, enCola) {
		if q.Origen == OrigenEstado {
			t.Fatalf("sin ⏳ de la persona no hay pregunta del estado: %+v", q)
		}
	}
}

// MEJ-004 (qdd-movile, 2026-11-03, datos reales): de 11 ⏳ publicados al teléfono, casi todos eran notas de avance o
// esperaban un pedido que la persona ya había armado en el panel; solo uno preguntaba algo. Un ⏳ llega como pregunta
// solo si pregunta (¿…?) y ningún pedido abierto lo cubre, y el teléfono ve solo la pregunta.

func TestEspera_UnaNotaDeAvanceNoEsPregunta_MEJ004(t *testing.T) {
	root := proyectoConTareas(t)
	for _, nota := range []string{"merge", "confirmado por la persona (2026-11-03): sale con el primer despliegue", "Falta crear el secreto en Supabase."} {
		conEspera(t, root, nota)
		if ps := delEstado(root); len(ps) != 0 {
			t.Fatalf("%q no pregunta nada: no va al teléfono: %+v", nota, ps)
		}
	}
}

func TestEspera_ElTelefonoVeSoloLaPregunta_MEJ004(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "Código: listo y probado con 12 casos. ¿Hacemos el gate en el Stop hook? Lo demás espera a eso.")
	p := esperaDe(t, root, enCola)
	if q := p.Preguntas[0].Texto; q != "¿Hacemos el gate en el Stop hook?" {
		t.Fatalf("el texto es la pregunta, no el historial de la tarea: %q", q)
	}
	conEspera(t, root, "¿Proveedor de correo? ¿Resend o SES?")
	if q := esperaDe(t, root, enCola).Preguntas[0].Texto; q != "¿Proveedor de correo? ¿Resend o SES?" {
		t.Fatalf("varias preguntas van todas: %q", q)
	}
}

func TestEspera_UnPedidoAbiertoLaCubre_MEJ004(t *testing.T) {
	root := proyectoConTareas(t)
	pd, err := Pedir(root, "aws configure", "Configurar AWS", root, enCola)
	if err != nil {
		t.Fatal(err)
	}
	conEspera(t, root, "¿Corres ["+pd.Tecla+"] para desplegar?")
	if ps := delEstado(root); len(ps) != 0 {
		t.Fatalf("el pedido [%s] ya la cubre en el panel: el teléfono no la vuelve a preguntar: %+v", pd.Tecla, ps)
	}
	conEspera(t, root, "¿Corres [z] para desplegar?")
	if ps := delEstado(root); len(ps) != 1 {
		t.Fatalf("[z] no es un pedido abierto: sí es pregunta: %+v", ps)
	}
}

// FND-124 (qdd-movile, 2026-11-03): sin una sesión abierta, la respuesta a un ⏳ quedaba como instrucción que vence a los
// 30 min, mientras la pregunta seguía «respondida» 24 h: la respuesta se perdía sin que nadie lo supiera.

func TestEspera_SinSesionLaRespuestaNoSePierde_FND124(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	p := esperaDe(t, root, enCola)
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"sí"}}`, "Pixel", enCola); !errors.Is(err, ErrPreguntaDesconocida) {
		t.Fatalf("sin sesión no hay quien la reciba: se rechaza: %v", err)
	}
	if e := EjecucionesA(root, enCola); len(e) != 0 {
		t.Fatalf("no queda una instrucción que vaya a vencer: %+v", e)
	}
	if r := esperaDe(t, root, enCola); r.Estado != PreguntaPendiente {
		t.Fatalf("sigue pendiente, no «respondida»: %+v", r)
	}
	conSesionAbierta(t, root)
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"sí"}}`, "Pixel", enCola); err != nil {
		t.Fatalf("con una sesión abierta, se responde: %v", err)
	}
}
