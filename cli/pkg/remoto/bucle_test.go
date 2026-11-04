package remoto

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): el bucle con un canal y un relay falsos. Sin teléfonos solo hay presence (inv. 12); con teléfonos,
// estado en sync y deltas agrupados (inv. 13); latido cada minuto con su plan; push de lo nuevo.

type enviado struct {
	Evento  string
	Payload json.RawMessage
}

type transporteFalso struct {
	mu        sync.Mutex
	enviados  []enviado
	tracks    []json.RawMessage
	eventos   chan Evento
	falla     error
	broadcast chan enviado
}

func nuevoTransporte() *transporteFalso {
	return &transporteFalso{eventos: make(chan Evento, 64), broadcast: make(chan enviado, 256)}
}

func (f *transporteFalso) Broadcast(ctx context.Context, evento string, payload any) error {
	f.mu.Lock()
	falla := f.falla
	f.mu.Unlock()
	if falla != nil {
		return falla
	}
	b, _ := json.Marshal(payload)
	e := enviado{Evento: evento, Payload: b}
	f.mu.Lock()
	f.enviados = append(f.enviados, e)
	f.mu.Unlock()
	f.broadcast <- e
	return nil
}

func (f *transporteFalso) Track(ctx context.Context, meta any) error {
	b, _ := json.Marshal(meta)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tracks = append(f.tracks, b)
	return nil
}

func (f *transporteFalso) Eventos() <-chan Evento { return f.eventos }

func (f *transporteFalso) Enviados() []enviado {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]enviado{}, f.enviados...)
}

func (f *transporteFalso) esperar(t *testing.T, plazo time.Duration, evento string) enviado {
	t.Helper()
	limite := time.After(plazo)
	for {
		select {
		case e := <-f.broadcast:
			if e.Evento == evento {
				return e
			}
		case <-limite:
			t.Fatalf("no se mandó %s", evento)
			return enviado{}
		}
	}
}

// contar dice cuántos eventos de ese tipo se mandaron en total.
func (f *transporteFalso) contar(evento string) int {
	n := 0
	for _, e := range f.Enviados() {
		if e.Evento == evento {
			n++
		}
	}
	return n
}

func (f *transporteFalso) nada(t *testing.T, plazo time.Duration) {
	t.Helper()
	select {
	case e := <-f.broadcast:
		t.Fatalf("no debía mandar nada y mandó %s: %s", e.Evento, e.Payload)
	case <-time.After(plazo):
	}
}

type aviso struct{ Tipo, Proyecto, Clave string }

type relayBucleFalso struct {
	mu       sync.Mutex
	plan     Plan
	err      error
	latidos  []map[string]int
	avisos   chan aviso
	errAviso error
}

func nuevoRelayBucle() *relayBucleFalso {
	return &relayBucleFalso{plan: Plan{Plan: "pro", PCs: 3, ConsolaPermitida: true}, avisos: make(chan aviso, 32)}
}

func (r *relayBucleFalso) Latido(ctx context.Context, topico, firma string, enCurso, consolaMin, mensajes int) (Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.latidos = append(r.latidos, map[string]int{"en_curso": enCurso, "consola_min": consolaMin, "mensajes": mensajes})
	return r.plan, r.err
}

func (r *relayBucleFalso) Avisar(ctx context.Context, topico, firma, tipo, proyecto, clave string) error {
	r.avisos <- aviso{tipo, proyecto, clave}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.errAviso
}

func (r *relayBucleFalso) Latidos() []map[string]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]map[string]int{}, r.latidos...)
}

func bucleDePrueba(t *testing.T, ajustes ...func(*Bucle)) (*Bucle, *transporteFalso, *relayBucleFalso, chan error) {
	t.Helper()
	tr, rl := nuevoTransporte(), nuevoRelayBucle()
	b := &Bucle{QDD: "v-prueba", Cred: Credenciales{Topico: topicoPrueba, Firma: "f"}, T: tr, R: rl,
		Agrupar: 30 * time.Millisecond, CadaLatido: time.Hour, CadaSondeo: 40 * time.Millisecond}
	for _, a := range ajustes {
		a(b)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	fin := make(chan error, 1)
	go func() { fin <- b.Correr(ctx) }()
	t.Cleanup(func() {
		cancelar()
		<-fin
	})
	esperarPrimerLatido()
	return b, tr, rl, fin
}

// esperarPrimerLatido espera (como mucho 1 s) a que el bucle tenga el plan del primer latido: con Pro el panel suma
// ejecutar, y ese cambio no debe caer en medio de lo que mide una prueba.
func esperarPrimerLatido() {
	limite := time.Now().Add(time.Second)
	for time.Now().Before(limite) {
		if _, err := LeerPlan(); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func telefono(dispositivo, mirando string) Evento {
	m, _ := json.Marshal(map[string]string{"rol": "movil", "dispositivo": dispositivo, "mirando": mirando, "phx_ref": "r-" + dispositivo})
	pc, _ := json.Marshal(map[string]string{"rol": "pc", "phx_ref": "r-pc"})
	return Evento{Tipo: TipoPresencia, Presencias: map[string][]json.RawMessage{"pc": {pc}, dispositivo: {m}}}
}

func sinTelefonos() Evento {
	pc, _ := json.Marshal(map[string]string{"rol": "pc", "phx_ref": "r-pc"})
	return Evento{Tipo: TipoPresencia, Presencias: map[string][]json.RawMessage{"pc": {pc}}}
}

func TestBucle_SinTelefonosSoloPresence(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	_, tr, rl, _ := bucleDePrueba(t)
	time.Sleep(60 * time.Millisecond)
	consola.Pedir(root, "echo hola", "Saludar", "", time.Now())
	tr.nada(t, 300*time.Millisecond)
	tr.mu.Lock()
	tracks := tr.tracks
	tr.mu.Unlock()
	var p PresenciaPC
	if len(tracks) == 0 || json.Unmarshal(tracks[0], &p) != nil || p.Rol != "pc" || p.VersionQDD != "v-prueba" || p.Desde == "" {
		t.Fatalf("presence del PC: %s", tracks)
	}
	if len(rl.Latidos()) == 0 {
		t.Fatal("el latido va aunque no haya teléfonos")
	}
}

func TestBucle_ConTelefonoEstadoSyncYDeltasAgrupados(t *testing.T) {
	homeTemporal(t)
	a, b := proyecto(t, "a", true), proyecto(t, "b", true)
	registro.Anotar(a, time.Now())
	registro.Anotar(b, time.Now().Add(-time.Minute))
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	e := tr.esperar(t, time.Second, "estado")
	var est Estado
	json.Unmarshal(e.Payload, &est)
	if est.Seq != 1 || est.Version != 1 || len(est.Proyectos) != 2 {
		t.Fatalf("al llegar un teléfono, el estado completo: %+v", est)
	}
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "sync", Payload: json.RawMessage(`{"version":1}`)}
	e = tr.esperar(t, time.Second, "estado")
	json.Unmarshal(e.Payload, &est)
	if est.Seq != 2 {
		t.Fatalf("sync responde el estado con el siguiente seq: %d", est.Seq)
	}
	consola.Pedir(a, "echo uno", "Uno", "", time.Now())
	consola.Pedir(a, "echo dos", "Dos", "", time.Now())
	e = tr.esperar(t, 2*time.Second, "delta")
	var d struct {
		Version   int
		Seq       uint64
		Proyectos map[string]*consola.DatosConsola
	}
	json.Unmarshal(e.Payload, &d)
	claveA, _ := registro.ClaveDe(a)
	if d.Version != 1 || d.Seq != 3 || len(d.Proyectos) != 1 || d.Proyectos[claveA] == nil {
		t.Fatalf("un delta con solo el proyecto que cambió: %s", e.Payload)
	}
	pedidos := 0
	for _, ac := range d.Proyectos[claveA].Acciones {
		if ac.Tipo == consola.TipoPedido {
			pedidos++
		}
	}
	if pedidos != 2 {
		t.Fatalf("los dos cambios juntos en 300 ms (inv. 13): %d pedidos", pedidos)
	}
	tr.nada(t, 200*time.Millisecond)
	tr.eventos <- sinTelefonos()
	time.Sleep(50 * time.Millisecond)
	consola.Pedir(a, "echo tres", "Tres", "", time.Now())
	tr.nada(t, 300*time.Millisecond)
}

func TestBucle_UnProyectoNuevoMandaElEstadoCompleto(t *testing.T) {
	homeTemporal(t)
	registro.Anotar(proyecto(t, "a", true), time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	registro.Anotar(proyecto(t, "nuevo", false), time.Now())
	e := tr.esperar(t, 2*time.Second, "estado")
	var est Estado
	json.Unmarshal(e.Payload, &est)
	if len(est.Proyectos) != 2 || est.Proyectos[0].Nombre != "nuevo" {
		t.Fatalf("lo que un delta no dice va en un estado: %+v", est.Proyectos)
	}
}

func TestMandarCambios_UnProyectoQueSaleVaComoNull(t *testing.T) {
	tr := nuevoTransporte()
	b := &Bucle{T: tr, telefonos: []Telefono{{Rol: "movil"}}, enviados: map[string]string{}}
	p := func(clave string) ProyectoEstado {
		return ProyectoEstado{Clave: clave, Nombre: clave, Panel: &consola.DatosConsola{Carpeta: clave}}
	}
	b.mandarEstado(context.Background(), Estado{Proyectos: []ProyectoEstado{p("x"), p("y")}})
	tr.esperar(t, time.Second, "estado")
	b.mandarCambios(context.Background(), Estado{Proyectos: []ProyectoEstado{p("x")}})
	e := tr.esperar(t, time.Second, "delta")
	if string(e.Payload) != `{"version":1,"seq":2,"proyectos":{"y":null}}` {
		t.Fatalf("salió y: %s", e.Payload)
	}
	b.mandarCambios(context.Background(), Estado{Proyectos: []ProyectoEstado{p("x")}})
	tr.nada(t, 50*time.Millisecond)
	tr.falla = errors.New("caído")
	cambiado := p("x")
	cambiado.Panel.Motivo = "otro"
	b.mandarCambios(context.Background(), Estado{Proyectos: []ProyectoEstado{cambiado}})
	if b.base {
		t.Fatal("si no se pudo mandar, los teléfonos ya no tienen base: el próximo es un estado")
	}
}

func TestBucle_ElLimiteDelPlanPublicaLosMasRecientes(t *testing.T) {
	homeTemporal(t)
	registro.Anotar(proyecto(t, "viejo", false), time.Now().Add(-time.Hour))
	registro.Anotar(proyecto(t, "reciente", false), time.Now())
	tr, rl := nuevoTransporte(), nuevoRelayBucle()
	uno := 1
	rl.plan = Plan{Plan: "gratis", Proyectos: &uno}
	b := &Bucle{QDD: "v", T: tr, R: rl, Agrupar: 20 * time.Millisecond, CadaLatido: time.Hour, CadaSondeo: time.Hour}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	go b.Correr(ctx)
	time.Sleep(100 * time.Millisecond)
	tr.eventos <- telefono("d1", "")
	e := tr.esperar(t, time.Second, "estado")
	var est Estado
	json.Unmarshal(e.Payload, &est)
	if len(est.Proyectos) != 1 || est.Proyectos[0].Nombre != "reciente" || est.FueraDelPlan != 1 {
		t.Fatalf("1 dentro (el más reciente), 1 fuera: %+v", est)
	}
}

func TestBucle_PushDeUnPedidoYUnaFichaNuevos(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "Mi App", true)
	consola.Pedir(root, "echo ya", "Ya estaba", "", time.Now())
	registro.Anotar(root, time.Now())
	_, _, rl, _ := bucleDePrueba(t)
	time.Sleep(100 * time.Millisecond)
	select {
	case a := <-rl.avisos:
		t.Fatalf("lo que ya esperaba al arrancar no se avisa: %+v", a)
	default:
	}
	consola.Pedir(root, "echo nuevo", "Nuevo", "", time.Now())
	clave, _ := registro.ClaveDe(root)
	select {
	case a := <-rl.avisos:
		if a.Tipo != AvisoPedido || a.Proyecto != "Mi App" || a.Clave != clave {
			t.Fatalf("push del pedido nuevo: %+v", a)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("un pedido nuevo pide un push")
	}
}

func TestBucle_SiElRelayYaNoReconoceLaMaquinaTermina(t *testing.T) {
	homeTemporal(t)
	tr, rl := nuevoTransporte(), nuevoRelayBucle()
	rl.err = ErrCredenciales
	b := &Bucle{T: tr, R: rl, CadaSondeo: time.Hour}
	fin := make(chan error, 1)
	go func() { fin <- b.Correr(context.Background()) }()
	select {
	case err := <-fin:
		if !errors.Is(err, ErrCredenciales) {
			t.Fatalf("termina con ErrCredenciales: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("una máquina quitada no sigue corriendo")
	}
}

func TestBucle_ElLatidoLlevaLoConsumido(t *testing.T) {
	homeTemporal(t)
	registro.Anotar(proyecto(t, "a", true), time.Now())
	tr, rl := nuevoTransporte(), nuevoRelayBucle()
	b := &Bucle{T: tr, R: rl, CadaLatido: 150 * time.Millisecond, CadaSondeo: time.Hour, Agrupar: 10 * time.Millisecond}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	go b.Correr(ctx)
	time.Sleep(30 * time.Millisecond)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	time.Sleep(300 * time.Millisecond)
	ls := rl.Latidos()
	total := 0
	for _, l := range ls {
		total += l["mensajes"]
	}
	if len(ls) < 2 || total < 2 {
		t.Fatalf("cada latido informa los mensajes desde el anterior (presence y estado): %v", ls)
	}
}

// Pedido de qdd-movile (2026-11-04): los límites de Claude van en el estado y en un delta, solo si cambió lo que se usa
// o cuándo se reinicia: actualizado solo no gasta mensajes del plan.
func TestBucle_LosLimitesVanEnEstadoYEnDeltaSoloSiCambian(t *testing.T) {
	homeTemporal(t)
	registro.Anotar(proyecto(t, "a", true), time.Now())
	entrada := `{"rate_limits":{"five_hour":{"used_percentage":12.2,"resets_at":1793000000}}}`
	consola.GuardarLimites([]byte(entrada), time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	e := tr.esperar(t, time.Second, "estado")
	var est Estado
	json.Unmarshal(e.Payload, &est)
	if est.Limites == nil || est.Limites.CincoHoras.Usado != 12.2 || est.Limites.Semana != nil {
		t.Fatalf("el estado trae los límites: %s", e.Payload)
	}
	consola.GuardarLimites([]byte(entrada), time.Now().Add(consola.CadaLimites+time.Second))
	tr.nada(t, 400*time.Millisecond)
	consola.GuardarLimites([]byte(`{"rate_limits":{"five_hour":{"used_percentage":14,"resets_at":1793000000}}}`), time.Now())
	e = tr.esperar(t, 2*time.Second, "delta")
	var d Delta
	json.Unmarshal(e.Payload, &d)
	if d.Limites == nil || d.Limites.CincoHoras.Usado != 14 || len(d.Proyectos) != 0 {
		t.Fatalf("un delta con solo los límites: %s", e.Payload)
	}
	tr.nada(t, 400*time.Millisecond)
}

func TestEstado_SinLimitesElCampoNoViaja(t *testing.T) {
	homeTemporal(t)
	b, err := json.Marshal(Armar("v", registro.Registro{}, time.Now(), 0))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "limites") {
		t.Fatalf("sin dato no viaja el campo: %s", b)
	}
}
