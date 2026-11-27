// Package cifrado es la construcción del RFC 0027: el teléfono cifra hacia el PC el contenido de lo que manda, y el
// relay (Supabase) lo ve pasar y lo guarda sin poder leerlo.
//
// Sellar es ECIES de una sola vía, la de HPKE en modo base: una X25519 efímera por mensaje, HKDF-SHA256 y
// AES-256-GCM. Todo sale de la biblioteca estándar. Los bytes exactos los fija el fixture compartido con qdd-movile
// (pkg/remoto/testdata/cifrado1.json).
package cifrado

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

const (
	// Prefijo marca un campo sellado: «c1.» y la base64url sin relleno de efímera (32 bytes) ‖ cifrado ‖ etiqueta (16).
	Prefijo = "c1."
	// Capacidad la declara el PC en la cabecera del estado: sin ella la app no le manda nada cifrado.
	Capacidad = "cifrado1"
	// info separa estas claves de cualquier otro uso de HKDF.
	info = "qdd cifrado1"
	// contextoClave es lo que firma la identidad junto a la X25519 y su fecha.
	contextoClave = "qdd cifrado1 clave"
	formatoDesde  = "2006-01-02T15:04:05Z"
	largoClave    = 32
	largoNonce    = 12
	largoEtiqueta = 16
)

var (
	// ErrNoAbre: el campo no está sellado, está mal formado, se selló hacia otra clave o no es de este mensaje. El PC lo
	// rechaza con el motivo clave y nunca lo ejecuta.
	ErrNoAbre = errors.New("no se pudo abrir: sellado hacia otra clave o para otro mensaje")
	// ErrClaveFirma: la X25519 no viene firmada por la identidad que dice.
	ErrClaveFirma = errors.New("la clave de cifrado no está firmada por la identidad del PC")
)

var b64 = base64.RawURLEncoding

// EsSellado dice si un campo viene sellado (empieza con Prefijo).
func EsSellado(campo string) bool {
	return strings.HasPrefix(campo, Prefijo)
}

// Sellar cifra claro hacia la X25519 pública del PC, atado a aad. azar nil usa crypto/rand; las pruebas y el fixture
// pasan una efímera fija.
func Sellar(azar io.Reader, publica, aad, claro []byte) (string, error) {
	pc, err := ecdh.X25519().NewPublicKey(publica)
	if err != nil {
		return "", fmt.Errorf("clave pública del PC inválida: %w", err)
	}
	if azar == nil {
		azar = rand.Reader
	}
	semilla := make([]byte, largoClave)
	if _, err := io.ReadFull(azar, semilla); err != nil {
		return "", fmt.Errorf("no se pudo generar la clave efímera: %w", err)
	}
	efimera, err := ecdh.X25519().NewPrivateKey(semilla)
	if err != nil {
		return "", err
	}
	compartido, err := efimera.ECDH(pc)
	if err != nil {
		return "", err
	}
	aead, nonce, err := derivar(compartido, efimera.PublicKey().Bytes(), publica)
	if err != nil {
		return "", err
	}
	sobre := append([]byte{}, efimera.PublicKey().Bytes()...)
	sobre = aead.Seal(sobre, nonce, claro, aad)
	return Prefijo + b64.EncodeToString(sobre), nil
}

// Abrir descifra un campo sellado con la privada del PC. Cualquier falla es ErrNoAbre.
func Abrir(privada *ecdh.PrivateKey, aad []byte, sellado string) ([]byte, error) {
	if !EsSellado(sellado) {
		return nil, ErrNoAbre
	}
	sobre, err := b64.DecodeString(strings.TrimPrefix(sellado, Prefijo))
	if err != nil || len(sobre) < largoClave+largoEtiqueta {
		return nil, ErrNoAbre
	}
	efimera, err := ecdh.X25519().NewPublicKey(sobre[:largoClave])
	if err != nil {
		return nil, ErrNoAbre
	}
	compartido, err := privada.ECDH(efimera)
	if err != nil {
		return nil, ErrNoAbre
	}
	aead, nonce, err := derivar(compartido, sobre[:largoClave], privada.PublicKey().Bytes())
	if err != nil {
		return nil, ErrNoAbre
	}
	claro, err := aead.Open(nil, nonce, sobre[largoClave:], aad)
	if err != nil {
		return nil, ErrNoAbre
	}
	return claro, nil
}

// derivar saca la clave AES y el nonce del secreto compartido: HKDF-SHA256 con sal efímera ‖ pública del PC.
func derivar(compartido, efimera, publica []byte) (cipher.AEAD, []byte, error) {
	sal := append(append([]byte{}, efimera...), publica...)
	material, err := hkdf.Key(sha256.New, compartido, sal, info, largoClave+largoNonce)
	if err != nil {
		return nil, nil, err
	}
	bloque, err := aes.NewCipher(material[:largoClave])
	if err != nil {
		return nil, nil, err
	}
	aead, err := cipher.NewGCM(bloque)
	if err != nil {
		return nil, nil, err
	}
	return aead, material[largoClave:], nil
}

// Datos asociados de cada campo: lo que el relay ve en claro y ata el sellado a su mensaje. Las partes van unidas por
// «|», como las firmas del relay.

// AADEjecutar es el de texto en un ejecutar.
func AADEjecutar(solicitud, proyecto, tipo string) []byte {
	return unir("ejecutar", solicitud, proyecto, tipo)
}

// AADResponder es el de respuesta en un responder.
func AADResponder(solicitud, proyecto, pregunta string) []byte {
	return unir("responder", solicitud, proyecto, pregunta)
}

// AADAccion es el de accion_id en una accion.
func AADAccion(solicitud, proyecto string) []byte {
	return unir("accion", solicitud, proyecto)
}

// AADFoto es el de datos en una parte de foto2.
func AADFoto(proyecto, id string, parte, de int) []byte {
	return unir("foto2", proyecto, id, strconv.Itoa(parte), strconv.Itoa(de))
}

func unir(partes ...string) []byte {
	return []byte(info + "|" + strings.Join(partes, "|"))
}

// Clave es la cabecera clave del estado y de la oferta de emparejamiento: la X25519 del PC firmada por su identidad.
type Clave struct {
	Identidad string `json:"identidad"`
	X25519    string `json:"x25519"`
	Firma     string `json:"firma"`
	Desde     string `json:"desde"`
}

// FirmarClave arma la cabecera: la identidad firma «qdd cifrado1 clave|<x25519>|<desde>», en base64url sin relleno.
func FirmarClave(identidad ed25519.PrivateKey, x25519 []byte, desde time.Time) Clave {
	c := Clave{
		Identidad: b64.EncodeToString(identidad.Public().(ed25519.PublicKey)),
		X25519:    b64.EncodeToString(x25519),
		Desde:     desde.UTC().Format(formatoDesde),
	}
	c.Firma = b64.EncodeToString(ed25519.Sign(identidad, mensajeClave(c)))
	return c
}

// VerificarClave devuelve la X25519 si su firma es de la identidad que trae. Que esa identidad sea la del QR (o la
// primera vista) lo decide la app.
func VerificarClave(c Clave) ([]byte, error) {
	identidad, err := b64.DecodeString(c.Identidad)
	if err != nil || len(identidad) != ed25519.PublicKeySize {
		return nil, ErrClaveFirma
	}
	x, err := b64.DecodeString(c.X25519)
	if err != nil || len(x) != largoClave {
		return nil, ErrClaveFirma
	}
	firma, err := b64.DecodeString(c.Firma)
	if err != nil || !ed25519.Verify(identidad, mensajeClave(c), firma) {
		return nil, ErrClaveFirma
	}
	return x, nil
}

func mensajeClave(c Clave) []byte {
	return []byte(contextoClave + "|" + c.X25519 + "|" + c.Desde)
}

// Huella es la huella corta de una identidad que muestran la app y qdd remoto dispositivos: los 8 primeros bytes de su
// sha256 en hex mayúscula, en cuatro grupos de cuatro.
func Huella(identidad ed25519.PublicKey) string {
	suma := sha256.Sum256(identidad)
	h := strings.ToUpper(hex.EncodeToString(suma[:8]))
	return h[0:4] + " " + h[4:8] + " " + h[8:12] + " " + h[12:16]
}
