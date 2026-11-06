package cmd

import (
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
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
