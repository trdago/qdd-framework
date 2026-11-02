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
	remotoEjecutor = func(n string, a ...string) error {
		corridos = append(corridos, n+" "+strings.Join(a, " "))
		return nil
	}
	remotoUsuario = func() (string, error) { return usuario, nil }
	remotoCadaEspera, remotoRelayURL, remotoClave, remotoSinServ = 5*time.Millisecond, url, "sb_publishable_prueba", false
	t.Cleanup(func() {
		remotoEjecutor, remotoUsuario, remotoCadaEspera, remotoRelayURL, remotoClave = ejecutarSistema, os.UserHomeDir, 0, "", ""
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
