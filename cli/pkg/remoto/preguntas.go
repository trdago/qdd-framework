package remoto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Preguntas desde el teléfono»: el teléfono responde una pregunta de la consola con qdd_remoto_responder;
// el relay sella el dispositivo y reenvía el evento responder con firma = hex(hmac_sha256(firma de la máquina,
// solicitud|proyecto|pregunta|respuesta|dispositivo|en)), donde respuesta es el texto JSON tal como lo firmó el relay.
// El PC lo trata como una acción: descarta lo que no trae firma válida o es viejo, aplica cada solicitud una sola vez
// (solicitudes.json), valida la respuesta contra la pregunta guardada (consola.ResponderPregunta) y la deja para el
// mod, que se la entrega a Claude Code. Responde accion_estado: recibida → hecha | ya_aprobado (con el lugar) |
// vencida | rechazada.

// EstadoVencida: la pregunta ya no espera respuesta (su sesión se cerró, o pasó su tiempo).
const EstadoVencida = "vencida"

// RespuestaFirmada es el evento responder que reenvía el relay.
type RespuestaFirmada struct {
	Version     int    `json:"version"`
	Solicitud   string `json:"solicitud"`
	Proyecto    string `json:"proyecto"`
	Pregunta    string `json:"pregunta"`
	Respuesta   string `json:"respuesta"`
	Dispositivo string `json:"dispositivo"`
	Nombre      string `json:"nombre"`
	En          string `json:"en"`
	Firma       string `json:"firma"`
}

// FirmaRespuesta es la firma que pone el relay a una respuesta (la misma clave que FirmaAccion).
func FirmaRespuesta(firmaMaquina, solicitud, proyecto, pregunta, respuesta, dispositivo, en string) string {
	return firmar(firmaMaquina, solicitud, proyecto, pregunta, respuesta, dispositivo, en)
}

// Verificar exige la firma de la máquina y que la respuesta sea de los últimos VigenciaAccion.
func (r RespuestaFirmada) Verificar(firmaMaquina string, now time.Time) error {
	if r.Solicitud == "" || r.Proyecto == "" || r.Pregunta == "" || r.Respuesta == "" || r.Dispositivo == "" || r.En == "" {
		return errors.New("respuesta incompleta")
	}
	esperada := FirmaRespuesta(firmaMaquina, r.Solicitud, r.Proyecto, r.Pregunta, r.Respuesta, r.Dispositivo, r.En)
	return verificarFirmada(esperada, r.Firma, r.En, now)
}

// alResponder recibe una respuesta del relay: la verifica, la recuerda antes de aplicarla y la aplica.
func (b *Bucle) alResponder(ctx context.Context, payload json.RawMessage) {
	var r RespuestaFirmada
	if err := json.Unmarshal(payload, &r); err != nil {
		b.logf("respuesta descartada: ilegible")
		return
	}
	id := consola.IDPregunta(r.Pregunta)
	accionID := "pregunta:" + id
	if err := r.Verificar(b.Cred.Firma, b.ahora()); err != nil {
		b.logf("respuesta %s descartada: %v", r.Solicitud, err)
		b.anotar(AccionFirmada{Solicitud: r.Solicitud, Proyecto: r.Proyecto, AccionID: accionID, Dispositivo: r.Dispositivo, DispositivoNombre: r.Nombre}, EstadoDescartada, err.Error())
		return
	}
	if s, ok := b.extras.solicitudes[r.Solicitud]; ok {
		b.responder(ctx, s)
		return
	}
	s := Solicitud{Solicitud: r.Solicitud, Proyecto: r.Proyecto, AccionID: accionID, Pregunta: id, Dispositivo: r.Dispositivo, Nombre: r.Nombre, Estado: EstadoRecibida, Recibida: b.ahora().UTC()}
	b.recordar(s)
	b.responder(ctx, s)
	b.contestar(ctx, s, r.Respuesta)
}

// contestar aplica la respuesta a la pregunta del proyecto, si este PC la publica y el teléfono no está revocado.
func (b *Bucle) contestar(ctx context.Context, s Solicitud, respuesta string) {
	if LeerRevocados()[strings.ToLower(s.Dispositivo)] {
		b.cerrar(ctx, s, EstadoRechazada, "este teléfono está revocado en el PC")
		return
	}
	p, ok := b.proyectoPublicado(s.Proyecto)
	if !ok {
		b.cerrar(ctx, s, EstadoRechazada, "el PC no publica ese proyecto")
		return
	}
	s.Ruta = p.Ruta
	if _, err := os.Stat(p.Ruta); err != nil {
		b.cerrar(ctx, s, EstadoRechazada, ErrCarpetaNoExiste)
		return
	}
	lugar := s.Nombre
	if lugar == "" {
		lugar = "un teléfono"
	}
	_, err := consola.ResponderPregunta(s.Ruta, s.Pregunta, respuesta, lugar, b.ahora())
	var cerrada *consola.PreguntaCerrada
	if errors.As(err, &cerrada) && cerrada.Pregunta.Estado == consola.PreguntaVencida {
		b.cerrar(ctx, s, EstadoVencida, "la pregunta venció: "+cerrada.Pregunta.Motivo)
		return
	}
	if errors.As(err, &cerrada) {
		s.Lugar = consola.LugarDe(cerrada.Pregunta)
		b.cerrar(ctx, s, EstadoYaAprobado, "ya respondida en "+s.Lugar)
		return
	}
	if errors.Is(err, consola.ErrPreguntaDesconocida) {
		b.cerrar(ctx, s, EstadoVencida, err.Error())
		return
	}
	if errors.Is(err, consola.ErrRespuestaInvalida) {
		b.cerrar(ctx, s, EstadoRechazada, err.Error())
		return
	}
	if err != nil {
		b.cerrar(ctx, s, EstadoFallo, err.Error())
		return
	}
	b.cerrar(ctx, s, EstadoHecha, "respuesta entregada a la consola del PC")
	b.marcar()
}

// seguirPreguntas corrige una respuesta del teléfono que el terminal le ganó: el mod la cerró en el PC después de que
// el teléfono la respondiera (vale lo que vio Claude Code), así que la solicitud pasa a ya_aprobado con lugar PC.
func (b *Bucle) seguirPreguntas(ctx context.Context) {
	for _, s := range b.extras.solicitudes {
		if s.Pregunta == "" || s.Estado != EstadoHecha || s.Ruta == "" || b.ahora().Sub(s.Recibida) > consola.VidaCerrada {
			continue
		}
		p, ok := consola.PreguntaDe(s.Ruta, s.Pregunta, b.ahora())
		if !ok || p.RespondidaEn != consola.EnPC {
			continue
		}
		s.Lugar = LugarPC
		b.cerrar(ctx, s, EstadoYaAprobado, "ya respondida en el PC: la consola respondió antes de recibir la del teléfono")
	}
}

// ocultarPreguntas pasa por el ocultador lo que se publica de cada pregunta (inv. 16): textos, opciones y el detalle
// de un permiso (el comando puede traer un token).
func ocultarPreguntas(root string, panel *consola.DatosConsola) {
	var o *Ocultador
	for i, a := range panel.Acciones {
		if a.Tipo != consola.TipoPregunta {
			continue
		}
		if o == nil {
			o = OcultadorDe(root)
		}
		panel.Acciones[i] = ocultarPregunta(o, a)
	}
}

func ocultarPregunta(o *Ocultador, a consola.Accion) consola.Accion {
	a.Etiqueta, a.Detalle, a.Resultado = o.Ocultar(a.Etiqueta), o.Ocultar(a.Detalle), o.Ocultar(a.Resultado)
	if a.Permiso != nil {
		permiso := *a.Permiso
		permiso.Detalle = o.Ocultar(permiso.Detalle)
		a.Permiso = &permiso
	}
	items := make([]consola.ItemPregunta, len(a.Preguntas))
	for i, q := range a.Preguntas {
		q.Encabezado, q.Texto = o.Ocultar(q.Encabezado), o.Ocultar(q.Texto)
		opciones := make([]consola.OpcionPregunta, len(q.Opciones))
		for j, op := range q.Opciones {
			opciones[j] = consola.OpcionPregunta{Valor: op.Valor, Etiqueta: o.Ocultar(op.Etiqueta), Descripcion: o.Ocultar(op.Descripcion)}
		}
		q.Opciones = opciones
		items[i] = q
	}
	a.Preguntas = items
	return a
}

// verificarFirmada compara la firma con hmac.Equal y exige que en sea de los últimos VigenciaAccion.
func verificarFirmada(esperada, firma, en string, now time.Time) error {
	if !firmaIgual(esperada, firma) {
		return errors.New("firma inválida")
	}
	t, err := time.Parse(formatoEn, en)
	if err != nil {
		return fmt.Errorf("hora inválida: %q", en)
	}
	if d := now.Sub(t); d > VigenciaAccion || d < -VigenciaAccion {
		return fmt.Errorf("vieja (%s)", en)
	}
	return nil
}
