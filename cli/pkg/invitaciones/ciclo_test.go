package invitaciones

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 3, en el tablero: cada propuesta es un pedido de la persona en el panel. Aprobarla la aplica a nombre
// del invitado; descartarla la rechaza. Lo que no cumple el esquema o el rol se rechaza solo, con su motivo. El relay
// se entera de cada resolución y, al aprobar, recibe la foto nueva.

const invPago = "7c1e0f2a-0000-4000-8000-000000000001"

type relayDePrueba struct {
	propuestas []Propuesta
	resueltas  map[string]string
	fotos      int
	caido      bool
}

func (r *relayDePrueba) Propuestas(_ context.Context, desde string, _ time.Time) ([]Propuesta, error) {
	if r.caido {
		return nil, errors.New("sin red")
	}
	var lista []Propuesta
	for _, p := range r.propuestas {
		if numeroDe(p.ID) > numeroDe(desde) {
			lista = append(lista, p)
		}
	}
	return lista, nil
}

func (r *relayDePrueba) Resolver(_ context.Context, propuesta, estado, motivo string, _ time.Time) error {
	if r.caido {
		return errors.New("sin red")
	}
	r.resueltas[propuesta] = estado + "|" + motivo
	return nil
}

func (r *relayDePrueba) SubirFoto(context.Context, string, json.RawMessage, time.Time) (int, error) {
	if r.caido {
		return 0, errors.New("sin red")
	}
	r.fotos++
	return 1, nil
}

type panelDePrueba struct {
	pedidos map[string]string // id → comando
	estados map[string]string // id → estado; sin entrada y sin pedido = descartado
	motivos []string
}

func (p *panelDePrueba) Pedir(comando, motivo string) (string, error) {
	id := "p" + itoa(len(p.pedidos)+1)
	p.pedidos[id], p.estados[id] = comando, "pendiente"
	p.motivos = append(p.motivos, motivo)
	return id, nil
}

func (p *panelDePrueba) Estado(id, comando string) string {
	if p.pedidos[id] != comando {
		return ""
	}
	return p.estados[id]
}

func (p *panelDePrueba) descartar(id string) {
	delete(p.pedidos, id)
	delete(p.estados, id)
}

func entornoDePrueba(t *testing.T) (Entorno, *relayDePrueba, *panelDePrueba) {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	f := flujo.Flujo{Formato: flujo.Formato, ID: "pago", Nombre: "Pago", Nodos: []flujo.Nodo{
		{ID: "a", Tipo: flujo.TipoPaso, Nombre: "Llega", Siguen: []string{"b"}},
		{ID: "b", Tipo: flujo.TipoPaso, Nombre: "Valida", Siguen: []string{"c"}},
		{ID: "c", Tipo: flujo.TipoResultado, Nombre: "Fin"},
	}}
	if err := flujo.Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	r := &relayDePrueba{resueltas: map[string]string{}}
	p := &panelDePrueba{pedidos: map[string]string{}, estados: map[string]string{}}
	e := Entorno{Root: root, Relay: r, Panel: p, Foto: func(string) (json.RawMessage, error) { return json.RawMessage(`{}`), nil }, Ahora: ahoraDePrueba}
	if err := Anotar(root, Invitacion{ID: invPago, Flujo: "pago", Rol: "cliente", Para: "Ana", Vence: ahoraDePrueba.Add(72 * time.Hour), Creada: ahoraDePrueba}); err != nil {
		t.Fatal(err)
	}
	return e, r, p
}

func propuestaDe(n int, autor, operacion string) Propuesta {
	return Propuesta{ID: "PRO-" + itoa(n), Invitacion: invPago, Flujo: "pago", Rol: "cliente", Autor: autor, Base: "r0", Operacion: json.RawMessage(operacion), Estado: Pendiente}
}

func TestCiclo_CadaPropuestaEsUnPedidoDeLaPersona(t *testing.T) {
	e, r, p := entornoDePrueba(t)
	r.propuestas = []Propuesta{propuestaDe(1, "Ana", `{"op":"agregar","flujo":"pago","nodo":"b","nombre":"Cobra"}`)}
	res, err := Ciclo(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}
	if res.Nuevas != 1 || len(p.pedidos) != 1 || p.pedidos["p1"] != "qdd flujo propuestas aprobar PRO-1" {
		t.Fatalf("un pedido para aprobarla: %+v %v", res, p.pedidos)
	}
	if !strings.Contains(p.motivos[0], "Ana") || !strings.Contains(p.motivos[0], "Cobra") {
		t.Fatalf("el motivo dice quién y qué: %q", p.motivos[0])
	}
	if _, err := Ciclo(context.Background(), e); err != nil || len(p.pedidos) != 1 {
		t.Fatalf("la misma propuesta no se pide dos veces: %v %v", err, p.pedidos)
	}
}

func TestCiclo_LoQueNoCumpleSeRechazaSoloConSuMotivo(t *testing.T) {
	e, r, p := entornoDePrueba(t)
	r.propuestas = []Propuesta{
		propuestaDe(1, "Ana", `{"op":"deshacer","flujo":"pago"}`),
		propuestaDe(2, "Ana", `{"op":"quitar","flujo":"otro","nodo":"a"}`),
		propuestaDe(3, "Ana", `{"op":"quitar","flujo":"pago","nodo":"a","extra":1}`),
		propuestaDe(4, "  ", `{"op":"quitar","flujo":"pago","nodo":"a"}`),
	}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(p.pedidos) != 0 || len(r.resueltas) != 4 {
		t.Fatalf("ninguna llega al panel y el relay sabe de las cuatro: %v %v", p.pedidos, r.resueltas)
	}
	for id, res := range r.resueltas {
		if !strings.HasPrefix(res, Rechazada+"|") || len(res) <= len(Rechazada)+1 {
			t.Fatalf("%s: rechazada con su motivo: %q", id, res)
		}
	}
}

func TestCiclo_DeOtraInvitacionNoSeToca(t *testing.T) {
	e, r, p := entornoDePrueba(t)
	ajena := propuestaDe(1, "Ana", `{"op":"quitar","flujo":"pago","nodo":"a"}`)
	ajena.Invitacion = "00000000-0000-4000-8000-000000000009"
	r.propuestas = []Propuesta{ajena}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(p.pedidos) != 0 || len(r.resueltas) != 0 {
		t.Fatal("una propuesta de una invitación de otro proyecto es de ese proyecto")
	}
}

func TestAprobar_LaAplicaASuNombreAvisaYSubeLaFoto(t *testing.T) {
	e, r, _ := entornoDePrueba(t)
	r.propuestas = []Propuesta{propuestaDe(1, "Ana", `{"op":"agregar","flujo":"pago","nodo":"b","nombre":"Cobra"}`)}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	linea, err := Aprobar(context.Background(), e, "PRO-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(linea, "Cobra") || r.resueltas["PRO-1"] != Aprobada+"|" || r.fotos != 1 {
		t.Fatalf("aplicada, avisada y con foto nueva: %q %v %d", linea, r.resueltas, r.fotos)
	}
	f, _ := flujo.Cargar(e.Root, "pago")
	if len(f.Nodos) != 4 || flujo.Historial(e.Root, "pago")[0].Autor != "cliente:Ana" {
		t.Fatalf("el flujo tiene el nodo, a nombre de Ana: %+v", f.Nodos)
	}
	if _, err := Aprobar(context.Background(), e, "PRO-1"); err == nil {
		t.Fatal("una aprobada no se aplica dos veces")
	}
}

func TestCiclo_UnPedidoDescartadoEsUnaRechazada(t *testing.T) {
	e, r, p := entornoDePrueba(t)
	r.propuestas = []Propuesta{propuestaDe(1, "Ana", `{"op":"quitar","flujo":"pago","nodo":"a"}`)}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	p.descartar("p1")
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.resueltas["PRO-1"], Rechazada) {
		t.Fatalf("descartar el pedido la rechaza: %v", r.resueltas)
	}
	f, _ := flujo.Cargar(e.Root, "pago")
	if len(f.Nodos) != 3 {
		t.Fatal("y no toca el flujo")
	}
}

func TestCiclo_UnPedidoRetiradoSePideDeNuevo(t *testing.T) {
	e, r, p := entornoDePrueba(t)
	r.propuestas = []Propuesta{propuestaDe(1, "Ana", `{"op":"quitar","flujo":"pago","nodo":"a"}`)}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	p.estados["p1"] = "retirado"
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(p.pedidos) != 2 || len(r.resueltas) != 0 {
		t.Fatalf("retirado no es rechazado: vuelve al panel: %v %v", p.pedidos, r.resueltas)
	}
}

func TestCiclo_AlPanelVanPocasALaVez(t *testing.T) {
	e, r, p := entornoDePrueba(t)
	for n := 1; n <= MaxAbiertas+2; n++ {
		r.propuestas = append(r.propuestas, propuestaDe(n, "Ana", `{"op":"comentar","flujo":"pago","nodo":"a","texto":"hola"}`))
	}
	res, err := Ciclo(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.pedidos) != MaxAbiertas || res.Esperan != 2 {
		t.Fatalf("el panel no se llena: %d pedidos, %d esperan", len(p.pedidos), res.Esperan)
	}
	if _, err := Aprobar(context.Background(), e, "PRO-1"); err != nil {
		t.Fatal(err)
	}
	p.estados["p1"] = "hecho"
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(p.pedidos) != MaxAbiertas+1 {
		t.Fatalf("al resolverse una, entra la siguiente: %d", len(p.pedidos))
	}
}

func TestCiclo_SinRedNoPierdeNadaYAvisaDespues(t *testing.T) {
	e, r, _ := entornoDePrueba(t)
	r.propuestas = []Propuesta{propuestaDe(1, "Ana", `{"op":"quitar","flujo":"pago","nodo":"a"}`)}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	r.caido = true
	if _, err := Aprobar(context.Background(), e, "PRO-1"); err != nil {
		t.Fatalf("aprobar no depende de la red: %v", err)
	}
	if _, err := Ciclo(context.Background(), e); err == nil {
		t.Fatal("sin red, el ciclo lo dice")
	}
	r.caido = false
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if r.resueltas["PRO-1"] != Aprobada+"|" || r.fotos != 1 {
		t.Fatalf("con red de vuelta, el relay se entera y recibe la foto: %v %d", r.resueltas, r.fotos)
	}
}

func TestRechazar_ConMotivoDesdeLaTerminal(t *testing.T) {
	e, r, _ := entornoDePrueba(t)
	r.propuestas = []Propuesta{propuestaDe(1, "Ana", `{"op":"quitar","flujo":"pago","nodo":"a"}`)}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if err := Rechazar(context.Background(), e, "PRO-1", "ese paso es obligatorio"); err != nil {
		t.Fatal(err)
	}
	if r.resueltas["PRO-1"] != Rechazada+"|ese paso es obligatorio" {
		t.Fatalf("el invitado ve el motivo: %v", r.resueltas)
	}
	if _, err := Aprobar(context.Background(), e, "PRO-1"); err == nil {
		t.Fatal("una rechazada ya no se aprueba")
	}
}

func TestSinInvitacionesVivas_NoLlamaAlRelay(t *testing.T) {
	e, r, _ := entornoDePrueba(t)
	e.Ahora = ahoraDePrueba.Add(100 * time.Hour)
	r.caido = true
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatalf("con la invitación vencida y nada por avisar, el ciclo no sale a la red: %v", err)
	}
}
