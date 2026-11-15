package reportes

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// RFC 0024: la llave es de la instalación (ed25519, <QDD_HOME>/reportes/llave, 0600) y nunca sale de la máquina. Lo
// enviado queda en enviados/ con el id del relay; lo que espera el «sí» de la persona, en pendientes/.

var (
	// ErrTope: el relay ya recibió los reportes de hoy de esta instalación, o los de esta hora de todas.
	ErrTope = errors.New("el relay alcanzó su tope de reportes (50 por día por instalación, y uno global por hora): prueba mañana")
	// ErrSinRelay: el relay todavía no tiene la función que recibe reportes.
	ErrSinRelay = errors.New("el relay todavía no recibe reportes: el reporte queda guardado; vuelve a intentarlo con una versión nueva")
)

func carpeta(partes ...string) (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(append([]string{home, "reportes"}, partes...)...)
	return dir, os.MkdirAll(dir, 0o700)
}

func rutaEn(nombre string) (string, error) {
	dir, err := carpeta()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, nombre), nil
}

// Llave devuelve la llave de esta instalación; la primera vez la crea.
func Llave() (ed25519.PrivateKey, error) {
	ruta, err := rutaEn("llave")
	if err != nil {
		return nil, err
	}
	if llave, err := leerLlave(ruta); err == nil {
		return llave, nil
	}
	semilla := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(semilla); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return leerLlave(ruta)
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.WriteString(base64.StdEncoding.EncodeToString(semilla) + "\n"); err != nil {
		return nil, err
	}
	return ed25519.NewKeyFromSeed(semilla), nil
}

func leerLlave(ruta string) (ed25519.PrivateKey, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	semilla, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(datos)))
	if err != nil || len(semilla) != ed25519.SeedSize {
		return nil, fmt.Errorf("la llave de reportes (%s) está dañada: bórrala y se crea otra", ruta)
	}
	return ed25519.NewKeyFromSeed(semilla), nil
}

// Publica es la llave pública de esta instalación, en base64: la que ve el relay.
func Publica() (string, error) {
	llave, err := Llave()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(llave.Public().(ed25519.PublicKey)), nil
}

// Enviado es un reporte que el relay recibió.
type Enviado struct {
	ID      string    `json:"id"`
	En      time.Time `json:"en"`
	Reporte Reporte   `json:"reporte"`
}

// Enviar firma el reporte ahora (el relay rechaza un «en» a más de 10 minutos), lo manda y lo guarda en enviados/.
func Enviar(ctx context.Context, cfg remoto.ConfigRelay, r Reporte, ahora time.Time) (string, error) {
	llave, err := Llave()
	if err != nil {
		return "", err
	}
	r.Instalacion = base64.StdEncoding.EncodeToString(llave.Public().(ed25519.PublicKey))
	r.En = ahora.UTC().Format(time.RFC3339)
	r.Firma = base64.StdEncoding.EncodeToString(ed25519.Sign(llave, []byte(r.mensaje())))
	id, err := post(ctx, cfg, r)
	if err != nil {
		return "", err
	}
	return id, guardarJSON(filepath.Join("enviados", ahora.UTC().Format("20060102T150405")+"-"+idNuevo()+".json"), Enviado{ID: id, En: ahora.UTC(), Reporte: r})
}

func post(ctx context.Context, cfg remoto.ConfigRelay, r Reporte) (string, error) {
	cuerpo, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.URL, "/")+"/functions/v1/reportar", bytes.NewReader(cuerpo))
	if err != nil {
		return "", err
	}
	req.Header.Set("apikey", cfg.Clave)
	req.Header.Set("Authorization", "Bearer "+cfg.Clave)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("no se pudo hablar con el relay: %w", err)
	}
	defer resp.Body.Close()
	datos, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var res struct {
		ID    string `json:"id"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(datos, &res)
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", ErrTope
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", ErrSinRelay
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("el relay respondió %d: %s", resp.StatusCode, res.Error)
	}
	if res.ID == "" {
		return "", fmt.Errorf("el relay respondió %d sin id", resp.StatusCode)
	}
	return res.ID, nil
}

// Enviados son los reportes que esta instalación mandó, del más viejo al más nuevo.
func Enviados() ([]Enviado, error) {
	dir, err := carpeta("enviados")
	if err != nil {
		return nil, err
	}
	archivos, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var out []Enviado
	for _, a := range archivos {
		var e Enviado
		datos, err := os.ReadFile(a)
		if err != nil || json.Unmarshal(datos, &e) != nil {
			continue
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].En.Before(out[j].En) })
	return out, nil
}

var idValido = regexp.MustCompile(`^[0-9a-f]{8}$`)

// ComandoEnviar es el comando del pedido que la persona confirma en el panel.
func ComandoEnviar(id string) string {
	return "qdd reportar enviar " + id
}

// GuardarPendiente deja el reporte esperando el «sí» de la persona y devuelve su id.
func GuardarPendiente(r Reporte, ahora time.Time) (string, error) {
	id := idNuevo()
	return id, guardarJSON(filepath.Join("pendientes", id+".json"), r)
}

// LeerPendiente lee un reporte que espera; el id no puede salir de pendientes/.
func LeerPendiente(id string) (Reporte, error) {
	if !idValido.MatchString(id) {
		return Reporte{}, fmt.Errorf("id de reporte inválido: %q", id)
	}
	dir, err := carpeta("pendientes")
	if err != nil {
		return Reporte{}, err
	}
	datos, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return Reporte{}, fmt.Errorf("no hay un reporte pendiente %s (ya se envió o se borró)", id)
	}
	var r Reporte
	return r, json.Unmarshal(datos, &r)
}

// EnviarPendiente manda un reporte que esperaba y, enviado, lo quita de pendientes/.
func EnviarPendiente(ctx context.Context, cfg remoto.ConfigRelay, id string, ahora time.Time) (string, error) {
	r, err := LeerPendiente(id)
	if err != nil {
		return "", err
	}
	enviado, err := Enviar(ctx, cfg, r, ahora)
	if err != nil {
		return "", err
	}
	dir, err := carpeta("pendientes")
	if err != nil {
		return enviado, err
	}
	return enviado, os.Remove(filepath.Join(dir, id+".json"))
}

func guardarJSON(relativa string, v any) error {
	dir, err := carpeta(filepath.Dir(relativa))
	if err != nil {
		return err
	}
	datos, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filepath.Base(relativa)), append(datos, '\n'), 0o600)
}

func idNuevo() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
