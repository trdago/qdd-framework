package consola

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9 (FND-081): la cola de pedidos es una por repositorio. doctor encuentra lo que quedó fuera de su
// lugar (la cola de un worktree de qdd 2.7 o anterior, o pedidos que corren en otro repositorio) y --fix lo junta en
// la cola que corresponde, sin ejecutar nada.

const nombreCola = "Cola de pedidos del panel (una por repositorio)"

// DiagnosticarCola revisa la cola de pedidos del proyecto.
func DiagnosticarCola(root string) Diagnostico {
	if n := len(colaSuelta(root)); n > 0 {
		return Diagnostico{Nombre: nombreCola, Detalle: fmt.Sprintf("%d pedido(s) en la cola propia de este worktree (qdd 2.7 o anterior): el panel ya no la lee. 'qdd doctor --fix' los pasa a la cola del repositorio", n)}
	}
	if n := len(ajenos(root)); n > 0 {
		return Diagnostico{Nombre: nombreCola, Detalle: fmt.Sprintf("%d pedido(s) corren en otro repositorio y se ven en el panel equivocado. 'qdd doctor --fix' los pasa a la cola de su repositorio", n)}
	}
	return Diagnostico{Nombre: nombreCola, OK: true}
}

// RepararCola junta la cola suelta del worktree con la del repositorio y pasa cada pedido ajeno a la cola de su
// repositorio. Devuelve lo que hizo.
func RepararCola(root string) ([]string, error) {
	var hecho []string
	if suelta := colaSuelta(root); len(suelta) > 0 {
		vieja, _ := RutaLocalSinRepositorio(root, "pedidos")
		if err := agregar(root, suelta); err != nil {
			return hecho, err
		}
		if err := os.Remove(vieja); err != nil {
			return hecho, err
		}
		hecho = append(hecho, fmt.Sprintf("%d pedido(s) de la cola del worktree pasaron a la del repositorio", len(suelta)))
	}
	porDestino := map[string][]Pedido{}
	for _, p := range ajenos(root) {
		destino := ColaDe(root, p.Carpeta)
		porDestino[destino] = append(porDestino[destino], p)
	}
	for destino, lista := range porDestino {
		if err := agregar(destino, lista); err != nil {
			return hecho, err
		}
		if err := quitar(root, lista); err != nil {
			return hecho, err
		}
		hecho = append(hecho, fmt.Sprintf("%d pedido(s) pasaron a la cola de %s", len(lista), destino))
	}
	return hecho, nil
}

// colaSuelta son los pedidos abiertos de la cola que este worktree tenía con la clave de qdd 2.7 (por su ruta).
func colaSuelta(root string) []Pedido {
	vieja, err := RutaLocalSinRepositorio(root, "pedidos")
	if err != nil {
		return nil
	}
	nueva, err := colaPath(root)
	if err != nil || nueva == vieja {
		return nil
	}
	b, err := os.ReadFile(vieja)
	if err != nil {
		return nil
	}
	var cola []Pedido
	if json.Unmarshal(b, &cola) != nil {
		return nil
	}
	return abiertos(normalizar(cola, time.Now()))
}

// ajenos son los pedidos abiertos de la cola que corren en otro repositorio.
func ajenos(root string) []Pedido {
	var out []Pedido
	for _, p := range abiertos(PedidosA(root, time.Now())) {
		if p.Estado != EstadoEjecutando && ColaDe(root, p.Carpeta) != root {
			out = append(out, p)
		}
	}
	return out
}

func abiertos(cola []Pedido) []Pedido {
	var out []Pedido
	for _, p := range cola {
		if p.Abierto() {
			out = append(out, p)
		}
	}
	return out
}

// agregar suma pedidos a la cola de root con ids y teclas nuevos; las dependencias entre ellos se conservan y las
// que apuntaban a otros se quitan.
func agregar(root string, lista []Pedido) error {
	return conCola(root, time.Now(), func(cola []Pedido) ([]Pedido, error) {
		usadas := map[string]bool{}
		for _, p := range cola {
			if p.Abierto() {
				usadas[p.Tecla] = true
			}
		}
		mayor, _ := strconv.Atoi(siguienteID(cola))
		nuevoID := map[string]string{}
		for i, p := range lista {
			nuevoID[p.ID] = strconv.Itoa(mayor + i)
		}
		for _, p := range lista {
			p.ID, p.Tecla = nuevoID[p.ID], teclaLibre(usadas)
			usadas[p.Tecla] = true
			p.Despues = remapear(p.Despues, nuevoID)
			cola = append(cola, p)
		}
		return cola, nil
	})
}

func remapear(despues []string, nuevoID map[string]string) []string {
	var out []string
	for _, d := range despues {
		if n, ok := nuevoID[d]; ok {
			out = append(out, n)
		}
	}
	return out
}

func quitar(root string, lista []Pedido) error {
	sale := map[string]bool{}
	for _, p := range lista {
		sale[p.ID] = true
	}
	return conCola(root, time.Now(), func(cola []Pedido) ([]Pedido, error) {
		resto := make([]Pedido, 0, len(cola))
		for _, p := range cola {
			if !sale[p.ID] {
				resto = append(resto, p)
			}
		}
		for id := range sale {
			resto = sinDependencia(resto, id)
		}
		return resto, nil
	})
}
