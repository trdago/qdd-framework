package remoto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Ejecutar desde el teléfono» (solo Pro, decisión de la persona, 2026-11-03): el teléfono llama a
// qdd_remoto_ejecutar; el relay exige Pro, sella el dispositivo y reenvía el evento ejecutar con firma =
// hex(hmac_sha256(firma de la máquina, solicitud|proyecto|tipo|texto|dispositivo|en)), como responder. El PC lo trata
// como una acción: descarta lo que no trae firma válida o es viejo (inv. 18), aplica cada solicitud una sola vez
// (solicitudes.json), exige Pro con el plan de su último latido y vuelve a validar el texto (nunca confía en el relay).
// Un comando qdd se parte sin shell, se compara con la lista permitida y corre con el qdd de este PC en la carpeta del
// proyecto; una instrucción queda en la cola del teléfono (pkg/consola) para la sesión de Claude Code del proyecto.
// Responde accion_estado {solicitud, accion_id: ejecutar, estado, resultado}: recibida → ejecutando → hecha | fallo,
// o rechazada (o vencida, una instrucción que nadie tomó) con el motivo.

const (
	// AccionEjecutar es el accion_id con que el PC responde una ejecución (no es una acción del panel).
	AccionEjecutar = "ejecutar"
	// MaxResultadoEjecutar es lo que vuelve al teléfono de la salida de un comando: su final, ya sin secretos.
	MaxResultadoEjecutar = 4000
	// MaxTextoQDD y MaxTextoIA son los topes del texto, como en el relay.
	MaxTextoQDD = 1000
	MaxTextoIA  = 4000
	// maxSalidaEjecutar es lo que se guarda de la salida mientras corre (el final): se le quitan los secretos entera.
	maxSalidaEjecutar = 64 << 10
	// planPro es el id del plan que permite ejecutar desde el teléfono.
	planPro = "pro"
	// metacaracteres son los de una shell: el PC nunca usa una, pero un texto que los trae se rechaza entero.
	metacaracteres = ";|&$`><()"
)

// Sugeridos son los comandos qdd que el PC ofrece en el teléfono (sin el «qdd»), en el orden en que se ofrecen.
// Solo leen o revisan: ninguno repara, aprueba, publica ni cambia el estado. Lo único que cambia algo está en permitidos
// y no aquí: encender y apagar el modo goal, que la app ofrece con biometría (RFC 0015).
var Sugeridos = []string{"audit", "doctor", "estado", "estado check", "modules list", "blueprint list", "blueprint quality"}

// permitido es un comando de la lista: sus palabras, las banderas que acepta (exactas, de solo lectura) y si admite un
// id de blueprint.
type permitido struct {
	palabras []string
	banderas []string
	conID    bool
	agrega   []string
}

// permitidos va del más largo al más corto: «estado check» antes que «estado».
var permitidos = []permitido{
	{palabras: []string{"estado", "check"}, banderas: []string{"--json"}},
	{palabras: []string{"estado"}, banderas: []string{"--json", "--alcance=abiertos", "--alcance=todos", "--formato=markdown", "--formato=mensajeria"}},
	{palabras: []string{"modules", "list"}},
	{palabras: []string{"blueprint", "list"}, banderas: []string{"--json"}},
	{palabras: []string{"blueprint", "quality"}, banderas: []string{"--json"}, conID: true},
	{palabras: []string{"audit"}},
	{palabras: []string{"doctor"}},
	{palabras: []string{"goal", "activar"}, agrega: []string{"--desde", "telefono"}},
	{palabras: []string{"goal", "desactivar"}},
}

var (
	idBlueprint  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,79}$`)
	qddDelantero = regexp.MustCompile(`(?i)^\s*qdd(\s|$)`)
)

// EjecucionFirmada es el evento ejecutar que reenvía el relay. nombre es solo para mostrar: no va firmado.
type EjecucionFirmada struct {
	Version     int    `json:"version"`
	Solicitud   string `json:"solicitud"`
	Proyecto    string `json:"proyecto"`
	Tipo        string `json:"tipo"`
	Texto       string `json:"texto"`
	Dispositivo string `json:"dispositivo"`
	Nombre      string `json:"nombre"`
	En          string `json:"en"`
	Firma       string `json:"firma"`
}

// FirmaEjecucion es la firma que pone el relay a una ejecución (la misma clave que FirmaAccion).
func FirmaEjecucion(firmaMaquina, solicitud, proyecto, tipo, texto, dispositivo, en string) string {
	return firmar(firmaMaquina, solicitud, proyecto, tipo, texto, dispositivo, en)
}

// Verificar exige la firma de la máquina y que la ejecución sea de los últimos VigenciaAccion.
func (e EjecucionFirmada) Verificar(firmaMaquina string, now time.Time) error {
	if e.Solicitud == "" || e.Proyecto == "" || e.Tipo == "" || e.Texto == "" || e.Dispositivo == "" || e.En == "" {
		return errors.New("ejecución incompleta")
	}
	esperada := FirmaEjecucion(firmaMaquina, e.Solicitud, e.Proyecto, e.Tipo, e.Texto, e.Dispositivo, e.En)
	return verificarFirmada(esperada, e.Firma, e.En, now)
}

// ValidarTexto vuelve a hacer en el PC lo que exige el relay: un comando qdd de la lista permitida (ArgsPermitidos);
// una instrucción de 1 a MaxTextoIA caracteres, sin más controles que saltos de línea y tabulaciones.
func ValidarTexto(tipo, texto string) error {
	if tipo == consola.EjecutarQDD {
		_, err := ArgsPermitidos(texto)
		return err
	}
	if tipo != consola.EjecutarIA {
		return fmt.Errorf("tipo «%s» desconocido: solo qdd o ia", tipo)
	}
	if !utf8.ValidString(texto) || strings.TrimSpace(texto) == "" {
		return errors.New("la instrucción está vacía o no es texto")
	}
	if utf8.RuneCountInString(texto) > MaxTextoIA {
		return fmt.Errorf("la instrucción tiene más de %d caracteres", MaxTextoIA)
	}
	for _, r := range texto {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return fmt.Errorf("la instrucción trae caracteres de control (%U)", r)
		}
	}
	return nil
}

// ArgsPermitidos parte un comando qdd sin shell (espacios y comillas simples o dobles) y lo compara con la lista
// permitida: devuelve los argumentos para qdd, o por qué no se acepta.
func ArgsPermitidos(texto string) ([]string, error) {
	args, err := partir(texto)
	if err != nil {
		return nil, err
	}
	for _, p := range permitidos {
		if !empiezaCon(args, p.palabras) {
			continue
		}
		if err := p.aceptar(args[len(p.palabras):]); err != nil {
			return args, err
		}
		return append(args, p.agrega...), nil
	}
	return nil, fmt.Errorf("«%s» no está permitido desde el teléfono: solo %s", strings.Join(args[:min(2, len(args))], " "), strings.Join(Sugeridos, ", "))
}

// partir separa las palabras de un texto de una sola línea, sin metacaracteres de shell ni la palabra qdd delante.
func partir(texto string) ([]string, error) {
	if strings.TrimSpace(texto) == "" {
		return nil, errors.New("el comando está vacío")
	}
	if !utf8.ValidString(texto) || utf8.RuneCountInString(texto) > MaxTextoQDD {
		return nil, fmt.Errorf("el comando tiene que ser texto de hasta %d caracteres", MaxTextoQDD)
	}
	if strings.ContainsAny(texto, "\r\n") {
		return nil, errors.New("el comando va en una sola línea")
	}
	if i := strings.IndexAny(texto, metacaracteres); i >= 0 {
		return nil, fmt.Errorf("«%c» no se permite: qdd corre sin shell", texto[i])
	}
	for _, r := range texto {
		if unicode.IsControl(r) && r != '\t' {
			return nil, fmt.Errorf("el comando trae caracteres de control (%U)", r)
		}
	}
	if qddDelantero.MatchString(texto) {
		return nil, errors.New("va sin la palabra qdd del principio")
	}
	return palabras(texto)
}

// palabras corta por espacios; lo que va entre comillas simples o dobles es una sola palabra.
func palabras(texto string) ([]string, error) {
	var out []string
	var actual strings.Builder
	comilla, hay := rune(0), false
	for _, r := range texto {
		if comilla != 0 && r == comilla {
			comilla = 0
			continue
		}
		if comilla != 0 {
			actual.WriteRune(r)
			continue
		}
		if r == '"' || r == '\'' {
			comilla, hay = r, true
			continue
		}
		if unicode.IsSpace(r) && (actual.Len() > 0 || hay) {
			out, hay = append(out, actual.String()), false
			actual.Reset()
			continue
		}
		if unicode.IsSpace(r) {
			continue
		}
		actual.WriteRune(r)
	}
	if comilla != 0 {
		return nil, errors.New("hay una comilla sin cerrar")
	}
	if actual.Len() > 0 || hay {
		out = append(out, actual.String())
	}
	return out, nil
}

func empiezaCon(args, palabras []string) bool {
	if len(args) < len(palabras) {
		return false
	}
	for i, p := range palabras {
		if args[i] != p {
			return false
		}
	}
	return true
}

// aceptar revisa lo que viene después de las palabras del comando: solo sus banderas y, si admite, un id.
func (p permitido) aceptar(resto []string) error {
	comando := "qdd " + strings.Join(p.palabras, " ")
	ids := 0
	for _, a := range resto {
		if p.palabras[0] == "doctor" && (strings.HasPrefix(a, "--fix") || (strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.Contains(a, "f"))) {
			return fmt.Errorf("«%s» no se permite desde el teléfono: %s --fix (-f) repara archivos", a, comando)
		}
		if contiene(p.banderas, a) {
			continue
		}
		if p.conID && !strings.HasPrefix(a, "-") && idBlueprint.MatchString(a) {
			ids++
			continue
		}
		return fmt.Errorf("«%s» no está permitido en %s desde el teléfono", a, comando)
	}
	if ids > 1 {
		return fmt.Errorf("%s acepta un solo id de blueprint", comando)
	}
	return nil
}

func contiene(lista []string, s string) bool {
	for _, x := range lista {
		if x == s {
			return true
		}
	}
	return false
}

// Pro dice si el plan permite ejecutar desde el teléfono.
func (p Plan) Pro() bool {
	return p.Plan == planPro
}

// planGuardado es <QDD_HOME>/remoto/plan.json: el plan del último latido, para qdd consola panel y qdd remoto estado,
// que corren en otro proceso.
type planGuardado struct {
	Formato int       `json:"formato"`
	Plan    Plan      `json:"plan"`
	En      time.Time `json:"en"`
}

// GuardarPlan deja el plan del último latido (0600).
func GuardarPlan(p Plan, now time.Time) error {
	return guardarPrivado("plan.json", planGuardado{Formato: 1, Plan: p, En: now.UTC()})
}

// LeerPlan es el plan del último latido de qdd remoto en este PC.
func LeerPlan() (Plan, error) {
	var g planGuardado
	if err := leerPrivado("plan.json", &g); err != nil {
		return Plan{}, err
	}
	return g.Plan, nil
}

// OpcionesEjecutar es el ejecutar del panel de un proyecto: nada sin Pro; con Pro, comandos qdd de la lista y, si hay
// una sesión de Claude Code abierta en el proyecto, instrucciones para la IA.
func OpcionesEjecutar(root string, plan Plan, now time.Time) *consola.OpcionesEjecutar {
	if !plan.Pro() {
		return nil
	}
	return &consola.OpcionesEjecutar{QDD: true, IA: consola.HaySesion(root, now), Sugeridos: append([]string{}, Sugeridos...)}
}

// ConEjecutar pone el ejecutar de cada panel del estado según el plan (sin Pro, ninguno).
func ConEjecutar(e *Estado, plan Plan, now time.Time) {
	for _, p := range e.Proyectos {
		if p.Panel == nil {
			continue
		}
		p.Panel.Ejecutar = OpcionesEjecutar(p.Ruta, plan, now)
	}
}

// alEjecutar recibe una ejecución del relay: la verifica, la recuerda antes de aplicarla y la aplica.
func (b *Bucle) alEjecutar(ctx context.Context, payload json.RawMessage) {
	var e EjecucionFirmada
	if err := json.Unmarshal(payload, &e); err != nil {
		b.logf("ejecución descartada: ilegible")
		return
	}
	if err := e.Verificar(b.Cred.Firma, b.ahora()); err != nil {
		b.logf("ejecución %s descartada: %v", e.Solicitud, err)
		b.anotar(AccionFirmada{Solicitud: e.Solicitud, Proyecto: e.Proyecto, AccionID: AccionEjecutar, Dispositivo: e.Dispositivo, DispositivoNombre: e.Nombre}, EstadoDescartada, err.Error())
		return
	}
	if s, ok := b.extras.solicitudes[e.Solicitud]; ok {
		b.responder(ctx, s)
		return
	}
	s := Solicitud{Solicitud: e.Solicitud, Proyecto: e.Proyecto, AccionID: AccionEjecutar, Ejecutar: e.Tipo, Dispositivo: e.Dispositivo, Nombre: e.Nombre, Estado: EstadoRecibida, Recibida: b.ahora().UTC()}
	b.recordar(s)
	b.responder(ctx, s)
	b.ejecutarDelTelefono(ctx, s, e)
}

// ejecutarDelTelefono decide qué hacer con una ejecución verificada y nueva.
func (b *Bucle) ejecutarDelTelefono(ctx context.Context, s Solicitud, e EjecucionFirmada) {
	if LeerRevocados()[strings.ToLower(s.Dispositivo)] {
		b.cerrar(ctx, s, EstadoRechazada, "este teléfono está revocado en el PC")
		return
	}
	p, ok := b.proyectoPublicado(s.Proyecto)
	if !ok {
		b.cerrar(ctx, s, EstadoRechazada, "el PC no publica ese proyecto")
		return
	}
	s.Ruta = p.Ruta
	if _, err := os.Stat(p.Ruta); err != nil {
		b.cerrar(ctx, s, EstadoRechazada, ErrCarpetaNoExiste)
		return
	}
	if !b.plan.Pro() {
		b.cerrar(ctx, s, EstadoRechazada, "la ejecución desde el teléfono requiere Pro (plan de este PC: "+b.plan.Plan+")")
		return
	}
	if e.Version != VersionProtocolo {
		b.cerrar(ctx, s, EstadoRechazada, fmt.Sprintf("versión %d del protocolo desconocida", e.Version))
		return
	}
	if err := ValidarTexto(e.Tipo, e.Texto); err != nil {
		b.cerrar(ctx, s, EstadoRechazada, err.Error())
		return
	}
	if e.Tipo == consola.EjecutarQDD && b.correEnElProyecto(s.Ruta) {
		b.cerrar(ctx, s, EstadoRechazada, "ya corre otro comando mandado desde el teléfono en este proyecto: espera a que termine")
		return
	}
	x := consola.Ejecucion{Solicitud: s.Solicitud, Tipo: e.Tipo, Texto: e.Texto, Dispositivo: s.Dispositivo, Nombre: s.Nombre}
	if _, err := consola.RegistrarEjecucion(s.Ruta, x, b.ahora()); err != nil {
		b.cerrar(ctx, s, EstadoFallo, err.Error())
		return
	}
	b.marcar()
	if e.Tipo == consola.EjecutarIA {
		b.encolarInstruccion(ctx, s)
		return
	}
	b.correrComando(ctx, s, e.Texto)
}

// correEnElProyecto dice si un comando qdd del teléfono corre ahora en esa carpeta: uno por proyecto a la vez (dos
// audit juntos se pisarían lo que escriben en .qdd).
func (b *Bucle) correEnElProyecto(ruta string) bool {
	for id := range b.extras.corriendo {
		if b.extras.solicitudes[id].Ruta == ruta {
			return true
		}
	}
	return false
}

// encolarInstruccion deja la instrucción para la sesión de Claude Code: si hay una abierta va en camino; si no, espera.
func (b *Bucle) encolarInstruccion(ctx context.Context, s Solicitud) {
	s.Resultado = "en cola: no hay una sesión de Claude Code abierta en el proyecto; se entrega cuando se abra una (espera " + consola.VidaInstruccion.String() + ")"
	if consola.HaySesion(s.Ruta, b.ahora()) {
		s.Estado, s.Resultado = EstadoEjecutando, "entregándola a la sesión de Claude Code abierta en el proyecto"
	}
	b.recordar(s)
	b.responder(ctx, s)
}

// correrComando corre qdd con los argumentos permitidos en segundo plano; el resultado vuelve por el bucle.
func (b *Bucle) correrComando(ctx context.Context, s Solicitud, texto string) {
	args, _ := ArgsPermitidos(texto)
	s.Estado = EstadoEjecutando
	b.recordar(s)
	b.responder(ctx, s)
	b.extras.corriendo[s.Solicitud] = true
	bin, ruta := b.Bin, s.Ruta
	go func() {
		codigo, salida := CorrerQDD(ctx, bin, ruta, args, consola.TopeComandoTelefono)
		select {
		case b.extras.resultados <- resultadoAccion{solicitud: s.Solicitud, codigo: codigo, salida: salida}:
		case <-ctx.Done():
		}
	}()
}

// terminarComando cierra un comando del teléfono: la salida sin secretos va a la cola del proyecto y su final al
// teléfono.
func (b *Bucle) terminarComando(ctx context.Context, s Solicitud, r resultadoAccion) {
	delete(b.extras.corriendo, s.Solicitud)
	salida := recortarFinal(b.ocultador(s.Ruta).Ocultar(strings.TrimSpace(r.salida)), MaxResultadoEjecutar)
	if salida == "" {
		salida = fmt.Sprintf("terminó con código %d, sin salida", r.codigo)
	}
	if err := consola.TerminarEjecucion(s.Ruta, s.Solicitud, r.codigo, salida, b.ahora()); err != nil {
		b.logf("cola del teléfono: %v", err)
	}
	estado := EstadoHecha
	if r.codigo != 0 {
		estado = EstadoFallo
	}
	s.Estado, s.Resultado = estado, salida
	b.recordar(s)
	b.responder(ctx, s)
	b.marcar()
}

// seguirEjecuciones cierra lo que el teléfono mandó a ejecutar y terminó fuera del bucle: una instrucción que una sesión
// tomó (hecha), venció, se retiró o no se pudo entregar; un comando que corría cuando qdd remoto se cerró (fallo).
func (b *Bucle) seguirEjecuciones(ctx context.Context) {
	for _, s := range b.extras.solicitudes {
		if s.Ejecutar == "" || s.terminada() || s.Ruta == "" {
			continue
		}
		if s.Ejecutar == consola.EjecutarQDD && !b.extras.corriendo[s.Solicitud] {
			_ = consola.TerminarEjecucion(s.Ruta, s.Solicitud, 1, "se cortó: qdd remoto se cerró mientras corría", b.ahora())
			b.cerrar(ctx, s, EstadoFallo, "se cortó: qdd remoto se cerró mientras corría")
			continue
		}
		if s.Ejecutar != consola.EjecutarIA {
			continue
		}
		b.seguirInstruccion(ctx, s)
	}
}

func (b *Bucle) seguirInstruccion(ctx context.Context, s Solicitud) {
	x, ok := consola.EjecucionDe(s.Ruta, s.Solicitud, b.ahora())
	if !ok {
		b.cerrar(ctx, s, EstadoVencida, "la instrucción ya no está en este PC")
		return
	}
	if x.Estado == consola.EstadoHecho {
		b.cerrar(ctx, s, EstadoHecha, "entregada a la sesión de Claude Code del proyecto")
		return
	}
	if x.Estado == consola.EstadoVencida {
		b.cerrar(ctx, s, EstadoVencida, x.Motivo)
		return
	}
	if x.Estado == consola.EstadoRetirada {
		b.cerrar(ctx, s, EstadoRechazada, "retirada desde el teléfono antes de entregarla")
		return
	}
	if x.Estado == consola.EstadoFallo {
		b.cerrar(ctx, s, EstadoFallo, "no se pudo entregar a la sesión: "+x.Motivo)
	}
}

// retirarInstruccion es el «Rechazar» del teléfono sobre una instrucción que todavía espera una sesión.
func (b *Bucle) retirarInstruccion(ctx context.Context, s Solicitud, solicitud string) {
	lugar := s.Nombre
	if lugar == "" {
		lugar = "un teléfono"
	}
	if _, err := consola.RetirarInstruccion(s.Ruta, solicitud, "retirada desde "+lugar, b.ahora()); err != nil {
		b.cerrar(ctx, s, EstadoRechazada, "esa instrucción ya no espera: "+err.Error())
		return
	}
	b.cerrar(ctx, s, EstadoHecha, "instrucción retirada")
	b.marcar()
}

// CorrerQDD corre bin con args en dir, sin shell, sin entrada, con el entorno sin las variables de git que eligen
// repositorio (FND-117) y cortado a los tope. Devuelve el código y el final de la salida (stdout y stderr juntos).
func CorrerQDD(ctx context.Context, bin, dir string, args []string, tope time.Duration) (int, string) {
	c, cancelar := context.WithTimeout(ctx, tope)
	defer cancelar()
	cmd := exec.CommandContext(c, bin, args...)
	// Lo mandó la persona desde el teléfono, no una IA, aunque qdd remoto se haya abierto desde una sesión de Claude Code.
	cmd.Dir, cmd.Env, cmd.WaitDelay = dir, append(gitenv.Limpio(os.Environ()), "CLAUDECODE="), 5*time.Second
	salida := &finalDeSalida{max: maxSalidaEjecutar}
	cmd.Stdout, cmd.Stderr = salida, salida
	err := cmd.Run()
	if errors.Is(c.Err(), context.DeadlineExceeded) {
		return 124, salida.String() + fmt.Sprintf("\n[!] se cortó a los %s", tope)
	}
	var salio *exec.ExitError
	if errors.As(err, &salio) {
		return salio.ExitCode(), salida.String()
	}
	if err != nil {
		return 1, salida.String() + "\n[!] no se pudo ejecutar qdd: " + err.Error()
	}
	return 0, salida.String()
}

// finalDeSalida guarda los últimos max bytes que se le escriben.
type finalDeSalida struct {
	max int
	b   []byte
}

func (f *finalDeSalida) Write(p []byte) (int, error) {
	f.b = append(f.b, p...)
	if len(f.b) > f.max {
		f.b = append([]byte{}, f.b[len(f.b)-f.max:]...)
	}
	return len(p), nil
}

func (f *finalDeSalida) String() string {
	return strings.ToValidUTF8(string(f.b), "")
}

// recortarFinal deja los últimos n caracteres de s (con «…» delante si cortó): el final dice cómo terminó.
func recortarFinal(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return "…" + string(r[len(r)-n:])
}

// PlanLocal es el plan para lo que corre en otro proceso que qdd remoto (qdd consola panel, qdd remoto estado): el
// del último latido si QDD Móvil está activado en este PC; si no, Gratis.
func PlanLocal() Plan {
	if _, err := LeerCredenciales(); err != nil {
		return PlanGratis()
	}
	p, err := LeerPlan()
	if err != nil {
		return PlanGratis()
	}
	return p
}
