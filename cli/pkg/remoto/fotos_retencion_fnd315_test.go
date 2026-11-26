package remoto

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/foto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// FND-315: la persona eligió que las fotos del teléfono se borren a los 7 días, y rc.124 solo las borraba al guardar
// otra en el mismo proyecto: sin una foto nueva se quedaban para siempre. qdd remoto las borra por su cuenta al arrancar
// y cada CadaPoda, en todos los proyectos del registro, sin teléfono conectado.

// fotoDeHace deja una imagen en tmp/fotos del proyecto con esa antigüedad.
func fotoDeHace(t *testing.T, root, nombre string, edad time.Duration) string {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(foto.Carpeta))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, nombre)
	if err := os.WriteFile(ruta, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cuando := time.Now().Add(-edad)
	if err := os.Chtimes(ruta, cuando, cuando); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestFotos_SeBorranALos7DiasSinQueLlegueOtra_FND315(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", true)
	registro.Anotar(root, time.Now())
	vieja := fotoDeHace(t, root, "movil-vieja.jpg", 8*24*time.Hour)
	reciente := fotoDeHace(t, root, "movil-reciente.jpg", 6*24*time.Hour)
	propia := fotoDeHace(t, root, "2026-11-01_10-00-00.png", 30*24*time.Hour)
	bucleDePrueba(t, func(b *Bucle) { b.CadaSondeo = 20 * time.Millisecond })

	limite := time.Now().Add(3 * time.Second)
	for time.Now().Before(limite) {
		if _, err := os.Stat(vieja); os.IsNotExist(err) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(vieja); !os.IsNotExist(err) {
		t.Fatal("la de hace 8 días se borra sin foto nueva y sin teléfono conectado")
	}
	if _, err := os.Stat(reciente); err != nil {
		t.Fatal("la de hace 6 días sigue")
	}
	if _, err := os.Stat(propia); err != nil {
		t.Fatal("las de qdd foto no se tocan")
	}
}

func TestFotos_LaPodaEsCadaHora_FND315(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", true)
	registro.Anotar(root, time.Now())
	now := time.Now()
	b := &Bucle{Ahora: func() time.Time { return now }}
	b.podarFotos()
	vieja := fotoDeHace(t, root, "movil-vieja.jpg", 8*24*time.Hour)
	now = now.Add(CadaPoda - time.Minute)
	b.podarFotos()
	if _, err := os.Stat(vieja); err != nil {
		t.Fatal("el sondeo es cada 5 s: no se relee el disco en cada uno")
	}
	now = now.Add(2 * time.Minute)
	b.podarFotos()
	if _, err := os.Stat(vieja); !os.IsNotExist(err) {
		t.Fatal("al cumplirse CadaPoda se borra")
	}
}

// registroDePrueba junta lo que anota el bucle, que escribe desde su goroutine.
type registroDePrueba struct {
	mu  sync.Mutex
	txt strings.Builder
}

func (r *registroDePrueba) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.txt.Write(p)
}

func (r *registroDePrueba) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.txt.String()
}

// esperarBorrada espera a que el bucle borre la ruta.
func esperarBorrada(t *testing.T, ruta string) {
	t.Helper()
	limite := time.Now().Add(3 * time.Second)
	for time.Now().Before(limite) {
		if _, err := os.Stat(ruta); os.IsNotExist(err) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no se borró %s", ruta)
}

// FND-320: qdd remoto borraba fotos de la persona sin anotarlo (lo vio qdd-movile, 2026-11-24). De paso prueba, con el
// bucle corriendo y sin reiniciarlo, la pasada de cada hora: en el PC real otra versión lo reiniciaba antes.
func TestFotos_CadaBorradoQuedaEnElRegistro_FND320(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", true)
	registro.Anotar(root, time.Now())
	var adelanto atomic.Int64
	reg := &registroDePrueba{}
	primera := fotoDeHace(t, root, "movil-primera.jpg", 8*24*time.Hour)
	reciente := fotoDeHace(t, root, "movil-reciente.jpg", 6*24*time.Hour)
	bucleDePrueba(t, func(b *Bucle) {
		b.CadaSondeo, b.Log = 20*time.Millisecond, reg
		b.Ahora = func() time.Time { return time.Now().Add(time.Duration(adelanto.Load())) }
	})
	esperarBorrada(t, primera)
	segunda := fotoDeHace(t, root, "movil-segunda.jpg", 8*24*time.Hour)
	time.Sleep(150 * time.Millisecond)
	if _, err := os.Stat(segunda); err != nil {
		t.Fatal("antes de CadaPoda no hay otra pasada")
	}
	adelanto.Store(int64(CadaPoda + time.Minute))
	esperarBorrada(t, segunda)
	limite := time.Now().Add(time.Second)
	for time.Now().Before(limite) && !strings.Contains(reg.String(), "movil-segunda.jpg") {
		time.Sleep(10 * time.Millisecond)
	}
	for _, n := range []string{"movil-primera.jpg", "movil-segunda.jpg"} {
		if strings.Count(reg.String(), "borrada: "+n) != 1 {
			t.Fatalf("%s queda una vez en el registro: %s", n, reg)
		}
	}
	if _, err := os.Stat(reciente); err != nil || strings.Contains(reg.String(), "movil-reciente") {
		t.Fatal("la de hace 6 días ni se borra ni se anota")
	}
}
