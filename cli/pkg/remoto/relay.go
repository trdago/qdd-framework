package remoto

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §Relay: el PC no tiene usuario, contraseña ni service key. Habla con el relay con la clave publicable (es
// anon) y se identifica con su canal y su firma, que viven en <QDD_HOME>/remoto/ con permisos 0600 (inv. 9). Todo pasa
// por funciones public.qdd_remoto_* de PostgREST y por la Edge Function avisar.

var (
	// ErrCredenciales: el relay no reconoce el canal y la firma (SQLSTATE 28000): la máquina se quitó desde el
	// teléfono o se desactivó.
	ErrCredenciales = errors.New("el relay no reconoce este PC (se quitó desde el teléfono o se desactivó): empareja de nuevo con 'qdd remoto activar'")
	// ErrNoDisponible: la función o la Edge Function no existe en el relay (404).
	ErrNoDisponible = errors.New("no disponible en el relay")
)

// ErrLimite es un límite del plan o del relay (SQLSTATE 53400), con su motivo.
type ErrLimite struct{ Motivo string }

func (e ErrLimite) Error() string { return e.Motivo }

// ErrRechazada: el relay rechazó la petición por inválida (400), como un tipo de aviso que un relay viejo no conoce.
type ErrRechazada struct{ Texto string }

func (e ErrRechazada) Error() string { return e.Texto }

// ConfigRelay es <QDD_HOME>/remoto/relay.json: la URL del relay y su clave publicable.
type ConfigRelay struct {
	URL   string `json:"url"`
	Clave string `json:"clave"`
}

// Credenciales es <QDD_HOME>/remoto/credenciales.json: la máquina, su canal y su firma (que solo conocen el PC y el
// relay; nunca se le entregan a un teléfono).
type Credenciales struct {
	Formato    int       `json:"formato"`
	Maquina    string    `json:"maquina"`
	Nombre     string    `json:"nombre"`
	Topico     string    `json:"topico"`
	Firma      string    `json:"firma"`
	Emparejada time.Time `json:"emparejada"`
}

// DirRemoto es <QDD_HOME>/remoto, privada.
func DirRemoto() (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "remoto"), nil
}

func rutaRemoto(nombre string) (string, error) {
	dir, err := DirRemoto()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, nombre), nil
}

// guardarPrivado escribe un JSON en <QDD_HOME>/remoto con permisos 0600, de forma atómica.
func guardarPrivado(nombre string, v any) error {
	ruta, err := rutaRemoto(nombre)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", ruta, os.Getpid())
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

func leerPrivado(nombre string, v any) error {
	ruta, err := rutaRemoto(nombre)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s ilegible: %w", ruta, err)
	}
	return nil
}

func borrarPrivado(nombre string) error {
	ruta, err := rutaRemoto(nombre)
	if err != nil {
		return err
	}
	if err := os.Remove(ruta); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// GuardarRelay valida la clave (nunca una secreta) y guarda la config del relay.
func GuardarRelay(c ConfigRelay) error {
	c.URL = strings.TrimRight(strings.TrimSpace(c.URL), "/")
	c.Clave = strings.TrimSpace(c.Clave)
	if err := ValidarURL(c.URL); err != nil {
		return err
	}
	if err := ValidarClave(c.Clave); err != nil {
		return err
	}
	return guardarPrivado("relay.json", c)
}

// RelayPorDefecto es el relay de QDD Móvil que qdd usa la primera vez (issue #102); lo opera qdd-movile. Lleva solo la
// clave publicable: se publica en npm.
var RelayPorDefecto = ConfigRelay{
	URL:   "https://vrozxrlkyvbeikxbgdgb.supabase.co",
	Clave: "sb_publishable_WFcy-WCo4ZtaV1JIlEaPkw_q4ECzs3T",
}

// LeerRelay devuelve la config guardada; vuelve a validar la clave por si alguien editó el archivo.
func LeerRelay() (ConfigRelay, error) {
	var c ConfigRelay
	if err := leerPrivado("relay.json", &c); err != nil {
		return ConfigRelay{}, err
	}
	if err := ValidarClave(c.Clave); err != nil {
		return ConfigRelay{}, err
	}
	return c, ValidarURL(c.URL)
}

// GuardarCredenciales deja la máquina emparejada.
func GuardarCredenciales(c Credenciales) error {
	c.Formato = 1
	return guardarPrivado("credenciales.json", c)
}

// LeerCredenciales devuelve la máquina emparejada.
func LeerCredenciales() (Credenciales, error) {
	var c Credenciales
	err := leerPrivado("credenciales.json", &c)
	if errors.Is(err, os.ErrNotExist) {
		return Credenciales{}, errors.New("este PC no está emparejado: usa 'qdd remoto activar'")
	}
	return c, err
}

// BorrarCredenciales olvida la máquina (qdd remoto desactivar).
func BorrarCredenciales() error {
	return borrarPrivado("credenciales.json")
}

// ValidarURL exige https:// (http:// solo para localhost, en pruebas).
func ValidarURL(u string) error {
	p, err := url.Parse(u)
	if err != nil || p.Host == "" {
		return fmt.Errorf("la URL del relay no es válida: %q", u)
	}
	local := p.Hostname() == "localhost" || p.Hostname() == "127.0.0.1"
	if p.Scheme == "https" || (p.Scheme == "http" && local) {
		return nil
	}
	return fmt.Errorf("la URL del relay tiene que ser https://: %q", u)
}

// ValidarClave acepta solo la clave publicable del relay: sb_publishable_… o un JWT con role anon. Una clave secreta
// (sb_secret_…) o un JWT service_role se rechaza sin guardarse (inv. 9).
func ValidarClave(clave string) error {
	if clave == "" {
		return errors.New("falta la clave publicable del relay (--clave)")
	}
	if strings.HasPrefix(clave, "sb_secret_") {
		return errors.New("esa es una clave secreta (sb_secret_…): el PC usa solo la publicable; no se guardó")
	}
	if strings.HasPrefix(clave, "sb_publishable_") {
		return nil
	}
	rol, ok := rolDelJWT(clave)
	if !ok {
		return errors.New("la clave no es publicable (sb_publishable_… o el JWT anon del proyecto); no se guardó")
	}
	if rol != "anon" {
		return fmt.Errorf("esa clave es de rol %q: el PC usa solo la publicable (anon); no se guardó", rol)
	}
	return nil
}

func rolDelJWT(token string) (string, bool) {
	partes := strings.Split(token, ".")
	if len(partes) != 3 || !strings.HasPrefix(partes[0], "eyJ") {
		return "", false
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(partes[1], "="))
	if err != nil {
		return "", false
	}
	var c struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(b, &c) != nil || c.Role == "" {
		return "", false
	}
	return c.Role, true
}

// Relay habla con las funciones del relay.
type Relay struct {
	URL   string
	Clave string
	HTTP  *http.Client
}

// NuevoRelay arma el cliente con un tope de 15 s por llamada.
func NuevoRelay(c ConfigRelay) *Relay {
	return &Relay{URL: strings.TrimRight(c.URL, "/"), Clave: c.Clave, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (r *Relay) post(ctx context.Context, ruta string, cuerpo any, out any) error {
	b, err := json.Marshal(cuerpo)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.URL+ruta, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", r.Clave)
	req.Header.Set("Authorization", "Bearer "+r.Clave)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := r.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo hablar con el relay: %w", err)
	}
	defer resp.Body.Close()
	datos, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return errorDelRelay(resp.StatusCode, datos)
	}
	if out == nil || len(bytes.TrimSpace(datos)) == 0 {
		return nil
	}
	return json.Unmarshal(datos, out)
}

// errorDelRelay traduce la respuesta de PostgREST: 28000 son credenciales malas, 53400 un límite.
func errorDelRelay(status int, datos []byte) error {
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Error   string `json:"error"` // las Edge Functions
	}
	_ = json.Unmarshal(datos, &e)
	if e.Code == "28000" {
		return ErrCredenciales
	}
	if status == http.StatusUnauthorized && e.Code == "" {
		return errors.New("el relay rechazó la llamada (401): revisa la clave publicable y el canal")
	}
	if e.Code == "53400" {
		return ErrLimite{Motivo: e.Message}
	}
	if status == http.StatusNotFound {
		return ErrNoDisponible
	}
	texto := fmt.Sprintf("el relay respondió %d", status)
	if e.Message != "" {
		texto += ": " + e.Message
	}
	if e.Error != "" {
		texto += ": " + e.Error
	}
	if status == http.StatusBadRequest {
		return ErrRechazada{Texto: texto}
	}
	return errors.New(texto)
}

func (r *Relay) rpc(ctx context.Context, fn string, args any, out any) error {
	return r.post(ctx, "/rest/v1/rpc/"+fn, args, out)
}

// Ofrecer abre una oferta de emparejamiento de 5 minutos.
func (r *Relay) Ofrecer(ctx context.Context, codigoHash, nombre, topico, firma string) error {
	return r.rpc(ctx, "qdd_remoto_ofrecer", map[string]string{"p_codigo_hash": codigoHash, "p_nombre": nombre, "p_topico": topico, "p_firma": firma}, nil)
}

// Emparejada devuelve el id de la máquina cuando el teléfono aceptó ("" si todavía no).
func (r *Relay) Emparejada(ctx context.Context, codigoHash string) (string, error) {
	var id *string
	if err := r.rpc(ctx, "qdd_remoto_emparejada", map[string]string{"p_codigo_hash": codigoHash}, &id); err != nil {
		return "", err
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

// Plan es lo que responde qdd_remoto_latido: el plan, sus límites, el consumo del mes y si la consola sigue permitida.
type Plan struct {
	Plan             string  `json:"plan"`
	PCs              int     `json:"pcs"`
	Proyectos        *int    `json:"proyectos"` // nil: sin límite
	ConsolaMinMes    int     `json:"consola_min_mes"`
	ConsolaMinUsados int     `json:"consola_min_usados"`
	MensajesMes      int64   `json:"mensajes_mes"`
	MensajesUsados   int64   `json:"mensajes_usados"`
	ConsolaPermitida bool    `json:"consola_permitida"`
	ConsolaPausada   bool    `json:"consola_pausada"`
	Motivo           *string `json:"motivo"`
	Aviso80          bool    `json:"aviso_80"`
	Maquina          string  `json:"maquina"`
}

// PlanGratis es lo que se supone antes del primer latido: el plan más chico, sin consola.
func PlanGratis() Plan {
	tres := 3
	return Plan{Plan: "gratis", PCs: 1, Proyectos: &tres}
}

// LimiteProyectos es el tope de proyectos del plan (0: sin límite).
func (p Plan) LimiteProyectos() int {
	if p.Proyectos == nil {
		return 0
	}
	return *p.Proyectos
}

// MotivoPausa dice por qué la consola está pausada.
func (p Plan) MotivoPausa() string {
	if p.Motivo == nil || *p.Motivo == "" {
		return "pausada por el relay"
	}
	return *p.Motivo
}

// Latido da el latido del minuto: tareas en curso, minutos de consola y mensajes desde el anterior.
func (r *Relay) Latido(ctx context.Context, topico, firma string, enCurso, consolaMin, mensajes int) (Plan, error) {
	var p Plan
	args := map[string]any{"p_topico": topico, "p_firma": firma, "p_en_curso": enCurso, "p_consola_min": consolaMin, "p_mensajes": mensajes}
	err := r.rpc(ctx, "qdd_remoto_latido", args, &p)
	return p, err
}

// Dispositivo es un teléfono del dueño de la máquina.
type Dispositivo struct {
	ID         string    `json:"id"`
	Nombre     string    `json:"nombre"`
	Plataforma string    `json:"plataforma"`
	Revocado   bool      `json:"revocado"`
	Visto      time.Time `json:"visto"`
}

// Dispositivos lista los teléfonos del dueño.
func (r *Relay) Dispositivos(ctx context.Context, topico, firma string) ([]Dispositivo, error) {
	var ds []Dispositivo
	err := r.rpc(ctx, "qdd_remoto_dispositivos", map[string]string{"p_topico": topico, "p_firma": firma}, &ds)
	return ds, err
}

// Revocar saca un teléfono: el relay rechaza su próxima acción.
func (r *Relay) Revocar(ctx context.Context, topico, firma, dispositivo string) (bool, error) {
	var ok bool
	err := r.rpc(ctx, "qdd_remoto_revocar", map[string]string{"p_topico": topico, "p_firma": firma, "p_dispositivo": dispositivo}, &ok)
	return ok, err
}

// Desactivar borra la máquina del relay: el canal deja de admitir a nadie.
func (r *Relay) Desactivar(ctx context.Context, topico, firma string) (bool, error) {
	var ok bool
	err := r.rpc(ctx, "qdd_remoto_desactivar", map[string]string{"p_topico": topico, "p_firma": firma}, &ok)
	return ok, err
}

// Tipos de aviso por push: cerrados, sin texto libre (inv. 17).
const (
	AvisoPedido       = "pedido"
	AvisoFicha        = "ficha"
	AvisoTerminoBien  = "termino_bien"
	AvisoTerminoFalla = "termino_falla"
	// El modo goal se detuvo o terminó (RFC 0015).
	AvisoGoalPausado   = "goal_pausado"
	AvisoGoalTerminado = "goal_terminado"
	// Ya no espera nada de ese tipo: el relay manda un push solo de datos y la app quita el aviso de la bandeja.
	AvisoPedidoResuelto = "pedido_resuelto"
	AvisoFichaResuelta  = "ficha_resuelto"
)

// NombreParaAviso deja el nombre visible del proyecto como lo acepta avisar (^[\p{L}\p{N} ._()-]{1,80}$): lo demás
// pasa a «-» y se recorta a 80; vacío, la clave.
func NombreParaAviso(nombre, clave string) string {
	var out []rune
	for _, r := range nombre {
		if len(out) == 80 {
			break
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) || strings.ContainsRune(" ._()-", r) {
			out = append(out, r)
			continue
		}
		out = append(out, '-')
	}
	if strings.TrimSpace(string(out)) == "" {
		return clave
	}
	return string(out)
}

// Avisar pide un push a los teléfonos del dueño: solo el tipo y el proyecto (nombre visible y clave).
func (r *Relay) Avisar(ctx context.Context, topico, firma, tipo, proyecto, clave string) error {
	cuerpo := map[string]string{"topico": topico, "firma": firma, "tipo": tipo, "proyecto": NombreParaAviso(proyecto, clave), "clave": clave}
	return r.post(ctx, "/functions/v1/avisar", cuerpo, nil)
}
