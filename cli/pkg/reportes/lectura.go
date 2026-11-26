package reportes

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// RFC 0024, fase 3: el relay entrega los reportes solo a una instalación lectora (su llave pública está en el secreto
// REPORTES_LECTORES). La consulta va firmada sobre «reportes|desde|en» y llega de a 100 en orden.

// ErrNoLector: la llave de esta instalación no está entre las lectoras del relay.
var ErrNoLector = errors.New("esta instalación no lee los reportes del relay: agrega la salida de 'qdd reportar llave' al secreto REPORTES_LECTORES del relay")

const (
	porPagina  = 100
	maxPaginas = 20
)

// Recibido es un reporte tal como lo entrega el relay a una instalación lectora.
type Recibido struct {
	ID      string `json:"id"`
	Tipo    string `json:"tipo"`
	Titulo  string `json:"titulo"`
	Detalle string `json:"detalle"`
	Version string `json:"version"`
	SO      string `json:"so"`
	Llego   string `json:"llego"`
}

// Leer trae los reportes posteriores a desde ("" = desde el principio), página por página.
func Leer(ctx context.Context, cfg remoto.ConfigRelay, desde string, ahora time.Time) ([]Recibido, error) {
	llave, err := Llave()
	if err != nil {
		return nil, err
	}
	var todos []Recibido
	for range maxPaginas {
		lote, err := leerPagina(ctx, cfg, llave, desde, ahora)
		if err != nil {
			return todos, err
		}
		todos = append(todos, lote...)
		if len(lote) < porPagina {
			return todos, nil
		}
		desde = lote[len(lote)-1].ID
	}
	return todos, nil
}

func leerPagina(ctx context.Context, cfg remoto.ConfigRelay, llave ed25519.PrivateKey, desde string, ahora time.Time) ([]Recibido, error) {
	en := ahora.UTC().Format(time.RFC3339)
	destino := strings.TrimRight(cfg.URL, "/") + "/functions/v1/reportes"
	if desde != "" {
		destino += "?desde=" + url.QueryEscape(desde)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, destino, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", cfg.Clave)
	req.Header.Set("Authorization", "Bearer "+cfg.Clave)
	req.Header.Set("x-qdd-instalacion", base64.StdEncoding.EncodeToString(llave.Public().(ed25519.PublicKey)))
	req.Header.Set("x-qdd-en", en)
	req.Header.Set("x-qdd-firma", base64.StdEncoding.EncodeToString(ed25519.Sign(llave, []byte("reportes|"+desde+"|"+en))))
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("no se pudo hablar con el relay: %w", err)
	}
	defer resp.Body.Close()
	datos, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	var res struct {
		Reportes []Recibido `json:"reportes"`
		Error    string     `json:"error"`
	}
	_ = json.Unmarshal(datos, &res)
	if resp.StatusCode == http.StatusForbidden {
		return nil, ErrNoLector
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrSinRelay
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("el relay respondió %d: %s", resp.StatusCode, res.Error)
	}
	return res.Reportes, nil
}
