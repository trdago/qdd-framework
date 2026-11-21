// Package enlaces publica la página de un flujo en un Worker de Cloudflare de la persona (RFC 0025, fase 3b): quien la
// recibe abre un enlace en vez de un archivo. La cuenta es de la persona y su token sale del baúl; qdd no guarda nada
// en la nube que no sea la página, y el token de la invitación nunca va en ella: viaja en el fragmento del enlace.
package enlaces

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

const (
	// NombreWorker y NombreEspacio son el Worker y el KV que qdd crea en la cuenta de la persona.
	NombreWorker  = "qdd-flujos"
	NombreEspacio = "qdd-flujos"

	// APICloudflare es la API v4; las pruebas la cambian por un servidor de mentira.
	APICloudflare = "https://api.cloudflare.com/client/v4"

	// MaxPagina es el tope de una página: KV acepta hasta 25 MiB y un flujo pesa mucho menos.
	MaxPagina = 5 << 20

	fechaDeCompatibilidad = "2026-01-01"
	tope                  = 45 * time.Second
)

//go:embed worker.js
var ScriptWorker string

var idValido = regexp.MustCompile(`^[a-z2-7]{26}$`)

// Cuenta es cómo hablar con Cloudflare. CuentaID elige una si el token ve varias.
type Cuenta struct {
	API      string
	Token    string
	CuentaID string
	HTTP     *http.Client
}

// Sitio es el Worker listo: su cuenta, su KV y la dirección donde responde.
type Sitio struct {
	Cuenta  string
	Espacio string
	Base    string
}

// NuevoID es el id de una página: 128 bits al azar en base32 minúscula, imposible de adivinar.
func NuevoID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("no hay azar para el id del enlace: %w", err)
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)), nil
}

// Preparar deja el Worker al día en la cuenta: busca o crea el KV, sube el script con su binding y lo publica en
// workers.dev. Es idempotente; correrlo cada vez mantiene el Worker en la versión de este qdd.
func (c Cuenta) Preparar(ctx context.Context) (Sitio, error) {
	cuenta, err := c.cuenta(ctx)
	if err != nil {
		return Sitio{}, err
	}
	var sub struct {
		Subdomain string `json:"subdomain"`
	}
	if err := c.llamar(ctx, "GET", "/accounts/"+cuenta+"/workers/subdomain", nil, "", &sub); err != nil || sub.Subdomain == "" {
		return Sitio{}, fmt.Errorf("la cuenta %s no tiene subdominio workers.dev: actívalo una vez en el panel de Cloudflare (Workers y Pages)", cuenta)
	}
	espacio, err := c.espacio(ctx, cuenta)
	if err != nil {
		return Sitio{}, err
	}
	if err := c.subirWorker(ctx, cuenta, espacio); err != nil {
		return Sitio{}, err
	}
	activar, _ := json.Marshal(map[string]bool{"enabled": true})
	if err := c.llamar(ctx, "POST", "/accounts/"+cuenta+"/workers/scripts/"+NombreWorker+"/subdomain", activar, "application/json", nil); err != nil {
		return Sitio{}, err
	}
	return Sitio{Cuenta: cuenta, Espacio: espacio, Base: "https://" + NombreWorker + "." + sub.Subdomain + ".workers.dev"}, nil
}

// Subir guarda la página bajo su id y devuelve su enlace (sin fragmento).
func (c Cuenta) Subir(ctx context.Context, s Sitio, id string, pagina []byte) (string, error) {
	if !idValido.MatchString(id) {
		return "", fmt.Errorf("id de página %q: lo genera qdd", id)
	}
	if len(pagina) > MaxPagina {
		return "", fmt.Errorf("la página pesa %d bytes: el tope es %d", len(pagina), MaxPagina)
	}
	if err := c.llamar(ctx, "PUT", c.valor(s, id), pagina, "text/html; charset=utf-8", nil); err != nil {
		return "", err
	}
	return s.Base + "/f/" + id, nil
}

// Quitar borra la página: el enlace deja de abrir.
func (c Cuenta) Quitar(ctx context.Context, s Sitio, id string) error {
	if !idValido.MatchString(id) {
		return fmt.Errorf("id de página %q: lo genera qdd", id)
	}
	return c.llamar(ctx, "DELETE", c.valor(s, id), nil, "", nil)
}

func (c Cuenta) valor(s Sitio, id string) string {
	return "/accounts/" + s.Cuenta + "/storage/kv/namespaces/" + s.Espacio + "/values/" + url.PathEscape("f/"+id)
}

func (c Cuenta) cuenta(ctx context.Context) (string, error) {
	var cuentas []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.llamar(ctx, "GET", "/accounts", nil, "", &cuentas); err != nil {
		return "", err
	}
	if len(cuentas) == 0 {
		return "", errors.New("el token de Cloudflare no ve ninguna cuenta: dale permiso de Workers Scripts y Workers KV")
	}
	var nombres []string
	for _, x := range cuentas {
		if c.CuentaID == x.ID || (c.CuentaID == "" && len(cuentas) == 1) {
			return x.ID, nil
		}
		nombres = append(nombres, x.ID+" ("+x.Name+")")
	}
	if c.CuentaID != "" {
		return "", fmt.Errorf("el token no ve la cuenta %s; ve %s", c.CuentaID, strings.Join(nombres, ", "))
	}
	return "", fmt.Errorf("el token ve %d cuentas: elige una con --cuenta (%s)", len(cuentas), strings.Join(nombres, ", "))
}

func (c Cuenta) espacio(ctx context.Context, cuenta string) (string, error) {
	var espacios []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := c.llamar(ctx, "GET", "/accounts/"+cuenta+"/storage/kv/namespaces?per_page=100", nil, "", &espacios); err != nil {
		return "", err
	}
	for _, e := range espacios {
		if e.Title == NombreEspacio {
			return e.ID, nil
		}
	}
	cuerpo, _ := json.Marshal(map[string]string{"title": NombreEspacio})
	var nuevo struct {
		ID string `json:"id"`
	}
	if err := c.llamar(ctx, "POST", "/accounts/"+cuenta+"/storage/kv/namespaces", cuerpo, "application/json", &nuevo); err != nil {
		return "", err
	}
	return nuevo.ID, nil
}

func (c Cuenta) subirWorker(ctx context.Context, cuenta, espacio string) error {
	var cuerpo bytes.Buffer
	partes := multipart.NewWriter(&cuerpo)
	metadatos, _ := json.Marshal(map[string]any{
		"main_module":        "worker.js",
		"compatibility_date": fechaDeCompatibilidad,
		"bindings":           []map[string]string{{"type": "kv_namespace", "name": "FLUJOS", "namespace_id": espacio}},
	})
	if err := partes.WriteField("metadata", string(metadatos)); err != nil {
		return err
	}
	cabecera := textproto.MIMEHeader{}
	cabecera.Set("Content-Disposition", `form-data; name="worker.js"; filename="worker.js"`)
	cabecera.Set("Content-Type", "application/javascript+module")
	parte, err := partes.CreatePart(cabecera)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(parte, ScriptWorker); err != nil {
		return err
	}
	if err := partes.Close(); err != nil {
		return err
	}
	return c.llamar(ctx, "PUT", "/accounts/"+cuenta+"/workers/scripts/"+NombreWorker, cuerpo.Bytes(), partes.FormDataContentType(), nil)
}

// llamar hace una llamada a la API y lee su sobre {success, errors, result}. El token nunca sale en un error.
func (c Cuenta) llamar(ctx context.Context, metodo, camino string, cuerpo []byte, tipo string, resultado any) error {
	ctx, cancel := context.WithTimeout(ctx, tope)
	defer cancel()
	base := c.API
	if base == "" {
		base = APICloudflare
	}
	req, err := http.NewRequestWithContext(ctx, metodo, base+camino, bytes.NewReader(cuerpo))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if tipo != "" {
		req.Header.Set("Content-Type", tipo)
	}
	cliente := c.HTTP
	if cliente == nil {
		cliente = &http.Client{Timeout: tope}
	}
	resp, err := cliente.Do(req)
	if err != nil {
		return fmt.Errorf("Cloudflare no respondió: %w", err)
	}
	defer resp.Body.Close()
	var sobre struct {
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&sobre); err != nil {
		return fmt.Errorf("Cloudflare respondió %d sin JSON a %s %s", resp.StatusCode, metodo, strings.SplitN(camino, "?", 2)[0])
	}
	if !sobre.Success || resp.StatusCode >= 300 {
		var mensajes []string
		for _, e := range sobre.Errors {
			mensajes = append(mensajes, fmt.Sprintf("%s (%d)", e.Message, e.Code))
		}
		return fmt.Errorf("Cloudflare respondió %d a %s %s: %s", resp.StatusCode, metodo, strings.SplitN(camino, "?", 2)[0], strings.Join(mensajes, "; "))
	}
	if resultado == nil || len(sobre.Result) == 0 {
		return nil
	}
	return json.Unmarshal(sobre.Result, resultado)
}
