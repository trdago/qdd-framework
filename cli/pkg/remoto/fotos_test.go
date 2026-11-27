package remoto

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
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
// Fotos por canal directo (fotos2; decisión de la persona en qdd-movile el 2026-11-26, contrato acordado con
// qdd-movile): el teléfono manda cada foto en partes por broadcast al tópico del PC, sin firma y sin que el relay la
// guarde. El id es el sha256 de la foto entera. El PC las junta en memoria con topes globales (8 fotos y 16 MB, 2 MB y
// 64 partes por foto; una incompleta se descarta a los 60 s y una completa sin ejecutar, a los 10 min), verifica huella
// y tipo y responde foto_ack. Solo un ejecutar firmado que la nombra la escribe en el disco: una foto que ninguno nombra
// nunca lo toca. El PC declara capacidades: [fotos2]; el evento foto viejo (qdd_remoto_foto) ya no se acepta.

// imagenPrueba es un PNG real de lado×lado: foto.Validar mira sus bytes mágicos.
func imagenPrueba(t *testing.T, lado int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewGray(image.Rect(0, 0, lado, lado))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func sha256Hex(datos []byte) string {
	suma := sha256.Sum256(datos)
	return hex.EncodeToString(suma[:])
}

// partesDe corta la foto en n partes como el teléfono: la primera trae tipo y bytes.
func partesDe(proyecto string, datos []byte, n int) []FotoParte {
	id := sha256Hex(datos)
	tam := (len(datos) + n - 1) / n
	var out []FotoParte
	for i := 0; i < n; i++ {
		desde, hasta := i*tam, min((i+1)*tam, len(datos))
		p := FotoParte{Version: 1, Proyecto: proyecto, ID: id, Parte: i + 1, De: n, Datos: base64.StdEncoding.EncodeToString(datos[desde:hasta])}
		if i == 0 {
			p.Tipo, p.Bytes = "image/png", len(datos)
		}
		out = append(out, p)
	}
	return out
}

func mandarFoto(tr *transporteFalso, p FotoParte) {
	b, _ := json.Marshal(p)
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: EventoFoto, Payload: b}
}

func ackDe(t *testing.T, tr *transporteFalso, plazo time.Duration) FotoAck {
	t.Helper()
	e := tr.esperar(t, plazo, EventoFotoAck)
	var a FotoAck
	json.Unmarshal(e.Payload, &a)
	return a
}

func ejecutarConFotos(t *testing.T, tr *transporteFalso, solicitud, proyecto, tipo string, fotos ...string) AccionEstado {
	t.Helper()
	texto := "mira esta captura"
	if tipo == consola.EjecutarQDD {
		texto = "audit"
	}
	e := ejecucionFirmada(solicitud, proyecto, tipo, texto, "d1", ahoraEn())
	e.Fotos = fotos
	e.Firma = FirmaEjecucion(firmaPrueba, e.Solicitud, e.Proyecto, e.Tipo, e.Texto, e.Dispositivo, e.En, e.Fotos...)
	mandarEjecucion(tr, e)
	estadoDe(t, tr, time.Second)
	return estadoDe(t, tr, 2*time.Second)
}

func carpetaFotos(root string) []string {
	entradas, _ := os.ReadDir(filepath.Join(root, "tmp", "fotos"))
	var out []string
	for _, e := range entradas {
		out = append(out, e.Name())
	}
	return out
}

func TestFotos_EnCualquierOrdenSeArmaYQuedaEnMemoria(t *testing.T) {
	datos := imagenPrueba(t, 40)
	partes := partesDe("p", datos, 4)
	var a armador
	en := time.Now()
	for _, i := range []int{2, 0, 3} {
		if m := a.agregar(partes[i], en); m != "" {
			t.Fatalf("parte %d: %s", partes[i].Parte, m)
		}
	}
	if a.lista("p", partes[0].ID) {
		t.Fatal("con una parte menos no está lista")
	}
	if m := a.agregar(partes[1], en); m != "" {
		t.Fatal(m)
	}
	if !a.lista("p", partes[0].ID) {
		t.Fatal("con todas las partes queda lista en memoria")
	}
	if m := a.agregar(partes[1], en); m != "" {
		t.Fatal("una parte repetida de una foto ya lista (el reenvío del teléfono) responde ok otra vez")
	}
}

func TestFotos_ElArmadorDiceElMotivo(t *testing.T) {
	datos := imagenPrueba(t, 8)
	en := time.Now()
	casos := []struct {
		nombre string
		cambia func(*FotoParte)
		motivo string
	}{
		{"id que no es un sha256", func(p *FotoParte) { p.ID = "../fuera" }, MotivoID},
		{"id en mayúscula", func(p *FotoParte) { p.ID = strings.ToUpper(p.ID) }, MotivoID},
		{"más de 64 partes", func(p *FotoParte) { p.De = MaxPartesFoto + 1 }, MotivoTope},
		{"más de 2 MB", func(p *FotoParte) { p.Bytes = MaxBytesFoto + 1 }, MotivoTope},
		{"datos de más", func(p *FotoParte) { p.Datos = strings.Repeat("A", MaxDatosParte+4) }, MotivoTope},
		{"parte fuera de rango", func(p *FotoParte) { p.Parte = 2 }, MotivoIncompleta},
		{"datos ilegibles", func(p *FotoParte) { p.Datos = "no es base64!" }, MotivoHuella},
		{"no es la foto del id", func(p *FotoParte) { p.ID = strings.Repeat("0", 64) }, MotivoHuella},
		{"bytes que no son los dichos", func(p *FotoParte) { p.Bytes++ }, MotivoHuella},
		{"no es una imagen", func(p *FotoParte) {
			texto := []byte("hola, no soy una imagen")
			p.ID, p.Bytes, p.Datos = sha256Hex(texto), len(texto), base64.StdEncoding.EncodeToString(texto)
		}, MotivoTipo},
	}
	for _, c := range casos {
		var a armador
		p := partesDe("p", datos, 1)[0]
		c.cambia(&p)
		if m := a.agregar(p, en); m != c.motivo {
			t.Errorf("%s: motivo %q, se esperaba %q", c.nombre, m, c.motivo)
		}
		if a.lista("p", p.ID) {
			t.Errorf("%s: no queda lista", c.nombre)
		}
	}
}

func TestFotos_LosTopesSonGlobalesYNoEchanAOtraFoto(t *testing.T) {
	en := time.Now()
	var a armador
	var primera string
	for i := range MaxFotosArmando {
		p := partesDe("p", imagenPrueba(t, 2+i), 2)[0]
		if i == 0 {
			primera = p.ID
		}
		if m := a.agregar(p, en); m != "" {
			t.Fatal(m)
		}
	}
	novena := partesDe("p", imagenPrueba(t, 50), 2)[0]
	if m := a.agregar(novena, en); m != MotivoTope {
		t.Fatalf("hasta %d fotos a la vez, vengan de donde vengan: %q", MaxFotosArmando, m)
	}
	resto := partesDe("p", imagenPrueba(t, 2), 2)[1]
	if m := a.agregar(resto, en); m != "" || !a.lista("p", primera) {
		t.Fatalf("la novena no echa a una que ya estaba: %q", m)
	}

	var b armador
	grande := partesDe("p", imagenPrueba(t, 3), 1)[0]
	b.ocupados = MaxBytesArmando - 10
	if m := b.agregar(grande, en); m != MotivoTope {
		t.Fatalf("hasta %d bytes en memoria entre todas: %q", MaxBytesArmando, m)
	}
}

func TestFotos_LasVencidasSeSueltan(t *testing.T) {
	en := time.Now()
	var a armador
	incompleta := partesDe("p", imagenPrueba(t, 5), 2)[0]
	completa := partesDe("p", imagenPrueba(t, 6), 1)[0]
	a.agregar(incompleta, en)
	a.agregar(completa, en)
	if v := a.vencidas(en.Add(VidaArmado - time.Second)); len(v) != 0 {
		t.Fatalf("antes de %v no vence: %+v", VidaArmado, v)
	}
	v := a.vencidas(en.Add(VidaArmado + time.Second))
	if len(v) != 1 || v[0].id != incompleta.ID {
		t.Fatalf("la incompleta se descarta a los %v, avisando: %+v", VidaArmado, v)
	}
	if !a.lista("p", completa.ID) {
		t.Fatal("la completa espera su ejecutar")
	}
	if v := a.vencidas(en.Add(VidaLista + time.Second)); len(v) != 0 || a.lista("p", completa.ID) {
		t.Fatalf("la completa sin ejecutar se suelta a los %v, sin ack: %+v", VidaLista, v)
	}
	if a.ocupados != 0 {
		t.Fatalf("lo soltado deja de contar para el tope: %d bytes", a.ocupados)
	}
}

func TestFotos_SoloElEjecutarFirmadoLaEscribeYLlegaALaSesion(t *testing.T) {
	root, clave, _, tr := bucleConQDDFalso(t)
	datos := imagenPrueba(t, 30)
	partes := partesDe(clave, datos, 3)
	for _, p := range partes {
		mandarFoto(tr, p)
	}
	ack := ackDe(t, tr, 2*time.Second)
	if !ack.OK || ack.ID != partes[0].ID || ack.Proyecto != clave || ack.Version != 1 {
		t.Fatalf("la foto llegó entera: %+v", ack)
	}
	if n := carpetaFotos(root); len(n) != 0 {
		t.Fatalf("sin un ejecutar que la nombre, la foto no toca el disco: %v", n)
	}

	got := ejecutarConFotos(t, tr, "d6b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", partes[0].ID)
	if got.Estado != EstadoRecibida {
		t.Fatalf("queda para la sesión: %+v", got)
	}
	ruta := filepath.Join(root, "tmp", "fotos", "movil-"+partes[0].ID+".png")
	guardada, err := os.ReadFile(ruta)
	if err != nil || !bytes.Equal(guardada, datos) {
		t.Fatalf("se guarda la foto tal cual: %v", err)
	}
	info, _ := os.Stat(ruta)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("0600: %v", info.Mode().Perm())
	}
	tomadas, _ := consola.TomarInstrucciones(root, "sesion-a", time.Now())
	if len(tomadas) != 1 {
		t.Fatalf("una instrucción: %+v", tomadas)
	}
	if msg := consola.MensajeInstruccion(tomadas[0]); !strings.HasSuffix(msg, "mira esta captura\n\nFotos adjuntas:\n"+ruta) {
		t.Fatalf("la sesión recibe la ruta absoluta al final: %q", msg)
	}

	otra := ejecutarConFotos(t, tr, "c6b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", partes[0].ID)
	if otra.Estado != EstadoRecibida {
		t.Fatalf("una foto ya escrita sirve para otro ejecutar: %+v", otra)
	}
}

func TestFotos_UnaQueFaltaTerminaEnFalloSinEscribirNinguna(t *testing.T) {
	root, clave, _, tr := bucleConQDDFalso(t)
	partes := partesDe(clave, imagenPrueba(t, 12), 1)
	mandarFoto(tr, partes[0])
	if ack := ackDe(t, tr, 2*time.Second); !ack.OK {
		t.Fatalf("%+v", ack)
	}
	nunca := strings.Repeat("a", 64)
	got := ejecutarConFotos(t, tr, "e6b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", partes[0].ID, nunca)
	if got.Estado != EstadoFallo || !strings.Contains(got.Resultado, nunca) {
		t.Fatalf("termina en fallo diciendo cuál: %+v", got)
	}
	if n := carpetaFotos(root); len(n) != 0 {
		t.Fatalf("si falta una, no escribe ninguna: %v", n)
	}
}

func TestFotos_RechazosConMotivo(t *testing.T) {
	_, clave, _, tr := bucleConQDDFalso(t)
	mandarFoto(tr, partesDe("proyecto-que-no-esta", imagenPrueba(t, 4), 1)[0])
	if ack := ackDe(t, tr, time.Second); ack.OK || ack.Motivo != MotivoProyecto {
		t.Fatalf("un proyecto que el PC no publica: %+v", ack)
	}
	mala := partesDe(clave, imagenPrueba(t, 4), 1)[0]
	mala.ID = "../fuera"
	mandarFoto(tr, mala)
	if ack := ackDe(t, tr, time.Second); ack.OK || ack.Motivo != MotivoID {
		t.Fatalf("un id que no es un sha256 no toca el disco: %+v", ack)
	}

	sha := strings.Repeat("b", 64)
	if got := ejecutarConFotos(t, tr, "a7b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", sha); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, "ia") {
		t.Fatalf("solo una instrucción para la IA lleva fotos: %+v", got)
	}
	cinco := []string{strings.Repeat("1", 64), strings.Repeat("2", 64), strings.Repeat("3", 64), strings.Repeat("4", 64), strings.Repeat("5", 64)}
	if got := ejecutarConFotos(t, tr, "b7b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", cinco...); got.Estado != EstadoRechazada {
		t.Fatalf("hasta %d fotos: %+v", MaxFotosEjecucion, got)
	}
	if got := ejecutarConFotos(t, tr, "b8b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", "f1a2b3c4"); got.Estado != EstadoRechazada {
		t.Fatalf("una foto se nombra por su sha256: %+v", got)
	}
}

func TestFotos_ElEventoViejoYaNoSeAcepta(t *testing.T) {
	root, clave, _, tr := bucleConQDDFalso(t)
	p := partesDe(clave, imagenPrueba(t, 4), 1)[0]
	b, _ := json.Marshal(p)
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "foto", Payload: b}
	limite := time.After(300 * time.Millisecond)
	for esperando := true; esperando; {
		select {
		case e := <-tr.broadcast:
			if e.Evento == EventoFotoAck {
				t.Fatalf("el evento foto de qdd_remoto_foto ya no se atiende: %s", e.Payload)
			}
		case <-limite:
			esperando = false
		}
	}
	if n := carpetaFotos(root); len(n) != 0 {
		t.Fatalf("%v", n)
	}
}

func TestEstado_ElPCDeclaraFotos2YNoFotos(t *testing.T) {
	e := Armar("v", registro.Registro{}, time.Now(), 3)
	b, _ := json.Marshal(e)
	if !strings.Contains(string(b), `"capacidades":["fotos2"]`) {
		t.Fatalf("la cabecera del estado dice que este PC recibe fotos por canal directo, y solo eso: %s", b)
	}
}

// leerFixtureFoto2 es test/fixtures/foto2_partes.json de qdd-movile (main, 2026-11-26), copiado sin cambios: el payload
// exacto que manda la app, una foto JPEG real de 175 150 bytes en 4 partes.
func leerFixtureFoto2(t *testing.T) (string, []FotoParte) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "foto2_partes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Evento string      `json:"evento"`
		Partes []FotoParte `json:"partes"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f.Evento, f.Partes
}

func TestFotos_ElFixtureDeQDDMovilLlegaYSeEscribeConSuEjecutar(t *testing.T) {
	evento, partes := leerFixtureFoto2(t)
	if evento != EventoFoto || len(partes) != 4 {
		t.Fatalf("el fixture es foto2 en 4 partes: %s, %d", evento, len(partes))
	}
	root, clave, _, tr := bucleConQDDFalso(t)
	for _, i := range []int{3, 1, 0, 2} {
		p := partes[i]
		p.Proyecto = clave
		mandarFoto(tr, p)
	}
	if ack := ackDe(t, tr, 2*time.Second); !ack.OK || ack.ID != partes[0].ID {
		t.Fatalf("la foto de la app llega entera: %+v", ack)
	}
	if got := ejecutarConFotos(t, tr, "f7b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", partes[0].ID); got.Estado != EstadoRecibida {
		t.Fatalf("%+v", got)
	}
	guardada, err := os.ReadFile(filepath.Join(root, "tmp", "fotos", "movil-"+partes[0].ID+".jpg"))
	if err != nil || len(guardada) != 175150 || sha256Hex(guardada) != partes[0].ID {
		t.Fatalf("queda la JPEG de 175 150 bytes con su sha256: %v", err)
	}
}
