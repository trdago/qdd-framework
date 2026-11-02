package remoto

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/modules"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.3): una acción solo vale con la firma de la máquina (inv. 18), se ejecuta una sola vez por solicitud
// (inv. 3), recibe su respuesta (inv. 4), queda en el log local (inv. 5) y un pedido ya aprobado en otro lugar responde
// ya_aprobado con el lugar (inv. 19).

// TestMain hace del binario de pruebas el «qdd consola correr» que lanza un pedido.
func TestMain(m *testing.M) {
	if root, id, ok := consola.EsCorrer(os.Args[1:]); ok {
		os.Exit(consola.CorrerLanzado(root, id, os.Stdout))
	}
	os.Exit(m.Run())
}

const firmaPrueba = "abababababababababababababababababababababababababababababababab"

func TestFirmaAccion_LaMismaQueElRelay(t *testing.T) {
	// hmac.new(("ab"*32).encode(), b"s|p|a|d|2026-11-02T10:00:00Z", sha256): lo mismo que pgcrypto hmac(mensaje, firma).
	if got := FirmaAccion(firmaPrueba, "s", "p", "a", "d", "2026-11-02T10:00:00Z"); got != "33128eceb2879a4fbf0ddc28a5ef7aa60d6d50b17e966171b247ffbaa8be910f" {
		t.Fatalf("firma: %s", got)
	}
	now := time.Date(2026, 11, 2, 10, 3, 0, 0, time.UTC)
	a := firmada("s", "p", "a", "d", "2026-11-02T10:00:00Z")
	if err := a.Verificar(firmaPrueba, now); err != nil {
		t.Fatal(err)
	}
	mala := a
	mala.AccionID = "auditar"
	if err := mala.Verificar(firmaPrueba, now); err == nil {
		t.Fatal("cambiar la acción rompe la firma")
	}
	if err := a.Verificar(firmaPrueba, now.Add(3*time.Minute)); err == nil {
		t.Fatal("más de 5 min es vieja")
	}
	if err := a.Verificar(firmaPrueba, now.Add(-9*time.Minute)); err == nil {
		t.Fatal("del futuro tampoco")
	}
	if err := a.Verificar("cd"+firmaPrueba[2:], now); err == nil {
		t.Fatal("con otra firma de máquina no vale")
	}
}

func firmada(solicitud, proyecto, accion, dispositivo, en string) AccionFirmada {
	return AccionFirmada{Version: 1, Solicitud: solicitud, Proyecto: proyecto, AccionID: accion, Dispositivo: dispositivo,
		DispositivoNombre: "Pixel de " + dispositivo, En: en, Firma: FirmaAccion(firmaPrueba, solicitud, proyecto, accion, dispositivo, en)}
}

func ahoraEn() string { return time.Now().UTC().Format(formatoEn) }

// proyectoConFicha arma un proyecto con una ficha en borrador y lo anota en el registro.
func proyectoConFicha(t *testing.T, id string) (string, string) {
	t.Helper()
	root := proyecto(t, "app", true)
	os.MkdirAll(filepath.Join(root, "src"), 0o755)
	m := modules.Module{ID: id, Nombre: "Módulo " + id, Estado: modules.EstadoBorrador, Rutas: []string{"src"},
		ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "p", CriterioDeExito: "c"},
		ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "a"},
		Pruebas:             modules.Pruebas{Comandos: []string{"go test ./..."}}}
	if _, problemas := modules.Save(root, m); len(problemas) > 0 {
		t.Fatal(problemas)
	}
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	return root, clave
}

func bucleConFirma(t *testing.T) (*Bucle, *transporteFalso, *relayBucleFalso) {
	t.Helper()
	b, tr, rl, _ := bucleDePrueba(t, func(b *Bucle) {
		b.Cred.Firma = firmaPrueba
		b.Bin = os.Args[0]
	})
	return b, tr, rl
}

func mandarAccion(tr *transporteFalso, a AccionFirmada) {
	p, _ := json.Marshal(a)
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "accion", Payload: p}
}

func estadoDe(t *testing.T, tr *transporteFalso, plazo time.Duration) AccionEstado {
	t.Helper()
	e := tr.esperar(t, plazo, "accion_estado")
	var ae AccionEstado
	json.Unmarshal(e.Payload, &ae)
	return ae
}

func lineasLog(t *testing.T) []EntradaLog {
	t.Helper()
	ruta, _ := rutaRemoto("log.jsonl")
	f, err := os.Open(ruta)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []EntradaLog
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e EntradaLog
		json.Unmarshal(sc.Bytes(), &e)
		out = append(out, e)
	}
	return out
}

func TestAccion_AprobarUnaFichaUnaSolaVezYQuedaEnElLog(t *testing.T) {
	homeTemporal(t)
	root, clave := proyectoConFicha(t, "pagos")
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	a := firmada("0b6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "aprobar:pagos", "d1", ahoraEn())
	mandarAccion(tr, a)
	for _, want := range []string{EstadoRecibida, EstadoEjecutando, EstadoHecha} {
		if got := estadoDe(t, tr, 5*time.Second); got.Estado != want || got.Solicitud != a.Solicitud || got.Version != 1 {
			t.Fatalf("esperaba %s: %+v", want, got)
		}
	}
	mods, _ := modules.Load(root)
	if mods[0].Estado != modules.EstadoAprobado {
		t.Fatalf("la ficha quedó aprobada: %s", mods[0].Estado)
	}
	mandarAccion(tr, a)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoHecha {
		t.Fatalf("la repetida responde el último estado sin re-ejecutar: %+v", got)
	}
	time.Sleep(150 * time.Millisecond)
	if n := tr.contar("accion_estado"); n != 4 {
		t.Fatalf("recibida, ejecutando, hecha y la respuesta a la repetida: %d", n)
	}
	log := lineasLog(t)
	if len(log) != 1 || log[0].Estado != EstadoHecha || log[0].Dispositivo != "d1" || log[0].Nombre != "Pixel de d1" || log[0].AccionID != "aprobar:pagos" || log[0].Proyecto != clave {
		t.Fatalf("una línea en el log con quién, qué y cómo terminó: %+v", log)
	}
	ruta, _ := rutaRemoto("log.jsonl")
	if info, _ := os.Stat(ruta); info.Mode().Perm() != 0o600 {
		t.Fatal("el log es 0600")
	}
}

func TestAccion_LoQueNoSeOfreceOLoQueNoVieneDelRelaySeRechaza(t *testing.T) {
	homeTemporal(t)
	_, clave := proyectoConFicha(t, "pagos")
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")

	mandarAccion(tr, firmada("s-no-ofrecida", clave, "aprobar:otra", "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, "no ofrece") {
		t.Fatalf("un id que el panel no ofrece (inv. 1): %+v", got)
	}
	mandarAccion(tr, firmada("s-proyecto", "no-existe-0123", "auditar", "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRechazada {
		t.Fatalf("un proyecto que el PC no publica: %+v", got)
	}
	falsa := firmada("s-falsa", clave, "aprobar:pagos", "d1", ahoraEn())
	falsa.Firma = strings.Repeat("0", 64)
	mandarAccion(tr, falsa)
	vieja := firmada("s-vieja", clave, "aprobar:pagos", "d1", time.Now().Add(-10*time.Minute).UTC().Format(formatoEn))
	antes := tr.contar("accion_estado")
	mandarAccion(tr, vieja)
	time.Sleep(200 * time.Millisecond)
	if tr.contar("accion_estado") != antes {
		t.Fatal("sin firma válida o vieja no hay respuesta")
	}
	descartadas := 0
	for _, l := range lineasLog(t) {
		if l.Estado == EstadoDescartada {
			descartadas++
		}
	}
	if descartadas != 2 {
		t.Fatalf("sin firma válida o vieja: sin respuesta y al log: %+v", lineasLog(t))
	}
	if err := Revocar("D2"); err != nil {
		t.Fatal(err)
	}
	mandarAccion(tr, firmada("s-revocado", clave, "aprobar:pagos", "d2", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, "revocado") {
		t.Fatalf("un teléfono revocado en el PC no puede, en el acto (inv. 18): %+v", got)
	}
}

func TestAccion_UnPedidoCorreUnaVezYOtroTelefonoVeYaAprobado(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	homeTemporal(t)
	root := proyecto(t, "web", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	marca := filepath.Join(t.TempDir(), "corridas")
	p, err := consola.Pedir(root, "echo x >> '"+marca+"'; echo TOKEN=supersecreto", "Publicar", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, tr, rl := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	a := firmada("sol-1", clave, "pedido:"+p.ID, "d1", ahoraEn())
	mandarAccion(tr, a)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRecibida {
		t.Fatalf("recibida: %+v", got)
	}
	if got := estadoDe(t, tr, 5*time.Second); got.Estado != EstadoEjecutando {
		t.Fatalf("ejecutando: %+v", got)
	}
	hecha := estadoDe(t, tr, 10*time.Second)
	if hecha.Estado != EstadoHecha || strings.Contains(hecha.Resultado, "supersecreto") || !strings.Contains(hecha.Resultado, Oculto) {
		t.Fatalf("hecha, con el resultado sin secretos (inv. 16): %+v", hecha)
	}
	select {
	case av := <-rl.avisos:
		if av.Tipo != AvisoTerminoBien || av.Clave != clave {
			t.Fatalf("push de cómo terminó: %+v", av)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("al terminar un pedido aprobado desde el teléfono, push")
	}
	pd, _ := pedidoDe(root, p.ID)
	if pd.Sesion != "remoto:d1" || pd.Avisado != true {
		t.Fatalf("corre con la sesión remoto:<dispositivo> y queda avisado: %+v", pd)
	}
	mandarAccion(tr, a)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoHecha {
		t.Fatalf("repetida, el último estado: %+v", got)
	}
	mandarAccion(tr, firmada("sol-2", clave, "pedido:"+p.ID, "d9", ahoraEn()))
	estadoDe(t, tr, time.Second)
	got := estadoDe(t, tr, time.Second)
	if got.Estado != EstadoYaAprobado || got.Lugar != "Pixel de d1" {
		t.Fatalf("otro teléfono: ya_aprobado en el teléfono que lo aprobó (inv. 19): %+v", got)
	}
	b, _ := os.ReadFile(marca)
	if strings.Count(string(b), "x") != 1 {
		t.Fatalf("corrió una sola vez: %q", b)
	}
}

func TestAccion_UnPedidoAprobadoEnElPCDiceLugarPC(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	homeTemporal(t)
	root := proyecto(t, "web", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	p, _ := consola.Pedir(root, "sleep 2", "Esperar", "", time.Now())
	a, _ := consola.BuscarAccion(consola.Panel(root, time.Now()).Acciones, "pedido:"+p.ID)
	if c := a.Lanzar(root, os.Args[0], "sesion-claude", os.Stderr); c != 0 {
		t.Fatal("lanzado desde el panel del PC")
	}
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	mandarAccion(tr, firmada("sol-pc", clave, "pedido:"+p.ID, "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoYaAprobado || got.Lugar != LugarPC {
		t.Fatalf("ya aprobado en el PC: %+v", got)
	}
}

func TestAccion_RechazarUnPedidoLoQuitaDeLaColaSinEjecutarlo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	homeTemporal(t)
	root := proyecto(t, "web", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	marca := filepath.Join(t.TempDir(), "corridas")
	p, err := consola.Pedir(root, "echo x >> '"+marca+"'", "Publicar", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	mandarAccion(tr, firmada("sol-r", clave, prefijoRechazar+"pedido:"+p.ID, "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoHecha {
		t.Fatalf("rechazado: %+v", got)
	}
	if _, ok := pedidoDe(root, p.ID); ok {
		t.Fatal("el pedido rechazado sigue en la cola")
	}
	if _, err := os.Stat(marca); err == nil {
		t.Fatal("rechazar no debe ejecutar el pedido")
	}
	ls := lineasLog(t)
	if len(ls) == 0 || ls[len(ls)-1].AccionID != prefijoRechazar+"pedido:"+p.ID {
		t.Fatalf("el rechazo queda en el log (inv. 5): %+v", ls)
	}
	// Ya no se ofrece: un segundo rechazo (otra solicitud) se rechaza.
	mandarAccion(tr, firmada("sol-r2", clave, prefijoRechazar+"pedido:"+p.ID, "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRechazada {
		t.Fatalf("un pedido que ya no está: %+v", got)
	}
}

func TestAccion_RechazarUnPedidoYaAprobadoDiceDonde(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	homeTemporal(t)
	root := proyecto(t, "web", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	p, _ := consola.Pedir(root, "sleep 2", "Esperar", "", time.Now())
	a, _ := consola.BuscarAccion(consola.Panel(root, time.Now()).Acciones, "pedido:"+p.ID)
	if c := a.Lanzar(root, os.Args[0], "sesion-claude", os.Stderr); c != 0 {
		t.Fatal("lanzado desde el panel del PC")
	}
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	mandarAccion(tr, firmada("sol-r", clave, prefijoRechazar+"pedido:"+p.ID, "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoYaAprobado || got.Lugar != LugarPC {
		t.Fatalf("no se rechaza lo que ya se aprobó: %+v", got)
	}
	if _, ok := pedidoDe(root, p.ID); !ok {
		t.Fatal("un pedido aprobado no sale de la cola por un rechazo")
	}
}
