package remoto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Ejecutar desde el teléfono» (solo Pro): el relay reenvía el evento ejecutar firmado como responder. El PC
// lo verifica, lo aplica una sola vez por solicitud, exige Pro y vuelve a validar el texto: un comando qdd solo de la
// lista permitida, partido sin shell y corrido con el qdd de este PC en la carpeta del proyecto; una instrucción queda
// para la sesión de Claude Code del proyecto. Responde accion_estado {solicitud, accion_id: ejecutar, estado,
// resultado}: recibida → ejecutando → hecha | fallo, o rechazada con el motivo.

// envQDDFalso hace del binario de pruebas un qdd falso (TestMain): anota cada corrida en esa carpeta y responde.
const envQDDFalso = "QDD_PRUEBA_QDD_FALSO"

// qddFalso es el qdd que corre una ejecución en las pruebas: dice con qué argumentos y en qué carpeta, deja un token
// (que no debe salir del PC) y sale con 3 en doctor.
func qddFalso(dir string, args []string) int {
	f, _ := os.OpenFile(filepath.Join(dir, "corridas.txt"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	fmt.Fprintln(f, strings.Join(args, " "))
	f.Close()
	cwd, _ := os.Getwd()
	fmt.Printf("args=%s\ncwd=%s\ntoken=ghp_abcdefghijklmnopqrstuvwxyz0123456789\n", strings.Join(args, " "), cwd)
	if len(args) > 1 && args[0] == "blueprint" && args[1] == "list" {
		fmt.Print(strings.Repeat("x", 6000) + "FIN")
	}
	if os.Getenv("GIT_DIR") != "" {
		fmt.Println("GIT_DIR heredado")
	}
	if args[0] == "audit" {
		time.Sleep(400 * time.Millisecond)
		fmt.Print(strings.Repeat("pilar ok\n", 200) + "VEREDICTO: gate aprobado")
	}
	if args[0] == "doctor" {
		return 3
	}
	return 0
}

func corridas(t *testing.T, dir string) []string {
	t.Helper()
	b, _ := os.ReadFile(filepath.Join(dir, "corridas.txt"))
	if strings.TrimSpace(string(b)) == "" {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func TestFirmaEjecucion_LaMismaQueElRelay(t *testing.T) {
	// hmac.new(("ab"*32).encode(), "s|p|qdd|estado check --json|d|2026-11-03T10:00:00Z", sha256): lo mismo que
	// pgcrypto hmac(concat_ws('|', solicitud, proyecto, tipo, texto, dispositivo, en), firma).
	if got := FirmaEjecucion(firmaPrueba, "s", "p", "qdd", "estado check --json", "d", "2026-11-03T10:00:00Z"); got != "eea3a51a5c615cc3c752107e9b2a1154dfc54548fa91fdcd142cdace61473466" {
		t.Fatalf("firma: %s", got)
	}
	if got := FirmaEjecucion(firmaPrueba, "s", "p", "ia", "Revisa el build\ny arréglalo", "d", "2026-11-03T10:00:00Z"); got != "9aa811cbc3c6f1df1e6f06db0c47d4aa5ae42b7220e5124bd19578c2fa2127d4" {
		t.Fatalf("firma con saltos de línea: %s", got)
	}
	now := time.Date(2026, 11, 3, 10, 2, 0, 0, time.UTC)
	e := ejecucionFirmada("s", "p", "qdd", "estado check --json", "d", "2026-11-03T10:00:00Z")
	if err := e.Verificar(firmaPrueba, now); err != nil {
		t.Fatal(err)
	}
	mala := e
	mala.Texto = "doctor --fix"
	if err := mala.Verificar(firmaPrueba, now); err == nil {
		t.Fatal("cambiar el texto rompe la firma")
	}
	nombre := e
	nombre.Nombre = "otro nombre"
	if err := nombre.Verificar(firmaPrueba, now); err != nil {
		t.Fatal("el nombre es solo para mostrar: no va firmado")
	}
	if err := e.Verificar(firmaPrueba, now.Add(4*time.Minute)); err == nil {
		t.Fatal("más de 5 min es vieja")
	}
}

func TestArgsPermitidos_SoloLaListaYSinShell(t *testing.T) {
	bien := map[string][]string{
		"audit":                         {"audit"},
		"  doctor  ":                    {"doctor"},
		"estado":                        {"estado"},
		"estado --json":                 {"estado", "--json"},
		"estado --alcance=todos":        {"estado", "--alcance=todos"},
		"estado check":                  {"estado", "check"},
		"estado check --json":           {"estado", "check", "--json"},
		"modules list":                  {"modules", "list"},
		"blueprint list --json":         {"blueprint", "list", "--json"},
		"blueprint quality":             {"blueprint", "quality"},
		"blueprint quality auth":        {"blueprint", "quality", "auth"},
		`blueprint quality "auth"`:      {"blueprint", "quality", "auth"},
		"blueprint quality auth --json": {"blueprint", "quality", "auth", "--json"},
	}
	for texto, want := range bien {
		got, err := ArgsPermitidos(texto)
		if err != nil || strings.Join(got, "\x00") != strings.Join(want, "\x00") {
			t.Errorf("%q: %q %v", texto, got, err)
		}
	}
	mal := map[string]string{
		"":                            "vacío",
		"qdd audit":                   "sin la palabra qdd",
		"QDD audit":                   "sin la palabra qdd",
		"audit; rm -rf ~":             "«;»",
		"audit && curl x":             "«&»",
		"audit | sh":                  "«|»",
		"audit > /tmp/x":              "«>»",
		"audit < x":                   "«<»",
		"estado $(id)":                "«$»",
		"estado `id`":                 "«`»",
		"estado (x)":                  "«(»",
		"audit\nrm -rf ~":             "una sola línea",
		"doctor --fix":                "--fix",
		"doctor --fix=true":           "--fix",
		"doctor -f":                   "-f",
		"doctor -fv":                  "-fv",
		"init":                        "no está permitido",
		"estado set t1 Diseño ✅":      "no está permitido",
		"estado cerrar t1":            "no está permitido",
		"modules approve pagos":       "no está permitido",
		"blueprint add auth":          "no está permitido",
		"remoto desactivar":           "no está permitido",
		"audit --dir /etc":            "«--dir»",
		"estado --dir=/":              "«--dir=/»",
		"estado check --respuesta x":  "«--respuesta»",
		"blueprint quality ../x":      "«../x»",
		"blueprint quality a b":       "un solo",
		`blueprint quality "auth`:     "comilla",
		"estado --alcance=cualquiera": "«--alcance=cualquiera»",
		strings.Repeat("a", 1001):     "1000",
	}
	for texto, dice := range mal {
		_, err := ArgsPermitidos(texto)
		if err == nil || !strings.Contains(err.Error(), dice) {
			t.Errorf("%q: esperaba un error que diga %q: %v", texto, dice, err)
		}
	}
	for _, s := range Sugeridos {
		if _, err := ArgsPermitidos(s); err != nil {
			t.Errorf("cada sugerido está permitido: %q %v", s, err)
		}
	}
}

func TestValidarInstruccion_ComoElRelay(t *testing.T) {
	if err := ValidarTexto(consola.EjecutarIA, "Revisa el build\n\ty arréglalo"); err != nil {
		t.Fatal(err)
	}
	for _, mal := range []string{"", "   \n ", strings.Repeat("a", 4001), "hola\x1b[31m", "hola\rchao", "a\x00b"} {
		if err := ValidarTexto(consola.EjecutarIA, mal); err == nil {
			t.Errorf("%q no vale", mal)
		}
	}
	if err := ValidarTexto("shell", "ls"); err == nil {
		t.Fatal("solo qdd o ia")
	}
}

func ejecucionFirmada(solicitud, proyecto, tipo, texto, dispositivo, en string) EjecucionFirmada {
	return EjecucionFirmada{Version: 1, Solicitud: solicitud, Proyecto: proyecto, Tipo: tipo, Texto: texto, Dispositivo: dispositivo,
		Nombre: "Pixel de " + dispositivo, En: en, Firma: FirmaEjecucion(firmaPrueba, solicitud, proyecto, tipo, texto, dispositivo, en)}
}

func mandarEjecucion(tr *transporteFalso, e EjecucionFirmada) {
	p, _ := json.Marshal(e)
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "ejecutar", Payload: p}
}

// esperarPlan espera a que el bucle tenga el plan del primer latido (lo guarda en plan.json).
func esperarPlan(t *testing.T) {
	t.Helper()
	limite := time.Now().Add(3 * time.Second)
	for time.Now().Before(limite) {
		if _, err := LeerPlan(); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("el bucle no guardó el plan del latido")
}

// bucleConQDDFalso es un bucle firmado que corre el qdd falso, con un teléfono en el canal y el plan ya recibido.
func bucleConQDDFalso(t *testing.T, ajustes ...func(*Bucle)) (string, string, string, *transporteFalso) {
	t.Helper()
	homeTemporal(t)
	falso := t.TempDir()
	t.Setenv(envQDDFalso, falso)
	t.Setenv("GIT_DIR", filepath.Join(falso, "no-es-un-repo"))
	root := proyecto(t, "app", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	_, tr, _, _ := bucleDePrueba(t, append([]func(*Bucle){func(b *Bucle) {
		b.Cred.Firma = firmaPrueba
		b.Bin = os.Args[0]
	}}, ajustes...)...)
	esperarPlan(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	return root, clave, falso, tr
}

func TestEjecutar_UnComandoQDDCorreUnaSolaVezSinShellNiSecretos(t *testing.T) {
	root, clave, falso, tr := bucleConQDDFalso(t)
	e := ejecucionFirmada("0c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", `estado check --json`, "d1", ahoraEn())
	mandarEjecucion(tr, e)
	var hecha AccionEstado
	for _, want := range []string{EstadoRecibida, EstadoEjecutando, EstadoHecha} {
		hecha = estadoDe(t, tr, 5*time.Second)
		if hecha.Estado != want || hecha.Solicitud != e.Solicitud || hecha.AccionID != AccionEjecutar || hecha.Proyecto != clave {
			t.Fatalf("esperaba %s: %+v", want, hecha)
		}
	}
	if !strings.Contains(hecha.Resultado, "args=estado check --json") || !strings.Contains(hecha.Resultado, "cwd="+root) {
		t.Fatalf("corre qdd con esos argumentos en la carpeta del proyecto: %q", hecha.Resultado)
	}
	if strings.Contains(hecha.Resultado, "ghp_") || !strings.Contains(hecha.Resultado, Oculto) {
		t.Fatalf("la salida sale sin secretos (inv. 16): %q", hecha.Resultado)
	}
	if strings.Contains(hecha.Resultado, "GIT_DIR heredado") {
		t.Fatal("corre con el entorno sin las variables de git (FND-117)")
	}
	mandarEjecucion(tr, e)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoHecha {
		t.Fatalf("la repetida responde el último estado: %+v", got)
	}
	time.Sleep(150 * time.Millisecond)
	if c := corridas(t, falso); len(c) != 1 || c[0] != "estado check --json" {
		t.Fatalf("corrió una sola vez: %q", c)
	}
	x, ok := consola.EjecucionDe(root, e.Solicitud, time.Now())
	if !ok || x.Estado != consola.EstadoHecho || x.Nombre != "Pixel de d1" || strings.Contains(x.Salida, "ghp_") {
		t.Fatalf("queda en la cola del proyecto, hecha y sin secretos: %+v", x)
	}
	var en *consola.Accion
	ev := tr.esperar(t, 2*time.Second, "delta")
	var d Delta
	json.Unmarshal(ev.Payload, &d)
	for i, a := range d.Proyectos[clave].Acciones {
		if a.Solicitud == e.Solicitud {
			en = &d.Proyectos[clave].Acciones[i]
		}
	}
	if en == nil || en.Origen != consola.OrigenTelefono || en.Dispositivo != "Pixel de d1" || strings.Contains(en.Resultado, "ghp_") {
		t.Fatalf("el panel del teléfono lo trae como pedido del teléfono: %+v", d.Proyectos[clave])
	}
	var ejecutadas int
	for _, l := range lineasLog(t) {
		if l.Solicitud == e.Solicitud && l.Estado == EstadoHecha && l.AccionID == AccionEjecutar {
			ejecutadas++
		}
	}
	if ejecutadas != 1 {
		t.Fatalf("queda una vez en el log local: %+v", lineasLog(t))
	}
}

func TestEjecutar_FallaConElFinalDeLaSalida(t *testing.T) {
	root, clave, _, tr := bucleConQDDFalso(t)
	e := ejecucionFirmada("1c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "doctor", "d1", ahoraEn())
	mandarEjecucion(tr, e)
	estadoDe(t, tr, time.Second)
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, 5*time.Second); got.Estado != EstadoFallo || !strings.Contains(got.Resultado, "args=doctor") {
		t.Fatalf("un código distinto de 0 es fallo, con la salida: %+v", got)
	}
	if x, _ := consola.EjecucionDe(root, e.Solicitud, time.Now()); x.Estado != consola.EstadoFallo || x.Codigo != 3 {
		t.Fatalf("la cola anota la falla y su código: %+v", x)
	}
	largo := ejecucionFirmada("2c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "blueprint list", "d1", ahoraEn())
	mandarEjecucion(tr, largo)
	estadoDe(t, tr, time.Second)
	estadoDe(t, tr, time.Second)
	got := estadoDe(t, tr, 5*time.Second)
	if got.Estado != EstadoHecha || !strings.HasSuffix(got.Resultado, "FIN") || len([]rune(got.Resultado)) > MaxResultadoEjecutar+1 {
		t.Fatalf("una salida larga llega con su final, hasta %d caracteres: %d %q", MaxResultadoEjecutar, len([]rune(got.Resultado)), got.Resultado[:40])
	}
}

func TestEjecutar_LoQueNoSePermiteSeRechazaYNoCorre(t *testing.T) {
	_, clave, falso, tr := bucleConQDDFalso(t)
	casos := []struct{ tipo, texto, dice string }{
		{"qdd", "doctor --fix", "--fix"},
		{"qdd", "init", "no está permitido"},
		{"qdd", "audit; rm -rf ~", "«;»"},
		{"qdd", "qdd audit", "qdd"},
		{"shell", "ls", "tipo"},
		{"ia", "hola\x1b[2J", "control"},
	}
	for i, c := range casos {
		mandarEjecucion(tr, ejecucionFirmada(fmt.Sprintf("3c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f5%d", i), clave, c.tipo, c.texto, "d1", ahoraEn()))
		estadoDe(t, tr, time.Second)
		if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, c.dice) {
			t.Errorf("%q: rechazada diciendo %q: %+v", c.texto, c.dice, got)
		}
	}
	mandarEjecucion(tr, ejecucionFirmada("4c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", "no-existe-0123", "qdd", "audit", "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, "no publica") {
		t.Fatalf("un proyecto que el PC no publica: %+v", got)
	}
	Revocar("d9")
	mandarEjecucion(tr, ejecucionFirmada("5c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "audit", "d9", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, "revocado") {
		t.Fatalf("un teléfono revocado: %+v", got)
	}
	falsa := ejecucionFirmada("6c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "audit", "d1", ahoraEn())
	falsa.Firma = strings.Repeat("0", 64)
	vieja := ejecucionFirmada("7c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "audit", "d1", time.Now().Add(-10*time.Minute).UTC().Format(formatoEn))
	antes := tr.contar("accion_estado")
	mandarEjecucion(tr, falsa)
	mandarEjecucion(tr, vieja)
	time.Sleep(300 * time.Millisecond)
	if tr.contar("accion_estado") != antes {
		t.Fatal("sin la firma del relay o vieja no hay respuesta (inv. 18)")
	}
	if c := corridas(t, falso); len(c) != 0 {
		t.Fatalf("nada de eso corrió: %q", c)
	}
	descartadas := 0
	for _, l := range lineasLog(t) {
		if l.Estado == EstadoDescartada && l.AccionID == AccionEjecutar {
			descartadas++
		}
	}
	if descartadas != 2 {
		t.Fatalf("quedan en el log como descartadas: %+v", lineasLog(t))
	}
	if _, ok := consola.EjecucionDe(proyectoDeClave(t, clave), falsa.Solicitud, time.Now()); ok {
		t.Fatal("lo que no trae firma no se recuerda: no le quita la solicitud a la de verdad")
	}
}

func proyectoDeClave(t *testing.T, clave string) string {
	t.Helper()
	reg, _ := registro.Leer()
	for _, p := range reg.Proyectos {
		if p.Clave == clave {
			return p.Ruta
		}
	}
	t.Fatalf("no está %s", clave)
	return ""
}

func TestEjecutar_UnComandoPorProyectoALaVez(t *testing.T) {
	_, clave, falso, tr := bucleConQDDFalso(t)
	primero := ejecucionFirmada("dc6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "audit", "d1", ahoraEn())
	mandarEjecucion(tr, primero)
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoEjecutando {
		t.Fatal(got)
	}
	mandarEjecucion(tr, ejecucionFirmada("ec6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "audit", "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, "ya corre") {
		t.Fatalf("dos audit juntos se pisarían: %+v", got)
	}
	if got := estadoDe(t, tr, 3*time.Second); got.Estado != EstadoHecha || got.Solicitud != primero.Solicitud {
		t.Fatalf("el primero termina: %+v", got)
	}
	if c := corridas(t, falso); len(c) != 1 {
		t.Fatalf("corrió uno: %q", c)
	}
}

// FND-118: «Auditar» desde el teléfono devolvía el principio de la salida (600 caracteres) y el veredicto, que va al
// final, nunca llegaba.
func TestAccion_AuditarDesdeElTelefonoDevuelveElFinalDeLaSalida(t *testing.T) {
	_, clave, _, tr := bucleConQDDFalso(t)
	a := firmada("fc6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "auditar", "d1", ahoraEn())
	mandarAccion(tr, a)
	estadoDe(t, tr, time.Second)
	estadoDe(t, tr, time.Second)
	got := estadoDe(t, tr, 5*time.Second)
	if got.Estado != EstadoHecha || !strings.HasSuffix(got.Resultado, "VEREDICTO: gate aprobado") || len([]rune(got.Resultado)) > maxResultado+1 {
		t.Fatalf("el final de la salida, hasta %d caracteres: %d %q", maxResultado, len([]rune(got.Resultado)), got.Resultado)
	}
}

func TestEjecutar_SinProSeRechaza(t *testing.T) {
	_, clave, falso, tr := bucleConQDDFalso(t, func(b *Bucle) {
		b.R = &relayBucleFalso{plan: PlanGratis(), avisos: make(chan aviso, 32)}
	})
	mandarEjecucion(tr, ejecucionFirmada("8c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "audit", "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, "requiere Pro") {
		t.Fatalf("sin Pro no se ejecuta nada desde el teléfono: %+v", got)
	}
	if c := corridas(t, falso); len(c) != 0 {
		t.Fatalf("no corrió: %q", c)
	}
}

func TestEjecutar_LaInstruccionEsperaUnaSesionYSeEntregaUnaVez(t *testing.T) {
	root, clave, _, tr := bucleConQDDFalso(t)
	e := ejecucionFirmada("9c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", "Revisa el build\ny arréglalo", "d1", ahoraEn())
	mandarEjecucion(tr, e)
	estadoDe(t, tr, time.Second)
	got := estadoDe(t, tr, 2*time.Second)
	if got.Estado != EstadoRecibida || !strings.Contains(got.Resultado, "sesión") {
		t.Fatalf("sin sesión abierta queda en cola y lo dice: %+v", got)
	}
	tomadas, _ := consola.TomarInstrucciones(root, "sesion-a", time.Now())
	if len(tomadas) != 1 || tomadas[0].Texto != e.Texto {
		t.Fatalf("la sesión la recibe tal cual: %+v", tomadas)
	}
	if got := estadoDe(t, tr, 3*time.Second); got.Estado != EstadoHecha || !strings.Contains(got.Resultado, "entregada") {
		t.Fatalf("entregada a la sesión: %+v", got)
	}

	consola.LatirSesion(root, "sesion-a", time.Now())
	otra := ejecucionFirmada("ac6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", "Y corre las pruebas", "d1", ahoraEn())
	mandarEjecucion(tr, otra)
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoEjecutando {
		t.Fatalf("con una sesión viva va en camino: %+v", got)
	}
}

func TestEjecutar_RechazarUnaInstruccionQueEspera(t *testing.T) {
	root, clave, _, tr := bucleConQDDFalso(t)
	e := ejecucionFirmada("bc6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", "hola", "d1", ahoraEn())
	mandarEjecucion(tr, e)
	estadoDe(t, tr, time.Second)
	estadoDe(t, tr, time.Second)
	mandarAccion(tr, firmada("cc6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "rechazar:telefono:"+e.Solicitud, "d1", ahoraEn()))
	vistos := map[string]AccionEstado{}
	for len(vistos) < 2 {
		got := estadoDe(t, tr, 3*time.Second)
		if got.Estado != EstadoRecibida {
			vistos[got.Solicitud] = got
		}
	}
	if r := vistos["cc6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50"]; r.Estado != EstadoHecha {
		t.Fatalf("el rechazo queda hecho: %+v", r)
	}
	if r := vistos[e.Solicitud]; r.Estado != EstadoRechazada || !strings.Contains(r.Resultado, "retir") {
		t.Fatalf("la instrucción queda rechazada: %+v", r)
	}
	if tomadas, _ := consola.TomarInstrucciones(root, "s", time.Now()); len(tomadas) != 0 {
		t.Fatalf("y ninguna sesión la recibe: %+v", tomadas)
	}
}

func TestArmar_EjecutarSoloConPro(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", true)
	if OpcionesEjecutar(root, PlanGratis(), time.Now()) != nil {
		t.Fatal("sin Pro no se ofrece")
	}
	o := OpcionesEjecutar(root, Plan{Plan: "pro"}, time.Now())
	if o == nil || !o.QDD || o.IA || len(o.Sugeridos) == 0 {
		t.Fatalf("con Pro, comandos qdd; instrucciones solo con una sesión abierta: %+v", o)
	}
	consola.LatirSesion(root, "s", time.Now())
	if o := OpcionesEjecutar(root, Plan{Plan: "pro"}, time.Now()); !o.IA {
		t.Fatalf("con una sesión viva acepta instrucciones: %+v", o)
	}
	for _, s := range o.Sugeridos {
		if strings.HasPrefix(s, "qdd") || strings.Contains(s, "status") {
			t.Fatalf("sugeridos sin la palabra qdd y solo comandos que existen: %q", s)
		}
	}
}

func TestBucle_GuardaElPlanYElPanelDelTelefonoDiceQueEjecuta(t *testing.T) {
	_, clave, _, tr := bucleConQDDFalso(t)
	p, err := LeerPlan()
	if err != nil || !p.Pro() {
		t.Fatalf("el plan del latido queda en plan.json: %+v %v", p, err)
	}
	ruta, _ := rutaRemoto("plan.json")
	if info, _ := os.Stat(ruta); info.Mode().Perm() != 0o600 {
		t.Fatal("plan.json es 0600")
	}
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "sync"}
	ev := tr.esperar(t, time.Second, "estado")
	var e Estado
	json.Unmarshal(ev.Payload, &e)
	if len(e.Proyectos) != 1 || e.Proyectos[0].Clave != clave || e.Proyectos[0].Panel.Ejecutar == nil || !e.Proyectos[0].Panel.Ejecutar.QDD {
		t.Fatalf("con Pro el panel publicado trae ejecutar: %s", ev.Payload)
	}
}

func TestPlanLocal_SoloConQDDMovilActivado(t *testing.T) {
	homeTemporal(t)
	GuardarPlan(Plan{Plan: "pro"}, time.Now())
	if PlanLocal().Pro() {
		t.Fatal("sin credenciales QDD Móvil no está activado: Gratis")
	}
	GuardarCredenciales(Credenciales{Maquina: "m", Topico: topicoPrueba, Firma: firmaPrueba})
	if !PlanLocal().Pro() {
		t.Fatal("activado y con el plan Pro del último latido")
	}
	borrarPrivado("plan.json")
	if PlanLocal().Pro() {
		t.Fatal("sin latido todavía: Gratis")
	}
}

func TestEjecutar_ElModoGoalSeEnciendeYApagaDesdeElTelefono(t *testing.T) {
	args, err := ArgsPermitidos("goal activar")
	if err != nil || strings.Join(args, " ") != "goal activar --desde telefono" {
		t.Fatalf("activar dice que vino del teléfono: %v %v", args, err)
	}
	if _, err := ArgsPermitidos("goal desactivar"); err != nil {
		t.Fatal(err)
	}
	for _, texto := range []string{"goal ciclo", "goal activar --desde panel", "goal desactivar --x"} {
		if _, err := ArgsPermitidos(texto); err == nil {
			t.Fatalf("«%s» no está permitido", texto)
		}
	}
	for _, s := range Sugeridos {
		if strings.HasPrefix(s, "goal") {
			t.Fatal("cambiar el modo no se sugiere como un comando más: la app lo ofrece con biometría")
		}
	}
}

func TestEjecutar_UnComandoDelTelefonoNoCorreComoIA(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	codigo, salida := CorrerQDD(context.Background(), "sh", t.TempDir(), []string{"-c", `printf "[%s]" "$CLAUDECODE"`}, 5*time.Second)
	if codigo != 0 || salida != "[]" {
		t.Fatalf("sin CLAUDECODE: %d %q", codigo, salida)
	}
}
