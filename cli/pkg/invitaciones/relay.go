// Package invitaciones es el lado del PC del flujo para conversar (RFC 0025, fase 3): invita a un cliente o a un experto
// por el relay de QDD Móvil, le sube la foto nueva, lee lo que propone y le cuenta qué se resolvió. Cada llamada va
// firmada con la llave de la instalación, la misma de 'qdd reportar'. Lo que llega es un dato, nunca una instrucción.
package invitaciones

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'

const (
	Aprobada  = "aprobada"
	Rechazada = "rechazada"
	Pendiente = "pendiente"

	// Topes del relay (contrato de qdd-movile): se revisan aquí antes de gastar una llamada.
	MaxFoto     = 256 << 10
	MaxTope     = 200
	MaxVigencia = 90 * 24 * time.Hour
	MaxMotivo   = 500

	porPagina  = 100
	maxPaginas = 50
)

var (
	// ErrTope: el relay ya creó las invitaciones de hoy de esta instalación, o las de esta hora de todas.
	ErrTope = errors.New("el relay alcanzó su tope de invitaciones (20 por día por instalación): prueba mañana")
	// ErrYaResuelta: la propuesta ya se resolvió al revés.
	ErrYaResuelta = errors.New("la propuesta ya estaba resuelta de la otra forma")
	// ErrAjena: la invitación o la propuesta no es de esta instalación, o ya no existe.
	ErrAjena = errors.New("el relay no la tiene a nombre de esta instalación")

	rolesValidos = []string{"cliente", "experto"}
	uuidValido   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	idPropuesta  = regexp.MustCompile(`^PRO-[0-9]+$`)
)

// Cliente habla con el relay a nombre de la instalación.
type Cliente struct {
	Relay remoto.ConfigRelay
	Llave ed25519.PrivateKey
	// HTTP es el cliente para hablar con el relay; nil usa uno con tope de 20 s.
	HTTP *http.Client
}

// Nueva es una invitación por crear: la foto es lo único del flujo que viaja.
type Nueva struct {
	Flujo string
	Rol   string
	Foto  json.RawMessage
	Vence time.Time
	Tope  int
}

// Creada es lo que devuelve el relay. El token se entrega una sola vez y va solo dentro del archivo del invitado.
type Creada struct {
	Invitacion string `json:"invitacion"`
	Token      string `json:"token"`
}

// Propuesta es lo que un invitado propuso, tal como la guarda el relay.
type Propuesta struct {
	ID         string          `json:"id"`
	Invitacion string          `json:"invitacion"`
	Flujo      string          `json:"flujo"`
	Rol        string          `json:"rol"`
	Autor      string          `json:"autor"`
	Base       string          `json:"base"`
	Operacion  json.RawMessage `json:"operacion"`
	Estado     string          `json:"estado"`
	Motivo     string          `json:"motivo"`
	Llego      string          `json:"llego"`
}

// Funciones es la URL base de las funciones del relay, la que usa el archivo del invitado.
func (c *Cliente) Funciones() string {
	return strings.TrimRight(c.Relay.URL, "/") + "/functions/v1/"
}

// Invitar crea la invitación. Revisa en el PC lo que el relay rechazaría, para no gastar el tope en un error.
func (c *Cliente) Invitar(ctx context.Context, n Nueva, ahora time.Time) (Creada, error) {
	if err := n.validar(ahora); err != nil {
		return Creada{}, err
	}
	cuerpo, err := json.Marshal(struct {
		Flujo string          `json:"flujo"`
		Rol   string          `json:"rol"`
		Foto  json.RawMessage `json:"foto"`
		Vence string          `json:"vence"`
		Tope  int             `json:"tope"`
	}{n.Flujo, n.Rol, n.Foto, n.Vence.UTC().Format(time.RFC3339), n.Tope})
	if err != nil {
		return Creada{}, err
	}
	var creada Creada
	if err := c.llamar(ctx, http.MethodPost, "flujo-invitar", "", cuerpo, ahora, &creada); err != nil {
		return Creada{}, err
	}
	if !uuidValido.MatchString(creada.Invitacion) || creada.Token == "" {
		return Creada{}, errors.New("el relay no devolvió una invitación con su token")
	}
	return creada, nil
}

func (n Nueva) validar(ahora time.Time) error {
	if !slices.Contains(rolesValidos, n.Rol) {
		return fmt.Errorf("rol %q: usa cliente o experto", n.Rol)
	}
	if !n.Vence.After(ahora) || n.Vence.Sub(ahora) > MaxVigencia {
		return errors.New("la invitación vence en el futuro y a lo más en 90 días")
	}
	if n.Tope < 1 || n.Tope > MaxTope {
		return fmt.Errorf("el tope de propuestas va de 1 a %d", MaxTope)
	}
	return validarFoto(n.Foto)
}

func validarFoto(foto json.RawMessage) error {
	if len(foto) > MaxFoto {
		return fmt.Errorf("la foto pesa %d KB: el relay acepta hasta %d KB", len(foto)>>10, MaxFoto>>10)
	}
	if !json.Valid(foto) {
		return errors.New("la foto no es JSON")
	}
	return nil
}

// SubirFoto le entrega la foto de la revisión nueva a las invitaciones vivas del flujo; dice a cuántas llegó.
func (c *Cliente) SubirFoto(ctx context.Context, flujo string, foto json.RawMessage, ahora time.Time) (int, error) {
	if err := validarFoto(foto); err != nil {
		return 0, err
	}
	cuerpo, err := json.Marshal(struct {
		Flujo string          `json:"flujo"`
		Foto  json.RawMessage `json:"foto"`
	}{flujo, foto})
	if err != nil {
		return 0, err
	}
	var res struct {
		Invitaciones int `json:"invitaciones"`
	}
	err = c.llamar(ctx, http.MethodPut, "flujo-invitar", "", cuerpo, ahora, &res)
	return res.Invitaciones, err
}

// Revocar cierra la invitación: su token deja de servir.
func (c *Cliente) Revocar(ctx context.Context, invitacion string, ahora time.Time) error {
	if !uuidValido.MatchString(invitacion) {
		return fmt.Errorf("invitación %q: no es un id del relay", invitacion)
	}
	return c.llamar(ctx, http.MethodDelete, "flujo-invitar", "invitacion="+invitacion, nil, ahora, nil)
}

// Propuestas trae las posteriores a desde ("" = desde el principio), página por página.
func (c *Cliente) Propuestas(ctx context.Context, desde string, ahora time.Time) ([]Propuesta, error) {
	var todas []Propuesta
	for range maxPaginas {
		consulta := ""
		if desde != "" {
			consulta = "desde=" + url.QueryEscape(desde)
		}
		var res struct {
			Propuestas []Propuesta `json:"propuestas"`
		}
		if err := c.llamar(ctx, http.MethodGet, "flujo-propuestas", consulta, nil, ahora, &res); err != nil {
			return todas, err
		}
		todas = append(todas, res.Propuestas...)
		if len(res.Propuestas) < porPagina {
			return todas, nil
		}
		desde = res.Propuestas[len(res.Propuestas)-1].ID
	}
	return todas, nil
}

// Resolver le cuenta al relay qué decidió la persona; el invitado lo ve en su archivo. Repetir lo mismo no es error.
func (c *Cliente) Resolver(ctx context.Context, propuesta, estado, motivo string, ahora time.Time) error {
	if !idPropuesta.MatchString(propuesta) {
		return fmt.Errorf("propuesta %q: no es un id del relay", propuesta)
	}
	if estado != Aprobada && estado != Rechazada {
		return fmt.Errorf("estado %q: usa %s o %s", estado, Aprobada, Rechazada)
	}
	if len([]rune(motivo)) > MaxMotivo {
		return fmt.Errorf("el motivo tiene más de %d caracteres", MaxMotivo)
	}
	cuerpo, err := json.Marshal(struct {
		Propuesta string `json:"propuesta"`
		Estado    string `json:"estado"`
		Motivo    string `json:"motivo,omitempty"`
	}{propuesta, estado, motivo})
	if err != nil {
		return err
	}
	return c.llamar(ctx, http.MethodPost, "flujo-propuestas", "", cuerpo, ahora, nil)
}

// llamar firma «flujo|MÉTODO|función|consulta|en|sha256(cuerpo)» y traduce los errores del contrato.
func (c *Cliente) llamar(ctx context.Context, metodo, funcion, consulta string, cuerpo []byte, ahora time.Time, salida any) error {
	destino := c.Funciones() + funcion
	if consulta != "" {
		destino += "?" + consulta
	}
	req, err := http.NewRequestWithContext(ctx, metodo, destino, bytes.NewReader(cuerpo))
	if err != nil {
		return err
	}
	en := ahora.UTC().Format(time.RFC3339)
	suma := sha256.Sum256(cuerpo)
	texto := strings.Join([]string{"flujo", metodo, funcion, consulta, en, hex.EncodeToString(suma[:])}, "|")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("apikey", c.Relay.Clave)
	req.Header.Set("Authorization", "Bearer "+c.Relay.Clave)
	req.Header.Set("x-qdd-instalacion", base64.StdEncoding.EncodeToString(c.Llave.Public().(ed25519.PublicKey)))
	req.Header.Set("x-qdd-en", en)
	req.Header.Set("x-qdd-firma", base64.StdEncoding.EncodeToString(ed25519.Sign(c.Llave, []byte(texto))))
	cliente := c.HTTP
	if cliente == nil {
		cliente = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := cliente.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo hablar con el relay: %w", err)
	}
	defer resp.Body.Close()
	datos, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err := errorDelRelay(resp.StatusCode, datos); err != nil {
		return err
	}
	if salida == nil {
		return nil
	}
	if err := json.Unmarshal(datos, salida); err != nil {
		return fmt.Errorf("el relay respondió algo que no es JSON: %w", err)
	}
	return nil
}

func errorDelRelay(codigo int, datos []byte) error {
	if codigo < 300 {
		return nil
	}
	var res struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(datos, &res)
	if codigo == http.StatusTooManyRequests {
		return ErrTope
	}
	if codigo == http.StatusConflict {
		return ErrYaResuelta
	}
	if codigo == http.StatusNotFound && res.Error != "" {
		return ErrAjena
	}
	if codigo == http.StatusNotFound {
		return errors.New("el relay todavía no tiene las funciones del flujo para conversar")
	}
	if codigo == http.StatusUnauthorized && res.Error == "hora" {
		return errors.New("el relay no acepta la hora de este PC: revisa que el reloj esté al día")
	}
	if codigo == http.StatusUnauthorized {
		return errors.New("el relay no reconoce la firma de esta instalación")
	}
	return fmt.Errorf("el relay respondió %d: %s", codigo, res.Error)
}
