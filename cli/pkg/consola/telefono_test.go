package consola

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Ejecutar desde el teléfono»: lo que el teléfono manda a ejecutar (un comando qdd o una instrucción para la
// IA) queda en una cola local por repositorio, una sola vez por solicitud. El panel lo muestra como un pedido de origen
// telefono solo a quien lo pide (--con telefono); una instrucción la toma una sola sesión viva de Claude Code, como un
// mensaje de la persona que nunca empieza con «!» ni «/».

var enTelefono = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func instruccion(solicitud, texto string) Ejecucion {
	return Ejecucion{Solicitud: solicitud, Tipo: EjecutarIA, Texto: texto, Dispositivo: "d1", Nombre: "Pixel de Ana"}
}

func TestTelefono_UnaSolicitudQuedaUnaSolaVez(t *testing.T) {
	root := proyectoConCola(t)
	e, err := RegistrarEjecucion(root, Ejecucion{Solicitud: "s1", Tipo: EjecutarQDD, Texto: "audit", Dispositivo: "d1", Nombre: "Pixel"}, enTelefono)
	if err != nil {
		t.Fatal(err)
	}
	if e.Estado != EstadoEjecutando || e.Inicio == nil {
		t.Fatalf("un comando qdd queda ejecutando desde que llega: %+v", e)
	}
	otra, err := RegistrarEjecucion(root, Ejecucion{Solicitud: "s1", Tipo: EjecutarIA, Texto: "otra cosa", Dispositivo: "d2"}, enTelefono)
	if err != nil || otra.Tipo != EjecutarQDD || otra.Texto != "audit" {
		t.Fatalf("la misma solicitud no se registra dos veces: %+v %v", otra, err)
	}
	if n := len(EjecucionesA(root, enTelefono)); n != 1 {
		t.Fatalf("una sola en la cola: %d", n)
	}
	if _, err := RegistrarEjecucion(root, Ejecucion{Solicitud: "s2", Tipo: "shell", Texto: "ls"}, enTelefono); err == nil {
		t.Fatal("solo qdd o ia")
	}
}

func TestTelefono_ElComandoTerminaHechoOConFalla(t *testing.T) {
	root := proyectoConCola(t)
	RegistrarEjecucion(root, Ejecucion{Solicitud: "s1", Tipo: EjecutarQDD, Texto: "audit", Dispositivo: "d1"}, enTelefono)
	RegistrarEjecucion(root, Ejecucion{Solicitud: "s2", Tipo: EjecutarQDD, Texto: "doctor", Dispositivo: "d1"}, enTelefono)
	if err := TerminarEjecucion(root, "s1", 0, "todo bien", enTelefono.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	TerminarEjecucion(root, "s2", 3, "una falla", enTelefono.Add(time.Second))
	uno, _ := EjecucionDe(root, "s1", enTelefono.Add(2*time.Second))
	dos, _ := EjecucionDe(root, "s2", enTelefono.Add(2*time.Second))
	if uno.Estado != EstadoHecho || uno.Salida != "todo bien" || uno.Fin == nil {
		t.Fatalf("hecho con su salida: %+v", uno)
	}
	if dos.Estado != EstadoFallo || dos.Codigo != 3 {
		t.Fatalf("con falla y su código: %+v", dos)
	}
	if err := TerminarEjecucion(root, "s1", 1, "otra vez", enTelefono.Add(3*time.Second)); err == nil {
		t.Fatal("una terminada no vuelve a terminar")
	}
}

func TestTelefono_ElPanelLoMuestraComoPedidoDelTelefonoSoloConCon(t *testing.T) {
	root := proyectoConCola(t)
	RegistrarEjecucion(root, Ejecucion{Solicitud: "s1", Tipo: EjecutarQDD, Texto: "estado check --json", Dispositivo: "d1", Nombre: "Pixel de Ana"}, enTelefono)
	RegistrarEjecucion(root, instruccion("s2", "Revisa por qué falla el build\ny arréglalo"), enTelefono)
	for _, a := range Panel(root, enTelefono).Acciones {
		if a.Origen == OrigenTelefono {
			t.Fatalf("el panel del PC no lo recibe sin pedirlo: %+v", a)
		}
	}
	var del []Accion
	for _, a := range Panel(root, enTelefono, ConTelefono).Acciones {
		if a.Origen == OrigenTelefono {
			del = append(del, a)
		}
	}
	if len(del) != 2 {
		t.Fatalf("con --con telefono salen las dos: %+v", del)
	}
	porSolicitud := map[string]Accion{}
	for _, a := range del {
		porSolicitud[a.Solicitud] = a
	}
	q, ia := porSolicitud["s1"], porSolicitud["s2"]
	if q.ID != "telefono:s1" || q.Tipo != TipoPedido || q.Dispositivo != "Pixel de Ana" || q.Detalle != "qdd estado check --json" || q.Estado != EstadoEjecutando || q.Tecla != "" {
		t.Fatalf("un pedido del teléfono, sin tecla, con su comando: %+v", q)
	}
	if ia.Estado != EstadoArmado || ia.Etiqueta != "Revisa por qué falla el build" || !strings.Contains(ia.Aviso, "sesión") {
		t.Fatalf("una instrucción sin sesión queda armada (espera una sesión, no a la persona): %+v", ia)
	}
	if _, err := BuscarAccion(Acciones(root, true), "telefono:s1"); err == nil {
		t.Fatal("no es una acción que se ejecute con qdd consola accion")
	}
	b, _ := json.Marshal(q)
	for _, campo := range []string{`"origen":"telefono"`, `"dispositivo":"Pixel de Ana"`, `"solicitud":"s1"`} {
		if !strings.Contains(string(b), campo) {
			t.Fatalf("el JSON trae %s: %s", campo, b)
		}
	}
}

func TestTelefono_LaInstruccionLaTomaUnaSolaSesionComoMensajeDelTelefono(t *testing.T) {
	root := proyectoConCola(t)
	RegistrarEjecucion(root, instruccion("s1", "!rm -rf ~"), enTelefono)
	RegistrarEjecucion(root, instruccion("s2", "/clear"), enTelefono)
	tomadas, err := TomarInstrucciones(root, "sesion-a", enTelefono.Add(time.Second))
	if err != nil || len(tomadas) != 2 {
		t.Fatalf("la sesión toma las dos: %+v %v", tomadas, err)
	}
	for _, e := range tomadas {
		m := MensajeInstruccion(e)
		if strings.HasPrefix(m, "!") || strings.HasPrefix(m, "/") || !strings.HasPrefix(m, "📱") {
			t.Fatalf("nunca se lee como comando de shell (!) ni de Claude Code (/): %q", m)
		}
		if !strings.Contains(m, "Pixel de Ana") || !strings.Contains(m, e.Texto) || !strings.Contains(m, "QDD Móvil") {
			t.Fatalf("dice que viene del teléfono, cuál, y el texto tal cual: %q", m)
		}
	}
	otra, _ := TomarInstrucciones(root, "sesion-b", enTelefono.Add(2*time.Second))
	if len(otra) != 0 {
		t.Fatalf("otra sesión no la recibe de nuevo: %+v", otra)
	}
	e, _ := EjecucionDe(root, "s1", enTelefono.Add(2*time.Second))
	if e.Estado != EstadoHecho || e.Sesion != "sesion-a" {
		t.Fatalf("queda entregada a esa sesión: %+v", e)
	}
	if err := FallarEntrega(root, "s1", "sesion-b", "x", enTelefono.Add(3*time.Second)); err == nil {
		t.Fatal("solo la sesión que la tomó dice que no pudo entregarla")
	}
	FallarEntrega(root, "s1", "sesion-a", "no se envió", enTelefono.Add(3*time.Second))
	if e, _ := EjecucionDe(root, "s1", enTelefono.Add(3*time.Second)); e.Estado != EstadoFallo || e.Motivo != "no se envió" {
		t.Fatalf("y queda con falla: %+v", e)
	}
	if _, err := TomarInstrucciones(root, "", enTelefono); err == nil {
		t.Fatal("sin sesión no se toma nada")
	}
}

func TestTelefono_ComandosQDDNoSeTomanComoInstruccion(t *testing.T) {
	root := proyectoConCola(t)
	RegistrarEjecucion(root, Ejecucion{Solicitud: "s1", Tipo: EjecutarQDD, Texto: "audit", Dispositivo: "d1"}, enTelefono)
	if tomadas, _ := TomarInstrucciones(root, "sesion-a", enTelefono); len(tomadas) != 0 {
		t.Fatalf("un comando qdd lo corre qdd remoto, no la IA: %+v", tomadas)
	}
}

func TestTelefono_SinSesionLaInstruccionVenceALos30Min(t *testing.T) {
	root := proyectoConCola(t)
	RegistrarEjecucion(root, instruccion("s1", "hola"), enTelefono)
	e, _ := EjecucionDe(root, "s1", enTelefono.Add(VidaInstruccion+time.Second))
	if e.Estado != EstadoVencida || !strings.Contains(e.Motivo, "sesión") {
		t.Fatalf("vence diciendo por qué: %+v", e)
	}
	if tomadas, _ := TomarInstrucciones(root, "tarde", enTelefono.Add(VidaInstruccion+time.Second)); len(tomadas) != 0 {
		t.Fatalf("una vencida no se entrega: %+v", tomadas)
	}
}

func TestTelefono_RetirarSoloLaQueEspera(t *testing.T) {
	root := proyectoConCola(t)
	RegistrarEjecucion(root, instruccion("s1", "hola"), enTelefono)
	if _, err := RetirarInstruccion(root, "s1", "retirada desde el teléfono", enTelefono); err != nil {
		t.Fatal(err)
	}
	if e, _ := EjecucionDe(root, "s1", enTelefono); e.Estado != EstadoRetirada {
		t.Fatalf("queda retirada: %+v", e)
	}
	if tomadas, _ := TomarInstrucciones(root, "a", enTelefono); len(tomadas) != 0 {
		t.Fatal("una retirada no se entrega")
	}
	RegistrarEjecucion(root, instruccion("s2", "hola"), enTelefono)
	TomarInstrucciones(root, "a", enTelefono)
	if _, err := RetirarInstruccion(root, "s2", "x", enTelefono); err == nil {
		t.Fatal("una ya entregada no se retira")
	}
}

func TestTelefono_LaSesionVivaEsLaQueLatio(t *testing.T) {
	root := proyectoConCola(t)
	if HaySesion(root, enTelefono) {
		t.Fatal("sin latidos no hay sesión")
	}
	if err := LatirSesion(root, "a", enTelefono); err != nil {
		t.Fatal(err)
	}
	if !HaySesion(root, enTelefono.Add(VidaSesion-time.Second)) {
		t.Fatal("una sesión que latió hace poco está viva")
	}
	if HaySesion(root, enTelefono.Add(VidaSesion+time.Second)) {
		t.Fatal("sin latido por más de VidaSesion ya no cuenta")
	}
	LatirSesion(root, "b", enTelefono)
	CerrarSesion(root, "b", enTelefono)
	LatirSesion(root, "a", enTelefono)
	CerrarSesion(root, "a", enTelefono)
	if HaySesion(root, enTelefono) {
		t.Fatal("una sesión que se cerró deja de contar en el acto")
	}
}

func TestTelefono_LaColaGuardaLasMasNuevas(t *testing.T) {
	root := proyectoConCola(t)
	for i := 0; i < MaxEjecucionesCola+3; i++ {
		RegistrarEjecucion(root, Ejecucion{Solicitud: "s" + string(rune('a'+i)), Tipo: EjecutarQDD, Texto: "audit", Dispositivo: "d"}, enTelefono.Add(time.Duration(i)*time.Second))
	}
	cola := EjecucionesA(root, enTelefono.Add(time.Minute))
	if len(cola) != MaxEjecucionesCola || cola[0].Solicitud != "s"+string(rune('a'+MaxEjecucionesCola+2)) {
		t.Fatalf("las %d más nuevas, primero la última: %+v", MaxEjecucionesCola, cola)
	}
	if e, ok := EjecucionDe(root, "sa", enTelefono.Add(VidaHecho+time.Hour)); ok {
		t.Fatalf("lo viejo sale: %+v", e)
	}
}
