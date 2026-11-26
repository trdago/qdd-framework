package consola

import (
	"os"
	"slices"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// RFC 0019: lo que espera a la persona recibe de qdd un número que no cambia ni se reutiliza. Panel y teléfono dibujan
// este orden: ninguno lo recalcula, así que la persona no tiene que adivinar qué contestar primero.

// turnoGuardado es el número de algo que sigue esperando y cuándo entró.
type turnoGuardado struct {
	Numero int       `json:"numero"`
	Desde  time.Time `json:"desde"`
}

// registroTurnos guarda el próximo número aparte de lo vivo: así uno que salió no se vuelve a usar.
type registroTurnos struct {
	Siguiente int                      `json:"siguiente"`
	Vivos     map[string]turnoGuardado `json:"vivos"`
}

// llevaTurno dice si la acción espera una respuesta de la persona. Lo que qdd ofrece por su cuenta no espera a nadie.
func llevaTurno(a Accion) bool {
	if a.Tipo == TipoAprobar {
		return true
	}
	if a.Tipo == TipoPregunta {
		return a.Estado == PreguntaPendiente
	}
	if a.Tipo != TipoPedido || a.Origen == OrigenTelefono || a.PR != 0 || a.VersionNueva != "" {
		return false
	}
	return a.Estado != EstadoHecho
}

// sePuedeContestar: un pedido ya confirmado conserva su número, pero la persona ya no tiene nada que hacer con él.
func sePuedeContestar(a Accion) bool {
	return a.Tipo != TipoPedido || a.Tecla != ""
}

func claveDeTurno(a Accion) string {
	if a.claveTurno != "" {
		return a.claveTurno
	}
	return a.ID
}

// conTurnos numera lo que espera a la persona y lo deja primero, en orden. Si el registro no se puede usar, el panel
// sale sin números: es mejor que no salir.
func conTurnos(root string, acciones []Accion, dependeDe map[string][]string, now time.Time) []Accion {
	var abiertas, resto []Accion
	claves := []string{}
	for _, a := range acciones {
		if !llevaTurno(a) {
			resto = append(resto, a)
			continue
		}
		abiertas = append(abiertas, a)
		claves = append(claves, claveDeTurno(a))
	}
	vivos, err := asignarTurnos(root, claves, now)
	if err != nil {
		return acciones
	}
	for i := range abiertas {
		t := vivos[claveDeTurno(abiertas[i])]
		desde := t.Desde
		abiertas[i].Turno, abiertas[i].TurnoDesde = t.Numero, &desde
	}
	marcarEsperas(abiertas, dependeDe)
	return append(enOrdenDeTurno(abiertas), resto...)
}

// marcarEsperas anota qué turnos espera cada una: los pedidos de su despues y lo abierto de las tareas de su depende_de.
func marcarEsperas(abiertas []Accion, dependeDe map[string][]string) {
	porPedido, porTarea := map[string]int{}, map[string][]int{}
	for _, a := range abiertas {
		if a.pedido != "" {
			porPedido[a.pedido] = a.Turno
		}
		if a.tareaTurno != "" {
			porTarea[a.tareaTurno] = append(porTarea[a.tareaTurno], a.Turno)
		}
	}
	for i, a := range abiertas {
		espera := []int{}
		for _, d := range a.despues {
			if n, ok := porPedido[d]; ok {
				espera = append(espera, n)
			}
		}
		for _, tarea := range dependeDe[a.tareaTurno] {
			espera = append(espera, porTarea[tarea]...)
		}
		slices.Sort(espera)
		abiertas[i].EsperaTurno = slices.Compact(espera)
	}
}

// enOrdenDeTurno va por número, con lo que cada una espera justo antes. Un ciclo no pierde nada, y «siguiente» es lo
// primero que se puede contestar.
func enOrdenDeTurno(abiertas []Accion) []Accion {
	slices.SortStableFunc(abiertas, func(a, b Accion) int { return a.Turno - b.Turno })
	porNumero := map[int]int{}
	for i, a := range abiertas {
		porNumero[a.Turno] = i
	}
	out := make([]Accion, 0, len(abiertas))
	visto := map[int]bool{}
	var poner func(i int)
	poner = func(i int) {
		a := abiertas[i]
		if visto[a.Turno] {
			return
		}
		visto[a.Turno] = true
		for _, n := range a.EsperaTurno {
			poner(porNumero[n])
		}
		out = append(out, a)
	}
	for i := range abiertas {
		poner(i)
	}
	if i := slices.IndexFunc(out, sePuedeContestar); i >= 0 {
		out[i].Siguiente = true
	}
	return out
}

// asignarTurnos da número a lo nuevo y olvida lo que ya no espera. Si nada cambió no toma el candado ni escribe: el panel
// lee a cada rato, y quien vigila la carpeta no debe ver un cambio que no hubo (FND-161).
func asignarTurnos(root string, claves []string, now time.Time) (map[string]turnoGuardado, error) {
	ruta, err := RutaLocal(root, "turnos")
	if err != nil {
		return nil, err
	}
	var leido registroTurnos
	if leerArchivoJSON(ruta, &leido) && mismasClaves(leido.Vivos, claves) {
		return leido.Vivos, nil
	}
	var vivos map[string]turnoGuardado
	err = conArchivo(ruta, "los turnos", func() error {
		reg := leerTurnos(ruta)
		vivos = map[string]turnoGuardado{}
		for _, c := range claves {
			t, ok := reg.Vivos[c]
			if !ok {
				t = turnoGuardado{Numero: reg.Siguiente, Desde: now.UTC()}
				reg.Siguiente++
			}
			vivos[c] = t
		}
		reg.Vivos = vivos
		return guardarJSON(ruta, reg)
	})
	return vivos, err
}

func mismasClaves(vivos map[string]turnoGuardado, claves []string) bool {
	if len(vivos) != len(claves) {
		return false
	}
	for _, c := range claves {
		if _, ok := vivos[c]; !ok {
			return false
		}
	}
	return true
}

// leerTurnos lee el registro; uno ilegible se aparta como .ilegible (no se pierde) y la numeración vuelve a empezar.
func leerTurnos(ruta string) registroTurnos {
	reg := registroTurnos{Siguiente: 1, Vivos: map[string]turnoGuardado{}}
	if !leerArchivoJSON(ruta, &reg) {
		_ = os.Rename(ruta, ruta+".ilegible")
		return registroTurnos{Siguiente: 1, Vivos: map[string]turnoGuardado{}}
	}
	return reg
}

// dependenciasDeTareas es el depende_de de cada tarea abierta.
func dependenciasDeTareas(e estado.Estado) map[string][]string {
	out := map[string][]string{}
	for _, f := range e.Abiertos() {
		out[f.ID] = f.DependeDe
	}
	return out
}
