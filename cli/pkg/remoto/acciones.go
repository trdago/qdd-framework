package remoto

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.3): acciones firmadas. El teléfono llama a qdd_remoto_accion; el relay sella el dispositivo, verifica que
// no esté revocado y la reenvía al canal con firma = hex(hmac_sha256(firma de la máquina, solicitud|proyecto|
// accion_id|dispositivo|en)). El PC descarta lo que no traiga una firma válida o sea viejo (inv. 18), vuelve a calcular
// el panel y ejecuta solo un id que ofrece (inv. 1), una sola vez por solicitud (inv. 3), y responde recibida →
// ejecutando → hecha | fallo, o ya_aprobado o rechazada (inv. 4, 19). Todo queda en <QDD_HOME>/remoto/log.jsonl (inv. 5).

const (
	// VigenciaAccion: una acción firmada hace más que esto (o en el futuro) se descarta.
	VigenciaAccion = 5 * time.Minute
	// maxResultado es el tope del resultado que se le manda al teléfono (ya sin secretos).
	maxResultado = 600
	// vidaSolicitud es cuánto se recuerda una solicitud para no ejecutarla dos veces.
	vidaSolicitud  = 7 * 24 * time.Hour
	maxSolicitudes = 1000
	formatoEn      = "2006-01-02T15:04:05Z"
	prefijoSesion  = "remoto:"
	// ventanaReenvio y maxReenvio acotan lo que un sync reenvía de las solicitudes (FND-111): lo que un teléfono puede
	// estar esperando todavía, sin volver a mandar el historial entero.
	ventanaReenvio = 30 * time.Minute
	maxReenvio     = 20
	// prefijoRechazar + el id de la acción de un pedido lo descarta de la cola sin ejecutarlo (el «Rechazar» del
	// teléfono). No es una acción del panel: el PC la acepta solo si el panel ofrece ese pedido.
	prefijoRechazar = "rechazar:"

	EstadoRecibida   = "recibida"
	EstadoEjecutando = "ejecutando"
	EstadoHecha      = "hecha"
	EstadoFallo      = "fallo"
	EstadoYaAprobado = "ya_aprobado"
	EstadoRechazada  = "rechazada"
	// EstadoDescartada solo va al log: una acción sin firma válida no recibe respuesta.
	EstadoDescartada = "descartada"
	// LugarPC es donde se aprobó un pedido que no aprobó un teléfono.
	LugarPC = "PC"
)

// AccionFirmada es el evento accion que reenvía el relay.
type AccionFirmada struct {
	Version           int    `json:"version"`
	Solicitud         string `json:"solicitud"`
	Proyecto          string `json:"proyecto"`
	AccionID          string `json:"accion_id"`
	Dispositivo       string `json:"dispositivo"`
	DispositivoNombre string `json:"dispositivo_nombre"`
	En                string `json:"en"`
	Firma             string `json:"firma"`
}

// FirmaAccion es la firma que pone el relay: la clave es el texto hex de la firma de la máquina (sus bytes UTF-8).
func FirmaAccion(firmaMaquina, solicitud, proyecto, accionID, dispositivo, en string) string {
	return firmar(firmaMaquina, solicitud, proyecto, accionID, dispositivo, en)
}

// firmar es hex(hmac_sha256(firma de la máquina, partes unidas por «|»)), como pgcrypto en el relay.
func firmar(firmaMaquina string, partes ...string) string {
	m := hmac.New(sha256.New, []byte(firmaMaquina))
	m.Write([]byte(strings.Join(partes, "|")))
	return hex.EncodeToString(m.Sum(nil))
}

// firmaIgual compara en tiempo constante (la firma puede venir en mayúsculas).
func firmaIgual(esperada, firma string) bool {
	return hmac.Equal([]byte(esperada), []byte(strings.ToLower(firma)))
}

// Verificar exige la firma de la máquina y que la acción sea de los últimos VigenciaAccion.
func (a AccionFirmada) Verificar(firmaMaquina string, now time.Time) error {
	if a.Solicitud == "" || a.Proyecto == "" || a.AccionID == "" || a.Dispositivo == "" || a.En == "" {
		return errors.New("acción incompleta")
	}
	esperada := FirmaAccion(firmaMaquina, a.Solicitud, a.Proyecto, a.AccionID, a.Dispositivo, a.En)
	if !firmaIgual(esperada, a.Firma) {
		return errors.New("firma inválida")
	}
	en, err := time.Parse(formatoEn, a.En)
	if err != nil {
		return fmt.Errorf("hora inválida: %q", a.En)
	}
	if d := now.Sub(en); d > VigenciaAccion || d < -VigenciaAccion {
		return fmt.Errorf("acción vieja (%s)", a.En)
	}
	return nil
}

// AccionEstado es la respuesta del PC a una acción.
type AccionEstado struct {
	Version   int    `json:"version"`
	Solicitud string `json:"solicitud"`
	Proyecto  string `json:"proyecto,omitempty"`
	AccionID  string `json:"accion_id,omitempty"`
	Estado    string `json:"estado"`
	Resultado string `json:"resultado,omitempty"`
	Lugar     string `json:"lugar,omitempty"`
}

// Solicitud es lo que el PC recuerda de una acción: su último estado y, si es un pedido, cuál.
type Solicitud struct {
	Solicitud   string    `json:"solicitud"`
	Proyecto    string    `json:"proyecto"`
	AccionID    string    `json:"accion_id"`
	Dispositivo string    `json:"dispositivo"`
	Nombre      string    `json:"dispositivo_nombre"`
	Estado      string    `json:"estado"`
	Resultado   string    `json:"resultado,omitempty"`
	Lugar       string    `json:"lugar,omitempty"`
	Ruta        string    `json:"ruta,omitempty"`
	Pedido      string    `json:"pedido,omitempty"`
	Pregunta    string    `json:"pregunta,omitempty"` // una respuesta a una pregunta de la consola (preguntas.go)
	Ejecutar    string    `json:"ejecutar,omitempty"` // lo mandado a ejecutar desde el teléfono: qdd o ia (ejecutar.go)
	Recibida    time.Time `json:"recibida"`
}

func (s Solicitud) respuesta() AccionEstado {
	return AccionEstado{Version: VersionProtocolo, Solicitud: s.Solicitud, Proyecto: s.Proyecto, AccionID: s.AccionID, Estado: s.Estado, Resultado: s.Resultado, Lugar: s.Lugar}
}

func (s Solicitud) terminada() bool {
	return s.Estado != EstadoRecibida && s.Estado != EstadoEjecutando
}

// leerSolicitudes carga las solicitudes recordadas; un archivo ilegible se aparta (no se pierde) y se empieza de cero.
func leerSolicitudes() map[string]Solicitud {
	out := map[string]Solicitud{}
	var lista []Solicitud
	err := leerPrivado("solicitudes.json", &lista)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		if ruta, e := rutaRemoto("solicitudes.json"); e == nil {
			_ = os.Rename(ruta, ruta+".ilegible")
		}
	}
	for _, s := range lista {
		out[s.Solicitud] = s
	}
	return out
}

// guardarSolicitudes deja las más recientes (hasta maxSolicitudes, de los últimos vidaSolicitud).
func guardarSolicitudes(m map[string]Solicitud, now time.Time) error {
	var lista []Solicitud
	for _, s := range m {
		if now.Sub(s.Recibida) <= vidaSolicitud {
			lista = append(lista, s)
		}
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Recibida.After(lista[j].Recibida) })
	if len(lista) > maxSolicitudes {
		lista = lista[:maxSolicitudes]
	}
	return guardarPrivado("solicitudes.json", lista)
}

// EntradaLog es una línea de <QDD_HOME>/remoto/log.jsonl: quién, qué, cuándo y cómo terminó.
type EntradaLog struct {
	En          time.Time `json:"en"`
	Solicitud   string    `json:"solicitud,omitempty"`
	Dispositivo string    `json:"dispositivo,omitempty"`
	Nombre      string    `json:"dispositivo_nombre,omitempty"`
	Proyecto    string    `json:"proyecto,omitempty"`
	AccionID    string    `json:"accion_id,omitempty"`
	Estado      string    `json:"estado"`
	Resultado   string    `json:"resultado,omitempty"`
}

// Anotar agrega una línea al log local (0600). Nunca va al relay.
func Anotar(e EntradaLog) error {
	ruta, err := rutaRemoto("log.jsonl")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	e.Resultado = Recortar(Ocultar(e.Resultado), maxResultado)
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(ruta, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// Revocados es <QDD_HOME>/remoto/revocados.json: los teléfonos que este PC ya no escucha, aunque el relay los deje.
type Revocados struct {
	Formato      int      `json:"formato"`
	Dispositivos []string `json:"dispositivos"`
}

// LeerRevocados devuelve los teléfonos revocados en este PC.
func LeerRevocados() map[string]bool {
	var r Revocados
	_ = leerPrivado("revocados.json", &r)
	out := map[string]bool{}
	for _, d := range r.Dispositivos {
		out[strings.ToLower(d)] = true
	}
	return out
}

// Revocar agrega un teléfono a revocados.json: su próxima acción se rechaza en el acto.
func Revocar(dispositivo string) error {
	r := LeerRevocados()
	r[strings.ToLower(strings.TrimSpace(dispositivo))] = true
	lista := Revocados{Formato: 1}
	for d := range r {
		lista.Dispositivos = append(lista.Dispositivos, d)
	}
	sort.Strings(lista.Dispositivos)
	return guardarPrivado("revocados.json", lista)
}

// resultadoAccion es lo que vuelve de una acción que corrió en segundo plano.
type resultadoAccion struct {
	solicitud string
	codigo    int
	salida    string
}

// alAccion recibe una acción del relay: la verifica, la recuerda antes de ejecutarla y la ejecuta como el panel.
func (b *Bucle) alAccion(ctx context.Context, payload json.RawMessage) {
	var a AccionFirmada
	if err := json.Unmarshal(payload, &a); err != nil {
		b.logf("acción descartada: ilegible")
		return
	}
	now := b.ahora()
	if err := a.Verificar(b.Cred.Firma, now); err != nil {
		b.logf("acción %s descartada: %v", a.Solicitud, err)
		b.anotar(a, EstadoDescartada, err.Error())
		return
	}
	if s, ok := b.extras.solicitudes[a.Solicitud]; ok {
		b.responder(ctx, s)
		return
	}
	s := Solicitud{Solicitud: a.Solicitud, Proyecto: a.Proyecto, AccionID: a.AccionID, Dispositivo: a.Dispositivo, Nombre: a.DispositivoNombre, Estado: EstadoRecibida, Recibida: now.UTC()}
	b.recordar(s)
	b.responder(ctx, s)
	b.ejecutar(ctx, s)
}

func (b *Bucle) anotar(a AccionFirmada, estado, resultado string) {
	e := EntradaLog{En: b.ahora().UTC(), Solicitud: a.Solicitud, Dispositivo: a.Dispositivo, Nombre: a.DispositivoNombre, Proyecto: a.Proyecto, AccionID: a.AccionID, Estado: estado, Resultado: resultado}
	if err := Anotar(e); err != nil {
		b.logf("log: %v", err)
	}
}

// recordar guarda la solicitud antes de seguir: si qdd se cae a mitad, al volver no la ejecuta de nuevo.
func (b *Bucle) recordar(s Solicitud) {
	b.extras.solicitudes[s.Solicitud] = s
	if err := guardarSolicitudes(b.extras.solicitudes, b.ahora()); err != nil {
		b.logf("solicitudes: %v", err)
	}
	if s.terminada() {
		b.anotar(AccionFirmada{Solicitud: s.Solicitud, Proyecto: s.Proyecto, AccionID: s.AccionID, Dispositivo: s.Dispositivo, DispositivoNombre: s.Nombre}, s.Estado, s.Resultado)
	}
}

// responder manda el estado de la solicitud a los teléfonos (sin teléfonos, nada: inv. 12; el teléfono reintenta con
// el mismo id y recibe este estado).
func (b *Bucle) responder(ctx context.Context, s Solicitud) {
	if len(b.telefonos) == 0 {
		return
	}
	b.broadcast(ctx, "accion_estado", s.respuesta())
}

// reenviarSolicitudes manda, al pedir sync un teléfono, cómo van las solicitudes recientes: una que terminó mientras
// no había teléfonos no le habría llegado nunca (FND-111). Solo las de la última media hora, como mucho 20.
func (b *Bucle) reenviarSolicitudes(ctx context.Context) {
	if len(b.telefonos) == 0 {
		return
	}
	var recientes []Solicitud
	for _, s := range b.extras.solicitudes {
		if b.ahora().Sub(s.Recibida) <= ventanaReenvio {
			recientes = append(recientes, s)
		}
	}
	sort.Slice(recientes, func(i, j int) bool { return recientes[i].Recibida.After(recientes[j].Recibida) })
	if len(recientes) > maxReenvio {
		recientes = recientes[:maxReenvio]
	}
	for _, s := range recientes {
		b.broadcast(ctx, "accion_estado", s.respuesta())
	}
}

func (b *Bucle) cerrar(ctx context.Context, s Solicitud, estado, resultado string) {
	s.Estado, s.Resultado = estado, Recortar(b.ocultador(s.Ruta).Ocultar(strings.TrimSpace(resultado)), maxResultado)
	b.recordar(s)
	b.responder(ctx, s)
}

// proyectoPublicado es el proyecto del registro con esa clave, si está dentro del plan.
func (b *Bucle) proyectoPublicado(clave string) (registro.Proyecto, bool) {
	reg, err := registro.Leer()
	if err != nil {
		return registro.Proyecto{}, false
	}
	for _, p := range Publicables(reg, b.plan.LimiteProyectos()) {
		if p.Clave == clave {
			return p, true
		}
	}
	return registro.Proyecto{}, false
}

// ejecutar decide qué hacer con una acción verificada y nueva.
func (b *Bucle) ejecutar(ctx context.Context, s Solicitud) {
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
	if pedido, ok := strings.CutPrefix(s.AccionID, prefijoRechazar); ok {
		b.rechazarPedido(ctx, s, pedido)
		return
	}
	accion, err := consola.BuscarAccion(consola.Panel(p.Ruta, b.ahora()).Acciones, s.AccionID)
	if err != nil {
		b.cerrar(ctx, s, EstadoRechazada, "el panel ya no ofrece esa acción")
		return
	}
	if accion.Tipo == consola.TipoAbrir || accion.Tipo == consola.TipoAtender {
		b.cerrar(ctx, s, EstadoRechazada, "esa acción solo tiene sentido en el PC")
		return
	}
	if accion.Tipo == consola.TipoPedido {
		b.lanzarPedido(ctx, s, accion)
		return
	}
	s.Estado = EstadoEjecutando
	b.recordar(s)
	b.responder(ctx, s)
	go func() {
		var salida bytes.Buffer
		codigo := accion.Ejecutar(p.Ruta, b.Bin, &salida)
		b.extras.resultados <- resultadoAccion{solicitud: s.Solicitud, codigo: codigo, salida: salida.String()}
	}()
}

// lanzarPedido corre un pedido como el panel (aparte, con la sesión remoto:<dispositivo>). Uno ya aprobado en otro
// lugar responde ya_aprobado con el lugar y no se ejecuta otra vez (FND-070, inv. 19).
func (b *Bucle) lanzarPedido(ctx context.Context, s Solicitud, accion consola.Accion) {
	s.Pedido = strings.TrimPrefix(accion.ID, "pedido:")
	if p, ok := pedidoDe(s.Ruta, s.Pedido); ok && !p.Ejecutable() {
		b.yaAprobado(ctx, s, p)
		return
	}
	var salida bytes.Buffer
	codigo := accion.Lanzar(s.Ruta, b.Bin, prefijoSesion+s.Dispositivo, &salida)
	if p, ok := pedidoDe(s.Ruta, s.Pedido); codigo != 0 && ok && !p.Ejecutable() && p.Sesion != prefijoSesion+s.Dispositivo {
		b.yaAprobado(ctx, s, p)
		return
	}
	if codigo != 0 {
		b.cerrar(ctx, s, EstadoFallo, salida.String())
		return
	}
	s.Estado, s.Resultado = EstadoEjecutando, Recortar(Ocultar(strings.TrimSpace(salida.String())), maxResultado)
	b.recordar(s)
	b.responder(ctx, s)
}

// rechazarPedido descarta de la cola un pedido que el panel ofrece y nadie aprobó todavía. Uno ya aprobado responde
// ya_aprobado con el lugar, como al aprobarlo (inv. 19). Una instrucción mandada desde el teléfono (telefono:<solicitud>)
// se retira si todavía espera una sesión (ejecutar.go).
func (b *Bucle) rechazarPedido(ctx context.Context, s Solicitud, accionID string) {
	if solicitud, ok := strings.CutPrefix(accionID, "telefono:"); ok {
		b.retirarInstruccion(ctx, s, solicitud)
		return
	}
	accion, err := consola.BuscarAccion(consola.Panel(s.Ruta, b.ahora()).Acciones, accionID)
	if err != nil || accion.Tipo != consola.TipoPedido {
		b.cerrar(ctx, s, EstadoRechazada, "el panel ya no ofrece ese pedido")
		return
	}
	s.Pedido = strings.TrimPrefix(accion.ID, "pedido:")
	if p, ok := pedidoDe(s.Ruta, s.Pedido); ok && !p.Ejecutable() {
		b.yaAprobado(ctx, s, p)
		return
	}
	if err := consola.Descartar(s.Ruta, s.Pedido); err != nil {
		b.cerrar(ctx, s, EstadoFallo, err.Error())
		return
	}
	b.cerrar(ctx, s, EstadoHecha, "pedido rechazado")
}

func pedidoDe(root, id string) (consola.Pedido, bool) {
	for _, p := range consola.Pedidos(root) {
		if p.ID == id {
			return p, true
		}
	}
	return consola.Pedido{}, false
}

func (b *Bucle) yaAprobado(ctx context.Context, s Solicitud, p consola.Pedido) {
	s.Lugar = b.lugarDe(p.Sesion)
	b.cerrar(ctx, s, EstadoYaAprobado, "ya aprobado en "+s.Lugar+" ("+p.Estado+")")
}

// lugarDe dice dónde se aprobó un pedido: el nombre del teléfono si fue desde uno, «PC» si no.
func (b *Bucle) lugarDe(sesion string) string {
	disp, ok := strings.CutPrefix(sesion, prefijoSesion)
	if !ok {
		return LugarPC
	}
	for _, s := range b.extras.solicitudes {
		if s.Dispositivo == disp && s.Nombre != "" {
			return s.Nombre
		}
	}
	return "otro teléfono"
}

// alResultado cierra una acción que corrió en segundo plano.
func (b *Bucle) alResultado(ctx context.Context, r resultadoAccion) {
	s, ok := b.extras.solicitudes[r.solicitud]
	if !ok {
		return
	}
	if s.Ejecutar != "" {
		b.terminarComando(ctx, s, r)
		return
	}
	estado := EstadoHecha
	if r.codigo != 0 {
		estado = EstadoFallo
	}
	// FND-118: de una salida va el final (el veredicto de audit está al final), ya sin secretos en toda ella.
	s.Estado, s.Resultado = estado, recortarFinal(b.ocultador(s.Ruta).Ocultar(strings.TrimSpace(r.salida)), maxResultado)
	b.recordar(s)
	b.responder(ctx, s)
	b.marcar()
}

// seguirSolicitudes cierra los pedidos aprobados desde el teléfono que ya terminaron: responde hecha o fallo, pide el
// push de cómo terminó y los marca avisados (así salen del tablero como uno que el panel ya avisó).
func (b *Bucle) seguirSolicitudes(ctx context.Context, e Estado) {
	for _, s := range b.extras.solicitudes {
		if s.Estado != EstadoEjecutando || s.Pedido == "" {
			continue
		}
		p, ok := pedidoDe(s.Ruta, s.Pedido)
		if !ok {
			b.cerrar(ctx, s, EstadoFallo, "el pedido ya no está en la cola")
			continue
		}
		if p.Estado == consola.EstadoPendiente || p.Estado == consola.EstadoRetirado {
			b.cerrar(ctx, s, EstadoFallo, "volvió a quedar pendiente en el panel: "+p.Aviso)
			continue
		}
		if p.Estado != consola.EstadoHecho && p.Estado != consola.EstadoFallo {
			continue
		}
		estado, tipo := EstadoHecha, AvisoTerminoBien
		if p.Estado == consola.EstadoFallo {
			estado, tipo = EstadoFallo, AvisoTerminoFalla
		}
		_ = consola.Avisado(s.Ruta, s.Pedido, p.Sesion)
		b.cerrar(ctx, s, estado, p.Salida)
		b.avisar(ctx, tipo, filepath.Base(s.Ruta), s.Proyecto)
	}
}

// ocultador es el de los .env del proyecto, releído cada minuto como mucho.
func (b *Bucle) ocultador(root string) *Ocultador {
	if root == "" {
		return nil
	}
	c, ok := b.extras.ocultadores[root]
	if ok && b.ahora().Sub(c.leido) < time.Minute {
		return c.o
	}
	o := OcultadorDe(root)
	b.extras.ocultadores[root] = ocultadorLeido{o: o, leido: b.ahora()}
	return o
}

type ocultadorLeido struct {
	o     *Ocultador
	leido time.Time
}
