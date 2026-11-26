package remoto

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// FND-209: Realtime descarta sin aviso un broadcast que pasa su tope (256 KB en el plan Free de Supabase). El estado
// completo de este PC medía 249 KB el 2026-11-15 y nadie lo medía: el día que pasara el tope, el teléfono se quedaría
// sin nada y doctor seguiría en verde. Ahora el servicio anota cuánto pesó cada estado completo y doctor lo compara.

func TestPeso_ElServicioAnotaCuantoPesaElEstadoCompleto_FND209(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	pesos := make(chan int, 4)
	_, tr, _, _ := bucleDePrueba(t, func(b *Bucle) { b.AlPesar = func(n int) { pesos <- n } })
	tr.eventos <- telefono("d1", "")
	e := tr.esperar(t, time.Second, "estado")
	select {
	case n := <-pesos:
		if n != len(e.Payload) {
			t.Fatalf("anota los bytes que salieron: %d, salió %d", n, len(e.Payload))
		}
	case <-time.After(time.Second):
		t.Fatal("no anotó el peso del estado")
	}
}

// Lo logrado era lo que más pesaba: 60 entradas con salidas de 2000 runas. Al teléfono va solo el final de cada
// salida, 500 runas con «…» delante; la cantidad y los días no cambian (cambian el contrato).
func TestPeso_LaSalidaDeCadaLogroVaRecortadaYElEstadoBaja_FND209(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	salida := strings.Repeat("línea de salida del pedido número n\n", 60) + "fin"
	salida = string([]rune(salida)[len([]rune(salida))-consola.TopeSalidaLogro:])
	grande := make([]consola.Logro, consola.TopeLogrado)
	for i := range grande {
		grande[i] = consola.Logro{Tipo: consola.LogroPedido, ID: fmt.Sprintf("pedido:%d", i), Titulo: "Publicar", Salida: salida}
	}
	antes, _ := json.Marshal(consola.DatosConsola{Logrado: &grande})
	panel := consola.DatosConsola{Logrado: &grande}
	ocultarDelPanel(t.TempDir(), &panel)
	despues, _ := json.Marshal(panel)
	for _, l := range *panel.Logrado {
		r := []rune(l.Salida)
		if len(r) != TopeSalidaTelefono || r[0] != '…' || !strings.HasSuffix(l.Salida, "fin") {
			t.Fatalf("la salida va recortada a %d runas con su final: %d %q", TopeSalidaTelefono, len(r), string(r[:10]))
		}
	}
	if len(*panel.Logrado) != consola.TopeLogrado {
		t.Fatalf("la cantidad no cambia: %d", len(*panel.Logrado))
	}
	t.Logf("lo logrado de un proyecto: %d bytes antes, %d después", len(antes), len(despues))
	if len(despues) > len(antes)/3 {
		t.Fatalf("el estado baja de forma medible: %d → %d bytes", len(antes), len(despues))
	}
	if corta := salidaParaTelefono("ok"); corta != "ok" {
		t.Fatalf("una salida corta no se toca: %q", corta)
	}
}

func TestPeso_SeGuardaYSeJuzgaContraElTope_FND209(t *testing.T) {
	homeTemporal(t)
	en := time.Date(2026, 11, 15, 6, 0, 0, 0, time.UTC)
	if err := GuardarPeso(249075, en); err != nil {
		t.Fatal(err)
	}
	p, err := LeerPeso()
	if err != nil || p.Bytes != 249075 || !p.En.Equal(en) {
		t.Fatalf("%+v %v", p, err)
	}
	if aviso := AvisoDePeso(249075); !strings.Contains(aviso, "243 KB") || !strings.Contains(aviso, "256 KB") {
		t.Fatalf("a más del 80 %% del tope avisa con los dos números: %q", aviso)
	}
	if aviso := AvisoDePeso(120 * 1024); aviso != "" {
		t.Fatalf("lejos del tope no avisa: %q", aviso)
	}
}
