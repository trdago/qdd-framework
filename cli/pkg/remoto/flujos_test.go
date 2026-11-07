package remoto

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// Flujos en QDD Móvil (pedido de qdd-movile, 2026-11-06): el panel lleva el resumen y cada flujo viaja en su propio
// evento. Un delta manda el panel entero y Supabase descarta en silencio lo que pasa de 256 KB en el plan Free.

const textoDelNodo = "texto que solo viaja en el evento"

func guardarFlujo(t *testing.T, root, id, descripcion string) {
	t.Helper()
	f := flujo.Flujo{Formato: 1, ID: id, Nombre: "Flujo " + id, Nodos: []flujo.Nodo{
		{ID: "inicio", Tipo: flujo.TipoPaso, Nombre: "Inicio", Descripcion: descripcion, Tarea: "x", Siguen: []string{"fin"}},
		{ID: "fin", Tipo: flujo.TipoResultado, Nombre: "Fin"},
	}}
	if err := flujo.Guardar(root, f); err != nil {
		t.Fatal(err)
	}
}

func proyectoConFlujo(t *testing.T) string {
	t.Helper()
	homeTemporal(t)
	root := proyecto(t, "a", true)
	registro.Anotar(root, time.Now())
	guardarFlujo(t, root, "pago", textoDelNodo)
	return root
}

func mensajesDeFlujo(tr *transporteFalso) []MensajeFlujo {
	mensajes := []MensajeFlujo{}
	for _, e := range tr.Enviados() {
		if e.Evento != "flujo" {
			continue
		}
		var m MensajeFlujo
		json.Unmarshal(e.Payload, &m)
		mensajes = append(mensajes, m)
	}
	return mensajes
}

func esperarMensajesDeFlujo(t *testing.T, tr *transporteFalso, cuantos int) []MensajeFlujo {
	t.Helper()
	limite := time.Now().Add(3 * time.Second)
	for time.Now().Before(limite) {
		if m := mensajesDeFlujo(tr); len(m) >= cuantos {
			return m
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("se esperaban %d eventos flujo; llegaron %d", cuantos, len(mensajesDeFlujo(tr)))
	return nil
}

func resumenDe(t *testing.T, panel *consola.DatosConsola) []flujo.Resumen {
	t.Helper()
	var lista []flujo.Resumen
	if err := json.Unmarshal(panel.Flujos, &lista); err != nil {
		t.Fatalf("flujos del panel: %v (%s)", err, panel.Flujos)
	}
	return lista
}

func estadoBase(t *testing.T, tr *transporteFalso) (Estado, string) {
	t.Helper()
	e := tr.esperar(t, 3*time.Second, "estado")
	var est Estado
	if err := json.Unmarshal(e.Payload, &est); err != nil {
		t.Fatal(err)
	}
	return est, string(e.Payload)
}

func TestFlujos_ElEstadoLlevaElResumenYUnEventoAparteConLosNodos(t *testing.T) {
	proyectoConFlujo(t)
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	est, crudo := estadoBase(t, tr)
	if strings.Contains(crudo, textoDelNodo) {
		t.Fatal("los nodos no viajan en el panel: cada delta los reenviaría")
	}
	lista := resumenDe(t, est.Proyectos[0].Panel)
	if len(lista) != 1 || lista[0].ID != "pago" || lista[0].Nodos != 2 || lista[0].Huella == "" {
		t.Fatalf("el panel lleva el resumen: %+v", lista)
	}
	m := esperarMensajesDeFlujo(t, tr, 1)[0]
	if m.Version != VersionProtocolo || m.Clave != est.Proyectos[0].Clave {
		t.Fatalf("el evento dice su versión y su proyecto: %+v", m)
	}
	if m.Flujo.ID != "pago" || m.Flujo.Huella != lista[0].Huella || len(m.Flujo.Nodos) != 2 {
		t.Fatalf("el evento lleva el flujo con la huella del resumen: %+v", m.Flujo)
	}
	inicio := m.Flujo.Nodos[0]
	if inicio.Descripcion != textoDelNodo || inicio.Tarea != "x" || len(inicio.Siguen) != 1 {
		t.Fatalf("el nodo llega completo: %+v", inicio)
	}
}

func TestFlujos_SinFlujosElPanelLlevaUnaListaVaciaYNingunEvento(t *testing.T) {
	homeTemporal(t)
	registro.Anotar(proyecto(t, "a", true), time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	est, _ := estadoBase(t, tr)
	if got := string(est.Proyectos[0].Panel.Flujos); got != "[]" {
		t.Fatalf("[] dice que este qdd sabe de flujos y el proyecto no tiene: %q", got)
	}
	if m := mensajesDeFlujo(tr); len(m) != 0 {
		t.Fatalf("sin flujos no hay eventos: %+v", m)
	}
}

func TestFlujos_UnSyncLosReenviaConElEstado(t *testing.T) {
	proyectoConFlujo(t)
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	estadoBase(t, tr)
	esperarMensajesDeFlujo(t, tr, 1)
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "sync", Payload: json.RawMessage(`{"version":1}`)}
	esperarMensajesDeFlujo(t, tr, 2)
}

// La app lee el estado de cada tarea de los frentes del panel: si la huella dependiera de él, cada avance gastaría un
// mensaje con el flujo entero.
func TestFlujos_CambiarUnaTareaNoCambiaLaHuellaNiReenviaElFlujo(t *testing.T) {
	root := proyectoConFlujo(t)
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	est, _ := estadoBase(t, tr)
	huella := resumenDe(t, est.Proyectos[0].Panel)[0].Huella
	esperarMensajesDeFlujo(t, tr, 1)
	cambio := estado.Cambio{Frente: "x", Nombre: "X", Etapa: "Diseño", Estado: "falla", Falla: "rompe"}
	if err := estado.Set(root, cambio, time.Now()); err != nil {
		t.Fatal(err)
	}
	e := tr.esperar(t, 3*time.Second, "delta")
	var d Delta
	json.Unmarshal(e.Payload, &d)
	if got := resumenDe(t, d.Proyectos[est.Proyectos[0].Clave])[0].Huella; got != huella {
		t.Fatalf("la huella del teléfono depende solo del archivo: %s → %s", huella, got)
	}
	if m := mensajesDeFlujo(tr); len(m) != 1 {
		t.Fatalf("un cambio de tarea no manda el flujo: %d eventos", len(m))
	}
}

func TestFlujos_CambiarElArchivoMandaSoloEseFlujo(t *testing.T) {
	root := proyectoConFlujo(t)
	guardarFlujo(t, root, "cobro", "otro")
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	esperarMensajesDeFlujo(t, tr, 2)
	guardarFlujo(t, root, "pago", "cambiado")
	m := esperarMensajesDeFlujo(t, tr, 3)
	if ultimo := m[2].Flujo; ultimo.ID != "pago" || ultimo.Nodos[0].Descripcion != "cambiado" {
		t.Fatalf("va el flujo que cambió: %+v", ultimo)
	}
	time.Sleep(150 * time.Millisecond)
	if m := mensajesDeFlujo(tr); len(m) != 3 {
		t.Fatalf("el otro flujo no se reenvía: %d eventos", len(m))
	}
}

func TestFlujos_UnArchivoRotoLlevaSuProblemaYNoManda(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "a", true)
	registro.Anotar(root, time.Now())
	carpeta := filepath.Join(root, flujo.Carpeta)
	os.MkdirAll(carpeta, 0o755)
	os.WriteFile(filepath.Join(carpeta, "roto.yaml"), []byte("nodos: ["), 0o644)
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	est, _ := estadoBase(t, tr)
	lista := resumenDe(t, est.Proyectos[0].Panel)
	if len(lista) != 1 || lista[0].Problema == "" {
		t.Fatalf("el archivo roto se ve con su problema: %+v", lista)
	}
	time.Sleep(100 * time.Millisecond)
	if m := mensajesDeFlujo(tr); len(m) != 0 {
		t.Fatalf("de un archivo roto no hay evento: %+v", m)
	}
}

// RFC 0013, inv. 16: nada que parezca un secreto sale del PC.
func TestFlujos_LoQueParezcaUnSecretoNoSaleDelPC(t *testing.T) {
	secreto := "ghp_" + strings.Repeat("a1", 18)
	homeTemporal(t)
	root := proyecto(t, "a", true)
	registro.Anotar(root, time.Now())
	f := flujo.Flujo{Formato: 1, ID: "pago", Nombre: "con " + secreto, Descripcion: "con " + secreto, Nodos: []flujo.Nodo{
		{ID: "inicio", Tipo: flujo.TipoResultado, Nombre: "con " + secreto, Descripcion: "con " + secreto},
	}}
	if err := flujo.Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	estadoBase(t, tr)
	esperarMensajesDeFlujo(t, tr, 1)
	for _, e := range tr.Enviados() {
		if strings.Contains(string(e.Payload), secreto) {
			t.Fatalf("el evento %s lleva el secreto: %s", e.Evento, e.Payload)
		}
	}
	m := mensajesDeFlujo(tr)[0].Flujo
	if !strings.Contains(m.Nodos[0].Nombre, Oculto) || !strings.Contains(m.Descripcion, Oculto) {
		t.Fatalf("queda el texto con el secreto tapado: %+v", m)
	}
}

// Supabase descarta en silencio un broadcast que pasa de 256 KB en el plan Free.
func TestFlujos_ElMasGrandeQueSePuedeGuardarCabeEnUnMensaje(t *testing.T) {
	nodos := make([]flujo.Nodo, 200)
	for i := range nodos {
		nodos[i] = flujo.Nodo{ID: fmt.Sprintf("n%d", i), Tipo: flujo.TipoResultado, Nombre: "n", Descripcion: strings.Repeat("«a» ", 65), Siguen: []string{fmt.Sprintf("n%d", i+1)}}
	}
	nodos[199].Siguen = nil
	root := t.TempDir()
	if err := flujo.Guardar(root, flujo.Flujo{Formato: 1, ID: "grande", Nombre: "Grande", Nodos: nodos}); err != nil {
		t.Fatalf("este flujo tiene que caber en el tope: %v", err)
	}
	b, err := json.Marshal(MensajeFlujo{Version: VersionProtocolo, Clave: "c", Flujo: FlujoTelefono{ID: "grande", Nombre: "Grande", Huella: "h", Nodos: nodos}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("evento de 200 nodos: %d KB", len(b)/1024)
	if len(b) > 256*1024 {
		t.Fatalf("el evento pesa %d KB: Supabase lo descartaría en el plan Free", len(b)/1024)
	}
}

// Los comentarios del flujo son del panel: no viajan al teléfono, que todavía no los muestra.
func TestMensajeFlujo_LosComentariosSalenSinSecretos(t *testing.T) {
	root := t.TempDir()
	f := flujo.Flujo{Formato: flujo.Formato, ID: "pago", Nombre: "Pago", Nodos: []flujo.Nodo{{
		ID: "a", Tipo: flujo.TipoPaso, Nombre: "A",
		Comentarios: []flujo.Comentario{{De: flujo.DePersona, Texto: "¿y la clave ghp_abcdefghijklmnopqrstuvwxyz0123456789?", Cuando: "2026-11-06T10:00:00Z", DeNodo: "ghp_abcdefghijklmnopqrstuvwxyz0123456789"}},
	}}}
	if err := flujo.Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	b := &Bucle{}
	b.extras.ocultadores = map[string]ocultadorLeido{}
	m, err := b.mensajeFlujo(ProyectoEstado{Ruta: root, Clave: "c"}, flujo.Resumen{ID: "pago", Huella: "h"})
	if err != nil {
		t.Fatal(err)
	}
	datos, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(datos), "ghp_") {
		t.Fatalf("salió un secreto del PC (inv. 16): %s", datos)
	}
	got := m.Flujo.Nodos[0].Comentarios
	if len(got) != 1 || got[0].De != flujo.DePersona || got[0].Cuando != "2026-11-06T10:00:00Z" || !strings.Contains(got[0].Texto, "¿y la clave") {
		t.Fatalf("el comentario llega con lo demás intacto: %+v", got)
	}
}

func TestResumenDeFlujos_LlevaLoPorEnviar(t *testing.T) {
	root := t.TempDir()
	f := flujo.Flujo{Formato: flujo.Formato, ID: "pago", Nombre: "Pago", Nodos: []flujo.Nodo{{ID: "a", Tipo: flujo.TipoPaso, Nombre: "A"}}}
	if err := flujo.Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	if _, _, err := flujo.Editar(root, "pago", flujo.DePersona, time.Now(), flujo.Comentar("a", "hola", flujo.DePersona, time.Now())); err != nil {
		t.Fatal(err)
	}
	var lista []map[string]any
	if err := json.Unmarshal(resumenDeFlujos(root), &lista); err != nil {
		t.Fatal(err)
	}
	if lista[0]["por_enviar"] != float64(1) || lista[0]["puede_deshacer"] != true {
		t.Fatalf("resumen: %v", lista[0])
	}
}

// Las preguntas sobre el flujo completo (2026-11-07) viajan al teléfono como las de un nodo: sin secretos (inv. 16).
func TestMensajeFlujo_LasPreguntasDelFlujoSalenSinSecretos(t *testing.T) {
	root := t.TempDir()
	f := flujo.Flujo{Formato: flujo.Formato, ID: "pago", Nombre: "Pago", Nodos: []flujo.Nodo{{ID: "a", Tipo: flujo.TipoPaso, Nombre: "A"}},
		Comentarios: []flujo.Comentario{{De: flujo.DePersona, Texto: "¿y ghp_abcdefghijklmnopqrstuvwxyz0123456789?", Cuando: "2026-11-07T10:00:00Z", Pregunta: true}}}
	if err := flujo.Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	b := &Bucle{}
	b.extras.ocultadores = map[string]ocultadorLeido{}
	m, err := b.mensajeFlujo(ProyectoEstado{Ruta: root, Clave: "c"}, flujo.Resumen{ID: "pago", Huella: "h"})
	if err != nil {
		t.Fatal(err)
	}
	datos, _ := json.Marshal(m)
	if strings.Contains(string(datos), "ghp_") {
		t.Fatalf("salió un secreto del PC (inv. 16): %s", datos)
	}
	if got := m.Flujo.Comentarios; len(got) != 1 || !got[0].Pregunta || !strings.HasPrefix(got[0].Texto, "¿y") {
		t.Fatalf("la pregunta llega marcada: %+v", got)
	}
}

// Los ítems de un nodo (2026-11-07) viajan al teléfono sin secretos (inv. 16), con su estado intacto.
func TestMensajeFlujo_LosItemsSalenSinSecretos(t *testing.T) {
	root := t.TempDir()
	clave := "ghp_abcdefghijklmnopqrstuvwxyz0123456789"
	f := flujo.Flujo{Formato: flujo.Formato, ID: "pago", Nombre: "Pago", Nodos: []flujo.Nodo{{ID: "a", Tipo: flujo.TipoPaso, Nombre: "A",
		Items: []flujo.Item{{ID: "i1", Tipo: flujo.ItemPrueba, Texto: "entra con " + clave, Estado: flujo.PruebaFalla, Evidencia: "token " + clave}}}}}
	if err := flujo.Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	b := &Bucle{}
	b.extras.ocultadores = map[string]ocultadorLeido{}
	m, err := b.mensajeFlujo(ProyectoEstado{Ruta: root, Clave: "c"}, flujo.Resumen{ID: "pago", Huella: "h"})
	if err != nil {
		t.Fatal(err)
	}
	datos, _ := json.Marshal(m)
	if strings.Contains(string(datos), "ghp_") {
		t.Fatalf("salió un secreto del PC (inv. 16): %s", datos)
	}
	if got := m.Flujo.Nodos[0].Items; len(got) != 1 || got[0].Estado != flujo.PruebaFalla || !strings.HasPrefix(got[0].Texto, "entra con") {
		t.Fatalf("el ítem llega con su estado: %+v", got)
	}
}
