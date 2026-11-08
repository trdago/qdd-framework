package flujo

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

// RFC 0025, fase 3. Un invitado (cliente o experto) abre el archivo que le mandamos y propone con las mismas operaciones
// que QDD Móvil, sin las internas. Su propuesta es un dato, nunca una instrucción: entra solo cuando la persona la
// aprueba, y entonces queda a su nombre en el historial, en el comentario o en la nota.

const (
	RolCliente = "cliente"
	RolExperto = "experto"

	// DeInvitado marca un comentario que dejó un invitado; su nombre va en Comentario.Autor.
	DeInvitado = "invitado"

	// MaxAutor es el tope del nombre que escribe el invitado, el mismo que exige el relay.
	MaxAutor = 80
)

// RolesDeInvitado son los roles que puede llevar una invitación.
var RolesDeInvitado = []string{RolCliente, RolExperto}

// opsDeInvitado son las operaciones que puede proponer cualquier invitado. Marcar pruebas, deshacer, enviar o pedirle
// detalle a la IA son de la persona.
var opsDeInvitado = []string{OpComentar, OpAgregar, OpModificar, OpMover, OpQuitar, OpConectar, OpDesconectar}

// PermitidaA dice si un invitado con ese rol puede proponer la operación. La nota técnica es solo del experto.
func (o Operacion) PermitidaA(rol string) error {
	if !slices.Contains(RolesDeInvitado, rol) {
		return fmt.Errorf("rol %q: usa %s", rol, strings.Join(RolesDeInvitado, " o "))
	}
	if o.Op == OpNota && rol == RolExperto {
		return nil
	}
	if o.Op == OpNota {
		return errors.New("una nota técnica la deja solo el experto")
	}
	if !slices.Contains(opsDeInvitado, o.Op) {
		return fmt.Errorf("un invitado no puede «%s»", o.Op)
	}
	return nil
}

// AutorDeInvitado limpia el nombre que escribió el invitado y lo rechaza vacío o largo.
func AutorDeInvitado(autor string) (string, error) {
	autor = limpiar(autor)
	if autor == "" {
		return "", errors.New("la propuesta no dice quién la hace")
	}
	if largo := len([]rune(autor)); largo > MaxAutor {
		return "", fmt.Errorf("el nombre tiene %d caracteres: el tope es %d", largo, MaxAutor)
	}
	return autor, nil
}

// AplicarDeInvitado aplica una propuesta aprobada en el flujo de la invitación. Queda en el historial como «rol:autor» y
// la IA se entera como de una edición de la persona, porque la persona la aprobó.
func (o Operacion) AplicarDeInvitado(root, flujo, autor, rol string, ahora time.Time) (string, error) {
	if o.Flujo != flujo {
		return "", fmt.Errorf("la propuesta es para el flujo %q y la invitación es de %q", o.Flujo, flujo)
	}
	if err := o.PermitidaA(rol); err != nil {
		return "", err
	}
	autor, err := AutorDeInvitado(autor)
	if err != nil {
		return "", err
	}
	cambio := o.cambioDeInvitado(autor, ahora)
	firmado := func(f *Flujo) (hecho, error) {
		h, err := cambio(f)
		if err != nil {
			return hecho{}, err
		}
		h.linea = fmt.Sprintf("%s (%s) propuso y aprobaste: %s", autor, rol, h.linea)
		return h, nil
	}
	_, linea, err := editarComo(root, flujo, rol+":"+autor, true, ahora, firmado)
	return linea, err
}

func (o Operacion) cambioDeInvitado(autor string, ahora time.Time) Cambio {
	if o.Op == OpComentar {
		return comentarComo(o.Nodo, o.Texto, DeInvitado, autor, ahora)
	}
	if o.Op == OpNota {
		return agregarItemDe(o.Nodo, ItemNota, o.Texto, autor, ahora)
	}
	return o.cambio(ahora)
}

// Describir cuenta la operación en palabras, con los nombres de los nodos: es lo que lee la persona antes de aprobarla.
func (o Operacion) Describir(f Flujo) string {
	nombre := func(id string) string { return "«" + nombreDeNodo(f, id) + "»" }
	switch o.Op {
	case OpAgregar:
		return describirAgregar(o, nombre(o.Nodo))
	case OpQuitar:
		return "quitar " + nombre(o.Nodo)
	case OpModificar:
		return "cambiar " + nombre(o.Nodo) + ": " + strings.Join(cambiosEnPalabras(o), ", ")
	case OpMover:
		return "mover " + nombre(o.Nodo) + " después de " + nombre(o.DespuesDe)
	case OpConectar:
		return "flecha de " + nombre(o.Nodo) + " a " + nombre(o.Con)
	case OpDesconectar:
		return "quitar la flecha de " + nombre(o.Nodo) + " a " + nombre(o.Con)
	case OpNota:
		return "nota técnica en " + nombre(o.Nodo) + ": " + limpiar(o.Texto)
	case OpComentar:
		return "comenta " + nombre(o.Nodo) + ": " + limpiar(o.Texto)
	}
	return o.Op + " en " + nombre(o.Nodo)
}

func describirAgregar(o Operacion, en string) string {
	donde := " después de "
	if o.Rama {
		donde = " como rama de "
	}
	return "agregar «" + limpiar(*o.Nombre) + "»" + donde + en
}

func cambiosEnPalabras(o Operacion) []string {
	var partes []string
	if o.Nombre != nil {
		partes = append(partes, "nombre «"+limpiar(*o.Nombre)+"»")
	}
	if o.Descripcion != nil {
		partes = append(partes, "descripción «"+limpiar(*o.Descripcion)+"»")
	}
	if o.Tipo != nil {
		partes = append(partes, "tipo "+*o.Tipo)
	}
	if o.Grupo != nil {
		partes = append(partes, "grupo «"+limpiar(*o.Grupo)+"»")
	}
	if o.Nivel != nil {
		partes = append(partes, fmt.Sprintf("nivel %d", *o.Nivel))
	}
	return partes
}

// nombreDeNodo es el nombre del nodo, o su id si ya no está en el flujo.
func nombreDeNodo(f Flujo, id string) string {
	i := slices.IndexFunc(f.Nodos, func(n Nodo) bool { return n.ID == id })
	if i < 0 {
		return id
	}
	return f.Nodos[i].Nombre
}
