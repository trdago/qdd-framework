package remoto

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): el PC habla con el relay solo con la clave publicable; las credenciales quedan en 0600; el
// emparejamiento genera canal y firma con el formato que exige el relay.

// relayFalso imita PostgREST y la Edge Function avisar: registra cada llamada y responde lo que diga respuestas.
type relayFalso struct {
	srv *httptest.Server

	mu         sync.Mutex
	llamadas   []llamada
	respuestas map[string]func(map[string]any) (int, any)
}

type llamada struct {
	Ruta   string
	Args   map[string]any
	Header http.Header
}

func nuevoRelayFalso(t *testing.T) *relayFalso {
	t.Helper()
	r := &relayFalso{respuestas: map[string]func(map[string]any) (int, any){}}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		var args map[string]any
		json.Unmarshal(b, &args)
		r.mu.Lock()
		r.llamadas = append(r.llamadas, llamada{Ruta: req.URL.Path, Args: args, Header: req.Header.Clone()})
		f := r.respuestas[req.URL.Path]
		r.mu.Unlock()
		if f == nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":"PGRST202","message":"no existe"}`))
			return
		}
		status, cuerpo := f(args)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(cuerpo)
	}))
	t.Cleanup(r.srv.Close)
	return r
}

func (r *relayFalso) Responder(ruta string, f func(map[string]any) (int, any)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.respuestas[ruta] = f
}

func (r *relayFalso) Llamadas(ruta string) []llamada {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []llamada
	for _, l := range r.llamadas {
		if l.Ruta == ruta {
			out = append(out, l)
		}
	}
	return out
}

func jwtDePrueba(claims string) string {
	return "eyJhbGciOiJIUzI1NiJ9." + base64.RawURLEncoding.EncodeToString([]byte(claims)) + ".firma"
}

func TestValidarClave_SoloLaPublicable(t *testing.T) {
	casos := []struct {
		clave string
		ok    bool
	}{
		{"sb_publishable_abc123", true},
		{jwtDePrueba(`{"role":"anon","iss":"supabase"}`), true},
		{"sb_secret_abc123", false},
		{jwtDePrueba(`{"role":"service_role"}`), false},
		{jwtDePrueba(`{"role":"authenticated"}`), false},
		{"cualquier-cosa", false},
		{"", false},
	}
	for _, c := range casos {
		err := ValidarClave(c.clave)
		if (err == nil) != c.ok {
			t.Errorf("%q: ok=%v, err=%v", c.clave, c.ok, err)
		}
	}
}

func TestGuardarRelay_NuncaGuardaUnaClaveSecretaYQuedaEn0600(t *testing.T) {
	home := homeTemporal(t)
	if err := GuardarRelay(ConfigRelay{URL: "https://x.supabase.co", Clave: "sb_secret_zzz"}); err == nil {
		t.Fatal("una clave secreta se rechaza")
	}
	if _, err := os.Stat(filepath.Join(home, "remoto", "relay.json")); err == nil {
		t.Fatal("y no se escribe nada")
	}
	if err := GuardarRelay(ConfigRelay{URL: "http://x.supabase.co", Clave: "sb_publishable_a"}); err == nil {
		t.Fatal("el relay va por https")
	}
	if err := GuardarRelay(ConfigRelay{URL: "https://x.supabase.co/", Clave: "sb_publishable_a"}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(filepath.Join(home, "remoto", "relay.json"))
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("0600: %v", info.Mode().Perm())
	}
	c, err := LeerRelay()
	if err != nil || c.URL != "https://x.supabase.co" {
		t.Fatalf("se lee sin la barra final: %+v %v", c, err)
	}
	os.WriteFile(filepath.Join(home, "remoto", "relay.json"), []byte(`{"url":"https://x.supabase.co","clave":"sb_secret_y"}`), 0o600)
	if _, err := LeerRelay(); err == nil {
		t.Fatal("una clave secreta escrita a mano tampoco se usa")
	}
}

func TestRelay_RPCConLaClavePublicableYErroresDelRelay(t *testing.T) {
	f := nuevoRelayFalso(t)
	f.Responder("/rest/v1/rpc/qdd_remoto_latido", func(a map[string]any) (int, any) {
		if a["p_firma"] != "f" {
			return 400, map[string]string{"code": "28000", "message": "credenciales de la máquina inválidas"}
		}
		return 200, map[string]any{"plan": "gratis", "pcs": 1, "proyectos": 3, "consola_permitida": true, "motivo": nil, "maquina": "m"}
	})
	r := NuevoRelay(ConfigRelay{URL: f.srv.URL, Clave: "sb_publishable_k"})
	p, err := r.Latido(context.Background(), "t", "f", 2, 1, 40)
	if err != nil || p.Plan != "gratis" || p.LimiteProyectos() != 3 || !p.ConsolaPermitida {
		t.Fatalf("latido: %+v %v", p, err)
	}
	l := f.Llamadas("/rest/v1/rpc/qdd_remoto_latido")[0]
	if l.Header.Get("apikey") != "sb_publishable_k" || l.Header.Get("Authorization") != "Bearer sb_publishable_k" {
		t.Fatalf("headers: %v", l.Header)
	}
	if l.Args["p_topico"] != "t" || l.Args["p_en_curso"] != float64(2) || l.Args["p_consola_min"] != float64(1) || l.Args["p_mensajes"] != float64(40) {
		t.Fatalf("argumentos: %v", l.Args)
	}
	if _, err := r.Latido(context.Background(), "t", "mala", 0, 0, 0); !errors.Is(err, ErrCredenciales) {
		t.Fatalf("28000 son credenciales: %v", err)
	}
	f.Responder("/rest/v1/rpc/qdd_remoto_ofrecer", func(map[string]any) (int, any) {
		return 400, map[string]string{"code": "53400", "message": "demasiados emparejamientos"}
	})
	var lim ErrLimite
	if err := r.Ofrecer(context.Background(), "h", "n", "t", "f"); !errors.As(err, &lim) || lim.Motivo != "demasiados emparejamientos" {
		t.Fatalf("53400 es un límite con su motivo: %v", err)
	}
	if err := r.Avisar(context.Background(), "t", "f", AvisoPedido, "p", "p-1"); !errors.Is(err, ErrNoDisponible) {
		t.Fatalf("404 es «no disponible»: %v", err)
	}
	f.Responder("/functions/v1/avisar", func(map[string]any) (int, any) {
		return 400, map[string]string{"error": "tipo inválido: x"}
	})
	var rechazada ErrRechazada
	if err := r.Avisar(context.Background(), "t", "f", AvisoPedido, "p", "p-1"); !errors.As(err, &rechazada) || !strings.Contains(rechazada.Error(), "tipo inválido") {
		t.Fatalf("400 es un rechazo con su motivo: %v", err)
	}
	sinLimite := PlanGratis()
	sinLimite.Proyectos = nil
	if sinLimite.LimiteProyectos() != 0 {
		t.Fatal("null es sin límite")
	}
}

func TestAvisar_SoloTipoYProyecto(t *testing.T) {
	f := nuevoRelayFalso(t)
	f.Responder("/functions/v1/avisar", func(map[string]any) (int, any) { return 200, map[string]int{"enviados": 1} })
	r := NuevoRelay(ConfigRelay{URL: f.srv.URL, Clave: "sb_publishable_k"})
	if err := r.Avisar(context.Background(), "t", "f", AvisoTerminoBien, "Mi proyecto/raro!", "mi-proyecto-1"); err != nil {
		t.Fatal(err)
	}
	a := f.Llamadas("/functions/v1/avisar")[0].Args
	if len(a) != 5 || a["tipo"] != "termino_bien" || a["proyecto"] != "Mi proyecto-raro-" || a["clave"] != "mi-proyecto-1" || a["topico"] != "t" || a["firma"] != "f" {
		t.Fatalf("cuerpo cerrado: %v", a)
	}
	if NombreParaAviso("", "c") != "c" || len([]rune(NombreParaAviso(strings.Repeat("á", 90), "c"))) != 80 || NombreParaAviso("qdd (v2)_x", "c") != "qdd (v2)_x" {
		t.Fatal("saneado del nombre")
	}
}

func TestNuevaOferta_FormatoQueExigeElRelay(t *testing.T) {
	o, err := NuevaOferta("mi-pc")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[A-Z2-7]{8}$`).MatchString(o.Codigo) {
		t.Fatalf("código base32 de 8: %s", o.Codigo)
	}
	if !regexp.MustCompile(`^qdd:pc:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}:[A-Za-z0-9_-]{22,64}$`).MatchString(o.Cred.Topico) {
		t.Fatalf("tópico: %s", o.Cred.Topico)
	}
	if !strings.HasPrefix(o.Cred.Topico, "qdd:pc:"+o.Cred.Maquina+":") || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(o.Cred.Firma) {
		t.Fatalf("el uuid del tópico es la máquina; firma de 64 hex: %+v", o.Cred)
	}
	otra, _ := NuevaOferta("mi-pc")
	if otra.Codigo == o.Codigo || otra.Cred.Firma == o.Cred.Firma || otra.Cred.Topico == o.Cred.Topico {
		t.Fatal("cada oferta es nueva")
	}
	suma := sha256.Sum256([]byte("ABCD2345"))
	if HashCodigo(" abcd2345 ") != hex.EncodeToString(suma[:]) {
		t.Fatal("hex(sha256(código en mayúsculas)), como el relay (upper(trim()))")
	}
	var qr map[string]any
	json.Unmarshal([]byte(o.ContenidoQR()), &qr)
	if qr["v"] != float64(1) || qr["codigo"] != o.Codigo || qr["topico"] != o.Cred.Topico || qr["nombre"] != "mi-pc" || len(qr) != 4 {
		t.Fatalf("QR: %v", qr)
	}
	if _, ok := qr["firma"]; ok {
		t.Fatal("la firma nunca va al teléfono")
	}
	dibujo, err := QRTerminal(o.ContenidoQR())
	if err != nil || strings.Count(dibujo, "\n") < 15 || !strings.Contains(dibujo, "▀") {
		t.Fatalf("QR en la terminal: %v\n%s", err, dibujo)
	}
}

func TestEmparejar_EsperaAQueElTelefonoAcepte(t *testing.T) {
	f := nuevoRelayFalso(t)
	var ofrecida map[string]any
	consultas := 0
	f.Responder("/rest/v1/rpc/qdd_remoto_ofrecer", func(a map[string]any) (int, any) {
		ofrecida = a
		return 204, nil
	})
	f.Responder("/rest/v1/rpc/qdd_remoto_emparejada", func(a map[string]any) (int, any) {
		consultas++
		if consultas < 3 {
			return 200, nil
		}
		return 200, strings.Split(ofrecida["p_topico"].(string), ":")[2]
	})
	r := NuevoRelay(ConfigRelay{URL: f.srv.URL, Clave: "sb_publishable_k"})
	var salida bytes.Buffer
	cred, err := Emparejar(context.Background(), r, "pc-de-prueba", &salida, 10*time.Millisecond, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if cred.Topico != ofrecida["p_topico"] || cred.Firma != ofrecida["p_firma"] || cred.Nombre != "pc-de-prueba" || cred.Emparejada.IsZero() {
		t.Fatalf("credenciales de lo ofrecido: %+v %v", cred, ofrecida)
	}
	codigo := regexp.MustCompile(`código: ([A-Z2-7]{8})`).FindStringSubmatch(salida.String())
	if codigo == nil || HashCodigo(codigo[1]) != ofrecida["p_codigo_hash"] {
		t.Fatalf("el código en texto es el del hash ofrecido:\n%s", salida.String())
	}
	f.Responder("/rest/v1/rpc/qdd_remoto_emparejada", func(map[string]any) (int, any) { return 200, nil })
	if _, err := Emparejar(context.Background(), r, "pc", io.Discard, 10*time.Millisecond, 50*time.Millisecond); !errors.Is(err, ErrOfertaVencida) {
		t.Fatalf("sin aceptación, vence: %v", err)
	}
}

func TestCredenciales_PrivadasYSeBorran(t *testing.T) {
	home := homeTemporal(t)
	if _, err := LeerCredenciales(); err == nil || !strings.Contains(err.Error(), "activar") {
		t.Fatalf("sin emparejar lo dice: %v", err)
	}
	if err := GuardarCredenciales(Credenciales{Maquina: "m", Topico: "t", Firma: "f"}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(filepath.Join(home, "remoto", "credenciales.json"))
	dir, _ := os.Stat(filepath.Join(home, "remoto"))
	if info.Mode().Perm() != 0o600 || dir.Mode().Perm() != 0o700 {
		t.Fatalf("0600 y carpeta 0700: %v %v", info.Mode().Perm(), dir.Mode().Perm())
	}
	c, err := LeerCredenciales()
	if err != nil || c.Formato != 1 || c.Firma != "f" {
		t.Fatalf("%+v %v", c, err)
	}
	BorrarCredenciales()
	if _, err := LeerCredenciales(); err == nil {
		t.Fatal("se borran")
	}
}

// Issue #102: qdd trae el relay de QDD Móvil por defecto. Hoy va vacío a propósito: publicarlo espera a que se cierren
// las tablas expuestas del proyecto. Si se llena, va completo y con una clave publicable.
// FND-133: sin relay de fábrica, qdd login fallaba en todo PC nuevo. Va completo y solo con la clave publicable.
func TestRelayPorDefecto_VaCompletoYPublicable_FND133(t *testing.T) {
	c := RelayPorDefecto
	if err := ValidarURL(c.URL); err != nil {
		t.Fatal(err)
	}
	if err := ValidarClave(c.Clave); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(c.Clave, "sb_publishable_") {
		t.Fatal("qdd se publica en npm: su relay de fábrica lleva solo la clave publicable")
	}
}
