package cifrado

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

// RFC 0027 fase 2: el teléfono cifra hacia la clave X25519 del PC el contenido de lo que manda; el PC la firma con su
// identidad Ed25519. Estas pruebas fijan la construcción; el fixture compartido con qdd-movile, sus bytes.

func clavesDePrueba(t *testing.T) (ed25519.PrivateKey, *ecdh.PrivateKey) {
	t.Helper()
	identidad := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, 32))
	cifrado, err := ecdh.X25519().NewPrivateKey(bytes.Repeat([]byte{2}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return identidad, cifrado
}

func TestCifrado_LoQueSellaElTelefonoLoAbreElPC(t *testing.T) {
	_, priv := clavesDePrueba(t)
	aad := AADEjecutar("0b1f7c1e-6f3a-4f43-9d55-3c8a1f0e2b10", "qdd-a45873ff129c", "qdd")
	sellado, err := Sellar(nil, priv.PublicKey().Bytes(), aad, []byte("audit --completo"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sellado, Prefijo) || strings.Contains(sellado, "audit") {
		t.Fatalf("sellado sin prefijo o con el texto en claro: %q", sellado)
	}
	claro, err := Abrir(priv, aad, sellado)
	if err != nil || string(claro) != "audit --completo" {
		t.Fatalf("abrir: %q %v", claro, err)
	}
}

// Lo que el relay sí puede ver (solicitud, proyecto, tipo) va como datos asociados: un texto cifrado no sirve en otra
// solicitud, otro proyecto u otro campo, así que quien administra el relay no puede repetir un comando guardado.
// contrato: remoto-cifrado#9be11cd3
func TestCifrado_UnSelladoNoSirveEnOtroMensaje(t *testing.T) {
	_, priv := clavesDePrueba(t)
	aad := AADEjecutar("s-1", "qdd-a45873ff129c", "qdd")
	sellado, err := Sellar(nil, priv.PublicKey().Bytes(), aad, []byte("audit"))
	if err != nil {
		t.Fatal(err)
	}
	otros := map[string][]byte{
		"otra solicitud": AADEjecutar("s-2", "qdd-a45873ff129c", "qdd"),
		"otro proyecto":  AADEjecutar("s-1", "otro-123", "qdd"),
		"otro tipo":      AADEjecutar("s-1", "qdd-a45873ff129c", "ia"),
		"otro campo":     AADAccion("s-1", "qdd-a45873ff129c"),
	}
	for nombre, otro := range otros {
		if _, err := Abrir(priv, otro, sellado); !errors.Is(err, ErrNoAbre) {
			t.Errorf("%s: se abrió o dio otro error: %v", nombre, err)
		}
	}
}

// Un mensaje sellado hacia una clave vieja (el PC rotó al volver a emparejar) no se abre: el PC responde el motivo clave.
// contrato: remoto-cifrado#47ddb269
func TestCifrado_ConOtraClaveNoAbre(t *testing.T) {
	_, priv := clavesDePrueba(t)
	vieja, err := ecdh.X25519().NewPrivateKey(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	aad := AADAccion("s-1", "p")
	sellado, err := Sellar(nil, vieja.PublicKey().Bytes(), aad, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Abrir(priv, aad, sellado); !errors.Is(err, ErrNoAbre) {
		t.Fatalf("una clave vieja: %v", err)
	}
}

// contrato: remoto-cifrado#47ddb269
func TestCifrado_LoMalFormadoNoSeAbre(t *testing.T) {
	_, priv := clavesDePrueba(t)
	casos := map[string]string{
		"en claro":        "audit",
		"base64 inválido": Prefijo + "%%%",
		"corto":           Prefijo + base64.RawURLEncoding.EncodeToString(make([]byte, 40)),
		"con relleno":     Prefijo + base64.URLEncoding.EncodeToString(make([]byte, 49)),
	}
	for nombre, sellado := range casos {
		if _, err := Abrir(priv, nil, sellado); !errors.Is(err, ErrNoAbre) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
}

func TestCifrado_EsSellado(t *testing.T) {
	if EsSellado("audit") || !EsSellado(Prefijo+"abc") {
		t.Fatal("EsSellado mira solo el prefijo")
	}
}

// La clave de cifrado viaja firmada con la identidad, que el teléfono conoce por el QR: el relay no puede cambiarla.
// contrato: remoto-cifrado#a6ab90fe
func TestCifrado_LaClaveFirmadaSeVerificaConLaIdentidad(t *testing.T) {
	identidad, priv := clavesDePrueba(t)
	desde := time.Date(2026, 11, 27, 15, 0, 0, 0, time.UTC)
	c := FirmarClave(identidad, priv.PublicKey().Bytes(), desde)
	if c.Desde != "2026-11-27T15:00:00Z" {
		t.Fatalf("desde: %q", c.Desde)
	}
	pub, err := VerificarClave(c)
	if err != nil || !bytes.Equal(pub, priv.PublicKey().Bytes()) {
		t.Fatalf("verificar: %v", err)
	}
	otra := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, 32))
	cambiada := c
	cambiada.X25519 = base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	ajena := FirmarClave(otra, priv.PublicKey().Bytes(), desde)
	ajena.Identidad = c.Identidad
	for nombre, mala := range map[string]Clave{"x25519 cambiada": cambiada, "firmada por otra identidad": ajena} {
		if _, err := VerificarClave(mala); !errors.Is(err, ErrClaveFirma) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
}

// La huella corta de la identidad es la misma en la app y en qdd remoto dispositivos.
func TestCifrado_HuellaDeLaIdentidad(t *testing.T) {
	identidad, _ := clavesDePrueba(t)
	h := Huella(identidad.Public().(ed25519.PublicKey))
	if len(h) != 19 || strings.Count(h, " ") != 3 {
		t.Fatalf("cuatro grupos de 4: %q", h)
	}
}
