package remoto

import (
	"bytes"
	"crypto/ecdh"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto/cifrado"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// RFC 0027 fase 3 (acordado con qdd-movile, 2026-11-27): el PC tiene su identidad Ed25519 y su X25519, publica la
// clave firmada en la cabecera del estado con la capacidad cifrado1, y abre lo que el teléfono selló después de
// verificar la firma del relay y antes de validar. Lo que no abre nunca se ejecuta: responde el motivo clave.

// conCifrado da claves al bucle de prueba (los ajustes corren en la goroutine de la prueba).
func conCifrado(t *testing.T) func(*Bucle) {
	return func(b *Bucle) {
		if err := GenerarCifrado(&b.Cred, false, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
}

// publicaDe es la X25519 que el teléfono lee de la cabecera del estado, verificada con la identidad.
func publicaDe(t *testing.T, cred Credenciales) []byte {
	t.Helper()
	c, err := cred.ClaveCifrado()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := cifrado.VerificarClave(*c)
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

func sellar(t *testing.T, publica, aad []byte, claro string) string {
	t.Helper()
	s, err := cifrado.Sellar(nil, publica, aad, []byte(claro))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func otraPublica(t *testing.T) []byte {
	t.Helper()
	p, err := ecdh.X25519().NewPrivateKey(bytes.Repeat([]byte{5}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return p.PublicKey().Bytes()
}

// publicaDeTransporte es la X25519 que el teléfono lee del estado completo que pide con sync.
func publicaDeTransporte(t *testing.T, tr *transporteFalso) []byte {
	t.Helper()
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "sync", Payload: json.RawMessage(`{"version":1}`)}
	var e Estado
	json.Unmarshal(tr.esperar(t, time.Second, "estado").Payload, &e)
	if e.Clave == nil {
		t.Fatal("el estado no trae la clave del PC")
	}
	pub, err := cifrado.VerificarClave(*e.Clave)
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

func accionSellada(solicitud, proyecto, accionID string) AccionFirmada {
	en := ahoraEn()
	return AccionFirmada{Version: 1, Solicitud: solicitud, Proyecto: proyecto, AccionID: accionID, Dispositivo: "d1", En: en,
		Firma: FirmaAccion(firmaPrueba, solicitud, proyecto, accionID, "d1", en)}
}

func ejecucionSellada(t *testing.T, publica []byte, solicitud, proyecto, tipo, texto string) EjecucionFirmada {
	t.Helper()
	return ejecucionFirmada(solicitud, proyecto, tipo, sellar(t, publica, cifrado.AADEjecutar(solicitud, proyecto, tipo, nil), texto), "d1", ahoraEn())
}

// contrato: remoto-cifrado#a6ab90fe
// contrato: remoto-cifrado#a57c289d
func TestCifrado_ElEstadoPublicaLaClaveFirmadaYLaCapacidad_RFC0027(t *testing.T) {
	var cred Credenciales
	_, _, _, tr := bucleConQDDFalso(t, conCifrado(t), func(b *Bucle) { cred = b.Cred })
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "sync", Payload: json.RawMessage(`{"version":1}`)}
	var e Estado
	json.Unmarshal(tr.esperar(t, time.Second, "estado").Payload, &e)
	if e.Clave == nil || !strings.Contains(strings.Join(e.Capacidades, ","), cifrado.Capacidad) {
		t.Fatalf("la cabecera trae clave y cifrado1: %+v %v", e.Clave, e.Capacidades)
	}
	if _, err := cifrado.VerificarClave(*e.Clave); err != nil {
		t.Fatalf("la clave publicada está firmada por la identidad: %v", err)
	}
	b, _ := json.Marshal(e)
	if cred.Identidad == "" || strings.Contains(string(b), cred.Identidad) || strings.Contains(string(b), cred.Cifrado) {
		t.Fatalf("ninguna privada va en el estado: %s", b)
	}
}

func TestCifrado_SinClavesNoDeclaraCifrado1(t *testing.T) {
	e := Armar("v", registro.Registro{}, time.Now(), 3)
	if e.Clave != nil || strings.Contains(strings.Join(e.Capacidades, ","), cifrado.Capacidad) {
		t.Fatalf("un estado sin claves no ofrece cifrado: %+v", e)
	}
}

func TestCifrado_UnComandoSelladoCorreComoEnClaro_RFC0027(t *testing.T) {
	_, clave, falso, tr := bucleConQDDFalso(t, conCifrado(t))
	b := publicaDeTransporte(t, tr)
	e := ejecucionSellada(t, b, "1c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "estado check --json")
	mandarEjecucion(tr, e)
	var hecha AccionEstado
	for _, want := range []string{EstadoRecibida, EstadoEjecutando, EstadoHecha} {
		if hecha = estadoDe(t, tr, 5*time.Second); hecha.Estado != want {
			t.Fatalf("esperaba %s: %+v", want, hecha)
		}
	}
	if c := corridas(t, falso); len(c) != 1 || c[0] != "estado check --json" {
		t.Fatalf("corre el comando abierto, una vez: %q", c)
	}
}

// contrato: remoto-cifrado#47ddb269
// contrato: remoto-cifrado#e0c6eaf6
func TestCifrado_LoQueNoAbreNoCorreYDiceClave_RFC0027(t *testing.T) {
	_, clave, falso, tr := bucleConQDDFalso(t, conCifrado(t))
	vieja := ejecucionSellada(t, otraPublica(t), "2c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "estado check --json")
	mandarEjecucion(tr, vieja)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRechazada || got.Motivo != MotivoClave {
		t.Fatalf("sellado hacia otra clave: %+v", got)
	}
	pub := publicaDeTransporte(t, tr)
	copiada := ejecucionSellada(t, pub, "3c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", "estado check --json")
	otra := ejecucionFirmada("4c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "qdd", copiada.Texto, "d1", ahoraEn())
	mandarEjecucion(tr, otra)
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRechazada || got.Motivo != MotivoClave {
		t.Fatalf("un sellado guardado no sirve en otra solicitud: %+v", got)
	}
	time.Sleep(150 * time.Millisecond)
	if c := corridas(t, falso); len(c) != 0 {
		t.Fatalf("nada corrió: %q", c)
	}
	// No se recordó: la app reconfirma la clave y reenvía la misma solicitud, que ahora corre.
	mandarEjecucion(tr, ejecucionSellada(t, pub, vieja.Solicitud, clave, "qdd", "estado check --json"))
	for _, want := range []string{EstadoRecibida, EstadoEjecutando, EstadoHecha} {
		if got := estadoDe(t, tr, 5*time.Second); got.Estado != want {
			t.Fatalf("el reenvío: esperaba %s: %+v", want, got)
		}
	}
	if c := corridas(t, falso); len(c) != 1 {
		t.Fatalf("corrió una vez, la reenviada: %q", c)
	}
}

// Las reglas del texto en claro las aplica el PC después de abrir: el relay ya no las puede ver.
func TestCifrado_ElTextoAbiertoSeValidaIgual_RFC0027(t *testing.T) {
	_, clave, falso, tr := bucleConQDDFalso(t, conCifrado(t))
	pub := publicaDeTransporte(t, tr)
	for i, texto := range []string{"qdd audit", "audit; rm -rf /", strings.Repeat("a", MaxTextoQDD+1)} {
		e := ejecucionSellada(t, pub, "5c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f5"+string(rune('0'+i)), clave, "qdd", texto)
		mandarEjecucion(tr, e)
		estadoDe(t, tr, time.Second)
		if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRechazada || got.Motivo == MotivoClave {
			t.Fatalf("%q: se abre y se rechaza por su texto: %+v", texto, got)
		}
	}
	if c := corridas(t, falso); len(c) != 0 {
		t.Fatalf("nada corrió: %q", c)
	}
}

func TestCifrado_UnaAccionSelladaSeAbreAntesDeBuscarla_RFC0027(t *testing.T) {
	_, clave, _, tr := bucleConQDDFalso(t, conCifrado(t))
	pub := publicaDeTransporte(t, tr)
	solicitud := "6c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50"
	sellado := sellar(t, pub, cifrado.AADAccion(solicitud, clave), "no-existe")
	mandarAccion(tr, accionSellada(solicitud, clave, sellado))
	estadoDe(t, tr, time.Second)
	got := estadoDe(t, tr, time.Second)
	if got.Estado != EstadoRechazada || got.Motivo == MotivoClave || got.AccionID != "no-existe" {
		t.Fatalf("abre el accion_id y lo busca en el panel: %+v", got)
	}
	mandarAccion(tr, accionSellada("7c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, sellar(t, otraPublica(t), cifrado.AADAccion("7c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave), "x")))
	if got := estadoDe(t, tr, time.Second); got.Estado != EstadoRechazada || got.Motivo != MotivoClave || strings.HasPrefix(got.AccionID, cifrado.Prefijo) {
		t.Fatalf("una acción que no abre: %+v", got)
	}
}

// Una respuesta sellada llega como el string JSON "c1.…" que firmó el relay; lo que abre es el objeto de siempre.
func TestCifrado_UnaRespuestaSelladaSeAbre_RFC0027(t *testing.T) {
	_, clave, _, tr := bucleConQDDFalso(t, conCifrado(t))
	pub := publicaDeTransporte(t, tr)
	for _, c := range []struct {
		solicitud string
		publica   []byte
		clave     bool
	}{
		{"8c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", pub, false},
		{"9c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", otraPublica(t), true},
	} {
		texto, _ := json.Marshal(sellar(t, c.publica, cifrado.AADResponder(c.solicitud, clave, "p-1"), `{"respuesta":"Sí"}`))
		r := RespuestaFirmada{Version: 1, Solicitud: c.solicitud, Proyecto: clave, Pregunta: "p-1", Respuesta: string(texto), Dispositivo: "d1", En: ahoraEn()}
		r.Firma = FirmaRespuesta(firmaPrueba, r.Solicitud, r.Proyecto, r.Pregunta, r.Respuesta, r.Dispositivo, r.En)
		p, _ := json.Marshal(r)
		tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "responder", Payload: p}
		var got AccionEstado
		for got = estadoDe(t, tr, time.Second); got.Estado == EstadoRecibida; got = estadoDe(t, tr, time.Second) {
		}
		if (got.Motivo == MotivoClave) != c.clave {
			t.Fatalf("%s: %+v", c.solicitud, got)
		}
	}
}

func TestCifrado_UnaFotoSelladaSeArma_RFC0027(t *testing.T) {
	_, clave, _, tr := bucleConQDDFalso(t, conCifrado(t))
	pub := publicaDeTransporte(t, tr)
	datos := imagenPrueba(t, 40)
	partes := partesDe(clave, datos, 2)
	for _, p := range partes {
		claro, _ := base64.StdEncoding.DecodeString(p.Datos)
		p.Datos = sellar(t, pub, cifrado.AADFoto(p.Proyecto, p.ID, p.Parte, p.De), string(claro))
		mandarFoto(tr, p)
	}
	if ack := ackDe(t, tr, time.Second); !ack.OK {
		t.Fatalf("una foto sellada se arma y verifica su huella: %+v", ack)
	}
	otra := partesDe(clave, imagenPrueba(t, 12), 1)[0]
	claro, _ := base64.StdEncoding.DecodeString(otra.Datos)
	otra.Datos = sellar(t, otraPublica(t), cifrado.AADFoto(otra.Proyecto, otra.ID, 1, 1), string(claro))
	mandarFoto(tr, otra)
	if ack := ackDe(t, tr, time.Second); ack.OK || ack.Motivo != MotivoClave {
		t.Fatalf("una parte que no abre: %+v", ack)
	}
}

// Una parte de 48 KB sellada pasa el tope de los datos en claro (acordado con qdd-movile): se acepta hasta
// MaxDatosParteSellada y, abierta, no puede pasar de MaxParteAbierta.
func TestCifrado_ElTopeDeUnaParteSellada_RFC0027(t *testing.T) {
	var cred Credenciales
	if err := GenerarCifrado(&cred, false, time.Now()); err != nil {
		t.Fatal(err)
	}
	pub := publicaDe(t, cred)
	priv, _ := cred.privadaCifrado()
	id := strings.Repeat("a", 64)
	entera := sellar(t, pub, cifrado.AADFoto("p", id, 1, 2), string(make([]byte, MaxParteAbierta)))
	if len(entera) != MaxDatosParteSellada {
		t.Fatalf("una parte entera sellada mide %d", len(entera))
	}
	if _, motivo := abrirParte(priv, FotoParte{Proyecto: "p", ID: id, Parte: 1, De: 2, Datos: entera}); motivo != "" {
		t.Fatalf("una parte de 48 KB: %s", motivo)
	}
	grande := sellar(t, pub, cifrado.AADFoto("p", id, 1, 2), string(make([]byte, MaxParteAbierta+1)))
	if _, motivo := abrirParte(priv, FotoParte{Proyecto: "p", ID: id, Parte: 1, De: 2, Datos: grande}); motivo != MotivoTope {
		t.Fatalf("una parte de más: %s", motivo)
	}
}

// La identidad no cambia al volver a emparejar; la X25519 sí (RFC 0027, rotación).
// contrato: remoto-cifrado#a57c289d
func TestCifrado_VolverAEmparejarRotaSoloLaDeCifrado_RFC0027(t *testing.T) {
	var cred Credenciales
	if err := GenerarCifrado(&cred, false, time.Now()); err != nil {
		t.Fatal(err)
	}
	antes := cred
	if err := GenerarCifrado(&cred, false, time.Now()); err != nil || cred != antes {
		t.Fatalf("sin rotar, unas claves que existen no cambian: %v", err)
	}
	if err := GenerarCifrado(&cred, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if cred.Identidad != antes.Identidad || cred.Cifrado == antes.Cifrado {
		t.Fatal("rotar cambia la X25519 y deja la identidad")
	}
	o := Oferta{Codigo: "ABCDEFGH", Cred: cred}
	qr := o.ContenidoQR()
	if !strings.Contains(qr, `"identidad":"`+cred.IdentidadPublica()+`"`) || strings.Contains(qr, cred.Identidad) || strings.Contains(qr, cred.Cifrado) {
		t.Fatalf("el QR lleva la identidad pública y ninguna privada: %s", qr)
	}
}

// Un PC emparejado antes del RFC 0027 genera sus claves al arrancar el servicio, una sola vez y sin perder nada.
func TestCifrado_UnPCYaEmparejadoGeneraSusClavesUnaVez_RFC0027(t *testing.T) {
	homeTemporal(t)
	if err := GuardarCredenciales(Credenciales{Formato: 1, Maquina: "m", Topico: "qdd:pc:m:x", Firma: firmaPrueba}); err != nil {
		t.Fatal(err)
	}
	primera, err := AsegurarCifrado(time.Now())
	if err != nil || primera.Identidad == "" || primera.Cifrado == "" || primera.Firma != firmaPrueba {
		t.Fatalf("genera las claves y deja la firma: %+v %v", primera, err)
	}
	segunda, err := AsegurarCifrado(time.Now().Add(time.Hour))
	guardada, _ := LeerCredenciales()
	if err != nil || segunda != primera || guardada != primera {
		t.Fatalf("la segunda vez no cambia nada: %v", err)
	}
}
