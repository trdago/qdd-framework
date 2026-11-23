package remoto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
// Fotos en los mensajes a la IA (ADR-0003 de qdd-movile, contrato acordado el 2026-11-23): el teléfono manda la foto
// en partes de 48 KB por qdd_remoto_foto (evento foto, firmado como ejecutar sobre id|proyecto|parte|de|huella|
// dispositivo|en). El PC las junta por teléfono e id, verifica bytes, huella y que sea una imagen, la guarda en
// tmp/fotos/movil-<id>.<ext> y responde foto_ack. Una ejecución ia con fotos (firma: lo de siempre más los ids unidos
// por coma) llega a la sesión con «Fotos adjuntas:» y una ruta absoluta por línea; si falta una, termina en fallo
// diciendo cuál. El PC declara capacidades: [fotos] en la cabecera del estado: la app solo ofrece adjuntar si la ve.

type fixtureFotos struct {
	Secreto string `json:"secreto_firma"`
	Foto    struct {
		ID     string `json:"id"`
		Tipo   string `json:"tipo"`
		Bytes  int    `json:"bytes"`
		Huella string `json:"huella"`
	} `json:"foto"`
	Partes   []FotoParte      `json:"partes"`
	Ejecutar EjecucionFirmada `json:"ejecutar"`
}

// leerFixtureFotos es test/fixtures/foto_partes.json de qdd-movile (commit 0989589), copiado sin cambios.
func leerFixtureFotos(t *testing.T) fixtureFotos {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "foto_partes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f fixtureFotos
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// fotoEntera junta los datos de las partes del fixture.
func fotoEntera(t *testing.T, partes []FotoParte) []byte {
	t.Helper()
	var out []byte
	for _, p := range partes {
		d, err := base64.StdEncoding.DecodeString(p.Datos)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, d...)
	}
	return out
}

// partesFirmadas son las partes del fixture para otro proyecto, teléfono y hora, firmadas con la firma de prueba.
func partesFirmadas(t *testing.T, proyecto, dispositivo, en string) []FotoParte {
	t.Helper()
	var out []FotoParte
	for _, p := range leerFixtureFotos(t).Partes {
		p.Proyecto, p.Dispositivo, p.En = proyecto, dispositivo, en
		p.Firma = FirmaFoto(firmaPrueba, p.ID, p.Proyecto, p.Parte, p.De, p.Huella, p.Dispositivo, p.En)
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

func TestFotos_ElFixtureDeQDDMovilSeVerificaYSeArma(t *testing.T) {
	f := leerFixtureFotos(t)
	en, _ := time.Parse(time.RFC3339, f.Partes[0].En)
	for _, p := range f.Partes {
		if err := p.Verificar(f.Secreto, en.Add(time.Minute)); err != nil {
			t.Fatalf("parte %d: %v", p.Parte, err)
		}
	}
	mala := f.Partes[1]
	mala.Huella = strings.Repeat("0", 64)
	if err := mala.Verificar(f.Secreto, en); err == nil {
		t.Fatal("cambiar la huella rompe la firma")
	}
	sinDatos := f.Partes[1]
	sinDatos.Datos = "AAAA"
	if err := sinDatos.Verificar(f.Secreto, en); err != nil {
		t.Fatal("los datos no van firmados: los cubre la huella")
	}

	var a armador
	var datos []byte
	for _, i := range []int{2, 0, 3, 1} {
		d, lista, motivo := a.agregar(f.Partes[i], en)
		if motivo != "" {
			t.Fatalf("parte %d: %s", f.Partes[i].Parte, motivo)
		}
		if lista {
			datos = d
		}
	}
	suma := sha256.Sum256(datos)
	if len(datos) != f.Foto.Bytes || hex.EncodeToString(suma[:]) != f.Foto.Huella {
		t.Fatalf("en cualquier orden arma la foto entera: %d bytes", len(datos))
	}

	if err := f.Ejecutar.Verificar(f.Secreto, en); err != nil {
		t.Fatalf("la ejecución con fotos: %v", err)
	}
	otra := f.Ejecutar
	otra.Fotos = []string{"otra"}
	if err := otra.Verificar(f.Secreto, en); err == nil {
		t.Fatal("las fotos van firmadas")
	}
	sin := f.Ejecutar
	sin.Fotos = nil
	sin.Firma = FirmaEjecucion(f.Secreto, sin.Solicitud, sin.Proyecto, sin.Tipo, sin.Texto, sin.Dispositivo, sin.En)
	if err := sin.Verificar(f.Secreto, en); err != nil {
		t.Fatal("sin fotos la firma es la de siempre")
	}
}

func TestFotos_ElArmadorDiceElMotivo(t *testing.T) {
	f := leerFixtureFotos(t)
	en, _ := time.Parse(time.RFC3339, f.Partes[0].En)

	var a armador
	mal := f.Partes[0]
	mal.De = MaxPartesFoto + 1
	if _, _, m := a.agregar(mal, en); m != MotivoTope {
		t.Fatalf("más de %d partes: %q", MaxPartesFoto, m)
	}
	grande := f.Partes[0]
	grande.Bytes = MaxBytesFoto + 1
	if _, _, m := a.agregar(grande, en); m != MotivoTope {
		t.Fatalf("más de 2 MB: %q", m)
	}

	a = armador{}
	otraHuella := strings.Repeat("0", 64)
	var m string
	for _, p := range f.Partes {
		p.Huella = otraHuella
		_, _, m = a.agregar(p, en)
	}
	if m != MotivoHuella {
		t.Fatalf("la foto no es la de la huella: %q", m)
	}

	a = armador{}
	for i := range MaxArmandoPorTelefono {
		p := f.Partes[0]
		p.ID = "f" + string(rune('a'+i))
		if _, _, m := a.agregar(p, en); m != "" {
			t.Fatal(m)
		}
	}
	quinta := f.Partes[0]
	quinta.ID = "fz"
	if _, _, m := a.agregar(quinta, en); m != MotivoTope {
		t.Fatalf("un teléfono arma hasta %d a la vez: %q", MaxArmandoPorTelefono, m)
	}
	otroTelefono := quinta
	otroTelefono.Dispositivo = "otro"
	if _, _, m := a.agregar(otroTelefono, en); m != "" {
		t.Fatal("el tope es por teléfono")
	}
	if v := a.vencidas(en.Add(VidaArmado - time.Second)); len(v) != 0 {
		t.Fatalf("antes de %v no vence: %+v", VidaArmado, v)
	}
	if v := a.vencidas(en.Add(VidaArmado + time.Second)); len(v) != MaxArmandoPorTelefono+1 {
		t.Fatalf("una incompleta se descarta a los %v: %d", VidaArmado, len(v))
	}
	if _, _, m := a.agregar(quinta, en.Add(VidaArmado+time.Second)); m != "" {
		t.Fatal("las vencidas ya no cuentan para el tope")
	}
}

func TestFotos_LlegaALaSesionConSuRutaAbsoluta(t *testing.T) {
	root, clave, _, tr := bucleConQDDFalso(t)
	partes := partesFirmadas(t, clave, "d1", ahoraEn())
	for _, p := range partes {
		mandarFoto(tr, p)
	}
	ack := ackDe(t, tr, 2*time.Second)
	if !ack.OK || ack.ID != "f1a2b3c4" || ack.Proyecto != clave || ack.Version != 1 {
		t.Fatalf("la foto llegó entera: %+v", ack)
	}
	ruta := filepath.Join(root, "tmp", "fotos", "movil-f1a2b3c4.jpg")
	guardada, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if string(guardada) != string(fotoEntera(t, partes)) {
		t.Fatal("se guarda la foto tal cual")
	}
	info, _ := os.Stat(ruta)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("0600: %v", info.Mode().Perm())
	}

	e := ejecucionFirmada("d6b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", "mira esta captura", "d1", ahoraEn())
	e.Fotos = []string{"f1a2b3c4"}
	e.Firma = FirmaEjecucion(firmaPrueba, e.Solicitud, e.Proyecto, e.Tipo, e.Texto, e.Dispositivo, e.En, e.Fotos...)
	mandarEjecucion(tr, e)
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoRecibida {
		t.Fatalf("queda para la sesión: %+v", got)
	}
	tomadas, _ := consola.TomarInstrucciones(root, "sesion-a", time.Now())
	if len(tomadas) != 1 || tomadas[0].Texto != "mira esta captura" {
		t.Fatalf("el texto queda tal cual: %+v", tomadas)
	}
	msg := consola.MensajeInstruccion(tomadas[0])
	if !strings.HasSuffix(msg, "mira esta captura\n\nFotos adjuntas:\n"+ruta) {
		t.Fatalf("la sesión recibe la ruta absoluta al final: %q", msg)
	}
}

func TestFotos_UnaQueFaltaODeOtroTelefonoTerminaEnFallo(t *testing.T) {
	_, clave, _, tr := bucleConQDDFalso(t)
	for _, p := range partesFirmadas(t, clave, "d1", ahoraEn()) {
		mandarFoto(tr, p)
	}
	if ack := ackDe(t, tr, 2*time.Second); !ack.OK {
		t.Fatalf("%+v", ack)
	}
	casos := []struct {
		solicitud, dispositivo string
		fotos                  []string
		falta                  string
	}{
		{"e6b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", "d1", []string{"f1a2b3c4", "nunca-llego"}, "nunca-llego"},
		{"f6b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", "d2", []string{"f1a2b3c4"}, "f1a2b3c4"},
	}
	for _, c := range casos {
		e := ejecucionFirmada(c.solicitud, clave, "ia", "mira", c.dispositivo, ahoraEn())
		e.Fotos = c.fotos
		e.Firma = FirmaEjecucion(firmaPrueba, e.Solicitud, e.Proyecto, e.Tipo, e.Texto, e.Dispositivo, e.En, e.Fotos...)
		mandarEjecucion(tr, e)
		estadoDe(t, tr, time.Second)
		got := estadoDe(t, tr, 2*time.Second)
		if got.Estado != EstadoFallo || !strings.Contains(got.Resultado, c.falta) {
			t.Fatalf("%s: termina en fallo diciendo cuál (%s): %+v", c.dispositivo, c.falta, got)
		}
	}
}

func TestFotos_RechazosConMotivo(t *testing.T) {
	_, clave, _, tr := bucleConQDDFalso(t)
	ajena := partesFirmadas(t, "proyecto-que-no-esta", "d1", ahoraEn())[0]
	mandarFoto(tr, ajena)
	if ack := ackDe(t, tr, time.Second); ack.OK || ack.Motivo != MotivoProyecto {
		t.Fatalf("un proyecto que el PC no publica: %+v", ack)
	}
	mala := partesFirmadas(t, clave, "d1", ahoraEn())[0]
	mala.ID = "../fuera"
	mala.Firma = FirmaFoto(firmaPrueba, mala.ID, mala.Proyecto, mala.Parte, mala.De, mala.Huella, mala.Dispositivo, mala.En)
	mandarFoto(tr, mala)
	if ack := ackDe(t, tr, time.Second); ack.OK || ack.Motivo != MotivoID {
		t.Fatalf("un id que no es [A-Za-z0-9_-] no toca el disco: %+v", ack)
	}

	e := ejecucionFirmada("a7b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "audit", "d1", ahoraEn())
	e.Fotos = []string{"f1a2b3c4"}
	e.Firma = FirmaEjecucion(firmaPrueba, e.Solicitud, e.Proyecto, e.Tipo, e.Texto, e.Dispositivo, e.En, e.Fotos...)
	mandarEjecucion(tr, e)
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoRechazada || !strings.Contains(got.Resultado, "ia") {
		t.Fatalf("solo una instrucción para la IA lleva fotos: %+v", got)
	}
	cinco := ejecucionFirmada("b7b1c1e0-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", "mira", "d1", ahoraEn())
	cinco.Fotos = []string{"a", "b", "c", "d", "e"}
	cinco.Firma = FirmaEjecucion(firmaPrueba, cinco.Solicitud, cinco.Proyecto, cinco.Tipo, cinco.Texto, cinco.Dispositivo, cinco.En, cinco.Fotos...)
	mandarEjecucion(tr, cinco)
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoRechazada {
		t.Fatalf("hasta %d fotos: %+v", MaxFotosEjecucion, got)
	}
}

func TestEstado_ElPCDeclaraQueRecibeFotos(t *testing.T) {
	e := Armar("v", registro.Registro{}, time.Now(), 3)
	b, _ := json.Marshal(e)
	if !strings.Contains(string(b), `"capacidades":["fotos"]`) {
		t.Fatalf("la cabecera del estado dice que este PC recibe fotos: %s", b)
	}
}
