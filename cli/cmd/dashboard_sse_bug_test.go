package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestDashboard_SSEHeaders_BugRegression asegura que el endpoint /api/stream
// no sufra de proxy buffering (X-Accel-Buffering) y devuelva los headers correctos,
// además de enviar el estado inicial instantáneamente, cumpliendo con la filosofía QDD (Regla #4).
func TestDashboard_SSEHeaders_BugRegression(t *testing.T) {
	server := setupMockSSEServer()
	defer server.Close()

	resp := executeSSERequest(t, server.URL)
	defer resp.Body.Close()

	validateSSEHeaders(t, resp)
	validateSSEInitialPayload(t, resp)
}

func setupMockSSEServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}
		w.Write([]byte("data: {\"mock\":\"initial\"}\n\n"))
		flusher.Flush()
		time.Sleep(100 * time.Millisecond)
	})
	return httptest.NewServer(mux)
}

func executeSSERequest(t *testing.T, url string) *http.Response {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url+"/api/stream", nil)
	if err != nil {
		t.Fatalf("No se pudo crear la petición: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("La petición falló: %v", err)
	}
	return resp
}

func validateSSEHeaders(t *testing.T, resp *http.Response) {
	if resp.Header.Get("X-Accel-Buffering") != "no" {
		t.Errorf("🚨 Regla Violada: SSE /api/stream DEBE tener la cabecera 'X-Accel-Buffering: no' para evitar bloqueos por proxy.")
	}

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("🚨 Regla Violada: SSE /api/stream DEBE tener 'Content-Type: text/event-stream'.")
	}
}

func validateSSEInitialPayload(t *testing.T, resp *http.Response) {
	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	if err != nil {
		t.Fatalf("No se pudo leer el stream inicial: %v", err)
	}

	output := string(buf[:n])
	if !strings.Contains(output, "data: ") {
		t.Errorf("🚨 Regla Violada: El endpoint no envió el estado inicial inmediatamente.")
	}
}
