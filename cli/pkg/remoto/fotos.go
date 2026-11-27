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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/foto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// Fotos en los mensajes a la IA (pedido de la persona desde QDD Móvil; ADR-0003 de qdd-movile, contrato acordado el
// 2026-11-23). El teléfono manda cada foto en partes de 48 KB por qdd_remoto_foto: el relay exige Pro y reenvía el
// evento foto firmado sobre id|proyecto|parte|de|huella|dispositivo|en (los datos no van firmados: los cubre la huella,
// SHA-256 de la foto entera). El PC las junta en memoria por teléfono e id, verifica bytes, huella y que sea una imagen,
// la guarda con pkg/foto y responde foto_ack. Una ejecución ia nombra hasta 4 fotos del mismo teléfono y proyecto.

const (
	// EventoFoto es una parte de una foto; EventoFotoAck, la respuesta del PC cuando la foto queda o se descarta.
	EventoFoto    = "foto"
	EventoFotoAck = "foto_ack"
	// CapacidadFotos la declara el PC en la cabecera del estado: la app solo ofrece adjuntar si la ve.
	CapacidadFotos = "fotos"
	// Topes del contrato (los mismos del relay).
	MaxBytesFoto          = 2 << 20
	MaxPartesFoto         = 64
	MaxDatosParte         = 65536
	MaxArmandoPorTelefono = 4
	MaxFotosEjecucion     = 4
	// VidaArmado es cuánto espera el PC las partes que faltan de una foto.
	VidaArmado = 60 * time.Second
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

// CapacidadesPC es lo que este PC declara en la cabecera del estado. Sin fotos mientras el relay las tenga cortadas
// (FND-059 de qdd-movile, desde el 2026-11-24): la app ofrecería adjuntar y el relay respondería 42501 (FND-333). El
// transporte nuevo, cuando se decida, va con otro nombre de capacidad (fotos2).
var CapacidadesPC = []string{}

// FotoParte es el evento foto que reenvía el relay. Tipo y Bytes vienen solo en la parte 1.
type FotoParte struct {
	Version     int    `json:"version"`
	Proyecto    string `json:"proyecto"`
	ID          string `json:"id"`
	Parte       int    `json:"parte"`
	De          int    `json:"de"`
	Huella      string `json:"huella"`
	Datos       string `json:"datos"`
	Tipo        string `json:"tipo,omitempty"`
	Bytes       int    `json:"bytes,omitempty"`
	Dispositivo string `json:"dispositivo"`
	En          string `json:"en"`
	Firma       string `json:"firma"`
}

// FotoAck es la respuesta del PC a una foto.
type FotoAck struct {
	Version  int    `json:"version"`
	Proyecto string `json:"proyecto"`
	ID       string `json:"id"`
	OK       bool   `json:"ok"`
	Motivo   string `json:"motivo,omitempty"`
}

// FirmaFoto es la firma que pone el relay a cada parte (la misma clave que FirmaAccion).
func FirmaFoto(firmaMaquina, id, proyecto string, parte, de int, huella, dispositivo, en string) string {
	return firmar(firmaMaquina, id, proyecto, strconv.Itoa(parte), strconv.Itoa(de), huella, dispositivo, en)
}

// Verificar exige la firma de la máquina y que la parte sea de los últimos VigenciaAccion.
func (p FotoParte) Verificar(firmaMaquina string, now time.Time) error {
	if p.ID == "" || p.Proyecto == "" || p.Huella == "" || p.Dispositivo == "" || p.En == "" {
		return errors.New("parte de foto incompleta")
	}
	esperada := FirmaFoto(firmaMaquina, p.ID, p.Proyecto, p.Parte, p.De, p.Huella, p.Dispositivo, p.En)
	return verificarFirmada(esperada, p.Firma, p.En, now)
}

// fotoEnArmado son las partes que llegaron de una foto.
type fotoEnArmado struct {
	id          string
	dispositivo string
	proyecto    string
	de          int
	huella      string
	bytes       int
	partes      map[int][]byte
	desde       time.Time
}

// fotoLista es una foto ya guardada: de qué proyecto es y dónde quedó.
type fotoLista struct {
	proyecto string
	ruta     string
	en       time.Time
}

// fotoVencida es una foto que no se completó a tiempo.
type fotoVencida struct {
	proyecto string
	id       string
}

// armador junta las partes por teléfono e id. Lo toca solo la goroutine del bucle.
type armador struct {
	armando map[string]*fotoEnArmado
	listas  map[string]fotoLista
}

func claveFoto(dispositivo, id string) string { return dispositivo + "\x00" + id }

// agregar suma una parte: devuelve la foto entera cuando está completa, o el motivo por el que se descarta.
func (a *armador) agregar(p FotoParte, now time.Time) ([]byte, bool, string) {
	if p.De < 1 || p.De > MaxPartesFoto || p.Bytes > MaxBytesFoto || len(p.Datos) > MaxDatosParte {
		return nil, false, MotivoTope
	}
	if p.Parte < 1 || p.Parte > p.De {
		return nil, false, MotivoIncompleta
	}
	datos, err := base64.StdEncoding.DecodeString(p.Datos)
	if err != nil {
		return nil, false, MotivoHuella
	}
	if a.armando == nil {
		a.armando = map[string]*fotoEnArmado{}
	}
	clave := claveFoto(p.Dispositivo, p.ID)
	f := a.armando[clave]
	if f == nil && a.enArmado(p.Dispositivo) >= MaxArmandoPorTelefono {
		return nil, false, MotivoTope
	}
	if f == nil {
		f = &fotoEnArmado{id: p.ID, dispositivo: p.Dispositivo, proyecto: p.Proyecto, de: p.De, huella: p.Huella, partes: map[int][]byte{}, desde: now}
		a.armando[clave] = f
	}
	if f.proyecto != p.Proyecto || f.de != p.De || f.huella != p.Huella {
		delete(a.armando, clave)
		return nil, false, MotivoHuella
	}
	f.partes[p.Parte] = datos
	if p.Parte == 1 {
		f.bytes = p.Bytes
	}
	if len(f.partes) < f.de || f.bytes == 0 {
		return nil, false, ""
	}
	delete(a.armando, clave)
	return f.armar()
}

// armar junta las partes en orden y las compara con bytes y huella.
func (f *fotoEnArmado) armar() ([]byte, bool, string) {
	var entera bytes.Buffer
	for i := 1; i <= f.de; i++ {
		entera.Write(f.partes[i])
	}
	if entera.Len() > MaxBytesFoto {
		return nil, false, MotivoTope
	}
	suma := sha256.Sum256(entera.Bytes())
	if entera.Len() != f.bytes || hex.EncodeToString(suma[:]) != f.huella {
		return nil, false, MotivoHuella
	}
	return entera.Bytes(), true, ""
}

// enArmado dice cuántas fotos arma un teléfono.
func (a *armador) enArmado(dispositivo string) int {
	n := 0
	for _, f := range a.armando {
		if f.dispositivo == dispositivo {
			n++
		}
	}
	return n
}

// vencidas saca las fotos que llevan más de VidaArmado sin completarse, y olvida las guardadas que ya se borraron del
// disco (foto.VidaTelefono).
func (a *armador) vencidas(now time.Time) []fotoVencida {
	for clave, f := range a.listas {
		if now.Sub(f.en) > foto.VidaTelefono {
			delete(a.listas, clave)
		}
	}
	var out []fotoVencida
	for clave, f := range a.armando {
		if now.Sub(f.desde) <= VidaArmado {
			continue
		}
		delete(a.armando, clave)
		out = append(out, fotoVencida{proyecto: f.proyecto, id: f.id})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

// guardar recuerda una foto ya guardada.
func (a *armador) guardar(dispositivo, id string, f fotoLista) {
	if a.listas == nil {
		a.listas = map[string]fotoLista{}
	}
	a.listas[claveFoto(dispositivo, id)] = f
}

// rutasDe da la ruta de cada foto que nombra una ejecución: del mismo teléfono y proyecto, y todavía en el disco.
func (a *armador) rutasDe(dispositivo, proyecto string, ids []string) ([]string, error) {
	var rutas []string
	for _, id := range ids {
		f, ok := a.listas[claveFoto(dispositivo, id)]
		if !ok || f.proyecto != proyecto {
			return nil, fmt.Errorf("falta la foto %s: no llegó a este PC desde este teléfono y proyecto", id)
		}
		if _, err := os.Stat(f.ruta); err != nil {
			return nil, fmt.Errorf("falta la foto %s: ya no está en el PC", id)
		}
		rutas = append(rutas, f.ruta)
	}
	return rutas, nil
}

// ValidarFotos vuelve a hacer en el PC lo que exige el relay: solo una instrucción para la IA lleva fotos, hasta
// MaxFotosEjecucion, distintas y con id válido.
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
		if !foto.IDTelefono.MatchString(id) || vistos[id] {
			return fmt.Errorf("foto «%s»: id inválido o repetido", id)
		}
		vistos[id] = true
	}
	return nil
}

// alFoto recibe una parte: la verifica como una acción y, si completa la foto, la guarda y responde foto_ack.
func (b *Bucle) alFoto(ctx context.Context, payload json.RawMessage) {
	var p FotoParte
	if err := json.Unmarshal(payload, &p); err != nil {
		b.logf("parte de foto descartada: ilegible")
		return
	}
	if err := p.Verificar(b.Cred.Firma, b.ahora()); err != nil {
		b.logf("foto %s descartada: %v", p.ID, err)
		return
	}
	if LeerRevocados()[strings.ToLower(p.Dispositivo)] || !b.plan.Pro() || p.Version != VersionProtocolo {
		b.logf("foto %s descartada: teléfono revocado, sin Pro o versión %d", p.ID, p.Version)
		return
	}
	if !foto.IDTelefono.MatchString(p.ID) {
		b.ackFoto(ctx, p.Proyecto, p.ID, MotivoID)
		return
	}
	proyecto, ok := b.proyectoPublicado(p.Proyecto)
	if !ok {
		b.ackFoto(ctx, p.Proyecto, p.ID, MotivoProyecto)
		return
	}
	datos, completa, motivo := b.fotos.agregar(p, b.ahora())
	if motivo != "" {
		b.ackFoto(ctx, p.Proyecto, p.ID, motivo)
		return
	}
	if !completa {
		return
	}
	if _, err := foto.Validar(datos); err != nil {
		b.ackFoto(ctx, p.Proyecto, p.ID, MotivoTipo)
		return
	}
	ruta, err := foto.GuardarDelTelefono(proyecto.Ruta, p.ID, datos, b.ahora())
	if err != nil {
		b.logf("foto %s: %v", p.ID, err)
		b.ackFoto(ctx, p.Proyecto, p.ID, MotivoDisco)
		return
	}
	b.fotos.guardar(p.Dispositivo, p.ID, fotoLista{proyecto: p.Proyecto, ruta: ruta, en: b.ahora()})
	b.logf("foto %s guardada (%d bytes)", p.ID, len(datos))
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
		b.logf("foto %s descartada: %s", id, motivo)
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
