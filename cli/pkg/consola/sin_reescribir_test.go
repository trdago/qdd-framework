package consola

import (
	"os"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-161: cada sesión latía cada 5 s y reescribía su cola del teléfono aunque siguiera vacía. Quien vigila esas carpetas
// (qdd remoto y el flujo de cada panel) lo veía como un cambio y volvía a leer todos los proyectos: qdd remoto gastaba del
// 13 al 21 % de un núcleo sin ningún teléfono conectado.

func archivoDe(t *testing.T, ruta string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestColaDelTelefono_SinCambiosNoSeReescribe_FND161(t *testing.T) {
	root := proyectoConCola(t)
	if _, err := TomarInstrucciones(root, "sesion-1", enCola); err != nil {
		t.Fatal(err)
	}
	ruta, err := rutaTelefono(root)
	if err != nil {
		t.Fatal(err)
	}
	antes := archivoDe(t, ruta)
	if _, err := TomarInstrucciones(root, "sesion-1", enCola.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(antes, archivoDe(t, ruta)) {
		t.Fatal("un latido sin instrucciones no debe reescribir la cola")
	}
}

func TestColaDePedidos_SinCambiosNoSeReescribe_FND161(t *testing.T) {
	root := proyectoConCola(t)
	mustPedir(t, root, "echo a", "a", "")
	ruta, err := colaPath(root)
	if err != nil {
		t.Fatal(err)
	}
	sinCambios := func(cola []Pedido) ([]Pedido, error) { return cola, nil }
	if err := conCola(root, enCola, sinCambios); err != nil {
		t.Fatal(err)
	}
	antes := archivoDe(t, ruta)
	if err := conCola(root, enCola.Add(5*time.Second), sinCambios); err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(antes, archivoDe(t, ruta)) {
		t.Fatal("una cola sin cambios no debe reescribirse")
	}
}

func TestColaDePedidos_ConCambiosSiSeEscribe_FND161(t *testing.T) {
	root := proyectoConCola(t)
	mustPedir(t, root, "echo a", "a", "")
	mustPedir(t, root, "echo b", "b", "")
	if got := len(PedidosA(root, enCola)); got != 2 {
		t.Fatalf("los dos pedidos tienen que quedar guardados: %d", got)
	}
}
