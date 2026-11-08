package rendimiento

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Procesos lee /proc: no lanza ningún programa, así que medir no cambia lo que se mide.
func Procesos() ([]Proceso, error) {
	dirs, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var lista []Proceso
	for _, d := range dirs {
		pid, err := strconv.Atoi(d.Name())
		if err != nil {
			continue
		}
		p, ok := leerStat(pid)
		if !ok {
			continue
		}
		lista = append(lista, p)
	}
	return lista, nil
}

// leerStat: el nombre va entre paréntesis y puede tener espacios; el estado y el padre vienen después del último «)».
func leerStat(pid int) (Proceso, bool) {
	b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return Proceso{}, false
	}
	s := string(b)
	abre, cierra := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
	if abre < 0 || cierra < abre {
		return Proceso{}, false
	}
	campos := strings.Fields(s[cierra+1:])
	if len(campos) < 2 {
		return Proceso{}, false
	}
	ppid, err := strconv.Atoi(campos[1])
	if err != nil {
		return Proceso{}, false
	}
	return Proceso{PID: pid, PPID: ppid, Zombie: campos[0] == "Z", Nombre: s[abre+1 : cierra]}, true
}
