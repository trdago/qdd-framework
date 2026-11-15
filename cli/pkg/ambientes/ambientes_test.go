package ambientes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0023, fase 3 (solo plan): .qdd/ambientes.yaml declara por ambiente la zona de Cloudflare y los registros DNS que
// espera, sin valores secretos. 'qdd ambiente levantar' lee lo que hay en Cloudflare y muestra el plan; no cambia nada.

const archivoDePrueba = `formato: 1
ambientes:
  dev:
    dns:
      zona: ejemplo.cl
      registros:
        - {tipo: CNAME, nombre: app, valor: app-dev.pages.dev, proxied: true}
        - {tipo: TXT, nombre: "@", valor: "v=spf1 include:resend.com ~all"}
        - {tipo: A, nombre: api, valor: 203.0.113.10}
  prod:
    dns:
      zona: ejemplo.cl
      registros:
        - {tipo: CNAME, nombre: www, valor: app.pages.dev, proxied: true}
`

func proyecto(t *testing.T, contenido string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if contenido == "" {
		return root
	}
	if err := os.WriteFile(filepath.Join(root, Ruta), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCargar_LeeLosAmbientesYNormalizaLosNombres(t *testing.T) {
	a, err := Cargar(proyecto(t, archivoDePrueba))
	if err != nil {
		t.Fatal(err)
	}
	dev, ok := a.Ambientes["dev"]
	if !ok || dev.DNS.Zona != "ejemplo.cl" || len(dev.DNS.Registros) != 3 {
		t.Fatalf("lee el ambiente dev: %+v", a)
	}
	nombres := []string{dev.DNS.Registros[0].Nombre, dev.DNS.Registros[1].Nombre}
	if nombres[0] != "app.ejemplo.cl" || nombres[1] != "ejemplo.cl" {
		t.Fatalf("un nombre corto o «@» queda con la zona, como lo devuelve Cloudflare: %v", nombres)
	}
}

func TestCargar_SinArchivoDiceComoEmpezar(t *testing.T) {
	_, err := Cargar(proyecto(t, ""))
	if err == nil || !strings.Contains(err.Error(), Ruta) {
		t.Fatalf("sin archivo, dice cuál crear: %v", err)
	}
}

func TestCargar_RechazaLoQueNoEntiende(t *testing.T) {
	casos := map[string]string{
		"ambiente desconocido": "formato: 1\nambientes:\n  staging:\n    dns: {zona: ejemplo.cl}\n",
		"formato nuevo":        "formato: 9\nambientes: {}\n",
		"tipo raro":            "formato: 1\nambientes:\n  dev:\n    dns:\n      zona: ejemplo.cl\n      registros: [{tipo: SRVX, nombre: a, valor: b}]\n",
		"sin zona":             "formato: 1\nambientes:\n  dev:\n    dns:\n      registros: [{tipo: A, nombre: a, valor: 203.0.113.1}]\n",
		"sin valor":            "formato: 1\nambientes:\n  dev:\n    dns:\n      zona: ejemplo.cl\n      registros: [{tipo: A, nombre: a}]\n",
		"repetido":             "formato: 1\nambientes:\n  dev:\n    dns:\n      zona: ejemplo.cl\n      registros: [{tipo: A, nombre: a, valor: 203.0.113.1}, {tipo: A, nombre: a.ejemplo.cl, valor: 203.0.113.2}]\n",
		"campo desconocido":    "formato: 1\nambientes:\n  dev:\n    token: abc\n",
	}
	for nombre, contenido := range casos {
		if _, err := Cargar(proyecto(t, contenido)); err == nil {
			t.Errorf("%s: se rechaza", nombre)
		}
	}
}

func TestPlanDNS_CreaLoQueFaltaCambiaLoDistintoYNoBorraLoQueNoDeclara(t *testing.T) {
	a, err := Cargar(proyecto(t, archivoDePrueba))
	if err != nil {
		t.Fatal(err)
	}
	actuales := []Registro{
		{Tipo: "CNAME", Nombre: "app.ejemplo.cl", Valor: "app-dev.pages.dev", Proxied: true},
		{Tipo: "TXT", Nombre: "ejemplo.cl", Valor: "v=spf1 ~all"},
		{Tipo: "MX", Nombre: "ejemplo.cl", Valor: "mx.ejemplo.cl"},
	}
	pasos := PlanDNS(a.Ambientes["dev"].DNS, actuales)
	got := map[string]string{}
	for _, p := range pasos {
		got[p.Registro.Tipo+" "+p.Registro.Nombre] = p.Accion
	}
	want := map[string]string{"CNAME app.ejemplo.cl": Igual, "TXT ejemplo.cl": Cambiar, "A api.ejemplo.cl": Crear}
	if len(got) != len(want) {
		t.Fatalf("un paso por registro declarado, nada para el MX que no declara: %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %s, quería %s", k, got[k], v)
		}
	}
	for _, p := range pasos {
		if p.Accion == Cambiar && p.Actual.Valor != "v=spf1 ~all" {
			t.Fatalf("un cambio muestra lo que hay hoy: %+v", p)
		}
	}
}

func TestLectorDNS_LeeLaZonaYSusRegistrosSinEscribir(t *testing.T) {
	var metodos []string
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metodos = append(metodos, r.Method)
		if r.Header.Get("Authorization") != "Bearer tok-secreto" {
			w.WriteHeader(401)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []map[string]any{{"code": 10000, "message": "auth"}}})
			return
		}
		camino := strings.TrimPrefix(r.URL.Path, "/client/v4")
		if camino == "/zones" && r.URL.Query().Get("name") == "ejemplo.cl" {
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []map[string]any{{"id": "z1", "name": "ejemplo.cl"}}})
			return
		}
		if camino == "/zones" {
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []any{}})
			return
		}
		if camino == "/zones/z1/dns_records" {
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []map[string]any{
				{"type": "CNAME", "name": "app.ejemplo.cl", "content": "app-dev.pages.dev", "proxied": true, "ttl": 1},
			}})
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()
	l := LectorDNS{API: s.URL + "/client/v4", Token: "tok-secreto", HTTP: s.Client()}
	regs, err := l.Registros(context.Background(), "ejemplo.cl")
	if err != nil || len(regs) != 1 || regs[0].Nombre != "app.ejemplo.cl" || !regs[0].Proxied {
		t.Fatalf("lee los registros de la zona: %+v %v", regs, err)
	}
	for _, m := range metodos {
		if m != http.MethodGet {
			t.Fatalf("solo plan: nunca escribe en Cloudflare (%s)", m)
		}
	}
	if _, err := l.Registros(context.Background(), "otra.cl"); err == nil || !strings.Contains(err.Error(), "otra.cl") {
		t.Fatalf("una zona que el token no ve se nombra: %v", err)
	}
	malo := LectorDNS{API: l.API, Token: "otro-token-malo", HTTP: s.Client()}
	if _, err := malo.Registros(context.Background(), "ejemplo.cl"); err == nil || strings.Contains(err.Error(), "otro-token-malo") {
		t.Fatalf("un rechazo se dice sin mostrar el token: %v", err)
	}
}
