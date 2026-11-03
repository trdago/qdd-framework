package remoto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Preguntas desde el teléfono»: el teléfono responde con qdd_remoto_responder; el relay sella el
// dispositivo y reenvía el evento responder firmado. El PC lo verifica como una acción (firma, ±5 min, una sola vez
// por solicitud), valida la respuesta contra la pregunta guardada, la deja para el mod y responde accion_estado:
// hecha, ya_aprobado (con el lugar), vencida o rechazada. Lo que se publica de una pregunta pasa por el ocultador.

func TestFirmaRespuesta_LaMismaQueElRelay(t *testing.T) {
	// hmac.new(("ab"*32).encode(), 's|p|pregunta:x|{"q1": {"opciones": ["a"]}}|d|2026-11-02T10:00:00Z', sha256): lo
	// mismo que pgcrypto hmac(mensaje, firma) con la respuesta como p_respuesta::text.
	got := FirmaRespuesta(firmaPrueba, "s", "p", "pregunta:x", `{"q1": {"opciones": ["a"]}}`, "d", "2026-11-02T10:00:00Z")
	if got != "70aec06b50d908e3156a20430522b72c421453844460f0bce0cd157b596ec94e" {
		t.Fatalf("firma: %s", got)
	}
	now := time.Date(2026, 11, 2, 10, 3, 0, 0, time.UTC)
	r := respuestaFirmada("s", "p", "pregunta:x", `{"q1": {"opciones": ["a"]}}`, "d", "2026-11-02T10:00:00Z")
	if err := r.Verificar(firmaPrueba, now); err != nil {
		t.Fatal(err)
	}
	mala := r
	mala.Respuesta = `{"q1": {"opciones": ["b"]}}`
	if err := mala.Verificar(firmaPrueba, now); err == nil {
		t.Fatal("cambiar la respuesta rompe la firma")
	}
	if err := r.Verificar(firmaPrueba, now.Add(3*time.Minute)); err == nil {
		t.Fatal("más de 5 min es vieja")
	}
}

func respuestaFirmada(solicitud, proyecto, pregunta, respuesta, dispositivo, en string) RespuestaFirmada {
	return RespuestaFirmada{Version: 1, Solicitud: solicitud, Proyecto: proyecto, Pregunta: pregunta, Respuesta: respuesta,
		Dispositivo: dispositivo, Nombre: "Pixel de " + dispositivo, En: en,
		Firma: FirmaRespuesta(firmaPrueba, solicitud, proyecto, pregunta, respuesta, dispositivo, en)}
}

func mandarRespuesta(tr *transporteFalso, r RespuestaFirmada) {
	p, _ := json.Marshal(r)
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "responder", Payload: p}
}

const entradaPregunta = `{"questions":[{"question":"¿Seguimos?","header":"Seguir","multiSelect":false,"options":[{"label":"Sí","description":"Adelante"},{"label":"No","description":""}]}]}`

// proyectoConPregunta registra una pregunta de la IA en un proyecto del registro.
func proyectoConPregunta(t *testing.T) (string, string, consola.Pregunta) {
	t.Helper()
	root := proyecto(t, "app", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	p, err := consola.RegistrarPregunta(root, consola.NuevaPregunta{Sesion: "s1", Herramienta: "AskUserQuestion", Entrada: json.RawMessage(entradaPregunta)}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return root, clave, p
}

func TestResponder_LaRespuestaLlegaUnaSolaVezYElOtroTelefonoVeDonde(t *testing.T) {
	homeTemporal(t)
	root, clave, p := proyectoConPregunta(t)
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	r := respuestaFirmada("1b6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "pregunta:"+p.ID, `{"q1": {"opciones": ["b"]}}`, "d1", ahoraEn())
	mandarRespuesta(tr, r)
	for _, want := range []string{EstadoRecibida, EstadoHecha} {
		if got := estadoDe(t, tr, 5*time.Second); got.Estado != want || got.Solicitud != r.Solicitud {
			t.Fatalf("esperaba %s: %+v", want, got)
		}
	}
	q, _ := consola.PreguntaDe(root, p.ID, time.Now())
	if q.Estado != consola.PreguntaRespondida || q.RespondidaEn != consola.EnTelefono || q.Lugar != "Pixel de d1" {
		t.Fatalf("la pregunta queda respondida en ese teléfono, para que el mod la entregue: %+v", q)
	}
	if e := consola.EntregaDe(q); e.Respuestas["¿Seguimos?"] != "No" {
		t.Fatalf("%+v", e)
	}
	mandarRespuesta(tr, r)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoHecha {
		t.Fatalf("la misma solicitud repetida responde su último estado: %+v", got)
	}
	otra := respuestaFirmada("2b6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, p.ID, `{"q1": {"opciones": ["a"]}}`, "d2", ahoraEn())
	mandarRespuesta(tr, otra)
	estadoDe(t, tr, 2*time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoYaAprobado || got.Lugar != "Pixel de d1" || !strings.Contains(got.Resultado, "Pixel de d1") {
		t.Fatalf("otro teléfono ve «ya respondida» en el primero: %+v", got)
	}
	if e := consola.EntregaDe(mustPreguntaRemota(t, root, p.ID)); e.Respuestas["¿Seguimos?"] != "No" {
		t.Fatalf("vale la primera: %+v", e)
	}
	var hechas int
	for _, l := range lineasLog(t) {
		if l.Solicitud == r.Solicitud && l.Estado == EstadoHecha && l.AccionID == "pregunta:"+p.ID {
			hechas++
		}
	}
	if hechas != 1 {
		t.Fatalf("queda una vez en el log local: %+v", lineasLog(t))
	}
}

func mustPreguntaRemota(t *testing.T, root, id string) consola.Pregunta {
	t.Helper()
	p, ok := consola.PreguntaDe(root, id, time.Now())
	if !ok {
		t.Fatalf("no está la pregunta %s", id)
	}
	return p
}

func TestResponder_InvalidaVencidaRevocadaOSinFirma(t *testing.T) {
	homeTemporal(t)
	root, clave, p := proyectoConPregunta(t)
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")

	casos := []struct {
		solicitud, pregunta, respuesta, estado, dice string
	}{
		{"3b6b1c1e-0000-4f39-9f4e-4a1c2d3e4f50", "pregunta:" + p.ID, `{"q1": {"opciones": ["z"]}}`, EstadoRechazada, "«z»"},
		{"4b6b1c1e-0000-4f39-9f4e-4a1c2d3e4f50", "pregunta:" + p.ID, `{"q1": {"opciones": ["a", "b"]}}`, EstadoRechazada, "una sola"},
		{"5b6b1c1e-0000-4f39-9f4e-4a1c2d3e4f50", "pregunta:" + p.ID, `{"q7": {"opciones": ["a"]}}`, EstadoRechazada, "q7"},
		{"6b6b1c1e-0000-4f39-9f4e-4a1c2d3e4f50", "pregunta:no-existe", `{"q1": {"opciones": ["a"]}}`, EstadoVencida, "ya no"},
	}
	for _, c := range casos {
		mandarRespuesta(tr, respuestaFirmada(c.solicitud, clave, c.pregunta, c.respuesta, "d1", ahoraEn()))
		estadoDe(t, tr, 2*time.Second)
		if got := estadoDe(t, tr, 2*time.Second); got.Estado != c.estado || !strings.Contains(got.Resultado, c.dice) {
			t.Errorf("%s: esperaba %s diciendo %q: %+v", c.respuesta, c.estado, c.dice, got)
		}
	}
	if q := mustPreguntaRemota(t, root, p.ID); q.Estado != consola.PreguntaPendiente {
		t.Fatalf("nada de eso la responde: %+v", q)
	}

	consola.VencerSesion(root, "s1", "se cerró la sesión", time.Now())
	mandarRespuesta(tr, respuestaFirmada("7b6b1c1e-0000-4f39-9f4e-4a1c2d3e4f50", clave, p.ID, `{"q1": {"opciones": ["a"]}}`, "d1", ahoraEn()))
	estadoDe(t, tr, 2*time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoVencida || !strings.Contains(got.Resultado, "se cerró la sesión") {
		t.Fatalf("una vencida dice por qué: %+v", got)
	}

	q, _ := consola.RegistrarPregunta(root, consola.NuevaPregunta{Sesion: "s1", Herramienta: "AskUserQuestion", Entrada: json.RawMessage(entradaPregunta)}, time.Now())
	Revocar("d9")
	mandarRespuesta(tr, respuestaFirmada("8b6b1c1e-0000-4f39-9f4e-4a1c2d3e4f50", clave, q.ID, `{"q1": {"opciones": ["a"]}}`, "d9", ahoraEn()))
	estadoDe(t, tr, 2*time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, "revocado") {
		t.Fatalf("un teléfono revocado no responde: %+v", got)
	}
	sinFirma := respuestaFirmada("9b6b1c1e-0000-4f39-9f4e-4a1c2d3e4f50", clave, q.ID, `{"q1": {"opciones": ["a"]}}`, "d1", ahoraEn())
	sinFirma.Firma = strings.Repeat("0", 64)
	mandarRespuesta(tr, sinFirma)
	time.Sleep(300 * time.Millisecond)
	for _, e := range tr.Enviados() {
		if e.Evento == "accion_estado" && strings.Contains(string(e.Payload), sinFirma.Solicitud) {
			t.Fatalf("lo que no trae la firma del relay no recibe respuesta: %s", e.Payload)
		}
	}
	if got := mustPreguntaRemota(t, root, q.ID); got.Estado != consola.PreguntaPendiente {
		t.Fatalf("lo que no trae la firma del relay no responde nada: %+v", got)
	}
}

func TestResponder_SiGanoElTerminalElTelefonoTerminaViendoPC(t *testing.T) {
	// El teléfono alcanzó a responder, pero el terminal respondió antes de que el mod lo entregara: el mod la cierra en
	// el PC y el PC le corrige al teléfono el estado de su solicitud (ya_aprobado, lugar PC).
	homeTemporal(t)
	root, clave, p := proyectoConPregunta(t)
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	r := respuestaFirmada("ab6b1c1e-0000-4f39-9f4e-4a1c2d3e4f50", clave, p.ID, `{"q1": {"opciones": ["a"]}}`, "d1", ahoraEn())
	mandarRespuesta(tr, r)
	estadoDe(t, tr, 2*time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoHecha {
		t.Fatal(got)
	}
	consola.CerrarPregunta(root, p.ID, consola.EnPC, "", time.Now())
	if got := estadoDe(t, tr, 3*time.Second); got.Estado != EstadoYaAprobado || got.Lugar != LugarPC || got.Solicitud != r.Solicitud {
		t.Fatalf("el teléfono termina viendo que valió la del PC: %+v", got)
	}
}

func TestArmar_LaPreguntaSalePublicadaSinSecretos(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", true)
	reg := registro.Registro{Proyectos: []registro.Proyecto{{Ruta: root, Clave: "app-1"}}}
	entrada, _ := json.Marshal(map[string]string{"command": "curl -H 'Authorization: Bearer ghp_abcdefghijklmnopqrstuvwxyz0123456789' https://api.github.com"})
	if _, err := consola.RegistrarPregunta(root, consola.NuevaPregunta{Sesion: "s1", Herramienta: "Bash", Entrada: entrada}, time.Now()); err != nil {
		t.Fatal(err)
	}
	e := Armar("v", reg, time.Now(), 0)
	var a *consola.Accion
	for i, x := range e.Proyectos[0].Panel.Acciones {
		if x.Tipo == consola.TipoPregunta {
			a = &e.Proyectos[0].Panel.Acciones[i]
		}
	}
	if a == nil || a.Permiso == nil || a.Origen != consola.OrigenPermiso {
		t.Fatalf("el teléfono recibe la pregunta en el panel del proyecto: %+v", e.Proyectos[0].Panel.Acciones)
	}
	b, _ := json.Marshal(a)
	if strings.Contains(string(b), "ghp_") || !strings.Contains(a.Permiso.Detalle, Oculto) || !strings.Contains(a.Detalle, Oculto) {
		t.Fatalf("el comando del permiso sale sin el secreto: %s", b)
	}
}
