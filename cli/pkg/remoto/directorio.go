package remoto

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0013 «El relay sale de un directorio firmado» (decisión de la persona, 2026-11-06): el relay deja de estar en el
// código y sale de una URL. Quien controle a qué relay se conecta un PC se queda con su firma y puede mandarle
// instrucciones, así que la respuesta va firmada con una clave cuya privada nunca está en el servidor que la sirve: entrar
// a esa cuenta no alcanza para redirigir a nadie. Serie solo sube: una respuesta vieja no lleva a un proyecto abandonado.

const (
	// URLDirectorio es de donde sale el relay de QDD Móvil.
	URLDirectorio = "https://qdd-movile.3ldago.workers.dev/"
	// ClavePublicaDirectorio es la ed25519 que firma el directorio, en base64.
	ClavePublicaDirectorio = "UPJa3bDOYZj49cdpGn18nNpsHm5/iwahV+ZxsQz8Am8="

	maxDirectorio  = 4096
	topeDirectorio = 10 * time.Second

	// CadaConsulta es cada cuánto se consulta como mucho sin conexión; EsperaConsulta, cuánto debe durar la caída antes.
	CadaConsulta   = 10 * time.Minute
	EsperaConsulta = time.Minute
)

// ErrRelayCambio: el directorio dio otro relay y hay que volver a conectar con él.
var ErrRelayCambio = errors.New("el directorio dio otro relay: qdd remoto vuelve a arrancar con él")

// Directorio es la URL que dice cuál es el relay y la clave que firma su respuesta.
type Directorio struct {
	URL     string
	Publica ed25519.PublicKey
}

// DirectorioDeFabrica es el que trae qdd.
func DirectorioDeFabrica() Directorio {
	publica, _ := base64.StdEncoding.DecodeString(ClavePublicaDirectorio)
	return Directorio{URL: URLDirectorio, Publica: publica}
}

// Consultar lee el directorio. No sigue redirecciones: la URL es esta y ninguna otra.
func (d Directorio) Consultar(ctx context.Context, serieMinima int) (ConfigRelay, error) {
	c, cancelar := context.WithTimeout(ctx, topeDirectorio)
	defer cancelar()
	pedido, err := http.NewRequestWithContext(c, http.MethodGet, d.URL, nil)
	if err != nil {
		return ConfigRelay{}, err
	}
	cliente := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r, err := cliente.Do(pedido)
	if err != nil {
		return ConfigRelay{}, errors.New("no se pudo leer el directorio del relay")
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return ConfigRelay{}, fmt.Errorf("el directorio del relay respondió %d", r.StatusCode)
	}
	datos, err := io.ReadAll(io.LimitReader(r.Body, maxDirectorio+1))
	if err != nil {
		return ConfigRelay{}, errors.New("no se pudo leer el directorio del relay")
	}
	if len(datos) > maxDirectorio {
		return ConfigRelay{}, errors.New("la respuesta del directorio es demasiado grande")
	}
	return LeerDirectorio(datos, d.Publica, serieMinima)
}

// LeerDirectorio acepta solo una respuesta exacta, firmada con esa clave y de esa serie o una mayor.
func LeerDirectorio(datos []byte, publica ed25519.PublicKey, serieMinima int) (ConfigRelay, error) {
	var r struct {
		URL   string `json:"url"`
		Clave string `json:"clave"`
		Serie int    `json:"serie"`
		Firma string `json:"firma"`
	}
	lector := json.NewDecoder(strings.NewReader(string(datos)))
	lector.DisallowUnknownFields()
	if err := lector.Decode(&r); err != nil {
		return ConfigRelay{}, errors.New("la respuesta del directorio no se entiende")
	}
	if _, err := lector.Token(); !errors.Is(err, io.EOF) {
		return ConfigRelay{}, errors.New("la respuesta del directorio trae datos de más")
	}
	firma, err := base64.StdEncoding.DecodeString(r.Firma)
	if err != nil || len(publica) != ed25519.PublicKeySize {
		return ConfigRelay{}, errors.New("la respuesta del directorio no trae una firma válida")
	}
	// El separador no puede venir en los campos: dos respuestas distintas nunca firman lo mismo.
	if strings.Contains(r.URL, "|") || strings.Contains(r.Clave, "|") {
		return ConfigRelay{}, errors.New("la respuesta del directorio trae un separador en sus campos")
	}
	mensaje := r.URL + "|" + r.Clave + "|" + strconv.Itoa(r.Serie)
	if !ed25519.Verify(publica, []byte(mensaje), firma) {
		return ConfigRelay{}, errors.New("la firma del directorio no es válida")
	}
	if r.Serie < 1 || r.Serie < serieMinima {
		return ConfigRelay{}, fmt.Errorf("la respuesta del directorio es vieja (serie %d; este PC ya vio la %d)", r.Serie, serieMinima)
	}
	if err := soloHost(r.URL); err != nil {
		return ConfigRelay{}, err
	}
	if err := ValidarClave(r.Clave); err != nil {
		return ConfigRelay{}, err
	}
	return ConfigRelay{URL: r.URL, Clave: r.Clave, Serie: r.Serie}, nil
}

func soloHost(u string) error {
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" || p.Host == "" || p.User != nil || p.Path != "" || p.RawQuery != "" || p.Fragment != "" {
		return errors.New("la URL del directorio tiene que ser https y sin ruta")
	}
	return nil
}

// ConsultaDirectorio es lo último que pasó al consultar, para qdd doctor.
type ConsultaDirectorio struct {
	En     time.Time `json:"en"`
	OK     bool      `json:"ok"`
	Serie  int       `json:"serie,omitempty"`
	Motivo string    `json:"motivo,omitempty"`
}

// LeerConsultaDirectorio devuelve la última, vacía si nunca se consultó.
func LeerConsultaDirectorio() ConsultaDirectorio {
	var c ConsultaDirectorio
	_ = leerPrivado("directorio.json", &c)
	return c
}

// ActualizarRelay consulta y guarda lo que cambió. Si la consulta falla, sigue con el recordado y dice por qué. Un relay
// propio no consulta nunca: no se pisa el de quien tiene el suyo.
func ActualizarRelay(ctx context.Context, consultar func(context.Context, int) (ConfigRelay, error), actual ConfigRelay) (ConfigRelay, bool, error) {
	if actual.Propio {
		return actual, false, nil
	}
	nuevo, err := consultar(ctx, actual.Serie)
	if err != nil {
		_ = guardarPrivado("directorio.json", ConsultaDirectorio{En: time.Now().UTC(), Motivo: err.Error()})
		return actual, false, err
	}
	if err := GuardarRelay(nuevo); err != nil {
		_ = guardarPrivado("directorio.json", ConsultaDirectorio{En: time.Now().UTC(), Motivo: err.Error()})
		return actual, false, err
	}
	_ = guardarPrivado("directorio.json", ConsultaDirectorio{En: time.Now().UTC(), OK: true, Serie: nuevo.Serie})
	return nuevo, nuevo.URL != actual.URL || nuevo.Clave != actual.Clave, nil
}

// anotarCaida guarda cuándo empezó: una caída que sigue no reinicia la cuenta.
func (b *Bucle) anotarCaida(now time.Time) {
	if b.caidoDesde.IsZero() {
		b.caidoDesde = now
	}
}

func (b *Bucle) anotarConexion() {
	b.caidoDesde = time.Time{}
}

// tocaConsultar: solo sin conexión, tras una caída que dura y sin pasar del tope. Un 28000 no pasa por aquí: el relay
// respondió, y eso nunca cambia de relay.
func (b *Bucle) tocaConsultar(now time.Time) bool {
	if b.caidoDesde.IsZero() || now.Sub(b.caidoDesde) < b.EsperaConsulta {
		return false
	}
	return b.consultadoEn.IsZero() || now.Sub(b.consultadoEn) >= b.CadaConsulta
}

// quizasConsultar consulta en segundo plano: el bucle no se detiene por la red.
func (b *Bucle) quizasConsultar(ctx context.Context) {
	now := b.ahora()
	if b.Consultar == nil || !b.tocaConsultar(now) {
		return
	}
	b.consultadoEn = now
	go func() {
		cambio, err := b.Consultar(ctx)
		select {
		case b.consultas <- resultadoConsulta{cambio: cambio, err: err}:
		case <-ctx.Done():
		}
	}()
}

type resultadoConsulta struct {
	cambio bool
	err    error
}
