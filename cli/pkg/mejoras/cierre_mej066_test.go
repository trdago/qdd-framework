package mejoras

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'

// MEJ-066 (2026-11-12): #149 y #155 se cerraron como resueltos antes de ser tarea y el ciclo los llamó «descartada».
// Corregidas a hecha, pedirCierre dejó dos pedidos «gh issue close» para issues ya cerrados. El ciclo solo lee los
// abiertos: uno que desaparece hay que preguntarlo, y uno que no está abierto no se pide cerrar.

func conIssuesYCierre(motivo string, err error, issues ...Issue) Fuentes {
	f := conIssues(issues...)
	f.Cerrado = func(string, string, int) (string, error) { return motivo, err }
	return f
}

func pedidosDeCierre(root string, now time.Time) []string {
	var out []string
	for _, p := range consola.PedidosA(root, now) {
		if strings.Contains(p.Comando, "gh issue close") {
			out = append(out, p.Comando)
		}
	}
	return out
}

func porRef(t *testing.T, root, ref string) Mejora {
	t.Helper()
	for _, m := range mustLista(t, root) {
		if m.Ref == ref {
			return m
		}
	}
	t.Fatalf("falta %s", ref)
	return Mejora{}
}

// tresEnGitHub deja #1 y #3 como tarea y #2 en la cola (MaxTareas).
func tresEnGitHub(t *testing.T, root string) (Issue, Issue) {
	t.Helper()
	viejo := Issue{Numero: 1, Titulo: "Una mejora vieja", Asociacion: "OWNER", Creado: t0}
	medio := Issue{Numero: 2, Titulo: "Otra mejora", Asociacion: "OWNER", Creado: t0.Add(time.Second)}
	seguridad := Issue{Numero: 3, Titulo: "Un token en la URL", Asociacion: "OWNER", Etiquetas: []string{"security"}, Creado: t0.Add(2 * time.Second)}
	ciclo(t, root, conIssues(viejo, medio, seguridad), t0.Add(time.Minute))
	if porRef(t, root, "github:trdago/qdd-framework#2").Estado != EnCola {
		t.Fatal("#2 espera en la cola")
	}
	return viejo, seguridad
}

func TestCiclo_UnIssueCerradoComoResueltoAntesDeSerTareaQuedaHechaSinPedirCierre_MEJ066(t *testing.T) {
	root := proyecto(t)
	viejo, seguridad := tresEnGitHub(t, root)
	ciclo(t, root, conIssuesYCierre("completed", nil, viejo, seguridad), t0.Add(2*time.Minute))
	m := porRef(t, root, "github:trdago/qdd-framework#2")
	if m.Estado != Hecha || !strings.Contains(m.Motivo, "resuelto") {
		t.Fatalf("cerrado como resuelto no es descartado: %+v", m)
	}
	ciclo(t, root, conIssuesYCierre("completed", nil, viejo, seguridad), t0.Add(3*time.Minute))
	if p := pedidosDeCierre(root, t0.Add(3*time.Minute)); len(p) != 0 {
		t.Fatalf("ya está cerrado: %v", p)
	}
}

func TestCiclo_UnIssueCerradoComoNoPlaneadoSeDescarta_MEJ066(t *testing.T) {
	root := proyecto(t)
	viejo, seguridad := tresEnGitHub(t, root)
	ciclo(t, root, conIssuesYCierre("not_planned", nil, viejo, seguridad), t0.Add(2*time.Minute))
	if m := porRef(t, root, "github:trdago/qdd-framework#2"); m.Estado != Descartada || !strings.Contains(m.Motivo, "GitHub") {
		t.Fatalf("%+v", m)
	}
}

func TestCiclo_SiNoSabeComoSeCerroLoDejaEnLaColaYLoDice_MEJ066(t *testing.T) {
	root := proyecto(t)
	viejo, seguridad := tresEnGitHub(t, root)
	r := ciclo(t, root, conIssuesYCierre("", errors.New("gh: sin red"), viejo, seguridad), t0.Add(2*time.Minute))
	if m := porRef(t, root, "github:trdago/qdd-framework#2"); m.Estado != EnCola {
		t.Fatalf("sin saberlo no se decide: %+v", m)
	}
	if !strings.Contains(strings.Join(r.Avisos, " "), "#2") {
		t.Fatalf("lo dice: %+v", r.Avisos)
	}
}

func TestCiclo_UnaMejoraHechaConElIssueYaCerradoNoPideCerrarlo_MEJ066(t *testing.T) {
	root := proyecto(t)
	issue := Issue{Numero: 12, Titulo: "Mejora del dueño", Asociacion: "OWNER", Creado: t0}
	ciclo(t, root, conIssues(issue), t0)
	if err := estado.Cerrar(root, mejora(t, root, "MEJ-001").Tarea, "commit abc123", false, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	ciclo(t, root, conIssuesYCierre("completed", nil), t0.Add(2*time.Minute))
	if p := pedidosDeCierre(root, t0.Add(2*time.Minute)); len(p) != 0 {
		t.Fatalf("el PR ya lo cerró: %v", p)
	}
	if m := mejora(t, root, "MEJ-001"); m.Estado != Hecha || !m.CierrePedido {
		t.Fatalf("queda hecha y no se vuelve a mirar: %+v", m)
	}
}

func TestCiclo_SinLeerGitHubNoPideCerrarACiegasYLuegoSi_MEJ066(t *testing.T) {
	root := proyecto(t)
	issue := Issue{Numero: 12, Titulo: "Mejora del dueño", Asociacion: "OWNER", Creado: t0}
	ciclo(t, root, conIssues(issue), t0)
	if err := estado.Cerrar(root, mejora(t, root, "MEJ-001").Tarea, "commit abc123", false, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	falla := Fuentes{GitHub: func(string) (string, []Issue, error) { return "", nil, errors.New("gh: sin red") }}
	ciclo(t, root, falla, t0.Add(2*time.Minute))
	if p := pedidosDeCierre(root, t0.Add(2*time.Minute)); len(p) != 0 {
		t.Fatalf("sin saber si sigue abierto no se pide: %v", p)
	}
	ciclo(t, root, conIssues(issue), t0.Add(3*time.Minute))
	if p := pedidosDeCierre(root, t0.Add(3*time.Minute)); len(p) != 1 {
		t.Fatalf("abierto, se pide una vez: %v", p)
	}
}
