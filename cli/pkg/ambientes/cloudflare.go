package ambientes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

// APICloudflare es la API v4 de Cloudflare.
const APICloudflare = "https://api.cloudflare.com/client/v4"

const topeCloudflare = 30 * time.Second

// LectorDNS lee el DNS de una zona de Cloudflare. Solo hace GET: armar el plan nunca cambia nada.
type LectorDNS struct {
	API   string
	Token string
	HTTP  *http.Client
}

// Registros devuelve los registros de la zona; una zona que el token no ve es un error que la nombra.
func (l LectorDNS) Registros(ctx context.Context, zona string) ([]Registro, error) {
	var zonas []struct {
		ID string `json:"id"`
	}
	if err := l.leer(ctx, "/zones?name="+url.QueryEscape(zona), &zonas); err != nil {
		return nil, err
	}
	if len(zonas) == 0 {
		return nil, fmt.Errorf("Cloudflare no muestra la zona %s a este token: revisa que tenga Zone:DNS:Read en ella", zona)
	}
	var crudos []struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		Proxied bool   `json:"proxied"`
	}
	if err := l.leer(ctx, "/zones/"+url.PathEscape(zonas[0].ID)+"/dns_records?per_page=5000", &crudos); err != nil {
		return nil, err
	}
	out := make([]Registro, 0, len(crudos))
	for _, c := range crudos {
		out = append(out, Registro{Tipo: c.Type, Nombre: c.Name, Valor: c.Content, Proxied: c.Proxied})
	}
	return out, nil
}

// leer hace un GET y lee el sobre {success, errors, result}. El token nunca sale en un error.
func (l LectorDNS) leer(ctx context.Context, camino string, resultado any) error {
	ctx, cancelar := context.WithTimeout(ctx, topeCloudflare)
	defer cancelar()
	base := l.API
	if base == "" {
		base = APICloudflare
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+camino, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+l.Token)
	cliente := l.HTTP
	if cliente == nil {
		cliente = &http.Client{Timeout: topeCloudflare}
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
	sinConsulta := strings.SplitN(camino, "?", 2)[0]
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&sobre); err != nil {
		return fmt.Errorf("Cloudflare respondió %d sin JSON a GET %s", resp.StatusCode, sinConsulta)
	}
	if !sobre.Success || resp.StatusCode >= 300 {
		var mensajes []string
		for _, e := range sobre.Errors {
			mensajes = append(mensajes, fmt.Sprintf("%s (%d)", e.Message, e.Code))
		}
		return fmt.Errorf("Cloudflare respondió %d a GET %s: %s", resp.StatusCode, sinConsulta, strings.Join(mensajes, "; "))
	}
	return json.Unmarshal(sobre.Result, resultado)
}
