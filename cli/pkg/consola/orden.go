package consola

import (
	"fmt"
	"strings"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9: el orden de los pedidos. Un pedido puede esperar a otros (Despues): el panel lo muestra justo después
// de lo que espera y no lo deja ejecutar hasta que eso quede hecho (FND-080). Sin dependencias, el orden es el de la
// cola, como siempre.

// EstadoEspera es el estado que el panel ve en un pedido pendiente (o con falla) que espera a otro que no está hecho.
// No se guarda en la cola: se calcula al leerla.
const EstadoEspera = "espera"

// Orden dice a qué pedidos de la cola espera uno nuevo (Despues) y cuáles lo esperan a él (Antes), por id o por tecla.
type Orden struct {
	Despues []string
	Antes   []string
}

func (o Orden) vacio() bool {
	return len(o.Despues) == 0 && len(o.Antes) == 0
}

// enOrden devuelve la cola con cada pedido después de lo que espera. Recorre la cola en su orden y, antes de cada
// pedido, pone lo que espera (en el orden de la cola): lo que va antes sube justo antes de lo que lo espera. Una
// dependencia que no está en la cola no cuenta, y un ciclo (una cola editada a mano) no pierde pedidos.
func enOrden(cola []Pedido) []Pedido {
	porID := map[string]int{}
	for i, p := range cola {
		porID[p.ID] = i
	}
	out := make([]Pedido, 0, len(cola))
	puesto, visitando := map[string]bool{}, map[string]bool{}
	var poner func(i int)
	poner = func(i int) {
		p := cola[i]
		if puesto[p.ID] || visitando[p.ID] {
			return
		}
		visitando[p.ID] = true
		for _, d := range p.Despues {
			if j, ok := porID[d]; ok {
				poner(j)
			}
		}
		puesto[p.ID] = true
		out = append(out, p)
	}
	for i := range cola {
		poner(i)
	}
	return out
}

// esperaA devuelve las teclas (o #id) de lo que el pedido espera y todavía no está hecho.
func esperaA(p Pedido, cola []Pedido) []string {
	var faltan []string
	for _, d := range p.Despues {
		for _, q := range cola {
			if q.ID == d && q.Estado != EstadoHecho {
				faltan = append(faltan, nombreDe(q))
			}
		}
	}
	return faltan
}

func nombreDe(p Pedido) string {
	if p.Tecla != "" {
		return p.Tecla
	}
	return "#" + p.ID
}

func errEspera(id string, faltan []string) error {
	return fmt.Errorf("el pedido %s espera a [%s]: ejecuta eso primero", id, strings.Join(faltan, "] ["))
}

// esperados son los ids que espera algún pedido abierto: un hecho esperado no se va de la cola, así lo que lo espera
// sabe que puede correr.
func esperados(cola []Pedido) map[string]bool {
	out := map[string]bool{}
	for _, p := range cola {
		if !p.Abierto() {
			continue
		}
		for _, d := range p.Despues {
			out[d] = true
		}
	}
	return out
}

// resolver convierte una referencia (id o tecla de un pedido abierto) en el id del pedido.
func resolver(cola []Pedido, ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	for i, p := range cola {
		if p.ID == ref {
			return i, nil
		}
	}
	for i, p := range cola {
		if p.Abierto() && p.Tecla == ref {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no hay un pedido '%s' en la cola (usa su id o su tecla: qdd consola pedidos)", ref)
}

// ordenar agrega a la cola las dependencias de orden del pedido en la posición i. Lo que va Antes tiene que poder
// esperar: no puede estar ejecutándose ni hecho. Si queda un ciclo, se rechaza.
func ordenar(cola []Pedido, i int, orden Orden) error {
	for _, ref := range orden.Despues {
		j, err := resolver(cola, ref)
		if err != nil {
			return err
		}
		if j == i {
			return fmt.Errorf("el pedido %s no puede esperarse a sí mismo", cola[i].ID)
		}
		cola[i].Despues = agregarUno(cola[i].Despues, cola[j].ID)
	}
	for _, ref := range orden.Antes {
		j, err := resolver(cola, ref)
		if err != nil {
			return err
		}
		if j == i {
			return fmt.Errorf("el pedido %s no puede ir antes de sí mismo", cola[i].ID)
		}
		if !cola[j].Ejecutable() {
			return fmt.Errorf("no puede ir antes del pedido %s: %v", cola[j].ID, errNoEjecutable(cola[j].ID, cola[j].Estado))
		}
		cola[j].Despues = agregarUno(cola[j].Despues, cola[i].ID)
	}
	if ciclo(cola) {
		return fmt.Errorf("ese orden deja pedidos esperándose entre sí: revisa --despues y --antes")
	}
	return nil
}

func agregarUno(lista []string, id string) []string {
	for _, x := range lista {
		if x == id {
			return lista
		}
	}
	return append(append([]string{}, lista...), id)
}

// ciclo dice si algún pedido termina esperándose a sí mismo.
func ciclo(cola []Pedido) bool {
	porID := map[string]Pedido{}
	for _, p := range cola {
		porID[p.ID] = p
	}
	estado := map[string]int{} // 1 visitando, 2 listo
	var visitar func(id string) bool
	visitar = func(id string) bool {
		if estado[id] == 1 {
			return true
		}
		if estado[id] == 2 {
			return false
		}
		estado[id] = 1
		for _, d := range porID[id].Despues {
			if _, ok := porID[d]; ok && visitar(d) {
				return true
			}
		}
		estado[id] = 2
		return false
	}
	for _, p := range cola {
		if visitar(p.ID) {
			return true
		}
	}
	return false
}

// sinDependencia quita id de lo que esperan los demás: lo que se descarta ya no bloquea a nadie.
func sinDependencia(cola []Pedido, id string) []Pedido {
	for i := range cola {
		resto := cola[i].Despues[:0:0]
		for _, d := range cola[i].Despues {
			if d != id {
				resto = append(resto, d)
			}
		}
		cola[i].Despues = resto
		if len(resto) == 0 {
			cola[i].Despues = nil
		}
	}
	return cola
}
