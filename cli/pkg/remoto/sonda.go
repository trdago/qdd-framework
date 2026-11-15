package remoto

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

// Rule: Prohibido usar 'else'
// Autodiagnóstico del canal con QDD Móvil (pedido de la persona desde el teléfono, 2026-11-15: «un test para auto
// diagnóstico y reparación usando qdd doctor; se activa automático cuando qdd tiene móvil activado»). La sonda abre su
// propia conexión al canal del PC, como lo haría el teléfono, y mide cada tramo: relay guardado, conexión, permiso del
// canal y Realtime. Si hay un teléfono en el canal, manda un broadcast «sonda» con eco y espera que vuelva: así prueba
// que lo que el PC publica llega. Sin teléfono no publica nada (inv. 12) ni hace track de presence. El broadcast lleva
// solo la versión y una marca al azar: nada del proyecto ni de la máquina. El motivo es un texto fijo por tramo (viaja al
// teléfono y a doctor); el error crudo, que puede traer la URL, queda solo en el registro del servicio.

const (
	EventoSonda = "sonda"

	TramoRelay    = "relay"    // sin relay guardado o sin emparejar: no hay canal que probar
	TramoConexion = "conexion" // no se llega al relay (red, DNS, URL)
	TramoCanal    = "canal"    // el relay no deja entrar al canal (permiso, credenciales, relay mudado)
	TramoRealtime = "realtime" // entra pero Realtime no responde o cierra el canal
	TramoEco      = "eco"      // con un teléfono en el canal, lo que el PC publica no vuelve

	ReparaDirectorio = "directorio"
	ReparaReiniciar  = "reiniciar"

	// TopeSonda es lo más que dura una sonda; CadaSonda, cada cuánto la corre el servicio, y PrimeraSonda, cuánto espera
	// al arrancar para no competir con la primera conexión.
	TopeSonda    = 20 * time.Second
	CadaSonda    = time.Hour
	PrimeraSonda = time.Minute
	// EntreReinicios: el servicio se reinicia por una sonda como mucho una vez en este tiempo.
	EntreReinicios = 6 * time.Hour
	// esperaPresencia: cuánto espera la sonda la lista de quién está en el canal, que Realtime manda tras el join.
	esperaPresencia = 2 * time.Second
)

var (
	// ErrSondaReinicia: la sonda falló en un tramo que se arregla volviendo a conectar; el servicio vuelve a arrancar.
	ErrSondaReinicia = errors.New("la prueba del canal con el teléfono falló: qdd remoto vuelve a arrancar para reconectar")

	motivos = map[string]string{
		TramoRelay:    "este PC no tiene relay guardado o no está emparejado",
		TramoConexion: "no se llega al relay: revisa la red de este PC",
		TramoCanal:    "el relay no deja entrar al canal de este PC (permiso o relay cambiado)",
		TramoRealtime: "el relay acepta la conexión pero su Realtime no responde",
		TramoEco:      "lo que este PC publica no vuelve por el relay: el teléfono no lo recibiría",
	}
	// CodigosMotivo es la lista cerrada que viaja al teléfono en diagnostico.motivo (pedido de qdd-movile: la app los
	// traduce), con los de avisar (sonda_avisar.go). El texto en español de motivos queda para doctor y la CLI. Un
	// código no se renombra: se agrega otro.
	CodigosMotivo = map[string]string{
		TramoRelay:    "sin_relay",
		TramoConexion: "sin_red",
		TramoCanal:    "canal_rechazado",
		TramoRealtime: "realtime_sin_respuesta",
		TramoEco:      "eco_perdido",
	}
)

// ResultadoSonda es una prueba del canal. Mensajes son los broadcasts que gastó (el enviado y su eco), para la cuota.
type ResultadoSonda struct {
	Formato   int       `json:"formato"`
	OK        bool      `json:"ok"`
	Tramo     string    `json:"tramo,omitempty"`
	Motivo    string    `json:"motivo,omitempty"`
	Codigo    string    `json:"codigo,omitempty"` // de la lista cerrada que viaja al teléfono (CodigosMotivo, avisar)
	Eco       bool      `json:"eco"`
	Telefonos int       `json:"telefonos"`
	MS        int64     `json:"ms"`
	Mensajes  int       `json:"mensajes"`
	En        time.Time `json:"en"`
	Reparado  string    `json:"reparado,omitempty"`
	// detalle es el error crudo: solo para el registro del servicio, nunca al teléfono ni al reporte.
	detalle string
}

// Detalle es el error crudo de la sonda, para el registro del servicio (que pasa por el ocultador).
func (r ResultadoSonda) Detalle() string { return r.detalle }

// MotivoDeTramo es el texto fijo de un tramo roto.
func MotivoDeTramo(tramo string) string { return motivos[tramo] }

func fallo(tramo string, err error, inicio time.Time) ResultadoSonda {
	r := ResultadoSonda{Formato: 1, Tramo: tramo, Motivo: MotivoDeTramo(tramo), Codigo: CodigosMotivo[tramo], MS: time.Since(inicio).Milliseconds(), En: time.Now().UTC()}
	if err != nil {
		r.detalle = err.Error()
	}
	return r
}

// Sondear prueba el canal del PC con su propia conexión y la cierra al terminar. Nunca dura más que tope.
func Sondear(ctx context.Context, c ConfigRelay, cred Credenciales, tope time.Duration) ResultadoSonda {
	inicio := time.Now()
	if c.URL == "" || c.Clave == "" || cred.Topico == "" {
		return fallo(TramoRelay, nil, inicio)
	}
	ctx, cancelar := context.WithTimeout(ctx, tope)
	defer cancelar()
	canal := NuevoCanal(ConfigCanal{Relay: c.URL, Clave: c.Clave, Topico: cred.Topico, PresenceKey: "sonda-" + marcaSonda(), Eco: true})
	fin := make(chan error, 1)
	go func() { fin <- canal.sesion(ctx) }()
	presencia, err := esperarConexion(ctx, canal, fin)
	if err != nil {
		return fallo(tramoDe(err), err, inicio)
	}
	telefonos := len(Telefonos(canal.Presencias()))
	if !presencia {
		telefonos = esperarTelefonos(ctx, canal)
	}
	if telefonos == 0 {
		return conAvisar(ctx, c, ResultadoSonda{Formato: 1, OK: true, MS: time.Since(inicio).Milliseconds(), En: time.Now().UTC()})
	}
	r := probarEco(ctx, canal, fin, inicio)
	r.Telefonos = telefonos
	return conAvisar(ctx, c, r)
}

// esperarConexion espera el join. La lista de presence puede llegar antes del aviso de conectado (la lee otra
// goroutine): si llegó, lo dice para no esperarla de nuevo.
func esperarConexion(ctx context.Context, canal *Canal, fin <-chan error) (bool, error) {
	presencia := false
	for {
		select {
		case <-ctx.Done():
			return presencia, errSinRespuesta
		case err := <-fin:
			return presencia, err
		case e := <-canal.Eventos():
			presencia = presencia || e.Tipo == TipoPresencia
			if e.Tipo == TipoConectado {
				return presencia, nil
			}
		}
	}
}

var errSinRespuesta = errors.New("el relay no respondió a tiempo")

// tramoDe clasifica el error de la conexión por lo que devolvió sesion.
func tramoDe(err error) string {
	if errors.Is(err, ErrConexionRelay) {
		return TramoConexion
	}
	if errors.Is(err, ErrCanalRechazado) {
		return TramoCanal
	}
	return TramoRealtime
}

// esperarTelefonos lee quién está en el canal: Realtime manda presence_state justo después del join.
func esperarTelefonos(ctx context.Context, canal *Canal) int {
	limite := time.NewTimer(esperaPresencia)
	defer limite.Stop()
	for {
		select {
		case <-ctx.Done():
			return len(Telefonos(canal.Presencias()))
		case <-limite.C:
			return len(Telefonos(canal.Presencias()))
		case e := <-canal.Eventos():
			if e.Tipo == TipoPresencia {
				return len(Telefonos(e.Presencias))
			}
		}
	}
}

func probarEco(ctx context.Context, canal *Canal, fin <-chan error, inicio time.Time) ResultadoSonda {
	marca := marcaSonda()
	if err := canal.Broadcast(ctx, EventoSonda, map[string]any{"version": VersionProtocolo, "marca": marca}); err != nil {
		return fallo(TramoRealtime, err, inicio)
	}
	for {
		select {
		case <-ctx.Done():
			r := fallo(TramoEco, errSinRespuesta, inicio)
			r.Mensajes = 1
			return r
		case err := <-fin:
			r := fallo(TramoRealtime, err, inicio)
			r.Mensajes = 1
			return r
		case e := <-canal.Eventos():
			if e.Tipo == TipoBroadcast && e.Evento == EventoSonda && marcaDe(e.Payload) == marca {
				return ResultadoSonda{Formato: 1, OK: true, Eco: true, MS: time.Since(inicio).Milliseconds(), Mensajes: 2, En: time.Now().UTC()}
			}
		}
	}
}

func marcaDe(payload json.RawMessage) string {
	var p struct {
		Marca string `json:"marca"`
	}
	_ = json.Unmarshal(payload, &p)
	return p.Marca
}

func marcaSonda() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ReparacionDe es la reparación segura de un tramo: con el canal rechazado, comparar el relay con su directorio firmado;
// si Realtime no responde o no devuelve, volver a conectar. Sin relay o sin red no hay nada que reparar desde aquí.
func ReparacionDe(r ResultadoSonda) string {
	if r.OK {
		return ""
	}
	switch r.Tramo {
	case TramoCanal:
		return ReparaDirectorio
	case TramoRealtime, TramoEco:
		return ReparaReiniciar
	}
	return ""
}

// PuedeReiniciarPorSonda: como mucho un reinicio cada EntreReinicios, para no quedar en un bucle si el relay sigue mal.
func PuedeReiniciarPorSonda(ultimo, ahora time.Time) bool {
	return ultimo.IsZero() || ahora.Sub(ultimo) >= EntreReinicios
}

// SondaGuardada es <QDD_HOME>/remoto/sonda.json: la última sonda y el último reinicio que pidió.
type SondaGuardada struct {
	Formato    int            `json:"formato"`
	Ultima     ResultadoSonda `json:"ultima"`
	ReinicioEn time.Time      `json:"reinicio_en,omitempty"`
}

// GuardarSonda anota la sonda; reinicio distinto de cero es el reinicio que pidió, y si es cero se conserva el anterior.
func GuardarSonda(r ResultadoSonda, reinicio time.Time) error {
	g := SondaGuardada{Formato: 1, Ultima: r, ReinicioEn: reinicio.UTC()}
	if reinicio.IsZero() {
		anterior, _ := LeerSonda()
		g.ReinicioEn = anterior.ReinicioEn
	}
	return guardarPrivado("sonda.json", g)
}

// AnotarReinicioCanal anota un reinicio por el canal que no pidió la sonda (doctor --fix por REM-07 o REM-18): es el
// mismo tope, así el servicio no se reinicia otra vez por la misma caída. Conserva la última sonda.
func AnotarReinicioCanal(now time.Time) error {
	g, _ := LeerSonda()
	g.Formato, g.ReinicioEn = 1, now.UTC()
	return guardarPrivado("sonda.json", g)
}

// LeerSonda es lo que dejó GuardarSonda.
func LeerSonda() (SondaGuardada, error) {
	var g SondaGuardada
	err := leerPrivado("sonda.json", &g)
	return g, err
}

// DiagnosticoCanal es lo que el teléfono ve de la última sonda que falló o reparó algo: tramo, motivo como código de CodigosMotivo y en RFC3339 UTC; sin datos.
type DiagnosticoCanal struct {
	OK       bool      `json:"ok"`
	Tramo    string    `json:"tramo,omitempty"`
	Motivo   string    `json:"motivo,omitempty"`
	Reparado string    `json:"reparado,omitempty"`
	En       time.Time `json:"en"`
}

// vigenciaDiagnostico: una falla o reparación se sigue mostrando un día.
const vigenciaDiagnostico = 24 * time.Hour

// diagnosticoDe da el diagnóstico para el estado, o nil si la sonda salió bien y no reparó nada, o si es de hace más
// de un día.
func diagnosticoDe(r *ResultadoSonda, ahora time.Time) *DiagnosticoCanal {
	if r == nil || (r.OK && r.Reparado == "") || ahora.Sub(r.En) > vigenciaDiagnostico {
		return nil
	}
	codigo := r.Codigo
	if codigo == "" {
		codigo = CodigosMotivo[r.Tramo] // una sonda.json de antes del código
	}
	return &DiagnosticoCanal{OK: r.OK, Tramo: r.Tramo, Motivo: codigo, Reparado: r.Reparado, En: r.En.UTC()}
}
