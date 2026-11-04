package consola

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/gitenv"
	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'
// Modo goal (decisión de la persona, 2026-11-04; RFC 0015): en un proyecto con el modo activo, el trabajo sigue hasta
// terminar sin esperarla. qdd decide todo y no llama a ningún modelo: elige la opción recomendada de cada pregunta, corre
// los pedidos de la IA y aprueba las fichas; con 2 min de silencio le recuerda a la IA lo que queda; tras 3 recordatorios
// sin avance real se detiene y avisa. Lo activa solo la persona (panel o teléfono). Las preguntas de diseño de un
// blueprint, las que no traen opción recomendada y los permisos de Claude Code siempre la esperan. Todo queda en
// <QDD_HOME>/claude/goal/<proyecto>.json, y cada elección, en Elegidas.

const (
	GoalTrabajando = "trabajando"
	GoalPausado    = "pausado"
	GoalTerminado  = "terminado"

	MotivoSinConsola    = "sin_consola"
	MotivoSinAvance     = "sin_avance"
	MotivoSinLimite     = "sin_limite"
	MotivoEsperaPersona = "espera_persona"

	// SilencioGoal es cuánto silencio de la consola espera antes de recordarle a la IA lo que queda.
	SilencioGoal = 2 * time.Minute
	// MaxRecordatoriosGoal son los recordatorios seguidos sin avance real antes de detenerse.
	MaxRecordatoriosGoal = 3
	// MaxElegidasGoal son las elecciones que se guardan: las últimas.
	MaxElegidasGoal = 20
	// LugarGoal es dónde se respondió una pregunta que eligió el modo.
	LugarGoal = "modo goal"

	AccionGoalActivar    = "goal-activar"
	AccionGoalDesactivar = "goal-desactivar"

	vigenciaLimites = 10 * time.Minute
	topeHuellaGoal  = 5 * time.Second
	maxTareasGoal   = 5
)

// Elegida es una decisión que tomó el modo: qué era, qué eligió y cuándo.
type Elegida struct {
	En       time.Time `json:"en"`
	Tipo     string    `json:"tipo"` // pregunta, pedido o ficha
	Etiqueta string    `json:"etiqueta"`
	Eleccion string    `json:"eleccion"`
}

// Goal es lo que el panel dice del modo. Con Activo false, este qdd lo ofrece y está apagado.
type Goal struct {
	Activo        bool       `json:"activo"`
	Estado        string     `json:"estado,omitempty"`
	Motivo        string     `json:"motivo,omitempty"` // solo con pausado
	Desde         *time.Time `json:"desde,omitempty"`
	Recordatorios int        `json:"recordatorios"`
	Elegidas      []Elegida  `json:"elegidas"`
}

type goalGuardado struct {
	Goal
	ActivadoDesde string     `json:"activado_desde,omitempty"`
	Huella        string     `json:"huella,omitempty"`
	Recordado     *time.Time `json:"recordado,omitempty"`
	OcupadaPor    string     `json:"ocupada_por,omitempty"`
	OcupadaEn     *time.Time `json:"ocupada_en,omitempty"`
}

// EntradaGoal es lo que el mod sabe de la sesión: cuánto lleva en silencio y si está haciendo algo.
type EntradaGoal struct {
	Sesion   string
	Silencio time.Duration
	Ocupada  bool
}

// ResultadoGoal es lo que el mod hace después de un ciclo: el mensaje para la IA y el aviso para la persona.
type ResultadoGoal struct {
	Mensaje  string    `json:"mensaje,omitempty"`
	Aviso    string    `json:"aviso,omitempty"`
	Elegidas []Elegida `json:"elegidas,omitempty"`
}

var (
	sufijoRecomendada = regexp.MustCompile(`(?i)\((recommended|recomendad[oa])\)\s*$`)
	// preguntasDeDiseno se puede reemplazar en las pruebas: el catálogo real necesita un blueprint instalado.
	preguntasDeDiseno = disenoReal
)

func disenoReal(root string) map[string]bool {
	out := map[string]bool{}
	for _, p := range blueprints.PreguntasDeDiseno(root) {
		out[normalizarPregunta(p)] = true
	}
	return out
}

func normalizarPregunta(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func esRecomendada(etiqueta string) bool {
	return sufijoRecomendada.MatchString(etiqueta)
}

func rutaGoal(root string) (string, error) {
	return RutaLocal(root, "goal")
}

func leerGoal(root string) (goalGuardado, error) {
	g := goalGuardado{Goal: Goal{Elegidas: []Elegida{}}}
	ruta, err := rutaGoal(root)
	if err != nil {
		return g, err
	}
	b, err := os.ReadFile(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return g, nil
	}
	if err != nil {
		return g, err
	}
	if err := json.Unmarshal(b, &g); err != nil {
		return goalGuardado{Goal: Goal{Elegidas: []Elegida{}}}, fmt.Errorf("%s ilegible: %v", ruta, err)
	}
	if g.Elegidas == nil {
		g.Elegidas = []Elegida{}
	}
	return g, nil
}

func conGoalGuardado(root string, cambiar func(*goalGuardado) error) error {
	ruta, err := rutaGoal(root)
	if err != nil {
		return err
	}
	return conArchivo(ruta, "modo goal", func() error {
		g, err := leerGoal(root)
		if err != nil {
			return err
		}
		antes, _ := json.Marshal(g)
		if err := cambiar(&g); err != nil {
			return err
		}
		ahora, _ := json.Marshal(g)
		if string(antes) == string(ahora) {
			return nil
		}
		return guardarJSON(ruta, g)
	})
}

// VistaGoal es lo que publican el panel y qdd remoto. Sin una sesión abierta se dice pausado: nadie puede avanzar, y
// vuelve solo cuando la sesión vuelve (no abre sesiones por su cuenta).
func VistaGoal(root string, now time.Time) Goal {
	g, err := leerGoal(root)
	if err != nil {
		return Goal{Elegidas: []Elegida{}}
	}
	v := g.Goal
	if v.Activo && v.Estado == GoalTrabajando && !HaySesion(root, now) {
		v.Estado, v.Motivo = GoalPausado, MotivoSinConsola
	}
	return v
}

// RevisarGoal es lo que mira doctor: si el modo está activo y si su archivo se puede leer.
func RevisarGoal(root string) (bool, error) {
	g, err := leerGoal(root)
	return g.Activo, err
}

// ApartarGoal aparta un archivo ilegible como .ilegible: el modo queda apagado y la persona lo activa de nuevo.
func ApartarGoal(root string) error {
	ruta, err := rutaGoal(root)
	if err != nil {
		return err
	}
	return os.Rename(ruta, ruta+".ilegible")
}

// ActivarGoal enciende el modo desde cero. desde dice dónde lo activó la persona. Sin tareas abiertas no hay qué terminar.
func ActivarGoal(root, desde string, now time.Time) error {
	e, err := estado.Load(root)
	if err != nil {
		return fmt.Errorf("el modo goal necesita el tablero del proyecto: %v", err)
	}
	if len(e.Abiertos()) == 0 {
		return errors.New("el tablero no tiene tareas abiertas: no hay nada que terminar")
	}
	return conGoalGuardado(root, func(g *goalGuardado) error {
		g.Activo, g.Estado, g.Motivo, g.Desde, g.Recordatorios = true, GoalTrabajando, "", &now, 0
		g.ActivadoDesde, g.Huella, g.Recordado = desde, "", nil
		return nil
	})
}

// DesactivarGoal lo apaga. Las elecciones quedan a la vista.
func DesactivarGoal(root string, now time.Time) error {
	return conGoalGuardado(root, func(g *goalGuardado) error {
		g.Activo, g.Estado, g.Motivo, g.Desde, g.Recordatorios, g.Huella, g.Recordado = false, "", "", nil, 0, "", nil
		return nil
	})
}

// CicloGoal es una vuelta del vigilante, con el archivo bloqueado: varias terminales no duplican nada.
func CicloGoal(root, bin string, in EntradaGoal, now time.Time) (ResultadoGoal, error) {
	var r ResultadoGoal
	g, err := leerGoal(root)
	if err != nil || !g.Activo {
		return r, err
	}
	err = conGoalGuardado(root, func(g *goalGuardado) error {
		r = vueltaGoal(root, bin, g, in, now)
		return nil
	})
	return r, err
}

func vueltaGoal(root, bin string, g *goalGuardado, in EntradaGoal, now time.Time) ResultadoGoal {
	var r ResultadoGoal
	if !g.Activo || g.Estado == GoalTerminado || (g.Estado == GoalPausado && g.Motivo == MotivoSinAvance) {
		return r
	}
	if in.Ocupada {
		g.OcupadaPor, g.OcupadaEn = in.Sesion, &now
	}
	if limiteAgotado(now) {
		r.Aviso = pausar(g, MotivoSinLimite, "🎯 Modo goal en pausa: se acabó el límite de uso de Claude. Sigue solo cuando se reinicie.")
		return r
	}
	elegidas, esperan := elegir(root, bin, in.Sesion, now)
	r.Elegidas = elegidas
	anotarElegidas(g, elegidas)
	if esperan > 0 {
		r.Aviso = pausar(g, MotivoEsperaPersona, "🎯 Modo goal en pausa: hay una pregunta o un permiso que solo tú puedes responder.")
		return r
	}
	g.Estado, g.Motivo = GoalTrabajando, ""
	if in.Ocupada || in.Silencio < SilencioGoal || len(elegidas) > 0 || otraOcupada(g, in.Sesion, now) || hayPedidoEnMarcha(root, now) {
		return r
	}
	e, err := estado.Load(root)
	if err != nil {
		return r
	}
	abiertas := e.Abiertos()
	if len(abiertas) == 0 {
		g.Estado = GoalTerminado
		r.Aviso = "🎯 Modo goal terminado: no quedan tareas abiertas."
		return r
	}
	if g.Recordado != nil && now.Sub(*g.Recordado) < SilencioGoal {
		return r
	}
	huella := huellaGoal(root)
	if huella != g.Huella {
		g.Huella, g.Recordatorios = huella, 0
	}
	if g.Recordatorios >= MaxRecordatoriosGoal {
		r.Aviso = pausar(g, MotivoSinAvance, fmt.Sprintf("🎯 Modo goal detenido: %d recordatorios sin avance. Actívalo de nuevo cuando quieras seguir.", MaxRecordatoriosGoal))
		return r
	}
	g.Recordatorios++
	g.Recordado = &now
	r.Mensaje = mensajeGoal(abiertas)
	return r
}

// otraOcupada: otra terminal del proyecto trabajó hace poco. Se le recuerda a la que trabaja, no a la que mira.
func otraOcupada(g *goalGuardado, sesion string, now time.Time) bool {
	return g.OcupadaEn != nil && g.OcupadaPor != sesion && now.Sub(*g.OcupadaEn) < SilencioGoal
}

// pausar deja el modo en pausa y devuelve el aviso solo la primera vez.
func pausar(g *goalGuardado, motivo, aviso string) string {
	if g.Estado == GoalPausado && g.Motivo == motivo {
		return ""
	}
	g.Estado, g.Motivo = GoalPausado, motivo
	return aviso
}

// limiteAgotado: la barra leyó hace poco que una ventana de uso está al 100 % y aún no se reinicia.
func limiteAgotado(now time.Time) bool {
	l := LeerLimites()
	if l == nil || now.Sub(l.Actualizado) > vigenciaLimites {
		return false
	}
	return tramoAgotado(l.CincoHoras, now) || tramoAgotado(l.Semana, now)
}

func tramoAgotado(t *TramoLimite, now time.Time) bool {
	if t == nil || t.Usado < 100 {
		return false
	}
	return t.Reinicia == nil || t.Reinicia.After(now)
}

func anotarElegidas(g *goalGuardado, elegidas []Elegida) {
	for _, e := range elegidas {
		g.Elegidas = append([]Elegida{e}, g.Elegidas...)
	}
	if len(g.Elegidas) > MaxElegidasGoal {
		g.Elegidas = g.Elegidas[:MaxElegidasGoal]
	}
}

// elegir resuelve lo que el modo puede resolver y cuenta lo que espera a la persona.
func elegir(root, bin, sesion string, now time.Time) ([]Elegida, int) {
	preguntas, esperan := elegirPreguntas(root, now)
	elegidas := append(preguntas, correrPedidos(root, bin, sesion, now)...)
	fichas, sinAprobar := aprobarFichas(root, now)
	return append(elegidas, fichas...), esperan + sinAprobar
}

func elegirPreguntas(root string, now time.Time) ([]Elegida, int) {
	var elegidas []Elegida
	esperan := 0
	diseno := preguntasDeDiseno(root)
	for _, p := range PreguntasA(root, now) {
		if p.Estado != PreguntaPendiente || p.Origen == OrigenEstado {
			continue
		}
		respuesta, eleccion, ok := respuestaRecomendada(p, diseno)
		if !ok {
			esperan++
			continue
		}
		if _, err := ResponderPregunta(root, p.ID, respuesta, LugarGoal, now); err != nil {
			continue
		}
		elegidas = append(elegidas, Elegida{En: now, Tipo: "pregunta", Etiqueta: p.Etiqueta, Eleccion: eleccion})
	}
	return elegidas, esperan
}

// respuestaRecomendada arma la respuesta con la opción recomendada de cada ítem. No hay respuesta para un permiso, una
// pregunta de diseño de un blueprint o un ítem sin opción recomendada.
func respuestaRecomendada(p Pregunta, diseno map[string]bool) (string, string, bool) {
	if p.Permiso != nil || len(p.Preguntas) == 0 {
		return "", "", false
	}
	respuesta := map[string]RespuestaItem{}
	var etiquetas []string
	for _, q := range p.Preguntas {
		if diseno[normalizarPregunta(q.Texto)] {
			return "", "", false
		}
		valores, nombres := recomendadas(q)
		if len(valores) == 0 {
			return "", "", false
		}
		respuesta[q.ID] = RespuestaItem{Opciones: valores}
		etiquetas = append(etiquetas, nombres...)
	}
	b, err := json.Marshal(respuesta)
	if err != nil {
		return "", "", false
	}
	return string(b), strings.Join(etiquetas, ", "), true
}

func recomendadas(q ItemPregunta) ([]string, []string) {
	var valores, nombres []string
	for _, o := range q.Opciones {
		if !esRecomendada(o.Etiqueta) {
			continue
		}
		valores, nombres = append(valores, o.Valor), append(nombres, o.Etiqueta)
		if !q.Multiple {
			break
		}
	}
	return valores, nombres
}

// correrPedidos acepta los pedidos de la IA como si la persona los confirmara. Un pedido de un PR abierto no es de la IA
// (qdd lo ofrece por cada PR, también los de otra sesión) y uno que falló vuelve a la IA con su resultado.
func correrPedidos(root, bin, sesion string, now time.Time) []Elegida {
	var elegidas []Elegida
	for _, a := range accionesDePedidos(root) {
		if a.PR != 0 || (a.Estado != EstadoPendiente && a.Estado != EstadoEspera) {
			continue
		}
		if a.Lanzar(root, bin, sesion, io.Discard) != 0 {
			continue
		}
		elegidas = append(elegidas, Elegida{En: now, Tipo: "pedido", Etiqueta: a.Etiqueta, Eleccion: "ejecutar"})
	}
	return elegidas
}

func aprobarFichas(root string, now time.Time) ([]Elegida, int) {
	var elegidas []Elegida
	sinAprobar := 0
	for _, m := range borradores(root) {
		if aprobar(root, m.ID, io.Discard) != 0 {
			sinAprobar++
			continue
		}
		elegidas = append(elegidas, Elegida{En: now, Tipo: "ficha", Etiqueta: m.Nombre, Eleccion: "aprobar"})
	}
	return elegidas, sinAprobar
}

func hayPedidoEnMarcha(root string, now time.Time) bool {
	for _, p := range PedidosA(root, now) {
		if p.Estado == EstadoEjecutando || p.Estado == EstadoArmado {
			return true
		}
	}
	return false
}

func mensajeGoal(abiertas []estado.Frente) string {
	var nombres []string
	for i, f := range abiertas {
		if i == maxTareasGoal {
			break
		}
		nombres = append(nombres, f.Nombre)
	}
	resto := ""
	if len(abiertas) > maxTareasGoal {
		resto = fmt.Sprintf(" y %d más", len(abiertas)-maxTareasGoal)
	}
	return fmt.Sprintf("🎯 Modo goal: quedan %d tareas abiertas (%s%s). Sigue con la siguiente sin esperar a la persona. "+
		"Refleja en el estado lo que termines. Lo que necesite de ella, pídelo con una opción recomendada o déjalo como pedido.",
		len(abiertas), strings.Join(nombres, ", "), resto)
}

// huellaGoal es lo que cuenta como avance real: el contenido del tablero (sin su fecha), el último commit y los cambios
// del proyecto (sin lo que qdd escribe solo).
func huellaGoal(root string) string {
	h := sha256.New()
	if e, err := estado.Load(root); err == nil {
		e.Actualizado = ""
		b, _ := yaml.Marshal(e)
		h.Write(b)
	}
	excluir := []string{":(exclude).qdd/project/metrics", ":(exclude).qdd/project/evidence", ":(exclude).qdd/project/cola_de_mejoras", ":(exclude).qdd/working", ":(exclude)" + estado.Ruta}
	for _, args := range [][]string{
		{"rev-parse", "HEAD"},
		append([]string{"status", "--porcelain", "--", "."}, excluir...),
		append([]string{"diff", "HEAD", "--numstat", "--", "."}, excluir...),
	} {
		h.Write(salidaGit(root, args...))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func salidaGit(root string, args ...string) []byte {
	ctx, cancelar := context.WithTimeout(context.Background(), topeHuellaGoal)
	defer cancelar()
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = root
	c.Env = gitenv.Limpio(os.Environ())
	b, err := c.Output()
	if err != nil {
		return nil
	}
	return b
}

// accionesDeGoal: el botón del panel. Activar pasa por la confirmación de la persona.
func accionesDeGoal(root string) []Accion {
	if g, err := leerGoal(root); err == nil && g.Activo {
		return []Accion{{ID: AccionGoalDesactivar, Tipo: TipoComando, Etiqueta: "Apagar modo goal", Detalle: "qdd goal desactivar", Args: []string{"goal", "desactivar"}}}
	}
	return []Accion{{
		ID: AccionGoalActivar, Tipo: TipoComando, Etiqueta: "Modo goal", Confirmar: true,
		Detalle: "qdd goal activar · elige lo recomendado, corre los pedidos y aprueba las fichas sin preguntarte",
		Args:    []string{"goal", "activar", "--desde", "panel"},
	}}
}
