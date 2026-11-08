package rendimiento

// Proceso es una fila de la tabla de procesos: lo justo para saber de quién es hijo y si quedó zombie.
type Proceso struct {
	PID    int    `json:"pid"`
	PPID   int    `json:"ppid"`
	Zombie bool   `json:"zombie,omitempty"`
	Nombre string `json:"nombre"`
}

// descendientes son los procesos de la lista que bajan de raiz (hijos, nietos…), sin raiz.
func descendientes(lista []Proceso, raiz int) []Proceso {
	hijos := map[int][]Proceso{}
	for _, p := range lista {
		hijos[p.PPID] = append(hijos[p.PPID], p)
	}
	var salida []Proceso
	pendientes := []int{raiz}
	vistos := map[int]bool{raiz: true}
	for len(pendientes) > 0 {
		pid := pendientes[0]
		pendientes = pendientes[1:]
		for _, h := range hijos[pid] {
			if vistos[h.PID] {
				continue
			}
			vistos[h.PID] = true
			salida = append(salida, h)
			pendientes = append(pendientes, h.PID)
		}
	}
	return salida
}
