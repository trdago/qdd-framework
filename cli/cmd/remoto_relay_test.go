package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): activar guarda el relay, empareja y deja el servicio; desactivar lo deshace. Con un relay falso y
// un ejecutor que solo anota: las pruebas no instalan nada.

type relayCmdFalso struct {
	mu       sync.Mutex
	llamadas []string
	topico   string
}

func nuevoRelayCmd(t *testing.T) (*relayCmdFalso, string) {
	t.Helper()
	f := &relayCmdFalso{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var a map[string]string
		json.Unmarshal(b, &a)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.llamadas = append(f.llamadas, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "qdd_remoto_ofrecer") {
			f.topico = a["p_topico"]
			w.WriteHeader(204)
			return
		}
		if strings.HasSuffix(r.URL.Path, "qdd_remoto_emparejada") {
			json.NewEncoder(w).Encode(strings.Split(f.topico, ":")[2])
			return
		}
		if strings.HasSuffix(r.URL.Path, "qdd_remoto_desactivar") {
			json.NewEncoder(w).Encode(true)
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)
	return f, srv.URL
}

func TestRemotoActivar_GuardaEmparejaYDejaElServicio(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("el servicio de prueba se genera para Linux y macOS")
	}
	t.Setenv("QDD_HOME", t.TempDir())
	usuario := t.TempDir()
	f, url := nuevoRelayCmd(t)
	var corridos []string
	antesEjecutor, antesUsuario := remotoEjecutor, remotoUsuario
	remotoEjecutor = func(n string, a ...string) error {
		corridos = append(corridos, n+" "+strings.Join(a, " "))
		return nil
	}
	remotoUsuario = func() (string, error) { return usuario, nil }
	remotoCadaEspera, remotoRelayURL, remotoClave, remotoSinServ = 5*time.Millisecond, url, "sb_publishable_prueba", false
	t.Cleanup(func() {
		remotoEjecutor, remotoUsuario, remotoCadaEspera, remotoRelayURL, remotoClave = antesEjecutor, antesUsuario, 0, "", ""
	})
	var out, errs bytes.Buffer
	if c := remotoActivar(context.Background(), &out, &errs); c != 0 {
		t.Fatalf("activar: %d\n%s\n%s", c, out.String(), errs.String())
	}
	cred, err := remoto.LeerCredenciales()
	if err != nil || cred.Topico != f.topico {
		t.Fatalf("credenciales del emparejamiento: %+v %v", cred, err)
	}
	if !strings.Contains(out.String(), "Escanea") || len(corridos) == 0 {
		t.Fatalf("muestra el QR y enciende el servicio:\n%s\n%v", out.String(), corridos)
	}
	s, _ := servicioRemoto()
	archivo, _ := s.Archivo()
	if !strings.HasPrefix(archivo, usuario) {
		t.Fatalf("el servicio va en la carpeta del usuario: %s", archivo)
	}
	if _, err := os.Stat(archivo); err != nil {
		t.Fatalf("queda escrito: %v", err)
	}
	remotoRelayURL, remotoClave = "", ""
	out.Reset()
	if c := remotoActivar(context.Background(), &out, &errs); c != 0 || strings.Contains(out.String(), "Escanea") {
		t.Fatalf("sin flags usa lo guardado y no vuelve a emparejar: %d\n%s", c, out.String())
	}
	if c := remotoEmparejar(context.Background(), &out, &errs); c != 1 {
		t.Fatal("emparejar con un PC ya emparejado lo dice")
	}
	if c := remotoDesactivar(context.Background(), &out, &errs); c != 0 {
		t.Fatalf("desactivar: %s", errs.String())
	}
	if _, err := remoto.LeerCredenciales(); err == nil {
		t.Fatal("borra las credenciales")
	}
	if _, err := os.Stat(archivo); err == nil {
		t.Fatal("borra el servicio")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.llamadas[len(f.llamadas)-1] != "/rest/v1/rpc/qdd_remoto_desactivar" {
		t.Fatalf("saca la máquina del relay: %v", f.llamadas)
	}
}

func TestRemotoActivar_RechazaUnaClaveSecreta(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	remotoRelayURL, remotoClave = "https://x.supabase.co", "sb_secret_nunca"
	t.Cleanup(func() { remotoRelayURL, remotoClave = "", "" })
	var out, errs bytes.Buffer
	if c := remotoActivar(context.Background(), &out, &errs); c != 1 || !strings.Contains(errs.String(), "secreta") {
		t.Fatalf("rechaza: %d %s", c, errs.String())
	}
	if _, err := os.Stat(home + "/remoto/relay.json"); err == nil {
		t.Fatal("no se guarda")
	}
}

func TestRemotoCorrer_UnaSolaVezPorMaquina(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	soltar, err := unaSolaVez()
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := remoto.DirRemoto()
	os.WriteFile(dir+"/correr.pid", []byte("1"), 0o600)
	if _, err := unaSolaVez(); err == nil && runtime.GOOS != "windows" {
		t.Fatal("con otro vivo (pid 1), no corre otro")
	}
	soltar()
}

func TestRemotoRevocar_LocalYEnElRelay(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	var revocado string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var a map[string]string
		json.Unmarshal(b, &a)
		if strings.HasSuffix(r.URL.Path, "qdd_remoto_revocar") {
			revocado = a["p_dispositivo"]
			json.NewEncoder(w).Encode(true)
			return
		}
		json.NewEncoder(w).Encode([]map[string]any{{"id": "d1", "nombre": "Pixel", "plataforma": "android", "revocado": false, "visto": "2026-11-02T10:00:00Z"}})
	}))
	defer srv.Close()
	remoto.GuardarRelay(remoto.ConfigRelay{URL: srv.URL, Clave: "sb_publishable_x"})
	remoto.GuardarCredenciales(remoto.Credenciales{Topico: "t", Firma: "f"})
	var out, errs bytes.Buffer
	if c := remotoDispositivos(context.Background(), &out, &errs); c != 0 || !strings.Contains(out.String(), "d1  Pixel (android)") {
		t.Fatalf("lista: %d %s %s", c, out.String(), errs.String())
	}
	if c := remotoRevocar(context.Background(), "d1", &out, &errs); c != 0 || revocado != "d1" {
		t.Fatalf("revoca en el relay: %d %s", c, errs.String())
	}
	if !remoto.LeerRevocados()["d1"] {
		t.Fatal("y en este PC, en el acto")
	}
	out.Reset()
	remotoDispositivos(context.Background(), &out, &errs)
	if !strings.Contains(out.String(), "✕ revocado") {
		t.Fatalf("la lista lo dice: %s", out.String())
	}
}

// FND-136: desactivar con otro QDD_HOME saca esa máquina, pero deja el servicio real y lo dice.
func TestRemoto_DesactivarDesdeOtroQDDHomeNoBorraElServicio_FND136(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("en Windows la tarea no lleva QDD_HOME")
	}
	prepararLogin(t, remoto.ConfigRelay{})
	t.Setenv("QDD_HOME", "/ruta/del/servicio/real")
	real, err := servicioRemoto()
	if err != nil {
		t.Fatal(err)
	}
	archivo, err := real.Instalar(remotoEjecutor)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("QDD_HOME", t.TempDir())
	var out, errs bytes.Buffer
	if c := remotoDesactivar(context.Background(), &out, &errs); c != 0 {
		t.Fatalf("desactivar: %d %s", c, errs.String())
	}
	if _, err := os.Stat(archivo); err != nil {
		t.Fatalf("el servicio real sigue: %v", err)
	}
	if strings.Contains(out.String(), "Servicio detenido") || !strings.Contains(out.String(), "otro QDD_HOME") {
		t.Fatalf("dice lo que pasó: %s", out.String())
	}
}

func TestRemoto_EmparejarSinRelayMandaAQddLogin_FND136(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{})
	var out, errs bytes.Buffer
	if c := remotoEmparejar(context.Background(), &out, &errs); c != 1 || !strings.Contains(errs.String(), "qdd login") {
		t.Fatalf("un PC nuevo se empareja con qdd login: %d %s", c, errs.String())
	}
}

func TestRemoto_EmparejarAvisaQueSinServicioNoPublica_FND136(t *testing.T) {
	_, url := nuevoRelayCmd(t)
	prepararLogin(t, remoto.ConfigRelay{})
	if err := remoto.GuardarRelay(remoto.ConfigRelay{URL: url, Clave: "sb_publishable_prueba"}); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if c := remotoEmparejar(context.Background(), &out, &errs); c != 0 || !strings.Contains(out.String(), "qdd login") {
		t.Fatalf("avisa qué falta: %d %s %s", c, out.String(), errs.String())
	}
}

// FND-171: emparejar tampoco puede dar un código de un relay retirado.
func TestRemoto_EmparejarUsaElRelayDelDirectorio_FND171(t *testing.T) {
	antiguo, urlAntiguo := nuevoRelayCmd(t)
	nuevo, urlNuevo := nuevoRelayCmd(t)
	prepararLogin(t, remoto.ConfigRelay{URL: urlNuevo, Clave: "sb_publishable_prueba", Serie: 1})
	if err := remoto.GuardarRelay(remoto.ConfigRelay{URL: urlAntiguo, Clave: "sb_publishable_prueba"}); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if c := remotoEmparejar(context.Background(), &out, &errs); c != 0 {
		t.Fatalf("empareja: %d %s", c, errs.String())
	}
	if len(antiguo.llamadas) != 0 || len(nuevo.llamadas) == 0 {
		t.Fatalf("el código sale del relay del directorio: antiguo %v, nuevo %v", antiguo.llamadas, nuevo.llamadas)
	}
}

func TestRemoto_EmparejarAvisaSiNoPudoConfirmarElRelay_FND171(t *testing.T) {
	_, url := nuevoRelayCmd(t)
	prepararLogin(t, remoto.ConfigRelay{})
	if err := remoto.GuardarRelay(remoto.ConfigRelay{URL: url, Clave: "sb_publishable_prueba"}); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if c := remotoEmparejar(context.Background(), &out, &errs); c != 0 || !strings.Contains(errs.String(), "no se pudo confirmar") {
		t.Fatalf("avisa antes del código: %d %q", c, errs.String())
	}
}
