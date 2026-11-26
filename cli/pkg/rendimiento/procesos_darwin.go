package rendimiento

import (
	"os/exec"
	"strconv"
	"strings"
)

// Procesos en macOS: ps es parte del sistema. Su propio proceso sale en la lista, pero no baja de lo que se mide.
func Procesos() ([]Proceso, error) {
	out, err := exec.Command("ps", "-A", "-o", "pid=,ppid=,stat=,comm=").Output()
	if err != nil {
		return nil, err
	}
	var lista []Proceso
	for _, linea := range strings.Split(string(out), "\n") {
		campos := strings.Fields(linea)
		if len(campos) < 4 {
			continue
		}
		pid, err1 := strconv.Atoi(campos[0])
		ppid, err2 := strconv.Atoi(campos[1])
		if err1 != nil || err2 != nil {
			continue
		}
		lista = append(lista, Proceso{PID: pid, PPID: ppid, Zombie: strings.HasPrefix(campos[2], "Z"), Nombre: strings.Join(campos[3:], " ")})
	}
	return lista, nil
}
