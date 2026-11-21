package remoto

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'

// bucleQueSeApaga arranca un bucle con la firma de prueba y devuelve con qué apagarlo, como un servicio que se reinicia.
func bucleQueSeApaga(t *testing.T) (*transporteFalso, func()) {
	t.Helper()
	tr, rl := nuevoTransporte(), nuevoRelayBucle()
	b := &Bucle{QDD: "v-prueba", Cred: Credenciales{Topico: topicoPrueba, Firma: firmaPrueba}, T: tr, R: rl, Bin: os.Args[0],
		Agrupar: 30 * time.Millisecond, CadaLatido: time.Hour, CadaSondeo: 40 * time.Millisecond}
	ctx, cancelar := context.WithCancel(context.Background())
	fin := make(chan error, 1)
	go func() { fin <- b.Correr(ctx) }()
	apagado := false
	apagar := func() {
		if apagado {
			return
		}
		apagado = true
		cancelar()
		<-fin
	}
	t.Cleanup(apagar)
	esperarPrimerLatido()
	return tr, apagar
}

// contrato: remoto#a52626e3
// recordar() guarda cada solicitud en <QDD_HOME>/remoto/solicitudes.json antes de seguir: con el servicio reiniciado,
// el teléfono que reintenta la misma solicitud recibe el último estado (hecha) y el pedido no corre de nuevo.
func TestAccion_LaMismaSolicitudTrasReiniciarNoSeEjecutaDeNuevo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	homeTemporal(t)
	root := proyecto(t, "web", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	marca := filepath.Join(t.TempDir(), "corridas")
	p, err := consola.Pedir(root, "echo x >> '"+marca+"'", "Publicar", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	esperarQueTermine(t, root, p.ID)
	tr, apagar := bucleQueSeApaga(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	a := firmada("sol-reinicio", clave, "pedido:"+p.ID, "d1", ahoraEn())
	mandarAccion(tr, a)
	for _, quiere := range []string{EstadoRecibida, EstadoEjecutando, EstadoHecha} {
		if got := estadoDe(t, tr, 10*time.Second); got.Estado != quiere {
			t.Fatalf("antes de reiniciar, %s: %+v", quiere, got)
		}
	}
	apagar()

	tr2, _ := bucleQueSeApaga(t)
	tr2.eventos <- telefono("d1", "")
	tr2.esperar(t, time.Second, "estado")
	mandarAccion(tr2, a)
	if got := estadoDe(t, tr2, 2*time.Second); got.Estado != EstadoHecha || got.Solicitud != a.Solicitud {
		t.Fatalf("tras reiniciar, la misma solicitud responde su último estado sin volver a mirarla: %+v", got)
	}
	time.Sleep(300 * time.Millisecond)
	if b, _ := os.ReadFile(marca); strings.Count(string(b), "x") != 1 {
		t.Fatalf("el pedido corrió una sola vez: %q", b)
	}
}
