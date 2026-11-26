package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/incidentes"
	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/reportes"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// RFC 0024: nada se envía sin el «sí» de la persona. En su terminal se le muestra el texto y se le pregunta; sin
// terminal (la IA, un pedido, el teléfono) queda un pedido DePersona que el modo goal no lanza.

var ahoraReporte = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

func entornoReporte(t *testing.T) (string, reportes.Reporte) {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := reportes.Nuevo(reportes.Datos{Titulo: "Se cae el panel", Detalle: "al abrir", Verifico: "abro el panel", Version: "v", SO: "linux/amd64"}, incidentes.Contexto{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return root, r
}

// contrato: reportes#f33c94d3
func TestReportar_SinTerminalDejaPedidoDeLaPersona(t *testing.T) {
	root, r := entornoReporte(t)
	enviado := false
	var salida bytes.Buffer
	codigo, err := ofrecerReporte(ofrecimiento{Reporte: r, Root: root, Enviar: func(reportes.Reporte) (string, error) { enviado = true; return "REP-1", nil }, Salida: &salida, Ahora: ahoraReporte})
	if err != nil || codigo != reportes.CodigoTodaviaNo || enviado {
		t.Fatalf("sin terminal no se envía: queda el pedido (75): %d %v enviado=%v", codigo, err, enviado)
	}
	pedidos := consola.PedidosA(root, ahoraReporte)
	if len(pedidos) != 1 || !pedidos[0].DePersona || !strings.HasPrefix(pedidos[0].Comando, "qdd reportar enviar ") || !strings.HasPrefix(pedidos[0].Motivo, "📮") {
		t.Fatalf("debe quedar un pedido 📮 que solo confirma la persona: %+v", pedidos)
	}
	id := strings.TrimPrefix(pedidos[0].Comando, "qdd reportar enviar ")
	if p, err := reportes.LeerPendiente(id); err != nil || p.Titulo != r.Titulo {
		t.Fatalf("el pendiente es el reporte que se mostró: %+v %v", p, err)
	}
	if !strings.Contains(salida.String(), "Se cae el panel") {
		t.Fatalf("muestra el texto exacto: %s", salida.String())
	}
}

func TestReportar_SinTerminalNiProyectoSeNiega(t *testing.T) {
	_, r := entornoReporte(t)
	_, err := ofrecerReporte(ofrecimiento{Reporte: r, Root: "", Salida: &bytes.Buffer{}, Ahora: ahoraReporte})
	if err == nil || !strings.Contains(err.Error(), "tu terminal") {
		t.Fatalf("sin terminal ni panel, dice dónde correrlo: %v", err)
	}
}

// contrato: reportes#f33c94d3
func TestReportar_EnLaTerminalPreguntaYRespetaElNo(t *testing.T) {
	root, r := entornoReporte(t)
	var visto reportes.Reporte
	enviar := func(x reportes.Reporte) (string, error) { visto = x; return "REP-2", nil }
	no := func(string) bool { return false }
	codigo, err := ofrecerReporte(ofrecimiento{Reporte: r, Root: root, Preguntar: no, Enviar: enviar, Salida: &bytes.Buffer{}, Ahora: ahoraReporte})
	if err != nil || codigo != 0 || visto.Titulo != "" || len(consola.PedidosA(root, ahoraReporte)) != 0 {
		t.Fatalf("con «no» no se envía ni queda pedido: %d %v %+v", codigo, err, visto)
	}
	si := func(string) bool { return true }
	var salida bytes.Buffer
	if _, err := ofrecerReporte(ofrecimiento{Reporte: r, Root: root, Preguntar: si, Enviar: enviar, Salida: &salida, Ahora: ahoraReporte}); err != nil {
		t.Fatal(err)
	}
	if visto != r || !strings.Contains(salida.String(), "REP-2") {
		t.Fatalf("con «sí» viaja exactamente lo que se mostró: %+v · %s", visto, salida.String())
	}
}

func TestReportar_SiFallaElEnvioQuedaPendiente(t *testing.T) {
	root, r := entornoReporte(t)
	falla := func(reportes.Reporte) (string, error) { return "", errors.New("sin red") }
	var salida bytes.Buffer
	_, err := ofrecerReporte(ofrecimiento{Reporte: r, Root: root, Preguntar: func(string) bool { return true }, Enviar: falla, Salida: &salida, Ahora: ahoraReporte})
	if err == nil || !strings.Contains(err.Error(), "qdd reportar enviar ") {
		t.Fatalf("el reporte no se pierde: queda pendiente y dice cómo reintentar: %v", err)
	}
}

// contrato: reportes#f33c94d3
func TestReportarEnviar_SoloDesdeElPedidoConfirmado(t *testing.T) {
	nunca := func(string, string) bool { return false }
	if err := autorizarEnvio("abcd1234", "", nil, nunca); err == nil {
		t.Fatal("sin pedido ni terminal no se envía")
	}
	if err := autorizarEnvio("abcd1234", "3", nil, nunca); err == nil {
		t.Fatal("un pedido que la persona no confirmó no envía")
	}
	confirmado := func(p, c string) bool { return p == "3" && c == "qdd reportar enviar abcd1234" }
	if err := autorizarEnvio("abcd1234", "3", nil, confirmado); err != nil {
		t.Fatalf("el pedido confirmado envía: %v", err)
	}
	if err := autorizarEnvio("abcd1234", "", func(string) bool { return true }, nunca); err != nil {
		t.Fatalf("en su terminal, la persona dice sí: %v", err)
	}
}

func TestReportar_NoLoFrenaElGatekeeper(t *testing.T) {
	for _, c := range []*cobra.Command{reportarCmd, reportarEnviarCmd, reportarLlaveCmd, reportarEnviadosCmd} {
		if !deReportar(c) {
			t.Fatalf("%s debe correr aunque el proyecto no tenga config", c.CommandPath())
		}
	}
	if deReportar(rootCmd) {
		t.Fatal("el resto sigue pasando por el gatekeeper")
	}
}

// salidaDe corre f con stdout y stderr a un tubo y devuelve lo que escribió.
func salidaDe(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	viejoOut, viejoErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	defer func() { os.Stdout, os.Stderr = viejoOut, viejoErr }()
	f()
	w.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

// qdd reportar llave muestra la pública y qdd reportar enviados lo enviado; ninguno imprime la llave privada.
// contrato: reportes#a48b9544
func TestReportarLlaveYEnviados_NuncaImprimenLaPrivada(t *testing.T) {
	_, r := entornoReporte(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"REP-9"}`)
	}))
	defer srv.Close()
	if _, err := reportes.Enviar(context.Background(), remoto.ConfigRelay{URL: srv.URL, Clave: "anon"}, r, time.Now()); err != nil {
		t.Fatal(err)
	}
	salida := salidaDe(t, func() {
		reportarLlaveCmd.Run(reportarLlaveCmd, nil)
		reportarEnviadosCmd.Run(reportarEnviadosCmd, nil)
	})
	publica, _ := reportes.Publica()
	if !strings.Contains(salida, publica) || !strings.Contains(salida, "REP-9") {
		t.Fatalf("la prueba necesita la pública y lo enviado en la salida: %q", salida)
	}
	llave, err := reportes.Llave()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{base64.StdEncoding.EncodeToString(llave.Seed()), hex.EncodeToString(llave.Seed()), base64.StdEncoding.EncodeToString(llave), hex.EncodeToString(llave)} {
		if strings.Contains(salida, f) {
			t.Fatal("qdd reportar imprimió la llave privada")
		}
	}
}
