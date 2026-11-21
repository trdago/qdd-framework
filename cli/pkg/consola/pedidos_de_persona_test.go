package consola

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-188 (2026-11-08): la persona confirmó «📮 Admitir MEJ-045» y «MEJ-046» en el panel y los dos fallaron con «un
// reporte del relay lo admite la persona». Un qdd mcp-server de días antes, que no conoce de_persona, reescribió la cola
// al dejar otro pedido y le borró esa marca a los dos. Desde ahora un campo que este qdd no conoce sobrevive a la
// reescritura, y pedir de nuevo el mismo comando como de la persona le devuelve la marca.

func TestPedidos_PedirDeNuevoComoDeLaPersonaLeDevuelveLaMarca_FND188(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "qdd mejoras admitir MEJ-045", "📮 Admitir MEJ-045", "")
	if p.DePersona {
		t.Fatal("uno de la IA no es de la persona")
	}
	q, err := PedirEnOrden(root, "qdd mejoras admitir MEJ-045", "📮 Admitir MEJ-045", "", Orden{DePersona: true}, enCola)
	if err != nil || q.ID != p.ID || !q.DePersona {
		t.Fatalf("el ciclo lo pide de nuevo y vuelve a ser de la persona, sin duplicarlo: %+v %v", q, err)
	}
	if _, err := aceptar(root, p.ID, "sesion-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if !PedidoConfirmado(root, p.ID, "qdd mejoras admitir MEJ-045", time.Now()) {
		t.Fatal("confirmado por la persona, el comando la reconoce")
	}
}

func TestPedidos_UnCampoQueEsteQddNoConoceSobreviveAReescribirLaCola_FND188(t *testing.T) {
	root := proyectoConCola(t)
	ruta, _ := colaPath(root)
	os.MkdirAll(filepath.Dir(ruta), 0o700)
	escrita := `[{"id":"1","comando":"qdd mejoras admitir MEJ-046","motivo":"📮 Admitir MEJ-046","carpeta":"` + root +
		`","creado":"2026-10-30T12:00:00Z","tecla":"a","estado":"pendiente","de_persona":true,"campo_de_un_qdd_nuevo":{"x":1}}]`
	if err := os.WriteFile(ruta, []byte(escrita), 0o600); err != nil {
		t.Fatal(err)
	}
	mustPedir(t, root, "git push origin v1", "Publicar", "")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"campo_de_un_qdd_nuevo"`) || !strings.Contains(string(b), `"de_persona": true`) {
		t.Fatalf("reescribir la cola no borra lo que trae otro qdd:\n%s", b)
	}
	if cola := leerCola(root); len(cola) != 2 || !cola[0].DePersona {
		t.Fatalf("la cola se lee igual: %+v", cola)
	}
}

// contrato: estado#fb185bd9
// La marca de la persona solo se agrega: pedir de nuevo el mismo comando como IA, aceptarlo, que falle, que diga
// «todavía no» y quede armado, dejar de esperarlo y reparar la cola nunca se la quitan. Sin ella, el modo goal lo
// lanzaría como uno de la IA.
func TestPedidos_NingunaEscrituraLeQuitaLaMarcaDeLaPersona_FND188(t *testing.T) {
	root := proyectoConCola(t)
	comando := "qdd mejoras admitir MEJ-047"
	p, err := PedirEnOrden(root, comando, "📮 Admitir MEJ-047", "", Orden{DePersona: true}, enCola)
	if err != nil || !p.DePersona {
		t.Fatalf("pedido como de la persona: %+v %v", p, err)
	}
	sigueDeLaPersona := func(paso string) {
		t.Helper()
		cola := leerCola(root)
		if len(cola) != 1 || !cola[0].DePersona {
			t.Fatalf("%s le quitó la marca de la persona: %+v", paso, cola)
		}
	}
	q, err := Pedir(root, comando, "📮 Admitir MEJ-047", "", enCola)
	if err != nil || q.ID != p.ID || !q.DePersona {
		t.Fatalf("pedirlo de nuevo como IA devuelve el mismo, todavía de la persona: %+v %v", q, err)
	}
	sigueDeLaPersona("pedirlo de nuevo como IA")
	if _, err := aceptar(root, p.ID, "sesion-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	sigueDeLaPersona("aceptarlo")
	terminar(root, p.ID, 1, "falló", false)
	if cola := leerCola(root); cola[0].Estado != EstadoFallo {
		t.Fatalf("quedó con falla: %+v", cola)
	}
	sigueDeLaPersona("una falla")
	if _, err := aceptar(root, p.ID, "sesion-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	terminar(root, p.ID, CodigoTodaviaNo, "todavía no", true)
	if cola := leerCola(root); cola[0].Estado != EstadoArmado {
		t.Fatalf("quedó armado: %+v", cola)
	}
	sigueDeLaPersona("armarlo")
	if err := Desarmar(root, p.ID); err != nil {
		t.Fatal(err)
	}
	sigueDeLaPersona("dejar de esperarlo")
	if _, err := Pedir(root, comando, "otra vez", "", enCola); err != nil {
		t.Fatal(err)
	}
	if _, err := RepararCola(root); err != nil {
		t.Fatal(err)
	}
	sigueDeLaPersona("reparar la cola")
}
