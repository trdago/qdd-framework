package reportes

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'

// formasDeLaPrivada son las maneras en que la llave privada podría salir escrita: su semilla (base64 y hex) y la llave
// entera de 64 bytes (base64 y hex).
func formasDeLaPrivada(t *testing.T) []string {
	t.Helper()
	llave, err := Llave()
	if err != nil {
		t.Fatal(err)
	}
	semilla := llave.Seed()
	return []string{base64.StdEncoding.EncodeToString(semilla), hex.EncodeToString(semilla),
		base64.StdEncoding.EncodeToString(llave), hex.EncodeToString(llave), base64.RawURLEncoding.EncodeToString(semilla)}
}

// Enviar y leer reportes firman con la llave, pero lo que llega al relay (ruta, cabeceras y cuerpo) solo lleva la
// pública y la firma; y fuera de <QDD_HOME>/reportes/llave ningún archivo la guarda.
// contrato: reportes#a48b9544
func TestLaLlavePrivadaNoViajaAlRelayNiQuedaEnOtroArchivo(t *testing.T) {
	entorno(t)
	var mu sync.Mutex
	var viajo strings.Builder
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cuerpo, _ := io.ReadAll(req.Body)
		mu.Lock()
		viajo.WriteString(req.URL.String() + "\n")
		for k, v := range req.Header {
			viajo.WriteString(k + ": " + strings.Join(v, ",") + "\n")
		}
		viajo.Write(cuerpo)
		mu.Unlock()
		if req.URL.Path == "/functions/v1/reportes" {
			_, _ = io.WriteString(w, `{"reportes":[]}`)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"REP-1"}`)
	}))
	defer srv.Close()
	cfg := remoto.ConfigRelay{URL: srv.URL, Clave: "anon"}
	if _, err := Enviar(context.Background(), cfg, nuevo(t, "Se cae el panel", "detalle"), ahora); err != nil {
		t.Fatal(err)
	}
	if _, err := Leer(context.Background(), cfg, "", ahora); err != nil {
		t.Fatal(err)
	}
	publica, _ := Publica()
	if !strings.Contains(viajo.String(), publica) {
		t.Fatal("la prueba necesita lo que viajó: la pública va en cada envío")
	}
	formas := formasDeLaPrivada(t)
	for _, f := range formas {
		if strings.Contains(viajo.String(), f) {
			t.Fatal("la llave privada viajó al relay")
		}
	}
	ruta, _ := rutaEn("llave")
	filepath.WalkDir(os.Getenv("QDD_HOME"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || p == ruta {
			return nil
		}
		b, _ := os.ReadFile(p)
		for _, f := range formas {
			if strings.Contains(string(b), f) {
				t.Errorf("%s guarda la llave privada", p)
			}
		}
		return nil
	})
	if info, err := os.Stat(ruta); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("la llave vive solo en %s, con 0600: %v %v", ruta, info, err)
	}
}
