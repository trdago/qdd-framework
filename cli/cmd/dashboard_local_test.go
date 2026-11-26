package cmd

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// FND-164: el dashboard acepta escrituras sin autenticación, así que solo puede atender a este PC y a su propia página.

func TestEscucharLocal_SoloEnEstePC(t *testing.T) {
	listener, port := escucharLocal(38099)
	defer listener.Close()
	addr := listener.Addr().(*net.TCPAddr)
	if !addr.IP.IsLoopback() || addr.Port != port {
		t.Fatalf("debe escuchar solo en loopback: %s (puerto %d)", addr, port)
	}
}

func TestSoloLocal(t *testing.T) {
	casos := []struct {
		nombre, metodo, host, origen, tipo string
		codigo                             int
	}{
		{"su página lee", http.MethodGet, "localhost:8099", "", "", http.StatusOK},
		{"su página escribe", http.MethodPost, "localhost:8099", "http://localhost:8099", "application/json", http.StatusOK},
		{"por 127.0.0.1", http.MethodPost, "127.0.0.1:8099", "http://127.0.0.1:8099", "application/json; charset=utf-8", http.StatusOK},
		{"un comando local sin Origin", http.MethodPost, "127.0.0.1:8099", "", "application/json", http.StatusOK},
		{"otro Host (rebinding de DNS)", http.MethodGet, "ejemplo.test:8099", "", "", http.StatusForbidden},
		{"una IP de la red", http.MethodGet, "192.168.1.20:8099", "", "", http.StatusForbidden},
		{"una página externa", http.MethodPost, "localhost:8099", "https://ejemplo.test", "application/json", http.StatusForbidden},
		{"Origin de otro puerto", http.MethodPost, "localhost:8099", "http://localhost:3000", "application/json", http.StatusForbidden},
		{"Origin null", http.MethodPost, "localhost:8099", "null", "application/json", http.StatusForbidden},
		{"un formulario sin JSON", http.MethodPost, "localhost:8099", "", "text/plain", http.StatusUnsupportedMediaType},
		{"un POST sin tipo", http.MethodPost, "localhost:8099", "", "", http.StatusUnsupportedMediaType},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			llego := false
			h := soloLocal(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llego = true }))
			r := httptest.NewRequest(c.metodo, "/api/file", strings.NewReader("{}"))
			r.Host = c.host
			if c.origen != "" {
				r.Header.Set("Origin", c.origen)
			}
			if c.tipo != "" {
				r.Header.Set("Content-Type", c.tipo)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != c.codigo || llego != (c.codigo == http.StatusOK) {
				t.Fatalf("código %d, llegó %v; se esperaba %d", w.Code, llego, c.codigo)
			}
		})
	}
}

func TestRutaDelProyecto(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "proy")
	got, ok := rutaDelProyecto(raiz, "a/b.txt")
	if !ok || got != filepath.Join(raiz, "a", "b.txt") {
		t.Fatalf("%q %v", got, ok)
	}
	// «../proy-otro» empieza igual que la raíz: comparar prefijos lo dejaba pasar.
	for _, fuera := range []string{"../x", "../proy-otro/x", "/etc/passwd", "a/../../x", ""} {
		if got, ok := rutaDelProyecto(raiz, fuera); ok {
			t.Fatalf("%q queda fuera del proyecto y dio %q", fuera, got)
		}
	}
}

// envDashboardDePrueba hace del binario de pruebas un «qdd dashboard» (TestMain).
const envDashboardDePrueba = "QDD_DASHBOARD_DE_PRUEBA"

// Caja negra del cableado: el «qdd dashboard» de verdad, no soloLocal suelto. Si http.Serve dejara de envolver sus
// rutas con soloLocal, otro Host, otro Origin o un POST sin JSON llegarían a /api/file y escribirían en el proyecto.
// contrato: dashboard#7ceb11f1
func TestDashboard_ElServidorDeVerdadSoloAtiendeAEstePC(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("lanza el binario de pruebas con un PATH vacío")
	}
	proyecto, home := t.TempDir(), t.TempDir()
	c := exec.Command(os.Args[0], "dashboard")
	c.Dir = proyecto
	c.Env = append(gitenv.Limpio(os.Environ()), envDashboardDePrueba+"=1", "HOME="+home, "QDD_HOME="+filepath.Join(home, "qdd"), "PATH="+t.TempDir())
	salida, err := c.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Process.Kill(); c.Wait() })
	puerto := make(chan string, 1)
	go func() {
		lineas := bufio.NewScanner(salida)
		for lineas.Scan() {
			if _, p, ok := strings.Cut(lineas.Text(), "escuchando en http://localhost:"); ok {
				puerto <- strings.TrimSpace(p)
			}
		}
	}()
	var p string
	select {
	case p = <-puerto:
	case <-time.After(30 * time.Second):
		t.Fatal("qdd dashboard no dijo en qué puerto escucha")
	}
	pedir := func(host, origen, tipo string) int {
		t.Helper()
		r, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:"+p+"/api/file", strings.NewReader(`{"path":"pwned.txt","content":"x"}`))
		r.Host = host
		if origen != "" {
			r.Header.Set("Origin", origen)
		}
		if tipo != "" {
			r.Header.Set("Content-Type", tipo)
		}
		res, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	local := "localhost:" + p
	casos := []struct {
		nombre, host, origen, tipo string
		codigo                     int
	}{
		{"otro Host", "evil.test", "", "application/json", http.StatusForbidden},
		{"otra página", local, "https://evil.test", "application/json", http.StatusForbidden},
		{"un POST sin JSON", local, "", "text/plain", http.StatusUnsupportedMediaType},
	}
	for _, caso := range casos {
		if got := pedir(caso.host, caso.origen, caso.tipo); got != caso.codigo {
			t.Errorf("%s: %d, se esperaba %d", caso.nombre, got, caso.codigo)
		}
	}
	if _, err := os.Stat(filepath.Join(proyecto, "pwned.txt")); err == nil {
		t.Fatal("una petición rechazada escribió en el proyecto")
	}
	r, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+p+"/", nil)
	r.Host = local
	res, err := http.DefaultClient.Do(r)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("su propia página sí se atiende: %v %v", res, err)
	}
	res.Body.Close()
}
