package consola

import (
	"errors"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Los PRs abiertos del repositorio a un clic en el panel, siempre opcionales (pedido del usuario, 2026-11-02: «las
// autorizaciones de pr también deberían quedar en el panel de la derecha para aceptar en un click (siempre opcional)»).

func pedidosDePR(t *testing.T, root string) map[int]Pedido {
	t.Helper()
	out := map[int]Pedido{}
	for _, p := range PedidosA(root, enCola) {
		if p.PR != 0 {
			out[p.PR] = p
		}
	}
	return out
}

func TestPRs_CadaPRAbiertoQuedaComoPedidoOpcional(t *testing.T) {
	root := proyectoConCola(t)
	abiertos := []PR{{Numero: 69, Titulo: "v2.8.0-rc.21: cada aporte de blueprint en su propio worktree (FND-113)"}, {Numero: 70, Titulo: "wip", Borrador: true}}
	if err := OfrecerPRs(root, abiertos, enCola); err != nil {
		t.Fatal(err)
	}
	prs := pedidosDePR(t, root)
	p, ok := prs[69]
	if !ok || p.Comando != ComandoMergearPR(69) || p.Estado != EstadoPendiente || p.Tecla == "" {
		t.Fatalf("el PR #69 queda como pedido con su tecla y el comando que lo mergea: %+v", prs)
	}
	if !strings.HasPrefix(p.Motivo, "🔀 Mergear #69") || len([]rune(p.Motivo)) > MaxMotivo {
		t.Fatalf("el motivo nombra el PR y cabe en el panel: %q", p.Motivo)
	}
	if _, ok := prs[70]; ok {
		t.Fatal("un borrador no se ofrece")
	}
	if err := OfrecerPRs(root, abiertos, enCola); err != nil || len(pedidosDePR(t, root)) != 1 {
		t.Fatalf("ofrecerlo de nuevo no lo duplica: %v", pedidosDePR(t, root))
	}
}

func TestPRs_UnPRQueSeCerroOSeMergeoAfueraSaleDelPanel(t *testing.T) {
	root := proyectoConCola(t)
	if err := OfrecerPRs(root, []PR{{Numero: 69, Titulo: "a"}, {Numero: 71, Titulo: "b"}}, enCola); err != nil {
		t.Fatal(err)
	}
	if err := OfrecerPRs(root, []PR{{Numero: 71, Titulo: "b"}}, enCola); err != nil {
		t.Fatal(err)
	}
	prs := pedidosDePR(t, root)
	if _, ok := prs[69]; ok || len(prs) != 1 {
		t.Fatalf("el #69 ya no está abierto: sale; el #71 queda: %v", prs)
	}
}

func TestPRs_UnPRQueLaIAYaPidioMergearNoSeRepite(t *testing.T) {
	root := proyectoConCola(t)
	mustPedir(t, root, "bash paso-rc21.sh", "🚀 Mergear #69 y publicar v2.8.0-rc.21 → npm preview", "")
	if err := OfrecerPRs(root, []PR{{Numero: 69, Titulo: "rc.21"}, {Numero: 6, Titulo: "otro"}}, enCola); err != nil {
		t.Fatal(err)
	}
	prs := pedidosDePR(t, root)
	if _, ok := prs[69]; ok {
		t.Fatal("el pedido de la IA ya nombra el #69: no se ofrece otro")
	}
	if _, ok := prs[6]; !ok {
		t.Fatal("«#69» no es «#6»: el #6 sí se ofrece")
	}
}

func TestPRs_UnPRDescartadoNoVuelve(t *testing.T) {
	root := proyectoConCola(t)
	if err := OfrecerPRs(root, []PR{{Numero: 69, Titulo: "a"}}, enCola); err != nil {
		t.Fatal(err)
	}
	if err := Descartar(root, pedidosDePR(t, root)[69].ID); err != nil {
		t.Fatal(err)
	}
	if err := OfrecerPRs(root, []PR{{Numero: 69, Titulo: "a"}}, enCola); err != nil {
		t.Fatal(err)
	}
	if _, ok := pedidosDePR(t, root)[69]; ok {
		t.Fatal("la persona lo descartó: es opcional y no vuelve a ofrecerse")
	}
}

func TestPRs_UnPRNuncaLeQuitaElLugarAUnPedidoDeLaIA(t *testing.T) {
	root := proyectoConCola(t)
	var abiertos []PR
	for n := 1; n <= MaxPedidos; n++ {
		abiertos = append(abiertos, PR{Numero: 100 + n, Titulo: "pr"})
	}
	if err := OfrecerPRs(root, abiertos, enCola); err != nil {
		t.Fatal(err)
	}
	if n := len(pedidosDePR(t, root)); n > MaxPRs {
		t.Fatalf("los PRs ocupan a lo más %d lugares: %d", MaxPRs, n)
	}
	for i := 0; i < MaxPedidos-MaxPRs; i++ {
		mustPedir(t, root, "echo "+string(rune('a'+i)), "pedido de la IA", "")
	}
	if _, err := Pedir(root, "echo lleno", "uno más de la IA", "", enCola); err != nil {
		t.Fatalf("con la cola llena, un PR pendiente cede su lugar al pedido de la IA: %v", err)
	}
	if n := len(pedidosDePR(t, root)); n != MaxPRs-1 {
		t.Fatalf("cedió uno: %d", n)
	}
}

func TestPRs_NoSonEsperasDeLaPersona(t *testing.T) {
	root := proyectoConCola(t)
	if err := OfrecerPRs(root, []PR{{Numero: 69, Titulo: "a"}}, enCola); err != nil {
		t.Fatal(err)
	}
	if esperas := Esperas(root, enCola); len(esperas) != 0 {
		t.Fatalf("un PR es opcional: no se le dice a la IA que el panel lo espera: %v", esperas)
	}
}

func TestMergearPR_DecideSegunElPRYSuCI(t *testing.T) {
	casos := []struct {
		nombre string
		vista  VistaPR
		codigo int
		dice   string
	}{
		{"ya mergeado", VistaPR{Estado: "MERGED"}, 0, "ya estaba mergeado"},
		{"cerrado", VistaPR{Estado: "CLOSED"}, 1, "cerrado"},
		{"borrador", VistaPR{Estado: "OPEN", Borrador: true}, CodigoTodaviaNo, "borrador"},
		{"CI corriendo", VistaPR{Estado: "OPEN", Checks: []CheckPR{{Estado: "IN_PROGRESS"}}}, CodigoTodaviaNo, "CI"},
		{"CI falló", VistaPR{Estado: "OPEN", Checks: []CheckPR{{Estado: "COMPLETED", Conclusion: "FAILURE"}}}, 1, "falló"},
		{"contexto pendiente", VistaPR{Estado: "OPEN", Checks: []CheckPR{{Contexto: "PENDING"}}}, CodigoTodaviaNo, "CI"},
		{"sin CI todavía en un repo con workflows", VistaPR{Estado: "OPEN", ConWorkflows: true}, CodigoTodaviaNo, "CI"},
		{"sin CI en un repo sin workflows", VistaPR{Estado: "OPEN"}, -1, ""},
		{"CI verde", VistaPR{Estado: "OPEN", Checks: []CheckPR{{Estado: "COMPLETED", Conclusion: "SUCCESS"}, {Estado: "COMPLETED", Conclusion: "SKIPPED"}}}, -1, ""},
	}
	for _, c := range casos {
		codigo, dice := veredictoPR(69, c.vista)
		if codigo != c.codigo || !strings.Contains(dice, c.dice) {
			t.Fatalf("%s: se esperaba %d «%s», salió %d «%s»", c.nombre, c.codigo, c.dice, codigo, dice)
		}
	}
}

func TestMergearPR_ConElCIVerdeMergea(t *testing.T) {
	var llamadas []string
	gh := func(dir string, args ...string) ([]byte, error) {
		llamadas = append(llamadas, strings.Join(args, " "))
		if args[1] == "view" {
			return []byte(`{"state":"OPEN","isDraft":false,"statusCheckRollup":[{"status":"COMPLETED","conclusion":"SUCCESS"}]}`), nil
		}
		return nil, nil
	}
	var salida strings.Builder
	if codigo := mergearPRCon(gh, t.TempDir(), 69, &salida); codigo != 0 {
		t.Fatalf("con el CI verde mergea: %d %s", codigo, salida.String())
	}
	if len(llamadas) != 2 || llamadas[1] != "pr merge 69 --merge" {
		t.Fatalf("mergea con un commit de merge: %v", llamadas)
	}
}

func TestMergearPR_SinGHLoDice(t *testing.T) {
	gh := func(dir string, args ...string) ([]byte, error) { return nil, errors.New("exec: gh: not found") }
	var salida strings.Builder
	if codigo := mergearPRCon(gh, t.TempDir(), 69, &salida); codigo != 1 || !strings.Contains(salida.String(), "gh") {
		t.Fatalf("sin gh falla diciendo por qué: %d %s", codigo, salida.String())
	}
}
