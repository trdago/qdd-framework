package remoto

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto/cifrado"
)

// Rule: Prohibido usar 'else'
// RFC 0027 fase 3 (acordado con qdd-movile, 2026-11-27): el PC guarda su identidad Ed25519 y su X25519 en
// credenciales.json (0600, junto a la firma de la máquina), publica la X25519 firmada en la cabecera del estado con la
// capacidad cifrado1 y abre lo que el teléfono selló: el texto de un ejecutar, la respuesta de un responder, el
// accion_id de una accion y los datos de una parte de foto2. Abre después de verificar la firma del relay (que firma
// el sellado tal cual) y antes de validar: las reglas del texto en claro las vuelve a aplicar el PC, porque el relay
// ya no las ve. Lo que no abre nunca se ejecuta ni se recuerda: responde rechazada con el motivo clave, y la app
// reconfirma la clave desde el estado y ofrece reenviar. Lo que llega en claro sigue valiendo hasta la fase 5.

const (
	// MotivoClave: el PC no pudo abrir lo sellado (la app usó una clave vieja, o el sellado es de otro mensaje).
	MotivoClave = "clave"
	// MaxParteAbierta es lo más que mide una parte de foto2 abierta (48 KB); MaxDatosParteSellada, esa parte sellada.
	MaxParteAbierta      = 48 << 10
	MaxDatosParteSellada = 65603
	// avisoClave es lo que ve la persona en el teléfono.
	avisoClave = "Tu PC cambió su clave; vuelve a mandarlo"
)

// GenerarCifrado da al PC su identidad (si no tiene) y su X25519 (si no tiene, o siempre con rotar: al volver a
// emparejar). La identidad no cambia mientras el PC no se borre.
func GenerarCifrado(c *Credenciales, rotar bool, now time.Time) error {
	if c.Identidad == "" {
		semilla, err := aleatorio(ed25519.SeedSize)
		if err != nil {
			return err
		}
		c.Identidad = base64.RawURLEncoding.EncodeToString(semilla)
	}
	if c.Cifrado != "" && !rotar {
		return nil
	}
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("no se pudo generar la clave de cifrado: %w", err)
	}
	c.Cifrado = base64.RawURLEncoding.EncodeToString(priv.Bytes())
	c.CifradoDesde = now.UTC().Truncate(time.Second)
	return nil
}

func (c Credenciales) identidad() (ed25519.PrivateKey, error) {
	semilla, err := base64.RawURLEncoding.DecodeString(c.Identidad)
	if err != nil || len(semilla) != ed25519.SeedSize {
		return nil, errors.New("identidad del PC ilegible")
	}
	return ed25519.NewKeyFromSeed(semilla), nil
}

func (c Credenciales) privadaCifrado() (*ecdh.PrivateKey, error) {
	b, err := base64.RawURLEncoding.DecodeString(c.Cifrado)
	if err != nil {
		return nil, errors.New("clave de cifrado del PC ilegible")
	}
	return ecdh.X25519().NewPrivateKey(b)
}

// IdentidadPublica es la identidad que va en el QR (base64url sin relleno), o "" si el PC no tiene.
func (c Credenciales) IdentidadPublica() string {
	id, err := c.identidad()
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(id.Public().(ed25519.PublicKey))
}

// HuellaIdentidad es la huella corta que muestran la app y qdd remoto dispositivos, o "" si el PC no tiene identidad.
func (c Credenciales) HuellaIdentidad() string {
	id, err := c.identidad()
	if err != nil {
		return ""
	}
	return cifrado.Huella(id.Public().(ed25519.PublicKey))
}

// ClaveCifrado es la cabecera clave del estado: la X25519 firmada por la identidad. Nunca lleva una privada.
func (c Credenciales) ClaveCifrado() (*cifrado.Clave, error) {
	id, err := c.identidad()
	if err != nil {
		return nil, err
	}
	priv, err := c.privadaCifrado()
	if err != nil {
		return nil, err
	}
	clave := cifrado.FirmarClave(id, priv.PublicKey().Bytes(), c.CifradoDesde)
	return &clave, nil
}

// conCifrado suma la clave y la capacidad cifrado1 al estado; un PC sin claves legibles no ofrece cifrado.
func (b *Bucle) conCifrado(e *Estado) {
	clave, err := b.Cred.ClaveCifrado()
	if err != nil {
		return
	}
	e.Clave = clave
	e.Capacidades = append(slices.Clone(e.Capacidades), cifrado.Capacidad)
}

// abrir devuelve el campo en claro: tal cual si no viene sellado, abierto si viene sellado, o ErrNoAbre.
func (b *Bucle) abrir(campo string, aad []byte) (string, error) {
	if !cifrado.EsSellado(campo) {
		return campo, nil
	}
	priv, err := b.Cred.privadaCifrado()
	if err != nil {
		return "", cifrado.ErrNoAbre
	}
	claro, err := cifrado.Abrir(priv, aad, campo)
	if err != nil {
		return "", err
	}
	return string(claro), nil
}

// abrirRespuesta abre una respuesta sellada, que el relay manda como el string JSON "c1.…" (firmado así); lo abierto
// es el texto JSON del objeto de siempre.
func (b *Bucle) abrirRespuesta(r RespuestaFirmada) (string, error) {
	if !strings.HasPrefix(r.Respuesta, `"`+cifrado.Prefijo) {
		return r.Respuesta, nil
	}
	var sellado string
	if err := json.Unmarshal([]byte(r.Respuesta), &sellado); err != nil {
		return "", cifrado.ErrNoAbre
	}
	return b.abrir(sellado, cifrado.AADResponder(r.Solicitud, r.Proyecto, r.Pregunta))
}

// abrirParte devuelve la parte con sus datos en claro (base64, como llegan sin sellar), o el motivo del foto_ack.
func abrirParte(priv *ecdh.PrivateKey, p FotoParte) (FotoParte, string) {
	if !cifrado.EsSellado(p.Datos) {
		return p, ""
	}
	if len(p.Datos) > MaxDatosParteSellada {
		return p, MotivoTope
	}
	if priv == nil {
		return p, MotivoClave
	}
	claro, err := cifrado.Abrir(priv, cifrado.AADFoto(p.Proyecto, p.ID, p.Parte, p.De), p.Datos)
	if err != nil {
		return p, MotivoClave
	}
	if len(claro) > MaxParteAbierta {
		return p, MotivoTope
	}
	p.Datos = base64.StdEncoding.EncodeToString(claro)
	return p, ""
}

// rechazarClave responde que no se pudo abrir, sin recordar la solicitud: no se ejecutó nada y la app puede reenviarla
// sellada hacia la clave nueva.
func (b *Bucle) rechazarClave(ctx context.Context, s Solicitud) {
	s.Estado, s.Motivo, s.Resultado = EstadoRechazada, MotivoClave, avisoClave
	b.logf("solicitud %s rechazada: no abre con la clave de este PC", s.Solicitud)
	b.anotar(AccionFirmada{Solicitud: s.Solicitud, Proyecto: s.Proyecto, AccionID: s.AccionID, Dispositivo: s.Dispositivo, DispositivoNombre: s.Nombre}, s.Estado, MotivoClave)
	b.responder(ctx, s)
}

// AsegurarCifrado lee las credenciales y, si a un PC emparejado antes del RFC 0027 le faltan las claves, las genera y
// las guarda; si ya las tiene, no escribe nada.
func AsegurarCifrado(now time.Time) (Credenciales, error) {
	c, err := LeerCredenciales()
	if err != nil {
		return c, err
	}
	if c.Identidad != "" && c.Cifrado != "" {
		return c, nil
	}
	if err := GenerarCifrado(&c, false, now); err != nil {
		return c, err
	}
	return c, GuardarCredenciales(c)
}
