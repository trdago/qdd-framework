package cifrado

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// RFC 0027 fase 2: el fixture compartido. Lo escribe qdd y qdd-movile lo copia sin cambios como prueba de la app: la
// app tiene que sellar exactamente estos bytes con la misma efímera y el PC, abrirlos. Se regenera con
// QDD_ACTUALIZAR_FIXTURE=1; si cambia un byte, se vuelve a acordar con qdd-movile.

const rutaFixture = "../testdata/cifrado1.json"

type fixtureClaves struct {
	IdentidadSemilla string `json:"identidad_semilla"`
	IdentidadPublica string `json:"identidad_publica"`
	Huella           string `json:"huella"`
	X25519Privada    string `json:"x25519_privada"`
	X25519Publica    string `json:"x25519_publica"`
	Clave            Clave  `json:"clave"`
}

type fixtureMensaje struct {
	Caso            string `json:"caso"`
	Evento          string `json:"evento"`
	Campo           string `json:"campo"`
	AAD             string `json:"aad"`
	EfimeraSemilla  string `json:"efimera_semilla"`
	ClaroBase64     string `json:"claro_base64"`
	ClaroTexto      string `json:"claro_texto,omitempty"`
	Sellado         string `json:"sellado"`
	Abre            bool   `json:"abre"`
	AADDelPC        string `json:"aad_del_pc,omitempty"`
	ClavePC         string `json:"clave_pc,omitempty"`
	ExplicacionMalo string `json:"por_que,omitempty"`
}

type fixture struct {
	Nota         string           `json:"_"`
	Construccion []string         `json:"construccion"`
	Claves       fixtureClaves    `json:"claves"`
	Mensajes     []fixtureMensaje `json:"mensajes"`
}

func semilla(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

// armarFixture sella cada caso con una efímera fija: mismos datos, mismos bytes.
func armarFixture(t *testing.T) fixture {
	t.Helper()
	identidad := ed25519.NewKeyFromSeed(semilla(0x11))
	priv, err := ecdh.X25519().NewPrivateKey(semilla(0x22))
	if err != nil {
		t.Fatal(err)
	}
	vieja, err := ecdh.X25519().NewPrivateKey(semilla(0x33))
	if err != nil {
		t.Fatal(err)
	}
	desde := time.Date(2026, 11, 27, 15, 0, 0, 0, time.UTC)
	f := fixture{
		Nota: "RFC 0027 (cifrar en el teléfono lo que va al PC), fase 2. Lo escribe qdd (pkg/remoto/cifrado, " +
			"QDD_ACTUALIZAR_FIXTURE=1) y qdd-movile lo copia sin cambios. Claves de prueba: nunca usar fuera de pruebas. " +
			"Todo en base64url sin relleno salvo claro_base64 (base64 estándar). Un caso con abre=false lo tiene que " +
			"rechazar el PC con el motivo clave, sin ejecutar nada.",
		Construccion: []string{
			"sellado = «c1.» + base64url_sin_relleno(efimera_publica(32) ‖ AES-256-GCM(clave, nonce, claro, aad) con su etiqueta de 16 bytes al final)",
			"efimera: X25519 nueva por campo (aquí, efimera_semilla como privada); compartido = X25519(efimera, x25519_publica del PC)",
			"clave ‖ nonce = HKDF-SHA256(ikm = compartido, sal = efimera_publica ‖ x25519_publica, info = «qdd cifrado1», 44 bytes): los primeros 32 son la clave y los 12 siguientes el nonce",
			"aad = «qdd cifrado1|» + evento + «|» + lo que el relay ve en claro, unido por «|»: ejecutar|solicitud|proyecto|tipo|ids de las fotos unidos por coma (vacío sin fotos) · responder|solicitud|proyecto|pregunta · accion|solicitud|proyecto · foto2|proyecto|id|parte|de",
			"campos sellados: texto de ejecutar, respuesta de responder, accion_id de accion y datos de cada parte de foto2 (los bytes de la parte, no su base64)",
			"clave del estado y de la oferta: {identidad, x25519, firma, desde}; firma = Ed25519(identidad, «qdd cifrado1 clave|» + x25519 + «|» + desde), con x25519 tal como viaja (base64url)",
			"huella = sha256(identidad_publica), 8 primeros bytes en hex mayúscula, en cuatro grupos de cuatro separados por un espacio",
			"el PC declara «cifrado1» en capacidades; sin ella la app no sella nada",
		},
		Claves: fixtureClaves{
			IdentidadSemilla: b64.EncodeToString(identidad.Seed()),
			IdentidadPublica: b64.EncodeToString(identidad.Public().(ed25519.PublicKey)),
			Huella:           Huella(identidad.Public().(ed25519.PublicKey)),
			X25519Privada:    b64.EncodeToString(priv.Bytes()),
			X25519Publica:    b64.EncodeToString(priv.PublicKey().Bytes()),
			Clave:            FirmarClave(identidad, priv.PublicKey().Bytes(), desde),
		},
	}
	const solicitud = "0b1f7c1e-6f3a-4f43-9d55-3c8a1f0e2b10"
	const proyecto = "qdd-a45873ff129c"
	foto := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	idFoto := "967abdd4db74e625512f058a8306fe96ee01e32e21dc94bfe565a8cb00e20537"
	buenos := []struct {
		caso, evento, campo string
		aad                 []byte
		claro               []byte
		texto               bool
		efimera             byte
	}{
		{"comando qdd", "ejecutar", "texto", AADEjecutar(solicitud, proyecto, "qdd", nil), []byte("audit --completo"), true, 0x41},
		{"instrucción con tildes, salto de línea y una foto", "ejecutar", "texto", AADEjecutar(solicitud, proyecto, "ia", []string{idFoto}), []byte("Revisa el «Para ti»\ny dime qué falta 🙂"), true, 0x42},
		{"respuesta a una pregunta", "responder", "respuesta", AADResponder(solicitud, proyecto, "p-7"), []byte(`{"respuesta":"Sí, publicar"}`), true, 0x43},
		{"acción del panel", "accion", "accion_id", AADAccion(solicitud, proyecto), []byte("pedido:a"), true, 0x44},
		{"parte de una foto", "foto2", "datos", AADFoto(proyecto, idFoto, 1, 4), foto, false, 0x45},
	}
	for _, c := range buenos {
		sellado, err := Sellar(bytes.NewReader(semilla(c.efimera)), priv.PublicKey().Bytes(), c.aad, c.claro)
		if err != nil {
			t.Fatal(err)
		}
		m := fixtureMensaje{
			Caso: c.caso, Evento: c.evento, Campo: c.campo, AAD: string(c.aad),
			EfimeraSemilla: b64.EncodeToString(semilla(c.efimera)), ClaroBase64: base64.StdEncoding.EncodeToString(c.claro),
			Sellado: sellado, Abre: true,
		}
		if c.texto {
			m.ClaroTexto = string(c.claro)
		}
		f.Mensajes = append(f.Mensajes, m)
	}
	comando := f.Mensajes[0]
	f.Mensajes = append(f.Mensajes,
		fixtureMensaje{
			Caso: "el mismo sellado en otra solicitud", Evento: "ejecutar", Campo: "texto", AAD: comando.AAD,
			EfimeraSemilla: comando.EfimeraSemilla, ClaroBase64: comando.ClaroBase64, Sellado: comando.Sellado,
			AADDelPC:        string(AADEjecutar("5d0c3a8e-1b2f-4c6d-8e9f-0a1b2c3d4e5f", proyecto, "qdd", nil)),
			ExplicacionMalo: "quien guarda el mensaje no puede repetirlo con otra solicitud",
		},
	)
	selladoViejo, err := Sellar(bytes.NewReader(semilla(0x46)), vieja.PublicKey().Bytes(), AADAccion(solicitud, proyecto), []byte("pedido:a"))
	if err != nil {
		t.Fatal(err)
	}
	f.Mensajes = append(f.Mensajes, fixtureMensaje{
		Caso: "sellado hacia una clave vieja", Evento: "accion", Campo: "accion_id", AAD: string(AADAccion(solicitud, proyecto)),
		EfimeraSemilla: b64.EncodeToString(semilla(0x46)), ClaroBase64: base64.StdEncoding.EncodeToString([]byte("pedido:a")),
		Sellado: selladoViejo, ClavePC: b64.EncodeToString(vieja.PublicKey().Bytes()),
		ExplicacionMalo: "la app usó la x25519 de antes de volver a emparejar: el PC responde clave y la app la reconfirma",
	})
	return f
}

// contrato: remoto-cifrado#10dc5b98
func TestFixture_EsLoQueSellaYAbreQDD_RFC0027(t *testing.T) {
	f := armarFixture(t)
	nuevo, err := json.MarshalIndent(f, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	nuevo = append(nuevo, '\n')
	if os.Getenv("QDD_ACTUALIZAR_FIXTURE") == "1" {
		if err := os.WriteFile(filepath.FromSlash(rutaFixture), nuevo, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	guardado, err := os.ReadFile(filepath.FromSlash(rutaFixture))
	if err != nil {
		t.Fatalf("falta el fixture (QDD_ACTUALIZAR_FIXTURE=1 lo escribe): %v", err)
	}
	if !bytes.Equal(guardado, nuevo) {
		t.Fatal("el fixture compartido no es lo que sella qdd: si el cambio es a propósito, se regenera y se acuerda con qdd-movile")
	}
}

// El PC abre cada caso bueno con su privada y rechaza los malos: es lo mismo que comprueba la app con sus propios
// sellados.
// contrato: remoto-cifrado#9be11cd3
func TestFixture_ElPCAbreLosBuenosYRechazaLosMalos_RFC0027(t *testing.T) {
	var f fixture
	b, err := os.ReadFile(filepath.FromSlash(rutaFixture))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	privBytes, _ := b64.DecodeString(f.Claves.X25519Privada)
	priv, err := ecdh.X25519().NewPrivateKey(privBytes)
	if err != nil {
		t.Fatal(err)
	}
	if pub, err := VerificarClave(f.Claves.Clave); err != nil || !bytes.Equal(pub, priv.PublicKey().Bytes()) {
		t.Fatalf("la clave del fixture no verifica: %v", err)
	}
	for _, m := range f.Mensajes {
		aad := m.AAD
		if m.AADDelPC != "" {
			aad = m.AADDelPC
		}
		claro, err := Abrir(priv, []byte(aad), m.Sellado)
		if !m.Abre {
			if !errors.Is(err, ErrNoAbre) {
				t.Errorf("%s: se abrió o dio otro error: %v", m.Caso, err)
			}
			continue
		}
		esperado, _ := base64.StdEncoding.DecodeString(m.ClaroBase64)
		if err != nil || !bytes.Equal(claro, esperado) {
			t.Errorf("%s: %q %v", m.Caso, claro, err)
		}
	}
}
