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
// solo si pregunta (¿…?) y ningún pedido abierto lo cubre, y el teléfono no ve lo que va antes de la pregunta.

// FND-155 (qdd-movile, compilación 69): el teléfono mostraba hasta el último «?», y lo que la persona tenía que hacer iba
// después.
func TestEspera_ElTelefonoVeLoQueSigueALaPregunta_FND155(t *testing.T) {
	root := proyectoConTareas(t)
	bloqueo := "¿Ves el modo goal arriba en el proyecto? Si lo ves, toca Apagar; si no, dime qué ves en esa pantalla."
	conEspera(t, root, bloqueo)
	if q := esperaDe(t, root, enCola).Preguntas[0].Texto; q != bloqueo {
		t.Fatalf("la instrucción va con la pregunta: %q", q)
	}
}

func TestEspera_UnaNotaDeAvanceNoEsPregunta_MEJ004(t *testing.T) {
	root := proyectoConTareas(t)
	for _, nota := range []string{"merge", "confirmado por la persona (2026-11-03): sale con el primer despliegue", "Falta crear el secreto en Supabase."} {
		conEspera(t, root, nota)
		if ps := delEstado(root); len(ps) != 0 {
			t.Fatalf("%q no pregunta nada: no va al teléfono: %+v", nota, ps)
		}
	}
}

func TestEspera_ElTelefonoNoVeElHistorial_MEJ004(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "Código: listo y probado con 12 casos. ¿Hacemos el gate en el Stop hook? Lo demás espera a eso.")
	p := esperaDe(t, root, enCola)
	if q := p.Preguntas[0].Texto; q != "¿Hacemos el gate en el Stop hook? Lo demás espera a eso." {
		t.Fatalf("el texto parte en la pregunta, sin el historial de la tarea: %q", q)
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

// Decisión de la persona (2026-11-03): el check marca los ⏳ de la persona que no llegan al teléfono, y esa cuenta es
// la medición que decide el gate en el Stop hook.
func TestEsperasSinPregunta_SoloLasQueNoPreguntanNiTienenPedido(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "falta tu visto bueno al despliegue")
	sin := EsperasSinPregunta(root, enCola)
	if len(sin) != 1 || sin[0].Tarea != "Barra" || sin[0].Etapa != "PR" || sin[0].Bloqueo != "falta tu visto bueno al despliegue" {
		t.Fatalf("no pregunta nada ni tiene pedido: se marca, con su tarea, etapa y bloqueo: %+v", sin)
	}
	if linea := (EsperaSinPregunta{Tarea: "Barra", Etapa: "PR", Bloqueo: strings.Repeat("x", 500)}).Linea(); !strings.HasPrefix(linea, "Barra · PR: x") || len([]rune(linea)) > maxLineaEspera {
		t.Fatalf("la línea nombra tarea y etapa, y recorta el bloqueo: %q", linea)
	}
	conEspera(t, root, "¿Le damos el visto bueno al despliegue?")
	if sin = EsperasSinPregunta(root, enCola); len(sin) != 0 {
		t.Fatalf("la que pregunta ya llega al teléfono: %+v", sin)
	}
	pd, err := Pedir(root, "aws configure", "Configurar AWS", root, enCola)
	if err != nil {
		t.Fatal(err)
	}
	conEspera(t, root, "falta correr ["+pd.Tecla+"] para desplegar")
	if sin = EsperasSinPregunta(root, enCola); len(sin) != 0 {
		t.Fatalf("la cubre el pedido [%s]: %+v", pd.Tecla, sin)
	}
	conEspera(t, root, "falta correr [z] para desplegar")
	if sin = EsperasSinPregunta(root, enCola); len(sin) != 1 {
		t.Fatalf("[z] no es un pedido abierto: se marca: %+v", sin)
	}
	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "PR", Estado: "⏳", Bloqueo: "falta el CI", DeQuien: "la IA"}, enCola); err != nil {
		t.Fatal(err)
	}
	if sin = EsperasSinPregunta(root, enCola); len(sin) != 0 {
		t.Fatalf("un ⏳ de otro no le pide nada a la persona: %+v", sin)
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

// FND-141: la respuesta a un ⏳ se guardaba como respondida aunque ninguna sesión la tomara, y ya no se podía volver a
// responder en 24 h. La pregunta recuerda qué instrucción lleva su respuesta para que qdd remoto la siga.
func TestEspera_LaRespuestaDiceQueInstruccionLaLleva_FND141(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	r, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"sí"}}`, "Pixel", enCola)
	if err != nil {
		t.Fatal(err)
	}
	ejecuciones := EjecucionesA(root, enCola)
	if len(ejecuciones) != 1 || r.Instruccion == "" || r.Instruccion != ejecuciones[0].Solicitud {
		t.Fatalf("la pregunta apunta a su instrucción: %q, %+v", r.Instruccion, ejecuciones)
	}
}

func TestEspera_UnaRespuestaQueNoLlegoSePuedeResponderDeNuevo_FND141(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	r, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"sí"}}`, "Pixel", enCola)
	if err != nil {
		t.Fatal(err)
	}
	if err := ReabrirEspera(root, p.ID, r.Instruccion, enCola.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if v := esperaDe(t, root, enCola.Add(time.Second)); v.Estado != PreguntaPendiente {
		t.Fatalf("vuelve a preguntarse: %+v", v)
	}
	otra := enCola.Add(2 * time.Second)
	if err := LatirSesion(root, "s1", otra); err != nil {
		t.Fatal(err)
	}
	segunda, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"sí, de nuevo"}}`, "Pixel", otra)
	if err != nil {
		t.Fatalf("se responde otra vez: %v", err)
	}
	if segunda.Instruccion == r.Instruccion {
		t.Fatalf("la segunda respuesta es otra instrucción: %q", segunda.Instruccion)
	}
}

func TestEspera_ReabrirNoTocaUnaRespuestaMasNueva_FND141(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"sí"}}`, "Pixel", enCola); err != nil {
		t.Fatal(err)
	}
	if err := ReabrirEspera(root, p.ID, "otra-instruccion", enCola); err != nil {
		t.Fatal(err)
	}
	if v := esperaDe(t, root, enCola); v.Estado != PreguntaRespondida {
		t.Fatalf("otra instrucción no reabre esta respuesta: %+v", v)
	}
}

// Issue #149: el panel solo mostraba el símbolo; «no veo las preguntas en el tablero». Muestra lo mismo que el teléfono,
// y una nota de la persona también, porque ahí tampoco se leía.
func TestEsperasDelPanel_TraeLaPreguntaEnteraYLasNotasSinID(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "Probé en Linux. ¿Pruebas tú en Windows? Hace falta un PC real.")
	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "Código", Estado: "⏳", Bloqueo: "Lo pruebas cuando puedas", DeQuien: "usuario"}, enCola); err != nil {
		t.Fatal(err)
	}
	got := EsperasDelPanel(root, enCola)
	if len(got) != 2 {
		t.Fatalf("una pregunta y una nota de la persona; el ⏳ de otro no: %+v", got)
	}
	p, n := got[0], got[1]
	if p.ID == "" || p.ID != esperaDe(t, root, enCola).ID || p.Donde != "Barra · PR" || p.Texto != "¿Pruebas tú en Windows? Hace falta un PC real." {
		t.Fatalf("la pregunta va primero, con el id del teléfono, y entera desde el ¿: %+v", p)
	}
	if n.ID != "" || n.Donde != "Barra · Código" || n.Texto != "Lo pruebas cuando puedas" {
		t.Fatalf("una nota no se responde: no trae id: %+v", n)
	}
}

func TestEsperasDelPanel_LoCubiertoPorUnPedidoNoSeRepite(t *testing.T) {
	root := proyectoConTareas(t)
	pd, err := Pedir(root, "aws configure", "Configurar AWS", root, enCola)
	if err != nil {
		t.Fatal(err)
	}
	conEspera(t, root, "¿Corres ["+pd.Tecla+"] para desplegar?")
	if got := EsperasDelPanel(root, enCola); len(got) != 0 {
		t.Fatalf("el pedido ya lo muestra el panel: %+v", got)
	}
}

func TestResponderEsperaEnPC_DaElMensajeYNoSeVuelveAPreguntar(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	p := esperaDe(t, root, enCola)

	mensaje, err := ResponderEsperaEnPC(root, p.ID, "  sí, mergea  ", enCola)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mensaje, "Barra · PR: ¿mergeamos el #8?") || !strings.Contains(mensaje, "\nsí, mergea\n") || !strings.Contains(mensaje, "qdd_estado") {
		t.Fatalf("la IA lee a qué ⏳ responde, la respuesta y cómo reflejarla: %q", mensaje)
	}
	if got := EsperasDelPanel(root, enCola); len(got) != 0 {
		t.Fatalf("ya respondida no se ofrece: %+v", got)
	}
	r := esperaDe(t, root, enCola)
	if r.Estado != PreguntaRespondida || r.RespondidaEn != EnPC || len(EjecucionesA(root, enCola)) != 0 {
		t.Fatalf("queda respondida en el PC, sin instrucción para otra sesión: %+v", r)
	}
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"no"}}`, "Pixel", enCola); err == nil || !strings.Contains(err.Error(), "ya respondida en PC") {
		t.Fatalf("el teléfono llega tarde y lo sabe: %v", err)
	}
}

func TestResponderEsperaEnPC_LaPrimeraGanaYLaVaciaNoCuenta(t *testing.T) {
	root := proyectoConTareas(t)
	conEspera(t, root, "¿mergeamos el #8?")
	conSesionAbierta(t, root)
	p := esperaDe(t, root, enCola)
	if _, err := ResponderEsperaEnPC(root, "espera-otra", "sí", enCola); !errors.Is(err, ErrPreguntaDesconocida) {
		t.Fatalf("un ⏳ que ya no existe: %v", err)
	}
	if _, err := ResponderEsperaEnPC(root, p.ID, "   ", enCola); !errors.Is(err, ErrRespuestaInvalida) {
		t.Fatalf("sin texto no hay respuesta: %v", err)
	}
	if e := esperaDe(t, root, enCola); e.Estado != PreguntaPendiente {
		t.Fatalf("una respuesta rechazada no la cierra: %+v", e)
	}
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"otra":"sí"}}`, "Pixel", enCola); err != nil {
		t.Fatal(err)
	}
	var cerrada *PreguntaCerrada
	if _, err := ResponderEsperaEnPC(root, p.ID, "no", enCola); !errors.As(err, &cerrada) || !strings.Contains(err.Error(), "Pixel") {
		t.Fatalf("dice que ya la respondió el teléfono: %v", err)
	}
}
