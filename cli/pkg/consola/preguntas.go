package consola

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/qdd-framework/qdd/pkg/candado"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Preguntas desde el teléfono»: lo que la consola del PC le pregunta a la persona (una pregunta de la IA
// con AskUserQuestion, un permiso de herramienta, o la pregunta de un comando qdd) queda en una cola local por
// repositorio (<QDD_HOME>/claude/preguntas/, RutaLocal, como los pedidos) y el panel la ofrece como acción
// «pregunta», solo a quien la pide (tipo opcional: qdd remoto, para el teléfono). Gana quien responde primero: el
// terminal (el mod la cierra «en el PC») o un teléfono (ResponderPregunta, que valida la respuesta contra la pregunta
// guardada). Una pregunta que su sesión dejó de esperar, o que lleva demasiado, vence.

const (
	TipoPregunta = "pregunta"

	OrigenIA      = "ia"
	OrigenPermiso = "permiso"
	OrigenQDD     = "qdd"

	PreguntaPendiente  = "pendiente"
	PreguntaRespondida = "respondida"
	PreguntaVencida    = "vencida"

	EnPC       = "pc"
	EnTelefono = "telefono"

	// PermitirUnaVez y Negar son las únicas respuestas a un permiso: desde el teléfono nunca se ofrece ni se aplica
	// «permitir siempre» ni un cambio de reglas (decisión del usuario, 2026-11-03).
	PermitirUnaVez = "permitir"
	Negar          = "negar"

	// LatidoPregunta: una pendiente cuya sesión no la volvió a esperar (qdd consola pregunta esperar) en este tiempo,
	// vence: Claude Code se cerró o se cayó sin avisar.
	LatidoPregunta = time.Minute
	// VidaPregunta es lo más que una pregunta espera al teléfono; el terminal la sigue esperando.
	VidaPregunta = 30 * time.Minute
	// VidaCerrada es cuánto se sigue viendo una respondida o vencida («ya respondida en el PC»).
	VidaCerrada = 5 * time.Minute
	// MaxPreguntasCola es el tope de la cola (las más nuevas).
	MaxPreguntasCola = 20
	// MaxOtra es el tope del texto de «Otra» (la app lo recorta a lo mismo).
	MaxOtra = 500
	// MaxRespuesta es el tope de la respuesta entera, como texto (el relay exige lo mismo).
	MaxRespuesta = 4096
	// MaxDetallePermiso es lo que se muestra del comando de un permiso; lo que pasa se cuenta.
	MaxDetallePermiso = 4000

	herramientaPreguntar = "AskUserQuestion"
	prefijoPregunta      = "pregunta:"
	valoresOpcion        = "abcdefghijklmnopqrstuvwxyz"
)

var (
	// ErrPreguntaDesconocida: la cola no tiene esa pregunta (nunca estuvo o ya salió).
	ErrPreguntaDesconocida = errors.New("esa pregunta ya no está en este PC")
	// ErrRespuestaInvalida envuelve el motivo por el que una respuesta no sirve para su pregunta.
	ErrRespuestaInvalida = errors.New("respuesta inválida")
)

// PreguntaCerrada es el error de responder una pregunta que ya no espera: trae cómo quedó (respondida dónde, o
// vencida y por qué).
type PreguntaCerrada struct{ Pregunta Pregunta }

func (e *PreguntaCerrada) Error() string {
	if e.Pregunta.Estado == PreguntaVencida {
		return "la pregunta venció: " + e.Pregunta.Motivo
	}
	return "ya respondida en " + LugarDe(e.Pregunta)
}

// LugarDe dice dónde se respondió una pregunta: «PC» o el nombre del teléfono.
func LugarDe(p Pregunta) string {
	if p.RespondidaEn != EnTelefono {
		return "PC"
	}
	if p.Lugar == "" {
		return "un teléfono"
	}
	return p.Lugar
}

// OpcionPregunta es una opción de una pregunta: el valor que manda el teléfono y lo que lee la persona.
type OpcionPregunta struct {
	Valor       string `json:"valor"`
	Etiqueta    string `json:"etiqueta"`
	Descripcion string `json:"descripcion"`
}

// ItemPregunta es una de las preguntas de una acción «pregunta».
type ItemPregunta struct {
	ID         string           `json:"id"`
	Encabezado string           `json:"encabezado"`
	Texto      string           `json:"texto"`
	Multiple   bool             `json:"multiple"`
	Otra       bool             `json:"otra"`
	Opciones   []OpcionPregunta `json:"opciones"`
}

// PermisoPregunta es lo que pide un permiso: la herramienta y su detalle (el comando exacto de Bash).
type PermisoPregunta struct {
	Herramienta string `json:"herramienta"`
	Detalle     string `json:"detalle"`
}

// RespuestaItem es la respuesta a una pregunta: las opciones elegidas o el texto de «Otra».
type RespuestaItem struct {
	Opciones []string `json:"opciones,omitempty"`
	Otra     *string  `json:"otra,omitempty"`
}

// Pregunta es una entrada de la cola de preguntas.
type Pregunta struct {
	ID           string                   `json:"id"`
	Origen       string                   `json:"origen"`
	Sesion       string                   `json:"sesion,omitempty"`
	Herramienta  string                   `json:"herramienta,omitempty"`
	Etiqueta     string                   `json:"etiqueta"`
	Preguntas    []ItemPregunta           `json:"preguntas"`
	Permiso      *PermisoPregunta         `json:"permiso,omitempty"`
	Estado       string                   `json:"estado"`
	RespondidaEn string                   `json:"respondida_en,omitempty"`
	Lugar        string                   `json:"lugar,omitempty"`  // el teléfono que la respondió
	Motivo       string                   `json:"motivo,omitempty"` // por qué venció
	Respuesta    map[string]RespuestaItem `json:"respuesta,omitempty"`
	Instruccion  string                   `json:"instruccion,omitempty"` // la instrucción que lleva la respuesta a un ⏳ (FND-141)
	Creada       time.Time                `json:"creada"`
	Vista        time.Time                `json:"vista"` // la última vez que su sesión la esperó
	Cerrada      *time.Time               `json:"cerrada,omitempty"`
}

// NuevaPregunta es lo que registra el mod (qdd consola pregunta registrar): la herramienta de Claude Code y su
// entrada (AskUserQuestion o un permiso), o un comando qdd con sus preguntas (origen qdd).
type NuevaPregunta struct {
	Origen      string          `json:"origen,omitempty"`
	Sesion      string          `json:"sesion,omitempty"`
	Herramienta string          `json:"herramienta,omitempty"`
	Entrada     json.RawMessage `json:"entrada,omitempty"`
	Etiqueta    string          `json:"etiqueta,omitempty"`
	Preguntas   []ItemPregunta  `json:"preguntas,omitempty"`
}

// Entrega es lo que el mod le da a Claude Code: el estado y, si la respondió un teléfono, la respuesta en la forma de
// la herramienta (answers por texto de la pregunta, o la decisión de un permiso).
type Entrega struct {
	ID           string            `json:"id"`
	Estado       string            `json:"estado"`
	RespondidaEn string            `json:"respondida_en,omitempty"`
	Lugar        string            `json:"lugar,omitempty"`
	Motivo       string            `json:"motivo,omitempty"`
	Respuestas   map[string]string `json:"answers,omitempty"`
	Decision     string            `json:"decision,omitempty"`
}

// EntregaDe arma lo que recibe Claude Code: en AskUserQuestion, cada respuesta por el texto de su pregunta (varias
// opciones separadas por coma, como el diálogo); en un permiso, permitir o negar.
func EntregaDe(p Pregunta) Entrega {
	e := Entrega{ID: p.ID, Estado: p.Estado, RespondidaEn: p.RespondidaEn, Lugar: p.Lugar, Motivo: p.Motivo}
	if p.Estado != PreguntaRespondida || p.RespondidaEn != EnTelefono {
		return e
	}
	if p.Origen == OrigenPermiso {
		e.Decision = p.Respuesta["q1"].Opciones[0]
		return e
	}
	e.Respuestas = map[string]string{}
	for _, q := range p.Preguntas {
		r := p.Respuesta[q.ID]
		if r.Otra != nil {
			e.Respuestas[q.Texto] = *r.Otra
			continue
		}
		var etiquetas []string
		for _, v := range r.Opciones {
			etiquetas = append(etiquetas, etiquetaDe(q, v))
		}
		e.Respuestas[q.Texto] = strings.Join(etiquetas, ", ")
	}
	return e
}

func etiquetaDe(q ItemPregunta, valor string) string {
	for _, o := range q.Opciones {
		if o.Valor == valor {
			return o.Etiqueta
		}
	}
	return valor
}

// RegistrarPregunta deja una pregunta pendiente en la cola del proyecto.
func RegistrarPregunta(root string, n NuevaPregunta, now time.Time) (Pregunta, error) {
	p, err := armarPregunta(n)
	if err != nil {
		return Pregunta{}, err
	}
	p.ID, p.Estado, p.Sesion, p.Creada, p.Vista = nuevoIDPregunta(), PreguntaPendiente, strings.TrimSpace(n.Sesion), now.UTC(), now.UTC()
	err = conPreguntas(root, now, func(cola []Pregunta) ([]Pregunta, error) {
		return append([]Pregunta{p}, cola...), nil
	})
	return p, err
}

func armarPregunta(n NuevaPregunta) (Pregunta, error) {
	if n.Origen == OrigenQDD {
		return preguntaDeQDD(n)
	}
	if n.Herramienta == herramientaPreguntar {
		return preguntaDeLaIA(n.Entrada)
	}
	if strings.TrimSpace(n.Herramienta) == "" {
		return Pregunta{}, errors.New("falta la herramienta (AskUserQuestion o la que pide permiso) o el origen qdd con sus preguntas")
	}
	return preguntaDePermiso(strings.TrimSpace(n.Herramienta), n.Entrada), nil
}

// preguntaDeLaIA traduce la entrada de AskUserQuestion: cada pregunta q1…q4, cada opción a, b, c… y siempre «Otra»
// (Claude Code la agrega sola).
func preguntaDeLaIA(entrada json.RawMessage) (Pregunta, error) {
	var in struct {
		Questions []struct {
			Question    string `json:"question"`
			Header      string `json:"header"`
			MultiSelect bool   `json:"multiSelect"`
			Options     []struct {
				Label       string `json:"label"`
				Description string `json:"description"`
			} `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(entrada, &in); err != nil {
		return Pregunta{}, fmt.Errorf("la entrada de AskUserQuestion no se entiende: %v", err)
	}
	if len(in.Questions) == 0 {
		return Pregunta{}, errors.New("AskUserQuestion sin preguntas")
	}
	p := Pregunta{Origen: OrigenIA, Herramienta: herramientaPreguntar}
	for i, q := range in.Questions {
		item := ItemPregunta{ID: fmt.Sprintf("q%d", i+1), Encabezado: q.Header, Texto: q.Question, Multiple: q.MultiSelect, Otra: true}
		for j, o := range q.Options {
			if j >= len(valoresOpcion) {
				break
			}
			item.Opciones = append(item.Opciones, OpcionPregunta{Valor: valoresOpcion[j : j+1], Etiqueta: o.Label, Descripcion: o.Description})
		}
		p.Preguntas = append(p.Preguntas, item)
	}
	if err := validarItems(p.Preguntas); err != nil {
		return Pregunta{}, err
	}
	p.Etiqueta = recortar("❓ "+resumenDe(p.Preguntas[0]), MaxMotivo)
	return p, nil
}

func resumenDe(q ItemPregunta) string {
	if q.Encabezado == "" {
		return q.Texto
	}
	return q.Encabezado + " · " + q.Texto
}

// preguntaDePermiso arma «¿Permites que Claude use <herramienta>?» con el detalle exacto y solo permitir o negar.
func preguntaDePermiso(herramienta string, entrada json.RawMessage) Pregunta {
	detalle, descripcion := detalleDePermiso(entrada)
	resumen := descripcion
	if resumen == "" {
		resumen = strings.SplitN(detalle, "\n", 2)[0]
	}
	return Pregunta{
		Origen: OrigenPermiso, Herramienta: herramienta,
		Etiqueta: recortar("🔐 "+herramienta+": "+resumen, MaxMotivo),
		Permiso:  &PermisoPregunta{Herramienta: herramienta, Detalle: detalle},
		Preguntas: []ItemPregunta{{
			ID: "q1", Encabezado: "Permiso", Texto: "¿Permites que Claude use " + herramienta + "?",
			Opciones: []OpcionPregunta{
				{Valor: PermitirUnaVez, Etiqueta: "Permitir una vez", Descripcion: "Solo esta vez; no cambia los permisos"},
				{Valor: Negar, Etiqueta: "Negar", Descripcion: "Claude no la ejecuta y sigue sin ella"},
			},
		}},
	}
}

// detalleDePermiso es lo que la persona tiene que ver para decidir: el comando de Bash, el archivo de una edición, la
// dirección de una consulta web o, si no, la entrada entera. Muy largo, se corta diciendo cuánto falta.
func detalleDePermiso(entrada json.RawMessage) (string, string) {
	var campos map[string]any
	_ = json.Unmarshal(entrada, &campos)
	descripcion, _ := campos["description"].(string)
	for _, clave := range []string{"command", "file_path", "notebook_path", "url", "path"} {
		if v, ok := campos[clave].(string); ok && v != "" {
			return cortarDetalle(v), strings.TrimSpace(descripcion)
		}
	}
	var compacto bytes.Buffer
	if json.Compact(&compacto, entrada) != nil {
		return cortarDetalle(string(entrada)), strings.TrimSpace(descripcion)
	}
	return cortarDetalle(compacto.String()), strings.TrimSpace(descripcion)
}

func cortarDetalle(s string) string {
	r := []rune(s)
	if len(r) <= MaxDetallePermiso {
		return s
	}
	return string(r[:MaxDetallePermiso]) + fmt.Sprintf("… (+%d caracteres)", len(r)-MaxDetallePermiso)
}

func preguntaDeQDD(n NuevaPregunta) (Pregunta, error) {
	etiqueta := strings.TrimSpace(n.Etiqueta)
	if etiqueta == "" {
		return Pregunta{}, errors.New("una pregunta de qdd necesita su etiqueta (el comando que pregunta)")
	}
	if len(n.Preguntas) == 0 {
		return Pregunta{}, errors.New("una pregunta de qdd necesita al menos una pregunta")
	}
	if err := validarItems(n.Preguntas); err != nil {
		return Pregunta{}, err
	}
	return Pregunta{Origen: OrigenQDD, Etiqueta: recortar(etiqueta, MaxMotivo), Preguntas: n.Preguntas}, nil
}

// validarItems: ids únicos, un texto, al menos una opción (o «Otra») y valores de opción únicos.
func validarItems(items []ItemPregunta) error {
	ids := map[string]bool{}
	for _, q := range items {
		if q.ID == "" || ids[q.ID] {
			return fmt.Errorf("la pregunta «%s» no tiene un id propio", q.Texto)
		}
		ids[q.ID] = true
		if strings.TrimSpace(q.Texto) == "" {
			return fmt.Errorf("la pregunta %s no tiene texto", q.ID)
		}
		if len(q.Opciones) == 0 && !q.Otra {
			return fmt.Errorf("la pregunta %s no tiene opciones ni «Otra»", q.ID)
		}
		valores := map[string]bool{}
		for _, o := range q.Opciones {
			if o.Valor == "" || valores[o.Valor] {
				return fmt.Errorf("la pregunta %s repite o deja vacío el valor de una opción («%s»)", q.ID, o.Valor)
			}
			valores[o.Valor] = true
		}
	}
	return nil
}

func nuevoIDPregunta() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// IDPregunta quita el prefijo «pregunta:» de un id de acción (el teléfono puede mandar cualquiera de los dos).
func IDPregunta(id string) string {
	return strings.TrimPrefix(strings.TrimSpace(id), prefijoPregunta)
}

// ResponderPregunta aplica la respuesta de un teléfono (lugar es su nombre) si la pregunta sigue pendiente y la
// respuesta le sirve. Una cerrada devuelve *PreguntaCerrada con cómo quedó; una inválida, ErrRespuestaInvalida con el
// motivo, y la cola no cambia.
func ResponderPregunta(root, id, respuesta, lugar string, now time.Time) (Pregunta, error) {
	id = IDPregunta(id)
	if strings.HasPrefix(id, prefijoEspera) {
		return responderEspera(root, id, respuesta, lugar, now)
	}
	var out Pregunta
	err := conPreguntas(root, now, func(cola []Pregunta) ([]Pregunta, error) {
		i := indicePregunta(cola, id)
		if i < 0 {
			return nil, ErrPreguntaDesconocida
		}
		if cola[i].Estado != PreguntaPendiente {
			return nil, &PreguntaCerrada{Pregunta: cola[i]}
		}
		r, err := validarRespuesta(cola[i], respuesta)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrRespuestaInvalida, err)
		}
		cerrada := now.UTC()
		cola[i].Estado, cola[i].RespondidaEn, cola[i].Lugar, cola[i].Respuesta, cola[i].Cerrada = PreguntaRespondida, EnTelefono, strings.TrimSpace(lugar), r, &cerrada
		out = cola[i]
		return cola, nil
	})
	return out, err
}

// validarRespuesta exige una respuesta para cada pregunta y nada más: opciones que la pregunta trae (una sola si no es
// múltiple, sin repetir) o el texto de «Otra» si la admite (no vacío, hasta MaxOtra).
func validarRespuesta(p Pregunta, texto string) (map[string]RespuestaItem, error) {
	if len(texto) > MaxRespuesta {
		return nil, fmt.Errorf("la respuesta pasa de %d bytes", MaxRespuesta)
	}
	var crudas map[string]json.RawMessage
	if err := json.Unmarshal([]byte(texto), &crudas); err != nil || crudas == nil {
		return nil, errors.New("la respuesta no es un objeto JSON")
	}
	conocidas := map[string]bool{}
	for _, q := range p.Preguntas {
		conocidas[q.ID] = true
	}
	for qid := range crudas {
		if !conocidas[qid] {
			return nil, fmt.Errorf("no hay pregunta «%s»", qid)
		}
	}
	out := map[string]RespuestaItem{}
	for _, q := range p.Preguntas {
		cruda, ok := crudas[q.ID]
		if !ok {
			return nil, fmt.Errorf("falta responder «%s»", nombreItem(q))
		}
		r, err := validarItem(q, cruda)
		if err != nil {
			return nil, err
		}
		out[q.ID] = r
	}
	return out, nil
}

func nombreItem(q ItemPregunta) string {
	if q.Encabezado != "" {
		return q.Encabezado
	}
	return q.ID
}

func validarItem(q ItemPregunta, cruda json.RawMessage) (RespuestaItem, error) {
	var r RespuestaItem
	dec := json.NewDecoder(bytes.NewReader(cruda))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return RespuestaItem{}, fmt.Errorf("«%s»: %v", nombreItem(q), err)
	}
	if r.Otra != nil && r.Opciones != nil {
		return RespuestaItem{}, fmt.Errorf("«%s»: o opciones o «Otra», no las dos", nombreItem(q))
	}
	if r.Otra != nil {
		return validarOtra(q, *r.Otra)
	}
	if len(r.Opciones) == 0 {
		return RespuestaItem{}, fmt.Errorf("«%s»: no se eligió ninguna opción", nombreItem(q))
	}
	if len(r.Opciones) > 1 && !q.Multiple {
		return RespuestaItem{}, fmt.Errorf("«%s» admite una sola opción", nombreItem(q))
	}
	vistas := map[string]bool{}
	for _, v := range r.Opciones {
		if vistas[v] {
			return RespuestaItem{}, fmt.Errorf("«%s» repite «%s»", nombreItem(q), v)
		}
		vistas[v] = true
		if !ofrece(q, v) {
			return RespuestaItem{}, fmt.Errorf("la opción «%s» no está en «%s»", v, nombreItem(q))
		}
	}
	return r, nil
}

func validarOtra(q ItemPregunta, texto string) (RespuestaItem, error) {
	if !q.Otra {
		return RespuestaItem{}, fmt.Errorf("«%s» no admite «Otra»", nombreItem(q))
	}
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return RespuestaItem{}, fmt.Errorf("«Otra» vacía en «%s»", nombreItem(q))
	}
	if utf8.RuneCountInString(texto) > MaxOtra {
		return RespuestaItem{}, fmt.Errorf("«Otra» pasa de %d caracteres en «%s»", MaxOtra, nombreItem(q))
	}
	return RespuestaItem{Otra: &texto}, nil
}

func ofrece(q ItemPregunta, valor string) bool {
	for _, o := range q.Opciones {
		if o.Valor == valor {
			return true
		}
	}
	return false
}

// CerrarPregunta la cierra desde el PC: en EnPC, el terminal la respondió (aunque un teléfono haya alcanzado a
// responder antes de que el mod lo entregara: vale lo que vio Claude Code); con otro valor, vence con el motivo. Una
// ya cerrada en el PC o vencida queda como está.
func CerrarPregunta(root, id, como, motivo string, now time.Time) (Pregunta, error) {
	id = IDPregunta(id)
	var out Pregunta
	err := conPreguntas(root, now, func(cola []Pregunta) ([]Pregunta, error) {
		i := indicePregunta(cola, id)
		if i < 0 {
			return nil, ErrPreguntaDesconocida
		}
		cola[i] = cerrarEnPC(cola[i], como, motivo, now)
		out = cola[i]
		return cola, nil
	})
	return out, err
}

func cerrarEnPC(p Pregunta, como, motivo string, now time.Time) Pregunta {
	if p.Estado == PreguntaVencida || (p.Estado == PreguntaRespondida && p.RespondidaEn == EnPC) {
		return p
	}
	cerrada := now.UTC()
	p.Cerrada = &cerrada
	if como == EnPC {
		p.Estado, p.RespondidaEn, p.Lugar, p.Respuesta = PreguntaRespondida, EnPC, "", nil
		return p
	}
	if p.Estado != PreguntaPendiente {
		return p
	}
	p.Estado, p.Motivo = PreguntaVencida, strings.TrimSpace(motivo)
	return p
}

// VencerSesion vence las pendientes de una sesión (se cerró): devuelve cuántas.
func VencerSesion(root, sesion, motivo string, now time.Time) (int, error) {
	sesion = strings.TrimSpace(sesion)
	if sesion == "" {
		return 0, errors.New("falta la sesión")
	}
	n := 0
	err := conPreguntas(root, now, func(cola []Pregunta) ([]Pregunta, error) {
		for i, p := range cola {
			if p.Sesion != sesion || p.Estado != PreguntaPendiente {
				continue
			}
			cola[i] = cerrarEnPC(p, PreguntaVencida, motivo, now)
			n++
		}
		return cola, nil
	})
	return n, err
}

// LatirPregunta anota que su sesión la sigue esperando y devuelve cómo está.
func LatirPregunta(root, id string, now time.Time) (Pregunta, error) {
	id = IDPregunta(id)
	var out Pregunta
	err := conPreguntas(root, now, func(cola []Pregunta) ([]Pregunta, error) {
		i := indicePregunta(cola, id)
		if i < 0 {
			return nil, ErrPreguntaDesconocida
		}
		if cola[i].Estado == PreguntaPendiente {
			cola[i].Vista = now.UTC()
		}
		out = cola[i]
		return cola, nil
	})
	return out, err
}

// PreguntaDe lee una pregunta a la hora now, sin escribir.
func PreguntaDe(root, id string, now time.Time) (Pregunta, bool) {
	cola := PreguntasA(root, now)
	i := indicePregunta(cola, IDPregunta(id))
	if i < 0 {
		return Pregunta{}, false
	}
	return cola[i], true
}

// PreguntasA es la cola a la hora now, con los ⏳ de la persona del estado (esperas_preguntas.go): las pendientes
// primero, después las cerradas, las más nuevas antes.
func PreguntasA(root string, now time.Time) []Pregunta {
	return conEsperas(normalizarPreguntas(leerPreguntas(root, false), now), esperasDeLaPersona(root, now))
}

func indicePregunta(cola []Pregunta, id string) int {
	for i, p := range cola {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// normalizarPreguntas vence las pendientes que su sesión dejó de esperar o que pasaron VidaPregunta, quita las
// cerradas hace más de VidaCerrada y deja las MaxPreguntasCola más nuevas.
func normalizarPreguntas(cola []Pregunta, now time.Time) []Pregunta {
	out := make([]Pregunta, 0, len(cola))
	for _, p := range cola {
		if p.Estado == PreguntaPendiente && now.Sub(p.Creada) > VidaPregunta {
			p = cerrarEnPC(p, PreguntaVencida, "pasaron "+VidaPregunta.String()+" sin respuesta desde el teléfono; sigue esperando en la consola del PC", now)
		}
		if p.Estado == PreguntaPendiente && now.Sub(p.Vista) > LatidoPregunta {
			p = cerrarEnPC(p, PreguntaVencida, "la sesión de Claude Code ya no la espera (se cerró o se cortó)", now)
		}
		vida := VidaCerrada
		if p.Origen == OrigenEstado {
			vida = VidaEsperaRespondida
		}
		if p.Cerrada != nil && now.Sub(*p.Cerrada) > vida {
			continue
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := out[i].Estado == PreguntaPendiente, out[j].Estado == PreguntaPendiente
		if pi != pj {
			return pi
		}
		return out[i].Creada.After(out[j].Creada)
	})
	if len(out) > MaxPreguntasCola {
		return out[:MaxPreguntasCola]
	}
	return out
}

func rutaPreguntas(root string) (string, error) {
	return RutaLocal(root, "preguntas")
}

// leerPreguntas lee la cola; una ilegible se aparta como .ilegible (no se pierde) si apartar, y se empieza de cero.
func leerPreguntas(root string, apartar bool) []Pregunta {
	ruta, err := rutaPreguntas(root)
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil
	}
	var cola []Pregunta
	if json.Unmarshal(b, &cola) == nil {
		return cola
	}
	if apartar {
		_ = os.Rename(ruta, ruta+".ilegible")
	}
	return nil
}

// conPreguntas lee, cambia y guarda la cola con el archivo bloqueado (pkg/candado): el mod que registra y espera, y
// qdd remoto que responde, nunca se pisan, y una pregunta no se responde dos veces.
func conPreguntas(root string, now time.Time, cambiar func([]Pregunta) ([]Pregunta, error)) error {
	ruta, err := rutaPreguntas(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	soltar, err := candado.Tomar(ruta+".lock", 2*time.Second, "la cola de preguntas")
	if err != nil {
		return err
	}
	defer soltar()
	cola, err := cambiar(normalizarPreguntas(leerPreguntas(root, true), now))
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(cola, "", "  ")
	if err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

// accionesDePreguntas ofrece cada pregunta de la cola como acción «pregunta» (tipo opcional): las pendientes y, un
// rato, las respondidas o vencidas, para que el teléfono diga «ya respondida en …».
func accionesDePreguntas(root string, now time.Time) []Accion {
	var out []Accion
	for _, p := range PreguntasA(root, now) {
		a := Accion{
			ID: prefijoPregunta + p.ID, Tipo: TipoPregunta, Etiqueta: p.Etiqueta, Detalle: p.Preguntas[0].Texto,
			Estado: p.Estado, Sesion: p.Sesion, Origen: p.Origen, RespondidaEn: p.RespondidaEn,
			Preguntas: p.Preguntas, Permiso: p.Permiso, Confirmar: true, Resultado: p.Motivo,
		}
		if p.Permiso != nil {
			a.Detalle = p.Permiso.Detalle
		}
		out = append(out, a)
	}
	return out
}
