package consola

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Preguntas desde el teléfono»: la consola del PC registra lo que le pregunta a la persona (una pregunta de
// la IA o un permiso de herramienta), el panel lo ofrece como acción «pregunta» (solo a quien la pide: tipo opcional) y
// gana quien responde primero, el terminal o un teléfono.

var enPregunta = time.Date(2026, 11, 3, 1, 0, 0, 0, time.UTC)

const entradaAskUser = `{"questions":[
 {"question":"¿Qué base de datos usamos?","header":"Base","multiSelect":false,
  "options":[{"label":"Postgres","description":"La de siempre"},{"label":"SQLite","description":"Un archivo"}]},
 {"question":"¿Qué pruebas corremos?","header":"Pruebas","multiSelect":true,
  "options":[{"label":"Unitarias","description":""},{"label":"Integración","description":"Con la base"},{"label":"E2E","description":""}]}]}`

func registrarAsk(t *testing.T, root string) Pregunta {
	t.Helper()
	p, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s1", Herramienta: "AskUserQuestion", Entrada: json.RawMessage(entradaAskUser)}, enPregunta)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func registrarBash(t *testing.T, root, comando string) Pregunta {
	t.Helper()
	entrada, _ := json.Marshal(map[string]string{"command": comando, "description": "Publica el tag"})
	p, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s1", Herramienta: "Bash", Entrada: entrada}, enPregunta)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPregunta_LaDeLaIAQuedaConSusOpcionesYSaleSoloConCon(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarAsk(t, root)
	if p.ID == "" || p.Origen != OrigenIA || p.Estado != PreguntaPendiente || p.Sesion != "s1" {
		t.Fatalf("pregunta: %+v", p)
	}
	if len(p.Preguntas) != 2 || p.Preguntas[0].ID != "q1" || p.Preguntas[1].ID != "q2" || !p.Preguntas[1].Multiple || p.Preguntas[0].Multiple {
		t.Fatalf("una por cada pregunta, con su id y si es múltiple: %+v", p.Preguntas)
	}
	q := p.Preguntas[0]
	if q.Encabezado != "Base" || q.Texto != "¿Qué base de datos usamos?" || !q.Otra || len(q.Opciones) != 2 {
		t.Fatalf("encabezado, texto, «Otra» y opciones: %+v", q)
	}
	if q.Opciones[0] != (OpcionPregunta{Valor: "a", Etiqueta: "Postgres", Descripcion: "La de siempre"}) || q.Opciones[1].Valor != "b" {
		t.Fatalf("cada opción con su valor, etiqueta y descripción: %+v", q.Opciones)
	}
	if n := len([]rune(p.Etiqueta)); n == 0 || n > MaxMotivo {
		t.Fatalf("la etiqueta es un resumen de ≤ %d: %q", MaxMotivo, p.Etiqueta)
	}

	if a := accionPregunta(Panel(root, enPregunta).Acciones); a != nil {
		t.Fatalf("un panel que no las pide no las ve (tipo opcional, FND-092): %+v", a)
	}
	a := accionPregunta(Panel(root, enPregunta, TipoPregunta).Acciones)
	if a == nil || a.ID != "pregunta:"+p.ID || a.Estado != PreguntaPendiente || a.Origen != OrigenIA || !a.Confirmar || a.Sesion != "s1" || len(a.Preguntas) != 2 {
		t.Fatalf("con --con pregunta, es una acción más del panel: %+v", a)
	}
	b, _ := json.Marshal(a)
	for _, campo := range []string{`"tipo":"pregunta"`, `"origen":"ia"`, `"preguntas":[`, `"encabezado":"Base"`, `"multiple":true`, `"otra":true`, `"valor":"a"`, `"descripcion":"La de siempre"`} {
		if !strings.Contains(string(b), campo) {
			t.Fatalf("el JSON del contrato lleva %s: %s", campo, b)
		}
	}
	entradas, _ := os.ReadDir(filepath.Join(root, ".qdd"))
	if len(entradas) != 0 {
		t.Fatalf("las preguntas viven fuera del proyecto: %v", entradas)
	}
}

func accionPregunta(acciones []Accion) *Accion {
	for i := range acciones {
		if acciones[i].Tipo == TipoPregunta {
			return &acciones[i]
		}
	}
	return nil
}

func TestPregunta_UnPermisoTraeLaHerramientaElComandoYSoloPermitirONegar(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarBash(t, root, "git push origin v2.9.0")
	if p.Origen != OrigenPermiso || p.Permiso == nil || p.Permiso.Herramienta != "Bash" || p.Permiso.Detalle != "git push origin v2.9.0" {
		t.Fatalf("permiso con la herramienta y el comando exacto: %+v", p)
	}
	if len(p.Preguntas) != 1 || p.Preguntas[0].Otra || p.Preguntas[0].Multiple {
		t.Fatalf("una sola pregunta, sin «Otra» ni múltiple: %+v", p.Preguntas)
	}
	var valores []string
	for _, o := range p.Preguntas[0].Opciones {
		valores = append(valores, o.Valor)
	}
	if strings.Join(valores, ",") != PermitirUnaVez+","+Negar {
		t.Fatalf("las opciones son exactamente permitir y negar (nunca «siempre»): %v", valores)
	}
	if !strings.Contains(p.Etiqueta, "Bash") {
		t.Fatalf("la etiqueta nombra la herramienta: %q", p.Etiqueta)
	}
	largo := strings.Repeat("x", 9000)
	if q := registrarBash(t, root, largo); len([]rune(q.Permiso.Detalle)) > MaxDetallePermiso+40 || !strings.Contains(q.Permiso.Detalle, "caracteres") {
		t.Fatalf("un comando muy largo se corta y dice cuánto falta: %d", len(q.Permiso.Detalle))
	}
}

func TestPregunta_ResponderDesdeElTelefonoUnaSolaVez(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarAsk(t, root)
	r, err := ResponderPregunta(root, p.ID, `{"q1": {"opciones": ["b"]}, "q2": {"opciones": ["a", "c"]}}`, "Pixel", enPregunta.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if r.Estado != PreguntaRespondida || r.RespondidaEn != EnTelefono || r.Lugar != "Pixel" {
		t.Fatalf("respondida en el teléfono: %+v", r)
	}
	e := EntregaDe(r)
	if e.Respuestas["¿Qué base de datos usamos?"] != "SQLite" || e.Respuestas["¿Qué pruebas corremos?"] != "Unitarias, E2E" {
		t.Fatalf("la respuesta para Claude va por texto de la pregunta, con las etiquetas (múltiple separada por coma): %+v", e)
	}
	_, err = ResponderPregunta(root, p.ID, `{"q1": {"opciones": ["a"]}, "q2": {"opciones": ["a"]}}`, "iPhone", enPregunta.Add(2*time.Second))
	var cerrada *PreguntaCerrada
	if !errors.As(err, &cerrada) || cerrada.Pregunta.Lugar != "Pixel" || cerrada.Pregunta.Estado != PreguntaRespondida {
		t.Fatalf("la segunda respuesta (otro teléfono) ve «ya respondida» en el primero: %v", err)
	}
	if got := EntregaDe(mustPregunta(t, root, p.ID)); got.Respuestas["¿Qué base de datos usamos?"] != "SQLite" {
		t.Fatalf("vale la primera: %+v", got)
	}
	if _, err := ResponderPregunta(root, "no-existe", `{}`, "Pixel", enPregunta); !errors.Is(err, ErrPreguntaDesconocida) {
		t.Fatalf("una pregunta que no está: %v", err)
	}
}

func mustPregunta(t *testing.T, root, id string) Pregunta {
	t.Helper()
	p, ok := PreguntaDe(root, id, enPregunta)
	if !ok {
		t.Fatalf("no está la pregunta %s", id)
	}
	return p
}

func TestPregunta_OtraYPermiso(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarAsk(t, root)
	r, err := ResponderPregunta(root, p.ID, `{"q1": {"otra": "  MariaDB  "}, "q2": {"opciones": ["b"]}}`, "Pixel", enPregunta)
	if err != nil {
		t.Fatal(err)
	}
	if e := EntregaDe(r); e.Respuestas["¿Qué base de datos usamos?"] != "MariaDB" || e.Respuestas["¿Qué pruebas corremos?"] != "Integración" {
		t.Fatalf("«Otra» va como texto, sin espacios de más: %+v", e)
	}
	permiso := registrarBash(t, root, "rm -rf tmp/")
	r, err = ResponderPregunta(root, permiso.ID, `{"q1":{"opciones":["negar"]}}`, "Pixel", enPregunta)
	if err != nil {
		t.Fatal(err)
	}
	if e := EntregaDe(r); e.Decision != Negar || len(e.Respuestas) != 0 {
		t.Fatalf("un permiso entrega la decisión: %+v", e)
	}
}

func TestPregunta_UnaRespuestaInvalidaSeRechazaSinCambiarNada(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarAsk(t, root)
	permiso := registrarBash(t, root, "git push")
	casos := []struct{ id, respuesta, motivo string }{
		{p.ID, `no es json`, "JSON"},
		{p.ID, `{"q1":{"opciones":["a"]}}`, "falta"},
		{p.ID, `{"q1":{"opciones":["a"]},"q2":{"opciones":["a"]},"q9":{"opciones":["a"]}}`, "q9"},
		{p.ID, `{"q1":{"opciones":["z"]},"q2":{"opciones":["a"]}}`, "«z»"},
		{p.ID, `{"q1":{"opciones":["a","b"]},"q2":{"opciones":["a"]}}`, "una sola"},
		{p.ID, `{"q1":{"opciones":[]},"q2":{"opciones":["a"]}}`, "ninguna"},
		{p.ID, `{"q1":{"opciones":["a"]},"q2":{"opciones":["a","a"]}}`, "repite"},
		{p.ID, `{"q1":{"otra":"   "},"q2":{"opciones":["a"]}}`, "vacía"},
		{p.ID, `{"q1":{"otra":"` + strings.Repeat("ñ", MaxOtra+1) + `"},"q2":{"opciones":["a"]}}`, "500"},
		{p.ID, `{"q1":{"otra":"x","opciones":["a"]},"q2":{"opciones":["a"]}}`, "o "},
		{p.ID, `{"q1":{"opciones":["a"],"extra":1},"q2":{"opciones":["a"]}}`, "extra"},
		{p.ID, `{"q1":{"otra":"` + strings.Repeat("x", MaxRespuesta) + `"}}`, "4096"},
		{permiso.ID, `{"q1":{"otra":"sí"}}`, "«Otra»"},
		{permiso.ID, `{"q1":{"opciones":["siempre"]}}`, "«siempre»"},
	}
	for _, c := range casos {
		_, err := ResponderPregunta(root, c.id, c.respuesta, "Pixel", enPregunta)
		if !errors.Is(err, ErrRespuestaInvalida) || !strings.Contains(err.Error(), c.motivo) {
			t.Errorf("%s: se rechaza diciendo %q: %v", c.respuesta[:min(len(c.respuesta), 60)], c.motivo, err)
		}
	}
	if got := mustPregunta(t, root, p.ID); got.Estado != PreguntaPendiente {
		t.Fatalf("una respuesta inválida no la cierra: %+v", got)
	}
}

func TestPregunta_ElTerminalGanaYElTelefonoLaVeRespondida(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarAsk(t, root)
	if _, err := CerrarPregunta(root, p.ID, EnPC, "", enPregunta); err != nil {
		t.Fatal(err)
	}
	_, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["a"]},"q2":{"opciones":["a"]}}`, "Pixel", enPregunta)
	var cerrada *PreguntaCerrada
	if !errors.As(err, &cerrada) || cerrada.Pregunta.RespondidaEn != EnPC {
		t.Fatalf("respondida en el PC: %v", err)
	}
	a := accionPregunta(Panel(root, enPregunta.Add(time.Minute), TipoPregunta).Acciones)
	if a == nil || a.Estado != PreguntaRespondida || a.RespondidaEn != EnPC {
		t.Fatalf("el panel la sigue mostrando un rato como respondida en el PC: %+v", a)
	}
	if a := accionPregunta(Panel(root, enPregunta.Add(VidaCerrada+time.Minute), TipoPregunta).Acciones); a != nil {
		t.Fatalf("pasado VidaCerrada sale del panel: %+v", a)
	}
}

func TestPregunta_SiElTerminalGanoDespuesDelTelefonoQuedaEnElPC(t *testing.T) {
	// El teléfono respondió, pero el terminal respondió antes de que el mod alcanzara a entregarlo: lo que vale es lo
	// que vio Claude Code, así que queda respondida en el PC (y el PC le corrige el estado al teléfono).
	root := proyectoConCola(t)
	p := registrarBash(t, root, "git push")
	if _, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["permitir"]}}`, "Pixel", enPregunta); err != nil {
		t.Fatal(err)
	}
	r, err := CerrarPregunta(root, p.ID, EnPC, "", enPregunta)
	if err != nil || r.RespondidaEn != EnPC || r.Lugar != "" || len(r.Respuesta) != 0 {
		t.Fatalf("queda respondida en el PC, sin la respuesta del teléfono: %+v %v", r, err)
	}
}

func TestPregunta_VenceSiLaSesionDejaDeEsperarla(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarAsk(t, root)
	if got := mustPregunta(t, root, p.ID); got.Estado != PreguntaPendiente {
		t.Fatal(got.Estado)
	}
	if _, err := LatirPregunta(root, p.ID, enPregunta.Add(50*time.Second)); err != nil {
		t.Fatal(err)
	}
	if got, _ := PreguntaDe(root, p.ID, enPregunta.Add(100*time.Second)); got.Estado != PreguntaPendiente {
		t.Fatalf("la sesión que la sigue esperando la mantiene viva: %+v", got)
	}
	got, _ := PreguntaDe(root, p.ID, enPregunta.Add(50*time.Second+LatidoPregunta+time.Second))
	if got.Estado != PreguntaVencida || got.Motivo == "" {
		t.Fatalf("sin latido de la sesión vence, con su motivo: %+v", got)
	}
	_, err := ResponderPregunta(root, p.ID, `{"q1":{"opciones":["a"]},"q2":{"opciones":["a"]}}`, "Pixel", enPregunta.Add(3*time.Minute))
	var cerrada *PreguntaCerrada
	if !errors.As(err, &cerrada) || cerrada.Pregunta.Estado != PreguntaVencida {
		t.Fatalf("una vencida no se responde: %v", err)
	}
}

// FND-165: la consola siguió detenida 2 h en un AskUserQuestion, pero a los 30 min la pregunta se fue del teléfono y
// nadie la podía responder desde allí. Mientras la sesión la espera, sigue viva.
func TestPregunta_MientrasLaSesionLaEsperaNoSeVaDelTelefono(t *testing.T) {
	root := proyectoConCola(t)
	q := registrarAsk(t, root)
	dosHoras := 2 * time.Hour
	for n := time.Duration(1); n*LatidoPregunta/2 <= dosHoras; n++ {
		LatirPregunta(root, q.ID, enPregunta.Add(n*LatidoPregunta/2))
	}
	if got, _ := PreguntaDe(root, q.ID, enPregunta.Add(dosHoras)); got.Estado != PreguntaPendiente {
		t.Fatalf("la sesión la sigue esperando: tiene que seguir pendiente: %+v", got)
	}
	if _, err := ResponderPregunta(root, q.ID, `{"q1":{"opciones":["a"]},"q2":{"opciones":["a"]}}`, "Pixel", enPregunta.Add(dosHoras)); err != nil {
		t.Fatalf("tiene que poder responderse desde el teléfono: %v", err)
	}
}

func TestPregunta_AlCerrarseLaSesionVencenLasSuyas(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarAsk(t, root)
	otra, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s2", Herramienta: "AskUserQuestion", Entrada: json.RawMessage(entradaAskUser)}, enPregunta)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := VencerSesion(root, "s1", "se cerró la sesión", enPregunta); err != nil || n != 1 {
		t.Fatalf("vence solo la de esa sesión: %d %v", n, err)
	}
	if got := mustPregunta(t, root, p.ID); got.Estado != PreguntaVencida || got.Motivo != "se cerró la sesión" {
		t.Fatalf("%+v", got)
	}
	if got := mustPregunta(t, root, otra.ID); got.Estado != PreguntaPendiente {
		t.Fatalf("la de otra sesión sigue: %+v", got)
	}
}

func TestPregunta_UnComandoQDDPreguntaConSusOpciones(t *testing.T) {
	root := proyectoConCola(t)
	n := NuevaPregunta{Origen: OrigenQDD, Etiqueta: "qdd blueprint apply auth", Preguntas: []ItemPregunta{{
		ID: "identificador", Encabezado: "Identificador", Texto: "¿Con qué se identifica un usuario?",
		Opciones: []OpcionPregunta{{Valor: "rut", Etiqueta: "RUT"}, {Valor: "correo", Etiqueta: "Correo"}},
	}}}
	p, err := RegistrarPregunta(root, n, enPregunta)
	if err != nil || p.Origen != OrigenQDD {
		t.Fatalf("%+v %v", p, err)
	}
	r, err := ResponderPregunta(root, p.ID, `{"identificador":{"opciones":["correo"]}}`, "Pixel", enPregunta)
	if err != nil || r.Respuesta["identificador"].Opciones[0] != "correo" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := RegistrarPregunta(root, NuevaPregunta{Origen: OrigenQDD, Etiqueta: "x"}, enPregunta); err == nil {
		t.Fatal("sin preguntas no se registra")
	}
	malas := NuevaPregunta{Origen: OrigenQDD, Etiqueta: "x", Preguntas: []ItemPregunta{{ID: "a", Texto: "¿?", Opciones: []OpcionPregunta{{Valor: "v", Etiqueta: "V"}, {Valor: "v", Etiqueta: "W"}}}}}
	if _, err := RegistrarPregunta(root, malas, enPregunta); err == nil {
		t.Fatal("dos opciones con el mismo valor no se registran")
	}
	if _, err := RegistrarPregunta(root, NuevaPregunta{Herramienta: "AskUserQuestion", Entrada: json.RawMessage(`{"questions":[]}`)}, enPregunta); err == nil {
		t.Fatal("una AskUserQuestion sin preguntas no se registra")
	}
}

func TestPregunta_UnaColaIlegibleSeAparta(t *testing.T) {
	root := proyectoConCola(t)
	ruta, _ := RutaLocal(root, "preguntas")
	os.MkdirAll(filepath.Dir(ruta), 0o700)
	os.WriteFile(ruta, []byte("{roto"), 0o600)
	p := registrarAsk(t, root)
	if _, ok := PreguntaDe(root, p.ID, enPregunta); !ok {
		t.Fatal("se registra en una cola nueva")
	}
	if _, err := os.Stat(ruta + ".ilegible"); err != nil {
		t.Fatalf("la ilegible queda aparte, no se pierde: %v", err)
	}
}

// FND-198 (MEJ-025, issue #147): la IA recibía la elección del modo goal igual que una respuesta de la persona, y dejó
// escrito en commits que la persona había aprobado lo que eligió el modo. La respuesta lo dice.
func TestPregunta_LaQueEligioElModoGoalLoDiceEnLaRespuesta_FND198(t *testing.T) {
	root := proyectoConCola(t)
	p := registrarAsk(t, root)
	r, err := ResponderPregunta(root, p.ID, `{"q1": {"opciones": ["b"]}, "q2": {"opciones": ["a", "c"]}}`, LugarGoal, enPregunta.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	e := EntregaDe(r)
	if e.Respuestas["¿Qué base de datos usamos?"] != "SQLite"+MarcaGoal || e.Respuestas["¿Qué pruebas corremos?"] != "Unitarias, E2E"+MarcaGoal {
		t.Fatalf("cada respuesta dice que la eligió el modo goal: %+v", e.Respuestas)
	}
	if !strings.Contains(MarcaGoal, "modo goal") || !strings.Contains(MarcaGoal, "no la persona") {
		t.Fatalf("la marca se entiende sola: %q", MarcaGoal)
	}
}
