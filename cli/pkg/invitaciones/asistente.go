package invitaciones

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'

// Lo que propone el asistente (RFC 0025, fase 4) entra a la misma cola que lo de los invitados: el ciclo lo deja como
// pedido de la persona y aprobarlo lo aplica a nombre del asistente. Nace en el PC, así que el relay no sabe de él;
// solo recibe la foto nueva cuando hay un invitado mirando ese flujo.

const (
	// PrefijoAsistente es el de los ids de sus propuestas: «ASI-n», para no cruzarse con las «PRO-n» del relay.
	PrefijoAsistente = "ASI-"
	// MaxDeAsistente es cuántas suyas esperan sin resolver: con el tope lleno, espera a que la persona resuelva.
	MaxDeAsistente = 30
)

// ProponerDeAsistente deja una propuesta del asistente en la cola y devuelve su id. El minuto (mm:ss) es el de la
// conversación que la respalda; vacío, es del asistente sin más. La misma operación sin resolver no se repite.
func ProponerDeAsistente(root string, raw json.RawMessage, minuto string, ahora time.Time) (string, error) {
	o, err := LeerOperacion(raw)
	if err != nil {
		return "", err
	}
	if err := o.PermitidaAlAsistente(); err != nil {
		return "", err
	}
	origen, err := flujo.OrigenDeAsistente(minuto)
	if err != nil {
		return "", err
	}
	f, err := flujo.Cargar(root, o.Flujo)
	if err != nil {
		return "", err
	}
	var compacto bytes.Buffer
	if err := json.Compact(&compacto, raw); err != nil {
		return "", err
	}
	var id string
	err = conLocal(root, func(l *Local) error {
		if previa := l.delAsistenteSinResolver(compacto.Bytes()); previa != "" {
			id = previa
			return nil
		}
		if l.sinResolverDelAsistente() >= MaxDeAsistente {
			return fmt.Errorf("el asistente ya tiene %d propuestas sin resolver: espera a que la persona las vea", MaxDeAsistente)
		}
		id = PrefijoAsistente + strconv.Itoa(l.siguienteDelAsistente())
		l.Propuestas = append(l.Propuestas, Vista{
			Propuesta: Propuesta{ID: id, Flujo: o.Flujo, Rol: flujo.RolAsistente, Autor: flujo.OrigenAsistente,
				Base: "r" + strconv.Itoa(f.Revision), Operacion: compacto.Bytes(), Estado: Pendiente, Llego: ahora.UTC().Format(time.RFC3339)},
			Situacion: PorPedir,
			Origen:    origen,
		})
		return nil
	})
	return id, err
}

// delAsistente: nació en este PC. Una del relay siempre trae su invitación, diga el rol que diga.
func delAsistente(v Vista) bool {
	return v.Invitacion == "" && strings.HasPrefix(v.ID, PrefijoAsistente) && v.Rol == flujo.RolAsistente
}

// delAsistenteSinResolver compara compactadas: el archivo local se guarda con sangría y la operación con él.
func (l *Local) delAsistenteSinResolver(operacion []byte) string {
	for _, v := range l.Propuestas {
		var guardada bytes.Buffer
		if !delAsistente(v) || resuelta(v) || json.Compact(&guardada, v.Operacion) != nil {
			continue
		}
		if bytes.Equal(guardada.Bytes(), operacion) {
			return v.ID
		}
	}
	return ""
}

func (l *Local) sinResolverDelAsistente() int {
	n := 0
	for _, v := range l.Propuestas {
		if delAsistente(v) && !resuelta(v) {
			n++
		}
	}
	return n
}

func (l *Local) siguienteDelAsistente() int {
	mayor := 0
	for _, v := range l.Propuestas {
		n, err := strconv.Atoi(strings.TrimPrefix(v.ID, PrefijoAsistente))
		if err == nil && strings.HasPrefix(v.ID, PrefijoAsistente) && n > mayor {
			mayor = n
		}
	}
	return mayor + 1
}

// encabezado es cómo empieza el pedido: quién propone y, del asistente, el minuto que la respalda.
func encabezado(v Vista) string {
	if !delAsistente(v) {
		return "📝 " + strings.TrimSpace(v.Autor) + ": "
	}
	minuto, ok := strings.CutPrefix(v.Origen, "transcripcion@")
	if !ok {
		return "🎙 asistente: "
	}
	return "🎙 asistente (" + minuto + "): "
}

// QuienPropone es cómo se nombra a quien la hizo en la lista de propuestas.
func QuienPropone(v Vista) string {
	if delAsistente(v) {
		return v.Origen
	}
	return v.Autor + " (" + v.Rol + ")"
}
