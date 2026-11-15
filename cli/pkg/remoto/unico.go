package remoto

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Rule: Prohibido usar 'else'
// Un solo qdd remoto correr por QDD_HOME (MEJ-079): dos en el mismo tópico publican dos veces el estado, duplican las
// respuestas y se pisan el seq. El candado es del sistema operativo (flock, LockFileEx) sobre
// <QDD_HOME>/remoto/correr.lock y lo suelta la muerte del proceso: un correr.pid que quedó o un pid reusado no
// bloquean, y dos que arrancan a la vez no pasan los dos. Es por QDD_HOME y no por máquina: el PC de muestra (otro
// QDD_HOME) corre junto al principal.

// ErrYaCorre: otro qdd remoto correr tiene el candado de este QDD_HOME.
var ErrYaCorre = errors.New("qdd remoto ya corre con este QDD_HOME")

// errOcupado: el candado lo tiene otro (lo devuelve bloquearArchivo de cada sistema).
var errOcupado = errors.New("candado ocupado")

// UnaSolaVez toma el candado de este QDD_HOME y anota el pid en correr.pid; devuelve con qué soltarlo.
func UnaSolaVez() (func(), error) {
	dir, err := DirRemoto()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "correr.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := bloquearArchivo(f); err != nil {
		f.Close()
		return nil, errDeCandado(err, dir)
	}
	ruta := filepath.Join(dir, "correr.pid")
	pid := strconv.Itoa(os.Getpid())
	if err := os.WriteFile(ruta, []byte(pid), 0o600); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		if b, err := os.ReadFile(ruta); err == nil && strings.TrimSpace(string(b)) == pid {
			os.Remove(ruta)
		}
		f.Close() // cerrar el archivo suelta el candado
	}, nil
}

// errDeCandado dice quién tiene el candado si está ocupado.
func errDeCandado(err error, dir string) error {
	if !errors.Is(err, errOcupado) {
		return fmt.Errorf("no se pudo tomar el candado de qdd remoto: %w", err)
	}
	pid := "desconocido"
	if b, err := os.ReadFile(filepath.Join(dir, "correr.pid")); err == nil && strings.TrimSpace(string(b)) != "" {
		pid = strings.TrimSpace(string(b))
	}
	return fmt.Errorf("%w (pid %s, %s): ciérralo antes de correr otro", ErrYaCorre, pid, dir)
}
