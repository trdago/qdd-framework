package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/cognitive"
	"github.com/qdd-framework/qdd/pkg/dashboard"
	"github.com/qdd-framework/qdd/ui"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"ui"},
	Short:   "Inicia el Centro de Comando Web (Frontend embebido)",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 Preparando QDD Dashboard...")

		go dashboard.RunWatcher()

		// Servir archivos estáticos embebidos (deshabilitando caché para evitar que el navegador guarde versiones antiguas de la UI)
		distFs, err := fs.Sub(ui.StaticFiles, "dist")
		if err != nil {
			fmt.Println("Error accediendo a los archivos estáticos de UI:", err)
			return
		}
		fileServer := http.FileServer(http.FS(distFs))
		http.HandleFunc("/", safeHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			fileServer.ServeHTTP(w, r)
		}))

		// Endpoint de API REST (Legacy/Fallback)
		http.HandleFunc("/api/state", safeHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
			response := dashboard.BuildState()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))

		// Endpoint SSE (Server-Sent Events) para Real-Time con Contrato Estricto
		http.HandleFunc("/api/stream", safeHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "SSE not supported", http.StatusInternalServerError)
				return
			}

			ch := dashboard.Broker.AddClient()
			defer dashboard.Broker.RemoveClient(ch)

			// Enviar estado inicial de inmediato para evitar que el UI se quede en mock si fsnotify falla (ej. Docker/WSL)
			initialState := dashboard.BuildState()
			initData, _ := json.Marshal(initialState)
			fmt.Fprintf(w, "data: %s\n\n", string(initData))
			flusher.Flush()

			for {
				select {
				case <-r.Context().Done():
					return
				case data := <-ch:
					fmt.Fprintf(w, "data: %s\n\n", string(data))
					flusher.Flush()
				}
			}
		}))

		http.HandleFunc("/api/policies", safeHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				cwd, _ := os.Getwd()
				p := audit.LoadPolicies(cwd)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(p)
				return
			}
			if r.Method == http.MethodPost {
				cwd, _ := os.Getwd()
				var p audit.QDDPolicies
				if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				audit.SavePolicies(cwd, p)

				// Re-run QDD Certify silently to update DB state with new rules
				exec.CommandContext(r.Context(), os.Args[0], "certify").Run()

				// Broadcast new state
				res := dashboard.BuildState()
				data, _ := json.Marshal(res)
				dashboard.Broker.Broadcast(data)

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}))

		type IntentRequest struct {
			Input string `json:"input"`
		}

		http.HandleFunc("/api/intent", safeHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req IntentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			cwd, _ := os.Getwd()
			execResult, err := cognitive.ExecuteLocalIntent(r.Context(), cwd, req.Input)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(execResult)
				return
			}

			// Broadcast updated state to all connected SSE clients
			res := dashboard.BuildState()
			data, _ := json.Marshal(res)
			dashboard.Broker.Broadcast(data)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(execResult)
		}))

		http.HandleFunc("/api/agent/execute", safeHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req IntentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			cwd, _ := os.Getwd()
			execResult, _ := cognitive.ExecuteLocalIntent(r.Context(), cwd, req.Input)

			res := dashboard.BuildState()
			data, _ := json.Marshal(res)
			dashboard.Broker.Broadcast(data)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(execResult)
		}))

		type FileRequest struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}

		http.HandleFunc("/api/file", safeHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req FileRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			cwd, _ := os.Getwd()
			targetPath := filepath.Join(cwd, req.Path)
			if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(cwd)) {
				http.Error(w, "Invalid path", http.StatusForbidden)
				return
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := os.WriteFile(targetPath, []byte(req.Content), 0644); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))

		port := 8099
		var listener net.Listener
		for {
			addr := fmt.Sprintf(":%d", port)
			listener, err = net.Listen("tcp", addr)
			if err == nil {
				break
			}
			port++
		}

		dashboardURL := fmt.Sprintf("http://localhost:%d", port)
		fmt.Printf("✅ Servidor listo y escuchando en %s\n", dashboardURL)

		go func() {
			http.Serve(listener, nil)
		}()

		if err := openBrowser(dashboardURL); err != nil {
			fmt.Printf("Por favor abre manualmente: %s\n", dashboardURL)
		}

		// Esperar señal de interrupción (Ctrl+C) para apagar el servidor
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		fmt.Println("\nApagando QDD Dashboard de forma segura. ¡Hasta pronto!")
	},
}

func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}

func safeHTTPHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("[🛑 DASHBOARD RECOVERY - ZERO PANIC] Pánico interceptado en endpoint %s: %v\n", r.URL.Path, rec)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":  fmt.Sprintf("Zero-Panic Guard: Error interno recuperado: %v", rec),
					"status": "PANIC_RECOVERED",
				})
			}
		}()
		next(w, r)
	}
}
