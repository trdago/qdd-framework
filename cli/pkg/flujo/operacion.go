package flujo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Rule: Prohibido usar 'else'

// Una operación es una edición de la persona escrita como un objeto JSON: así las manda QDD Móvil (RFC 0013, tipo flujo).
// El relay solo revisa que sea JSON; quien decide es este esquema, que no admite nada que no conozca. El «de» no es un
// campo: lo que llega por aquí siempre es de la persona.

const (
	OpComentar     = "comentar"
	OpPedirDetalle = "pedir-detalle"
	OpPreguntar    = "preguntar"
	OpItemAgregar  = "item-agregar"
	OpItemMarcar   = "item-marcar"
	OpItemQuitar   = "item-quitar"
	OpAgregar      = "agregar"
	OpModificar    = "modificar"
	OpMover        = "mover"
	OpQuitar       = "quitar"
	OpConectar     = "conectar"
	OpDesconectar  = "desconectar"
	OpDeshacer     = "deshacer"
	OpEnviar       = "enviar"

	// MaxOperacion es el tope del JSON entero, el mismo que exige el relay.
	MaxOperacion = 8000
)

// Operacion es una edición leída. Nombre, Descripcion y Tipo son punteros porque modificar distingue «no viene» de «vacío».
type Operacion struct {
	Op          string  `json:"op"`
	Flujo       string  `json:"flujo"`
	Nodo        string  `json:"nodo"`
	Nombre      *string `json:"nombre"`
	Descripcion *string `json:"descripcion"`
	Tipo        *string `json:"tipo"`
	DespuesDe   string  `json:"despues_de"`
	Con         string  `json:"con"`
	Rama        bool    `json:"rama"`
	Texto       string  `json:"texto"`
	Item        string  `json:"item"`
	Estado      string  `json:"estado"`
	Evidencia   string  `json:"evidencia"`
	Grupo       *string `json:"grupo"`
	Nivel       *int    `json:"nivel"`
}

// camposDe dice qué exige y qué admite cada operación, además de op y flujo.
var camposDe = map[string]struct{ exige, admite []string }{
	OpComentar:     {exige: []string{"nodo", "texto"}},
	OpPedirDetalle: {exige: []string{"nodo"}},
	OpPreguntar:    {exige: []string{"texto"}, admite: []string{"nodo"}},
	OpItemAgregar:  {exige: []string{"nodo", "tipo", "texto"}},
	OpItemMarcar:   {exige: []string{"nodo", "item", "estado"}, admite: []string{"evidencia"}},
	OpItemQuitar:   {exige: []string{"nodo", "item"}},
	OpAgregar:      {exige: []string{"nodo", "nombre"}, admite: []string{"descripcion", "tipo", "rama"}},
	OpModificar:    {exige: []string{"nodo"}, admite: []string{"nombre", "descripcion", "tipo", "grupo", "nivel"}},
	OpMover:        {exige: []string{"nodo", "despues_de"}},
	OpQuitar:       {exige: []string{"nodo"}},
	OpConectar:     {exige: []string{"nodo", "con"}},
	OpDesconectar:  {exige: []string{"nodo", "con"}},
	OpDeshacer:     {},
	OpEnviar:       {},
}

// LeerOperacion interpreta el JSON y rechaza lo que no sea exactamente una operación del esquema.
func LeerOperacion(texto string) (Operacion, error) {
	if !utf8.ValidString(texto) || utf8.RuneCountInString(texto) > MaxOperacion {
		return Operacion{}, fmt.Errorf("la operación no es texto o tiene más de %d caracteres", MaxOperacion)
	}
	if strings.ContainsFunc(texto, unicode.IsControl) {
		return Operacion{}, errors.New("la operación trae caracteres de control")
	}
	var o Operacion
	lector := json.NewDecoder(strings.NewReader(texto))
	lector.DisallowUnknownFields()
	if err := lector.Decode(&o); err != nil {
		return Operacion{}, fmt.Errorf("la operación no cumple el esquema: %w", err)
	}
	if _, err := lector.Token(); !errors.Is(err, io.EOF) {
		return Operacion{}, errors.New("la operación trae datos después del objeto")
	}
	if err := o.validar(); err != nil {
		return Operacion{}, err
	}
	return o, nil
}

func (o Operacion) validar() error {
	campos, ok := camposDe[o.Op]
	if !ok {
		return fmt.Errorf("operación «%s» desconocida", o.Op)
	}
	if !idValido.MatchString(o.Flujo) {
		return fmt.Errorf("id de flujo %q: usa minúsculas, números y guiones", o.Flujo)
	}
	presentes := o.presentes()
	for _, c := range presentes {
		if !slices.Contains(campos.exige, c) && !slices.Contains(campos.admite, c) {
			return fmt.Errorf("«%s» no lleva el campo %s", o.Op, c)
		}
	}
	for _, c := range campos.exige {
		if !slices.Contains(presentes, c) {
			return fmt.Errorf("«%s» necesita el campo %s", o.Op, c)
		}
	}
	if o.Op == OpModificar && len(presentes) == 1 {
		return errors.New("«modificar» necesita nombre, descripcion, tipo, grupo o nivel")
	}
	for _, id := range []string{o.Nodo, o.DespuesDe, o.Con, o.Item} {
		if id != "" && !idValido.MatchString(id) {
			return fmt.Errorf("id de nodo %q: usa minúsculas, números y guiones", id)
		}
	}
	if o.Op == OpItemAgregar && !slices.Contains(tiposDeItem, *o.Tipo) {
		return fmt.Errorf("tipo de ítem %q: usa %s", *o.Tipo, strings.Join(tiposDeItem, ", "))
	}
	if o.Op != OpItemAgregar && o.Tipo != nil && !tipoConocido(*o.Tipo) {
		return fmt.Errorf("tipo %q: ninguna plantilla lo conoce (proceso: paso o resultado)", *o.Tipo)
	}
	return nil
}

// presentes lista los campos que trae, sin contar los que solo traen espacios: un nombre o un comentario así no dice nada.
func (o Operacion) presentes() []string {
	var lista []string
	for _, c := range []struct {
		campo string
		viene bool
	}{
		{"nodo", o.Nodo != ""},
		{"nombre", o.Nombre != nil && (o.Op != OpAgregar || limpiar(*o.Nombre) != "")},
		{"descripcion", o.Descripcion != nil},
		{"tipo", o.Tipo != nil},
		{"despues_de", o.DespuesDe != ""},
		{"con", o.Con != ""},
		{"rama", o.Rama},
		{"texto", limpiar(o.Texto) != ""},
		{"item", o.Item != ""},
		{"estado", o.Estado != ""},
		{"evidencia", o.Evidencia != ""},
		{"grupo", o.Grupo != nil},
		{"nivel", o.Nivel != nil},
	} {
		if c.viene {
			lista = append(lista, c.campo)
		}
	}
	return lista
}

// LlamaALaIA dice si la operación pide respuesta, como en el panel: una edición espera a «enviar», para no gastar un turno
// por cambio.
func (o Operacion) LlamaALaIA() bool {
	return o.Op == OpComentar || o.Op == OpPedirDetalle || o.Op == OpPreguntar
}

// Aplicar hace la edición como de la persona y dice en una línea qué hizo. Enviar no se aplica aquí: reclama lo pendiente
// y quien la recibe debe entregarlo.
func (o Operacion) Aplicar(root string, ahora time.Time) (string, error) {
	if o.Op == OpEnviar {
		return "", errors.New("«enviar» no es una edición")
	}
	if o.Op == OpDeshacer {
		return "se deshizo la última edición", Deshacer(root, o.Flujo)
	}
	if o.Op == OpPedirDetalle {
		return "se pidió más detalle de " + o.Nodo, PedirDetalle(root, o.Flujo, o.Nodo, ahora)
	}
	_, linea, err := Editar(root, o.Flujo, DePersona, ahora, o.cambio(ahora))
	return linea, err
}

func (o Operacion) cambio(ahora time.Time) Cambio {
	switch o.Op {
	case OpAgregar:
		return Agregar(o.Nodo, *o.Nombre, valorDe(o.Tipo), valorDe(o.Descripcion), o.Rama)
	case OpModificar:
		return ModificarNodo(o.Nodo, Campos{Nombre: o.Nombre, Descripcion: o.Descripcion, Tipo: o.Tipo, Grupo: o.Grupo, Nivel: o.Nivel})
	case OpMover:
		return Mover(o.Nodo, o.DespuesDe)
	case OpQuitar:
		return Quitar(o.Nodo)
	case OpConectar:
		return Conectar(o.Nodo, o.Con)
	case OpDesconectar:
		return Desconectar(o.Nodo, o.Con)
	case OpPreguntar:
		return Preguntar(o.Nodo, o.Texto, ahora)
	case OpItemAgregar:
		return AgregarItem(o.Nodo, *o.Tipo, o.Texto, ahora)
	case OpItemMarcar:
		return MarcarItem(o.Nodo, o.Item, o.Estado, o.Evidencia, ahora)
	case OpItemQuitar:
		return QuitarItem(o.Nodo, o.Item)
	}
	return Comentar(o.Nodo, o.Texto, DePersona, ahora)
}

func valorDe(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
