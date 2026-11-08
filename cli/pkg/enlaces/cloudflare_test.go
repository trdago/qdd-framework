package enlaces

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 3b (propuesta de la persona, 2026-11-08): con un token de Cloudflare del baúl, el flujo se comparte
// con un enlace. qdd instala un Worker en la cuenta de la persona, guarda cada página en KV y la borra al quitarla.
// Las pruebas hablan con un Cloudflare de mentira: nunca con la cuenta real.

type cloudflareFalso struct {
	mu         sync.Mutex
	cuentas    []string
	subdominio string
	espacios   []string
	creados    int
	script     string
	metadatos  map[string]any
	valores    map[string]string
	activado   bool
	llamadas   []string
}

func (c *cloudflareFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.llamadas = append(c.llamadas, r.Method+" "+r.URL.Path)
	if r.Header.Get("Authorization") != "Bearer tok" {
		w.WriteHeader(403)
		io.WriteString(w, `{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`)
		return
	}
	ok := func(resultado any) {
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": resultado})
	}
	camino := strings.TrimPrefix(r.URL.Path, "/client/v4")
	switch {
	case camino == "/accounts":
		lista := []map[string]string{}
		for _, id := range c.cuentas {
			lista = append(lista, map[string]string{"id": id, "name": "Cuenta " + id})
		}
		ok(lista)
	case camino == "/accounts/a1/workers/subdomain":
		if c.subdominio == "" {
			w.WriteHeader(404)
			io.WriteString(w, `{"success":false,"errors":[{"code":10007,"message":"no subdomain"}]}`)
			return
		}
		ok(map[string]string{"subdomain": c.subdominio})
	case camino == "/accounts/a1/storage/kv/namespaces" && r.Method == "GET":
		lista := []map[string]string{}
		for i, t := range c.espacios {
			lista = append(lista, map[string]string{"id": "ns" + string(rune('0'+i)), "title": t})
		}
		ok(lista)
	case camino == "/accounts/a1/storage/kv/namespaces" && r.Method == "POST":
		var cuerpo map[string]string
		json.NewDecoder(r.Body).Decode(&cuerpo)
		c.espacios = append(c.espacios, cuerpo["title"])
		c.creados++
		ok(map[string]string{"id": "ns" + string(rune('0'+len(c.espacios)-1)), "title": cuerpo["title"]})
	case camino == "/accounts/a1/workers/scripts/"+NombreWorker && r.Method == "PUT":
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		partes := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := partes.NextPart()
			if err != nil {
				break
			}
			b, _ := io.ReadAll(p)
			if p.FormName() == "metadata" {
				json.Unmarshal(b, &c.metadatos)
				continue
			}
			c.script = string(b)
		}
		ok(map[string]string{"id": NombreWorker})
	case camino == "/accounts/a1/workers/scripts/"+NombreWorker+"/subdomain":
		c.activado = true
		ok(map[string]bool{"enabled": true})
	case strings.HasPrefix(camino, "/accounts/a1/storage/kv/namespaces/ns0/values/"):
		clave := strings.TrimPrefix(camino, "/accounts/a1/storage/kv/namespaces/ns0/values/")
		if r.Method == "DELETE" {
			delete(c.valores, clave)
			ok(nil)
			return
		}
		b, _ := io.ReadAll(r.Body)
		c.valores[clave] = string(b)
		ok(nil)
	default:
		w.WriteHeader(404)
		io.WriteString(w, `{"success":false,"errors":[{"code":7003,"message":"no route"}]}`)
	}
}

func conFalso(t *testing.T, c *cloudflareFalso) Cuenta {
	t.Helper()
	if c.valores == nil {
		c.valores = map[string]string{}
	}
	s := httptest.NewTLSServer(c)
	t.Cleanup(s.Close)
	return Cuenta{API: s.URL + "/client/v4", Token: "tok", HTTP: s.Client()}
}

func TestPreparar_InstalaElWorkerConSuKVYDaSuDireccion(t *testing.T) {
	falso := &cloudflareFalso{cuentas: []string{"a1"}, subdominio: "ana"}
	c := conFalso(t, falso)
	sitio, err := c.Preparar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sitio.Base != "https://"+NombreWorker+".ana.workers.dev" || sitio.Cuenta != "a1" || sitio.Espacio != "ns0" {
		t.Fatalf("sitio: %+v", sitio)
	}
	if falso.creados != 1 || !falso.activado || !strings.Contains(falso.script, "FLUJOS.get") {
		t.Fatalf("crea el KV, sube el Worker y lo publica en workers.dev: %+v", falso)
	}
	enlaces, _ := falso.metadatos["bindings"].([]any)
	if len(enlaces) != 1 || !strings.Contains(toJSON(enlaces[0]), `"namespace_id":"ns0"`) || falso.metadatos["main_module"] != "worker.js" {
		t.Fatalf("el Worker lee el KV por su binding: %+v", falso.metadatos)
	}
	if _, err := c.Preparar(context.Background()); err != nil || falso.creados != 1 {
		t.Fatalf("la segunda vez usa el mismo KV: %v, creados %d", err, falso.creados)
	}
}

func TestPreparar_DiceQueFaltaSinSubdominioOConVariasCuentas(t *testing.T) {
	c := conFalso(t, &cloudflareFalso{cuentas: []string{"a1"}})
	if _, err := c.Preparar(context.Background()); err == nil || !strings.Contains(err.Error(), "workers.dev") {
		t.Fatalf("sin subdominio, dice dónde activarlo: %v", err)
	}
	c = conFalso(t, &cloudflareFalso{cuentas: []string{"a1", "b2"}, subdominio: "ana"})
	if _, err := c.Preparar(context.Background()); err == nil || !strings.Contains(err.Error(), "a1") || !strings.Contains(err.Error(), "b2") {
		t.Fatalf("con dos cuentas, las nombra para elegir: %v", err)
	}
	c.CuentaID = "a1"
	if _, err := c.Preparar(context.Background()); err != nil {
		t.Fatalf("con la cuenta elegida, sigue: %v", err)
	}
	c = conFalso(t, &cloudflareFalso{cuentas: []string{"a1"}, subdominio: "ana"})
	c.Token = "otro"
	if _, err := c.Preparar(context.Background()); err == nil || !strings.Contains(err.Error(), "Authentication error") {
		t.Fatalf("un token rechazado se dice con el error de Cloudflare: %v", err)
	}
}

func TestSubirYQuitar_LaPaginaVaAKVPorSuID(t *testing.T) {
	falso := &cloudflareFalso{cuentas: []string{"a1"}, subdominio: "ana"}
	c := conFalso(t, falso)
	sitio, _ := c.Preparar(context.Background())
	id, err := NuevoID()
	otro, _ := NuevoID()
	if err != nil || len(id) != 26 || !idValido.MatchString(id) || id == otro {
		t.Fatalf("un id al azar, de 26 letras: %q", id)
	}
	url, err := c.Subir(context.Background(), sitio, id, []byte("<!doctype html>hola"))
	if err != nil || url != sitio.Base+"/f/"+id || falso.valores["f%2F"+id] != "<!doctype html>hola" && falso.valores["f/"+id] != "<!doctype html>hola" {
		t.Fatalf("sube: %q %v %+v", url, err, falso.valores)
	}
	if err := c.Quitar(context.Background(), sitio, id); err != nil || len(falso.valores) != 0 {
		t.Fatalf("quita: %v %+v", err, falso.valores)
	}
	if _, err := c.Subir(context.Background(), sitio, "../otro", nil); err == nil {
		t.Fatal("un id que no generó qdd no se sube")
	}
}

func TestWorker_SoloSirveLasPaginasPorSuIDYSinCache(t *testing.T) {
	for _, debe := range []string{`/^\/f\/([a-z2-7]{26})$/`, "no-store", "noindex", "no-referrer", "nosniff", "DENY", "'f/' + m[1]"} {
		if !strings.Contains(ScriptWorker, debe) {
			t.Fatalf("el Worker no tiene %q:\n%s", debe, ScriptWorker)
		}
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
