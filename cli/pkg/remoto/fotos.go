package remoto

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/foto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// Fotos en los mensajes a la IA por canal directo (fotos2). La persona eligió en qdd-movile (2026-11-26) que las fotos
// no se guarden en Supabase: el evento foto viejo pasaba por qdd_remoto_foto y realtime.send, que las guardaba de 3 a 4
// días (FND-059 de qdd-movile), y el relay lo cortó el 24-11. Ahora el teléfono las manda en partes por broadcast al
// tópico del PC, sin firma: el id es el sha256 de la foto entera, y lo que la firma cubre es el ejecutar que la nombra.
// El PC las junta en memoria con topes globales (el dispositivo de un broadcast se puede falsificar, así que no van por
// teléfono), verifica huella y tipo y responde foto_ack. Solo un ejecutar firmado que la nombra la escribe en el disco:
// una foto que ninguno nombra nunca lo toca. Al llenarse responde tope y no echa a otra foto.

const (
	// EventoFoto es una parte de una foto; EventoFotoAck, la respuesta del PC cuando la foto queda o se descarta.
	EventoFoto    = "foto2"
	EventoFotoAck = "foto_ack"
	// CapacidadFotos la declara el PC en la cabecera del estado: la app solo ofrece adjuntar si la ve.
	CapacidadFotos = "fotos2"
	// Topes del contrato con qdd-movile.
	MaxBytesFoto      = 2 << 20
	MaxPartesFoto     = 64
	MaxDatosParte     = 65536
	MaxFotosArmando   = 8
	MaxBytesArmando   = 16 << 20
	MaxFotosEjecucion = 4
	// VidaArmado es cuánto espera el PC las partes que faltan de una foto; VidaLista, cuánto guarda en memoria una foto
	// completa a la espera del ejecutar que la nombre.
	VidaArmado = 60 * time.Second
	VidaLista  = 10 * time.Minute
	// CadaPoda es cada cuánto borra qdd remoto las fotos del teléfono de más de foto.VidaTelefono (FND-315).
	CadaPoda = time.Hour
)

// Motivos de un foto_ack con ok false.
const (
	MotivoHuella     = "huella"
	MotivoIncompleta = "incompleta"
	MotivoProyecto   = "proyecto"
	MotivoDisco      = "disco"
	MotivoTipo       = "tipo"
	MotivoTope       = "tope"
	MotivoID         = "id"
)

// IDFoto es el id de una foto: el sha256 hex de la foto entera, en minúscula. Es parte del nombre del archivo.
var IDFoto = regexp.MustCompile(`^[0-9a-f]{64}$`)

// CapacidadesPC es lo que este PC declara en la cabecera del estado.
var CapacidadesPC = []string{CapacidadFotos}

// FotoParte es el evento foto2 que manda el teléfono. Tipo y Bytes vienen solo en la parte 1.
type FotoParte struct {
	Version  int    `json:"version"`
	Proyecto string `json:"proyecto"`
	ID       string `json:"id"`
	Parte    int    `json:"parte"`
	De       int    `json:"de"`
	Datos    string `json:"datos"`
	Tipo     string `json:"tipo,omitempty"`
	Bytes    int    `json:"bytes,omitempty"`
}

// FotoAck es la respuesta del PC a una foto.
type FotoAck struct {
	Version  int    `json:"version"`
	Proyecto string `json:"proyecto"`
	ID       string `json:"id"`
	OK       bool   `json:"ok"`
	Motivo   string `json:"motivo,omitempty"`
}

// fotoEnArmado son las partes que llegaron de una foto.
type fotoEnArmado struct {
	proyecto string
	id       string
	de       int
	bytes    int
	partes   map[int][]byte
	desde    time.Time
}

// fotoLista es una foto completa en memoria, a la espera del ejecutar que la nombre.
type fotoLista struct {
	datos []byte
	en    time.Time
}

// fotoEscrita es una foto que un ejecutar ya escribió: otro ejecutar puede volver a nombrarla.
type fotoEscrita struct {
	ruta string
	en   time.Time
}

// fotoVencida es una foto que no se completó a tiempo.
type fotoVencida struct {
	proyecto string
	id       string
}

// armador junta las partes en memoria. Lo toca solo la goroutine del bucle.
type armador struct {
	armando  map[string]*fotoEnArmado
	listas   map[string]fotoLista
	escritas map[string]fotoEscrita
	ocupados int
}

func claveFoto(proyecto, id string) string { return proyecto + "\x00" + id }

// agregar suma una parte. Devuelve el motivo por el que se descarta, o "" si sigue o quedó lista.
func (a *armador) agregar(p FotoParte, now time.Time) string {
	if !IDFoto.MatchString(p.ID) {
		return MotivoID
	}
	clave := claveFoto(p.Proyecto, p.ID)
	if _, ok := a.listas[clave]; ok {
		return ""
	}
	if p.De < 1 || p.De > MaxPartesFoto || p.Bytes > MaxBytesFoto || len(p.Datos) > MaxDatosParte {
		return MotivoTope
	}
	if p.Parte < 1 || p.Parte > p.De {
		return MotivoIncompleta
	}
	datos, err := base64.StdEncoding.DecodeString(p.Datos)
	if err != nil {
		return MotivoHuella
	}
	f, err := a.armadoPara(p, len(datos), now)
	if err != nil {
		return err.Error()
	}
	if f.de != p.De {
		a.soltarArmado(clave)
		return MotivoHuella
	}
	if _, repetida := f.partes[p.Parte]; !repetida {
		f.partes[p.Parte] = datos
		a.ocupados += len(datos)
	}
	if p.Parte == 1 {
		f.bytes = p.Bytes
	}
	if len(f.partes) < f.de || f.bytes == 0 {
		return ""
	}
	entera, motivo := f.armar()
	a.soltarArmado(clave)
	if motivo != "" {
		return motivo
	}
	a.listas[clave] = fotoLista{datos: entera, en: now}
	a.ocupados += len(entera)
	return ""
}

// armadoPara da la foto en armado de esa parte, o la empieza si caben una foto y sus bytes más.
func (a *armador) armadoPara(p FotoParte, nuevos int, now time.Time) (*fotoEnArmado, error) {
	if a.armando == nil {
		a.armando, a.listas = map[string]*fotoEnArmado{}, map[string]fotoLista{}
	}
	if a.ocupados+nuevos > MaxBytesArmando {
		return nil, errors.New(MotivoTope)
	}
	clave := claveFoto(p.Proyecto, p.ID)
	if f := a.armando[clave]; f != nil {
		return f, nil
	}
	if len(a.armando)+len(a.listas) >= MaxFotosArmando {
		return nil, errors.New(MotivoTope)
	}
	f := &fotoEnArmado{proyecto: p.Proyecto, id: p.ID, de: p.De, partes: map[int][]byte{}, desde: now}
	a.armando[clave] = f
	return f, nil
}

// armar junta las partes en orden y las compara con los bytes, el id y el tipo.
func (f *fotoEnArmado) armar() ([]byte, string) {
	var entera bytes.Buffer
	for i := 1; i <= f.de; i++ {
		entera.Write(f.partes[i])
	}
	if entera.Len() > MaxBytesFoto {
		return nil, MotivoTope
	}
	suma := sha256.Sum256(entera.Bytes())
	if entera.Len() != f.bytes || hex.EncodeToString(suma[:]) != f.id {
		return nil, MotivoHuella
	}
	if _, err := foto.Validar(entera.Bytes()); err != nil {
		return nil, MotivoTipo
	}
	return entera.Bytes(), ""
}

func (a *armador) soltarArmado(clave string) {
	for _, d := range a.armando[clave].partes {
		a.ocupados -= len(d)
	}
	delete(a.armando, clave)
}

func (a *armador) soltarLista(clave string) {
	a.ocupados -= len(a.listas[clave].datos)
	delete(a.listas, clave)
}

// lista dice si la foto está completa en memoria.
func (a *armador) lista(proyecto, id string) bool {
	_, ok := a.listas[claveFoto(proyecto, id)]
	return ok
}

// vencidas suelta las fotos que llevan más de VidaArmado sin completarse (las devuelve, para avisar), las completas que
// llevan más de VidaLista sin ejecutar y olvida las escritas que ya se borraron del disco (foto.VidaTelefono).
func (a *armador) vencidas(now time.Time) []fotoVencida {
	for clave, f := range a.escritas {
		if now.Sub(f.en) > foto.VidaTelefono {
			delete(a.escritas, clave)
		}
	}
	for clave, f := range a.listas {
		if now.Sub(f.en) > VidaLista {
			a.soltarLista(clave)
		}
	}
	var out []fotoVencida
	for clave, f := range a.armando {
		if now.Sub(f.desde) <= VidaArmado {
			continue
		}
		a.soltarArmado(clave)
		out = append(out, fotoVencida{proyecto: f.proyecto, id: f.id})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

// rutasDe escribe en el proyecto las fotos que nombra un ejecutar firmado y da sus rutas. Si falta una, no escribe
// ninguna: dice cuál falta.
func (a *armador) rutasDe(root, proyecto string, ids []string, now time.Time) ([]string, error) {
	for _, id := range ids {
		if a.lista(proyecto, id) {
			continue
		}
		if _, err := os.Stat(a.escritas[claveFoto(proyecto, id)].ruta); err != nil {
			return nil, fmt.Errorf("falta la foto %s: no llegó a este PC para este proyecto, o ya no está", id)
		}
	}
	var rutas []string
	for _, id := range ids {
		ruta, err := a.escribir(root, proyecto, id, now)
		if err != nil {
			return nil, fmt.Errorf("foto %s: %v", id, err)
		}
		rutas = append(rutas, ruta)
	}
	return rutas, nil
}

// escribir deja en el disco una foto lista (o da la ruta de una ya escrita) y la suelta de la memoria.
func (a *armador) escribir(root, proyecto, id string, now time.Time) (string, error) {
	clave := claveFoto(proyecto, id)
	if e, ok := a.escritas[clave]; ok && !a.lista(proyecto, id) {
		return e.ruta, nil
	}
	ruta, err := foto.GuardarDelTelefono(root, id, a.listas[clave].datos, now)
	if err != nil {
		return "", err
	}
	a.soltarLista(clave)
	if a.escritas == nil {
		a.escritas = map[string]fotoEscrita{}
	}
	a.escritas[clave] = fotoEscrita{ruta: ruta, en: now}
	return ruta, nil
}

// ValidarFotos vuelve a hacer en el PC lo que exige el relay: solo una instrucción para la IA lleva fotos, hasta
// MaxFotosEjecucion, distintas y nombradas por su sha256.
func ValidarFotos(tipo string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if tipo != consola.EjecutarIA {
		return errors.New("solo una instrucción para la IA (ia) lleva fotos")
	}
	if len(ids) > MaxFotosEjecucion {
		return fmt.Errorf("una instrucción lleva hasta %d fotos", MaxFotosEjecucion)
	}
	vistos := map[string]bool{}
	for _, id := range ids {
		if !IDFoto.MatchString(id) || vistos[id] {
			return fmt.Errorf("foto «%s»: id inválido o repetido", id)
		}
		vistos[id] = true
	}
	return nil
}

// alFoto recibe una parte y, si completa la foto, responde foto_ack. La foto queda en memoria hasta que la nombre un
// ejecutar firmado.
func (b *Bucle) alFoto(ctx context.Context, payload json.RawMessage) {
	var p FotoParte
	if err := json.Unmarshal(payload, &p); err != nil {
		b.logf("parte de foto descartada: ilegible")
		return
	}
	if !b.plan.Pro() || p.Version != VersionProtocolo {
		b.logf("foto %.12s descartada: sin Pro o versión %d", p.ID, p.Version)
		return
	}
	if !IDFoto.MatchString(p.ID) {
		b.ackFoto(ctx, p.Proyecto, p.ID, MotivoID)
		return
	}
	if _, ok := b.proyectoPublicado(p.Proyecto); !ok {
		b.ackFoto(ctx, p.Proyecto, p.ID, MotivoProyecto)
		return
	}
	priv, _ := b.Cred.privadaCifrado()
	p, motivo := abrirParte(priv, p)
	if motivo != "" {
		b.ackFoto(ctx, p.Proyecto, p.ID, motivo)
		return
	}
	if motivo := b.fotos.agregar(p, b.ahora()); motivo != "" {
		b.ackFoto(ctx, p.Proyecto, p.ID, motivo)
		return
	}
	if !b.fotos.lista(p.Proyecto, p.ID) {
		return
	}
	b.logf("foto %.12s lista en memoria", p.ID)
	b.ackFoto(ctx, p.Proyecto, p.ID, "")
}

// descartarVencidas responde incompleta por cada foto que no se completó a tiempo.
func (b *Bucle) descartarVencidas(ctx context.Context) {
	for _, v := range b.fotos.vencidas(b.ahora()) {
		b.ackFoto(ctx, v.proyecto, v.id, MotivoIncompleta)
	}
}

// ackFoto responde una foto: ok sin motivo, o el motivo por el que se descartó.
func (b *Bucle) ackFoto(ctx context.Context, proyecto, id, motivo string) {
	if motivo != "" {
		b.logf("foto %.12s descartada: %s", id, motivo)
	}
	if len(b.telefonos) == 0 {
		return
	}
	b.broadcast(ctx, EventoFotoAck, FotoAck{Version: VersionProtocolo, Proyecto: proyecto, ID: id, OK: motivo == "", Motivo: motivo})
}

// podarFotos borra, al arrancar y cada CadaPoda, las fotos del teléfono que ya cumplieron foto.VidaTelefono en todos los
// proyectos del registro (FND-315): sin esperar a que llegue otra foto ni a que haya un teléfono.
func (b *Bucle) podarFotos() {
	now := b.ahora()
	if !b.podadoEn.IsZero() && now.Sub(b.podadoEn) < CadaPoda {
		return
	}
	b.podadoEn = now
	reg, err := registro.Leer()
	if err != nil {
		return
	}
	for _, p := range reg.Proyectos {
		for _, nombre := range foto.PodarDelTelefono(p.Ruta, now) {
			b.logf("foto del teléfono borrada: %s de %s (más de 7 días)", nombre, registro.NombreDe(p.Ruta))
		}
	}
}
