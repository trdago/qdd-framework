package flujo

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 1: la foto que va al cliente y el archivo HTML de solo lectura con N1 · N2 · N3. La foto nunca lleva
// tareas, ítems ni comentarios internos, y su texto pasa por la limpieza que le da quien la arma.

func flujoConGrupos() Flujo {
	f := deNodos(
		Nodo{ID: "inicio", Tipo: TipoResultado, Nombre: "Llega", Siguen: []string{"a"}},
		Nodo{ID: "a", Tipo: TipoPaso, Nombre: "Revisa", Grupo: "Ingreso", Tarea: "t-1", Siguen: []string{"b", "rechazo"},
			Descripcion: "lo ve en /home/ana/proyecto", Items: []Item{{ID: "i1", Tipo: "nota", Texto: "interna"}},
			Comentarios: []Comentario{{De: DeIA, Texto: "comentario interno", Cuando: "x"}}},
		Nodo{ID: "b", Tipo: TipoPaso, Nombre: "Valida", Grupo: "Ingreso", Siguen: []string{"fin"}},
		Nodo{ID: "rechazo", Tipo: TipoResultado, Nombre: "Rechazado"},
		Nodo{ID: "fin", Tipo: TipoResultado, Nombre: "Listo"},
	)
	f.Revision = 7
	f.Versiones = []Version{{N: 2, Nombre: "acordada", Revision: 5}}
	return f
}

func limpiezaDePrueba(s string) string {
	return strings.ReplaceAll(s, "/home/ana/proyecto", "<proyecto>")
}

func TestFoto_SoloLoQueElClientePuedeVerYLimpio(t *testing.T) {
	foto := ArmarFoto(flujoConGrupos(), limpiezaDePrueba, ahoraDePrueba)
	datos, _ := json.Marshal(foto)
	texto := string(datos)
	for _, prohibido := range []string{"t-1", "interna", "comentario interno", "/home/ana"} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("la foto lleva %q: %s", prohibido, texto)
		}
	}
	if foto.Nodos[1].Descripcion != "lo ve en <proyecto>" || foto.Revision != 7 || foto.Version == nil || foto.Version.N != 2 {
		t.Fatalf("limpia y con su versión: %+v", foto)
	}
	if foto.Plantilla.ID != PlantillaProceso || foto.Plantilla.Niveles[0] != "Hitos" {
		t.Fatalf("lleva la plantilla para la leyenda y los niveles: %+v", foto.Plantilla)
	}
}

func TestFoto_LosTresNivelesVanYaPlegados(t *testing.T) {
	foto := ArmarFoto(flujoConGrupos(), limpiezaDePrueba, ahoraDePrueba)
	n1, n3 := foto.Vistas[0], foto.Vistas[2]
	if len(n3.Nodos) != 5 || len(n1.Nodos) >= len(n3.Nodos) {
		t.Fatalf("N3 es el flujo entero y N1 el panorama: N1 %d, N3 %d", len(n1.Nodos), len(n3.Nodos))
	}
	caja := false
	for _, n := range n1.Nodos {
		caja = caja || (n.Grupo && n.Nombre == "Ingreso" && n.Resumen != "")
	}
	if !caja {
		t.Fatalf("en N1 el grupo es una caja con su resumen: %+v", n1.Nodos)
	}
}

var scriptEnLinea = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

func TestArchivo_UnSoloHTMLSinNadaDeAfueraYConCSPPorHash(t *testing.T) {
	f := flujoConGrupos()
	f.Nodos[2].Nombre = `</script><script>alert(1)</script>`
	html, err := ArchivoHTML(ArmarFoto(f, limpiezaDePrueba, ahoraDePrueba))
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)
	if strings.Contains(s, "alert(1)</script>") {
		t.Fatal("un nombre no puede cerrar el script: los datos van escapados")
	}
	for _, afuera := range []string{"src=\"http", "href=\"http", "@import", "url(http"} {
		if strings.Contains(s, afuera) {
			t.Fatalf("el archivo no carga nada de afuera (%s)", afuera)
		}
	}
	m := scriptEnLinea.FindStringSubmatch(s)
	if m == nil {
		t.Fatal("sin el script del archivo")
	}
	suma := sha256.Sum256([]byte(m[1]))
	if !strings.Contains(s, "'sha256-"+base64.StdEncoding.EncodeToString(suma[:])+"'") || !strings.Contains(s, "default-src 'none'") {
		t.Fatal("la CSP solo deja correr el script del archivo, por su hash")
	}
	if !strings.Contains(s, `<title>Pago`) || !strings.Contains(s, `id="foto"`) {
		t.Fatal("lleva el título y los datos")
	}
}
