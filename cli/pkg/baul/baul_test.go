package baul

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/zalando/go-keyring"
)

// Rule: Prohibido usar 'else'
// RFC 0023, fase 1. Ninguna prueba toca el llavero ni el 1Password reales: llavero de prueba, op falso, QDD_HOME y
// HOME temporales.

var ahora = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

const secreto = "cf-SECRETO-1234567890"

var raizActual string

// latir: la sesión s1 está abierta en el proyecto (el panel late por ella).
func latir(t *testing.T) {
	t.Helper()
	if err := consola.LatirSesion(raizActual, "s1", ahora); err != nil {
		t.Fatal(err)
	}
}

func entorno(t *testing.T) string {
	t.Helper()
	keyring.MockInit()
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	raizActual = root
	return root
}

func guardarCloudflare(t *testing.T) {
	t.Helper()
	latir(t)
	if err := Guardar(Entrada{Nombre: "cf", Tipo: "cloudflare", Donde: EnLlavero}, secreto, ahora); err != nil {
		t.Fatal(err)
	}
}

func TestGuardar_ElIndiceNoTieneElValor(t *testing.T) {
	entorno(t)
	guardarCloudflare(t)
	ruta, _ := rutaEn("indice.json")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), secreto) || !strings.Contains(string(b), `"cf"`) {
		t.Fatalf("el índice anota el token sin su valor: %s", b)
	}
	e, err := Buscar("cf")
	if err != nil || e.Variable != "CLOUDFLARE_API_TOKEN" {
		t.Fatalf("la variable sale del tipo: %+v %v", e, err)
	}
}

func TestGuardar_TipoDesconocidoYOtroSinVariable(t *testing.T) {
	entorno(t)
	if err := Guardar(Entrada{Nombre: "x", Tipo: "nada", Donde: EnLlavero}, "v", ahora); err == nil {
		t.Fatal("un tipo desconocido se rechaza")
	}
	if err := Guardar(Entrada{Nombre: "x", Tipo: "otro", Donde: EnLlavero}, "v", ahora); err == nil {
		t.Fatal("«otro» necesita la variable")
	}
	if err := Guardar(Entrada{Nombre: "../x", Tipo: "github", Donde: EnLlavero}, "v", ahora); err == nil {
		t.Fatal("el nombre es simple")
	}
}

func opcionesUsar(root string, salida *bytes.Buffer, args ...string) OpcionesUsar {
	return OpcionesUsar{Root: root, Nombre: "cf", Fin: "crear la zona de ejemplo.cl", Sesion: "s1", Comando: args, Salida: salida, Ahora: ahora}
}

func TestUsar_SinPermisoDejaElPedidoDeLaPersonaYNoCorre(t *testing.T) {
	root := entorno(t)
	guardarCloudflare(t)
	marca := filepath.Join(root, "corrio")
	var salida bytes.Buffer
	_, err := Usar(opcionesUsar(root, &salida, "touch", marca))
	if !errors.Is(err, ErrSinPermiso) {
		t.Fatalf("sin permiso: %v", err)
	}
	if _, err := os.Stat(marca); err == nil {
		t.Fatal("sin permiso no corre nada")
	}
	pedidos := consola.PedidosA(root, ahora)
	if len(pedidos) != 1 || !pedidos[0].DePersona || !strings.HasPrefix(pedidos[0].Comando, "qdd baul permitir ") {
		t.Fatalf("queda un pedido de la persona: %+v", pedidos)
	}
	if !strings.Contains(pedidos[0].Motivo, "cf") {
		t.Fatalf("el pedido dice qué token: %q", pedidos[0].Motivo)
	}
	again, _ := Usar(opcionesUsar(root, &salida, "touch", marca))
	if again != CodigoTodaviaNo || len(consola.PedidosA(root, ahora)) != 1 {
		t.Fatal("pedirlo otra vez no duplica el pedido")
	}
}

func permitirTodo(t *testing.T, root string) {
	t.Helper()
	p := consola.PedidosA(root, ahora)[0]
	id := strings.TrimPrefix(p.Comando, "qdd baul permitir ")
	confirmado := func(pedido, comando string) bool { return pedido == p.ID && comando == p.Comando }
	if err := Permitir(root, id, p.ID, confirmado, ahora); err != nil {
		t.Fatal(err)
	}
}

func TestPermitir_SinElSiDeLaPersonaNoPermite(t *testing.T) {
	root := entorno(t)
	guardarCloudflare(t)
	var salida bytes.Buffer
	Usar(opcionesUsar(root, &salida, "true"))
	p := consola.PedidosA(root, ahora)[0]
	id := strings.TrimPrefix(p.Comando, "qdd baul permitir ")
	nunca := func(string, string) bool { return false }
	if err := Permitir(root, id, p.ID, nunca, ahora); err == nil {
		t.Fatal("sin el «Sí» de la persona en el panel no hay permiso")
	}
	if err := Permitir(root, id, "", nunca, ahora); err == nil {
		t.Fatal("fuera de un pedido no hay permiso")
	}
}

func TestUsar_ConPermisoCorreConElTokenYLoTapa(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("usa sh")
	}
	root := entorno(t)
	guardarCloudflare(t)
	var salida bytes.Buffer
	Usar(opcionesUsar(root, &salida, "true"))
	permitirTodo(t, root)
	salida.Reset()
	codigo, err := Usar(opcionesUsar(root, &salida, "sh", "-c", `printf 'token=%s\n' "$CLOUDFLARE_API_TOKEN"; printf '%s' "$CLOUDFLARE_API_TOKEN" >&2`))
	if err != nil || codigo != 0 {
		t.Fatalf("con permiso corre: %d %v %s", codigo, err, salida.String())
	}
	if strings.Contains(salida.String(), secreto) || !strings.Contains(salida.String(), "token="+Tapado) {
		t.Fatalf("la salida tapa el valor: %q", salida.String())
	}
	reg, _ := rutaEn("registro.jsonl")
	b, _ := os.ReadFile(reg)
	if strings.Contains(string(b), secreto) || !strings.Contains(string(b), "crear la zona de ejemplo.cl") {
		t.Fatalf("el registro anota el uso, nunca el valor: %s", b)
	}
	otroFin := opcionesUsar(root, &salida, "true")
	otroFin.Fin = "borrar la zona"
	if _, err := Usar(otroFin); !errors.Is(err, ErrSinPermiso) {
		t.Fatalf("un fin distinto es otro permiso: %v", err)
	}
}

func TestUsar_ElPermisoMuereAlCerrarLaSesion(t *testing.T) {
	root := entorno(t)
	guardarCloudflare(t)
	var salida bytes.Buffer
	Usar(opcionesUsar(root, &salida, "true"))
	permitirTodo(t, root)
	if codigo, err := Usar(opcionesUsar(root, &salida, "true")); err != nil || codigo != 0 {
		t.Fatalf("vale mientras la sesión vive: %d %v", codigo, err)
	}
	if err := consola.CerrarSesion(root, "s1", ahora); err != nil {
		t.Fatal(err)
	}
	if codigo, err := Usar(opcionesUsar(root, &salida, "true")); err == nil || codigo == 0 {
		t.Fatalf("cerrada la consola, el permiso murió: %d %v", codigo, err)
	}
	if err := consola.LatirSesion(root, "s1", ahora); err != nil {
		t.Fatal(err)
	}
	if _, err := Usar(opcionesUsar(root, &salida, "true")); !errors.Is(err, ErrSinPermiso) {
		t.Fatalf("y no vuelve aunque la sesión lata de nuevo: %v", err)
	}
}

func TestUsar_UnaSesionQueNoLateNoPuedePedir(t *testing.T) {
	root := entorno(t)
	guardarCloudflare(t)
	var salida bytes.Buffer
	o := opcionesUsar(root, &salida, "true")
	o.Sesion = "s9"
	if _, err := Usar(o); err == nil || errors.Is(err, ErrSinPermiso) || !strings.Contains(err.Error(), "consola") {
		t.Fatalf("sin el panel no hay dónde pedir ni cuánto dura: %v", err)
	}
	if len(consola.PedidosA(root, ahora)) != 0 {
		t.Fatal("no deja un pedido que nadie verá")
	}
}

func TestUsar_SinSesionNiTerminalSeNiega(t *testing.T) {
	root := entorno(t)
	guardarCloudflare(t)
	var salida bytes.Buffer
	o := opcionesUsar(root, &salida, "true")
	o.Sesion = ""
	if _, err := Usar(o); err == nil || errors.Is(err, ErrSinPermiso) {
		t.Fatalf("sin sesión de Claude Code ni terminal no hay a quién pedirle: %v", err)
	}
}

func TestUsar_EnLaTerminalDeLaPersonaPreguntaAhi(t *testing.T) {
	root := entorno(t)
	guardarCloudflare(t)
	var salida bytes.Buffer
	o := opcionesUsar(root, &salida, "true")
	o.Sesion = ""
	o.Terminal = &Terminal{PID: os.Getpid(), Preguntar: func(string) bool { return true }}
	if codigo, err := Usar(o); err != nil || codigo != 0 {
		t.Fatalf("la persona dice sí en su terminal: %d %v", codigo, err)
	}
	o.Terminal.Preguntar = func(string) bool { t.Fatal("ya tiene permiso en esta terminal"); return false }
	if codigo, err := Usar(o); err != nil || codigo != 0 {
		t.Fatalf("el permiso dura lo que la terminal: %d %v", codigo, err)
	}
}

func TestRegistro_BorrarUnaLineaSeDetecta(t *testing.T) {
	entorno(t)
	for _, fin := range []string{"uno", "dos", "tres"} {
		if err := Anotar(Linea{Accion: "usar", Token: "cf", Fin: fin}, ahora); err != nil {
			t.Fatal(err)
		}
	}
	if err := VerificarRegistro(); err != nil {
		t.Fatalf("intacto: %v", err)
	}
	ruta, _ := rutaEn("registro.jsonl")
	b, _ := os.ReadFile(ruta)
	lineas := strings.Split(strings.TrimSpace(string(b)), "\n")
	if err := os.WriteFile(ruta, []byte(lineas[0]+"\n"+lineas[2]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerificarRegistro(); err == nil || !strings.Contains(err.Error(), "línea 2") {
		t.Fatalf("falta una línea y se nota dónde: %v", err)
	}
}

func TestOcultador_TapaElValorAunqueLlegueCortado(t *testing.T) {
	var out bytes.Buffer
	o := NuevoOcultador(&out, []string{secreto})
	o.Write([]byte("antes " + secreto[:6]))
	o.Write([]byte(secreto[6:] + " después\notra línea "))
	o.Write([]byte(secreto))
	o.Close()
	if strings.Contains(out.String(), secreto) || out.String() != "antes "+Tapado+" después\notra línea "+Tapado {
		t.Fatalf("salida: %q", out.String())
	}
}

func TestVerificar_CloudflareYRechazado(t *testing.T) {
	entorno(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secreto {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"success":true,"result":{"id":"abc","status":"active","expires_on":"2027-01-01T00:00:00Z"}}`))
	}))
	defer srv.Close()
	t.Cleanup(func(u string) func() { return func() { apis["cloudflare"] = u } }(apis["cloudflare"]))
	apis["cloudflare"] = srv.URL
	v, err := VerificarValor("cloudflare", secreto)
	if err != nil || v.Vence == nil || v.Vence.Year() != 2027 || v.Alcance == "" {
		t.Fatalf("verificado: %+v %v", v, err)
	}
	if _, err := VerificarValor("cloudflare", "malo"); !errors.Is(err, ErrRechazado) {
		t.Fatalf("un token que la API rechaza: %v", err)
	}
}

func TestVerificar_GithubAnotaDuenioYAlcances(t *testing.T) {
	entorno(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-OAuth-Scopes", "repo, workflow")
		w.Write([]byte(`{"login":"trdago"}`))
	}))
	defer srv.Close()
	t.Cleanup(func(u string) func() { return func() { apis["github"] = u } }(apis["github"]))
	apis["github"] = srv.URL
	v, err := VerificarValor("github", "gh")
	if err != nil || v.Duenio != "trdago" || v.Alcance != "repo, workflow" {
		t.Fatalf("%+v %v", v, err)
	}
}

func TestUnoPassword_GuardaYLeeConOp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("op falso en sh")
	}
	entorno(t)
	dir := t.TempDir()
	op := filepath.Join(dir, "op")
	guion := "#!/bin/sh\nset -e\ncase \"$1 $2\" in\n\"item create\") cat > " + filepath.Join(dir, "item.json") + "; echo '{}' ;;\n\"item get\") grep -o 'cf-[A-Z0-9-]*' " + filepath.Join(dir, "item.json") + " ;;\n*) exit 2 ;;\nesac\n"
	if err := os.WriteFile(op, []byte(guion), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func(v string) func() { return func() { binOp = v } }(binOp))
	binOp = op
	if err := Guardar(Entrada{Nombre: "cf", Tipo: "cloudflare", Donde: EnUnoPass}, secreto, ahora); err != nil {
		t.Fatal(err)
	}
	argv, _ := os.ReadFile(filepath.Join(dir, "item.json"))
	if !strings.Contains(string(argv), secreto) {
		t.Fatalf("el valor va por la entrada de op, no por sus argumentos: %s", argv)
	}
	e, _ := Buscar("cf")
	v, err := leerValor(e)
	if err != nil || v != secreto {
		t.Fatalf("se lee de 1Password: %q %v", v, err)
	}
}

func TestEnvs_ListaLasClavesConValorSinImprimirlos(t *testing.T) {
	root := t.TempDir()
	contenido := "# comentario\nexport CLOUDFLARE_API_TOKEN=" + secreto + "\nPORT=3000\nSUPABASE_KEY=\nAPI_SECRET=\"otro-valor\"\n"
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env.example"), []byte("API_SECRET=xxx\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	hallados := EnvsConValores(root)
	if len(hallados) != 1 || hallados[0].Archivo != ".env" {
		t.Fatalf("solo el .env real: %+v", hallados)
	}
	claves := strings.Join(hallados[0].Claves, ",")
	if claves != "CLOUDFLARE_API_TOKEN,API_SECRET" {
		t.Fatalf("las claves de secretos con valor, sin PORT ni las vacías: %q", claves)
	}
	if strings.Contains(claves, secreto) || strings.Contains(claves, "otro-valor") {
		t.Fatal("nunca un valor")
	}
}

// RFC 0024: un reporte a QDD tapa los valores del baúl; se leen solo los del llavero (1Password pediría desbloquear).
func TestValoresDelLlavero_SoloLosDelLlavero(t *testing.T) {
	entorno(t)
	guardarCloudflare(t)
	valores := ValoresDelLlavero()
	if len(valores) != 1 || valores[0] != secreto {
		t.Fatalf("debe dar el valor del llavero para taparlo: %d valores", len(valores))
	}
}
