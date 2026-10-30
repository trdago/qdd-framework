package estado

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Rule: Prohibido usar 'else'
// Tarea en curso (RFC 0006 §3.13, blueprint estado-por-tareas): el agente declara la tarea que trabaja y cada señal de
// vida (una herramienta del agente, un subagente o un comando externo) la hace latir. Es actividad, no estado: vive en
// RutaLocal(root, CarpetaActividad), fuera del repositorio, y estado.yaml no cambia. No va en .qdd/working: ese es el
// archivo con el comando qdd en curso (FND-079). Sin pulso durante VidaActividad, la tarea
// deja de estar en curso, así un agente que se cae no la deja encendida.

const (
	CarpetaActividad = "actividad"
	VidaActividad = 120 * time.Second

	FuenteAgente    = "agente"
	FuenteSubagente = "subagente"
	FuenteComando   = "comando"
)

// Actividad es la señal de vida de una tarea; las horas van en RFC 3339.
type Actividad struct {
	Desde  string `json:"desde"`
	Pulso  string `json:"pulso"`
	Fuente string `json:"fuente"`
	Quien  string `json:"quien,omitempty"`
}

// Actividades es el archivo: Ultima es la última tarea activada, la que late con un pulso sin id.
type Actividades struct {
	Formato int                  `json:"formato"`
	Ultima  string               `json:"ultima,omitempty"`
	Tareas  map[string]Actividad `json:"tareas"`
}

// LeerActividad devuelve la actividad del proyecto; sin archivo legible no hay tareas en curso.
func LeerActividad(root string) Actividades {
	a := Actividades{Formato: 1, Tareas: map[string]Actividad{}}
	ruta, err := RutaLocal(root, CarpetaActividad)
	if err != nil {
		return a
	}
	data, err := os.ReadFile(ruta)
	if err != nil {
		return a
	}
	var leida Actividades
	if json.Unmarshal(data, &leida) != nil || leida.Tareas == nil {
		return a
	}
	leida.Formato = 1
	return leida
}

// viva dice si el último pulso es de hace menos de VidaActividad.
func (a Actividad) viva(now time.Time) bool {
	t, err := time.Parse(time.RFC3339, a.Pulso)
	return err == nil && now.Sub(t) < VidaActividad
}

// EnCurso devuelve las tareas con un pulso de hace menos de VidaActividad.
func EnCurso(a Actividades, now time.Time) map[string]Actividad {
	en := map[string]Actividad{}
	for id, act := range a.Tareas {
		if act.viva(now) {
			en[id] = act
		}
	}
	return en
}

// tareaAbierta comprueba que el id sea una tarea abierta de estado.yaml.
func tareaAbierta(root, id string) error {
	e, err := Load(root)
	if err != nil {
		return err
	}
	i := e.buscar(id)
	if i < 0 {
		return fmt.Errorf("no hay una tarea %q", id)
	}
	if e.Frentes[i].Estado != EstadoAbierto {
		return fmt.Errorf("la tarea %q está %s", id, e.Frentes[i].Estado)
	}
	return nil
}

// Activar deja la tarea en curso: la declara el agente al empezar a trabajarla.
func Activar(root, id, quien string, now time.Time) error {
	if err := tareaAbierta(root, id); err != nil {
		return err
	}
	a := LeerActividad(root)
	hora := now.Format(time.RFC3339)
	a.Tareas[id] = Actividad{Desde: hora, Pulso: hora, Fuente: FuenteAgente, Quien: quien}
	a.Ultima = id
	return guardarActividad(root, a, now)
}

// Terminar apaga la tarea; si no estaba en curso no hace nada.
func Terminar(root, id string) error {
	a := LeerActividad(root)
	if _, ok := a.Tareas[id]; !ok {
		return nil
	}
	delete(a.Tareas, id)
	if a.Ultima == id {
		a.Ultima = ""
	}
	return guardarActividad(root, a, time.Time{})
}

// Pulsar deja una señal de vida y devuelve la tarea que latió. Sin id late la última activada si sigue en curso (y si
// no, no hace nada: el mod pulsa en cada herramienta sin saber cuál es). Con id, la tarea tiene que estar abierta y el
// pulso la enciende si estaba apagada (un comando externo que trabaja en ella).
func Pulsar(root, id, fuente string, now time.Time) (string, error) {
	if fuente != FuenteAgente && fuente != FuenteSubagente && fuente != FuenteComando {
		return "", fmt.Errorf("fuente %q: debe ser %s, %s o %s", fuente, FuenteAgente, FuenteSubagente, FuenteComando)
	}
	a := LeerActividad(root)
	hora := now.Format(time.RFC3339)
	if id == "" {
		act, ok := a.Tareas[a.Ultima]
		if !ok || !act.viva(now) {
			return "", nil
		}
		act.Pulso, act.Fuente = hora, fuente
		a.Tareas[a.Ultima] = act
		return a.Ultima, guardarActividad(root, a, now)
	}
	if err := tareaAbierta(root, id); err != nil {
		return "", err
	}
	act, ok := a.Tareas[id]
	if !ok || !act.viva(now) {
		act = Actividad{Desde: hora}
	}
	act.Pulso, act.Fuente = hora, fuente
	a.Tareas[id] = act
	return id, guardarActividad(root, a, now)
}

// guardarActividad limpia las tareas vencidas (con now cero no limpia) y escribe de forma atómica.
func guardarActividad(root string, a Actividades, now time.Time) error {
	if !now.IsZero() {
		for id, act := range a.Tareas {
			if !act.viva(now) {
				delete(a.Tareas, id)
			}
		}
	}
	if _, ok := a.Tareas[a.Ultima]; !ok {
		a.Ultima = ""
	}
	a.Formato = 1
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	dest, err := RutaLocal(root, CarpetaActividad)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".actividad-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dest)
}
