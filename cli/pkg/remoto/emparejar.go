package remoto

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"rsc.io/qr"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §Emparejar: el PC genera su id (uuid), el secreto de su canal (128 bits) y su firma (256 bits) con
// crypto/rand, abre una oferta de 5 minutos con el hash de un código de un solo uso y muestra un QR con
// {"v":1,"codigo","topico","nombre"} (y el código en texto). El teléfono acepta el código; el PC consulta hasta ver la
// máquina y recién entonces guarda sus credenciales.

const (
	// VigenciaOferta es lo que dura un código de emparejamiento.
	VigenciaOferta = 5 * time.Minute
	cadaConsulta   = 2 * time.Second
	maxNombre      = 80
)

// Oferta es un emparejamiento en curso: lo que muestra el QR y las credenciales que quedarán si el teléfono acepta.
type Oferta struct {
	Codigo string
	Cred   Credenciales
}

// NuevaOferta genera el código (8 caracteres base32 en mayúsculas, 40 bits), el id, el canal y la firma.
func NuevaOferta(nombre string) (Oferta, error) {
	codigo, err := aleatorio(5)
	if err != nil {
		return Oferta{}, err
	}
	id, err := uuidV4()
	if err != nil {
		return Oferta{}, err
	}
	secreto, err := aleatorio(16)
	if err != nil {
		return Oferta{}, err
	}
	firma, err := aleatorio(32)
	if err != nil {
		return Oferta{}, err
	}
	return Oferta{
		Codigo: base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(codigo),
		Cred: Credenciales{
			Formato: 1, Maquina: id, Nombre: recortar(nombre, maxNombre),
			Topico: "qdd:pc:" + id + ":" + base64.RawURLEncoding.EncodeToString(secreto),
			Firma:  hex.EncodeToString(firma),
		},
	}, nil
}

func aleatorio(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("no se pudo generar un valor aleatorio: %w", err)
	}
	return b, nil
}

func uuidV4() (string, error) {
	b, err := aleatorio(16)
	if err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32], nil
}

func recortar(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) == 0 {
		return "PC"
	}
	if len(r) > n {
		return string(r[:n])
	}
	return string(r)
}

// HashCodigo es lo que guarda el relay: hex(sha256(código en mayúsculas)).
func HashCodigo(codigo string) string {
	suma := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(codigo))))
	return hex.EncodeToString(suma[:])
}

// ContenidoQR es el JSON del QR: versión, código, canal y nombre del PC.
func (o Oferta) ContenidoQR() string {
	b, _ := json.Marshal(struct {
		V      int    `json:"v"`
		Codigo string `json:"codigo"`
		Topico string `json:"topico"`
		Nombre string `json:"nombre"`
	}{1, o.Codigo, o.Cred.Topico, o.Cred.Nombre})
	return string(b)
}

// QRTerminal dibuja el QR con medios bloques (dos filas por línea), con su margen claro: se lee en una terminal oscura.
func QRTerminal(texto string) (string, error) {
	c, err := qr.Encode(texto, qr.M)
	if err != nil {
		return "", err
	}
	margen := 2
	oscuro := func(x, y int) bool {
		if x < 0 || y < 0 || x >= c.Size || y >= c.Size {
			return false
		}
		return c.Black(x, y)
	}
	var sb strings.Builder
	for y := -margen; y < c.Size+margen; y += 2 {
		for x := -margen; x < c.Size+margen; x++ {
			sb.WriteString(medioBloque(oscuro(x, y), oscuro(x, y+1)))
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// medioBloque: lo claro se dibuja (█), lo oscuro queda en el fondo de la terminal.
func medioBloque(arribaOscuro, abajoOscuro bool) string {
	if !arribaOscuro && !abajoOscuro {
		return "█"
	}
	if !arribaOscuro {
		return "▀"
	}
	if !abajoOscuro {
		return "▄"
	}
	return " "
}

// RelayEmparejar es lo que usa Emparejar del relay.
type RelayEmparejar interface {
	Ofrecer(ctx context.Context, codigoHash, nombre, topico, firma string) error
	Emparejada(ctx context.Context, codigoHash string) (string, error)
}

// ErrOfertaVencida: nadie aceptó el código en VigenciaOferta.
var ErrOfertaVencida = errors.New("el código venció sin que un teléfono lo aceptara: vuelve a intentarlo con 'qdd remoto emparejar'")

// Emparejar ofrece la máquina, muestra el QR y el código y espera hasta plazo a que un teléfono lo acepte. Devuelve las
// credenciales (sin guardarlas). cada es cada cuánto consulta (0: 2 s).
func Emparejar(ctx context.Context, r RelayEmparejar, nombre string, salida io.Writer, cada, plazo time.Duration) (Credenciales, error) {
	o, err := NuevaOferta(nombre)
	if err != nil {
		return Credenciales{}, err
	}
	hash := HashCodigo(o.Codigo)
	if err := r.Ofrecer(ctx, hash, o.Cred.Nombre, o.Cred.Topico, o.Cred.Firma); err != nil {
		return Credenciales{}, fmt.Errorf("no se pudo abrir el emparejamiento: %w", err)
	}
	dibujo, err := QRTerminal(o.ContenidoQR())
	if err != nil {
		return Credenciales{}, err
	}
	fmt.Fprintf(salida, "Escanea este código con QDD Móvil (Emparejar PC):\n\n%s\nO escribe el código: %s  (vale %d minutos, una sola vez)\n\nEsperando al teléfono…\n", dibujo, o.Codigo, int(VigenciaOferta.Minutes()))
	if cada <= 0 {
		cada = cadaConsulta
	}
	return esperarAceptacion(ctx, r, hash, o.Cred, cada, plazo)
}

func esperarAceptacion(ctx context.Context, r RelayEmparejar, hash string, cred Credenciales, cada, plazo time.Duration) (Credenciales, error) {
	limite := time.Now().Add(plazo)
	t := time.NewTicker(cada)
	defer t.Stop()
	for time.Now().Before(limite) {
		select {
		case <-ctx.Done():
			return Credenciales{}, ctx.Err()
		case <-t.C:
		}
		id, err := r.Emparejada(ctx, hash)
		if err != nil {
			return Credenciales{}, err
		}
		if id == "" {
			continue
		}
		if !strings.EqualFold(id, cred.Maquina) {
			return Credenciales{}, fmt.Errorf("el relay emparejó otra máquina (%s, esperaba %s)", id, cred.Maquina)
		}
		cred.Emparejada = time.Now().UTC()
		return cred, nil
	}
	return Credenciales{}, ErrOfertaVencida
}
