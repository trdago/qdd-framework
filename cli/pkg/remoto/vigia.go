package remoto

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): un solo watcher para todos los proyectos. Mira las carpetas de las que sale el panel: el estado y
// las fichas de cada proyecto (.qdd/project y .qdd/project/modules), y lo local de QDD (la cola de pedidos, la
// actividad, el ciclo y el registro). Si fsnotify no está, el sondeo del bucle alcanza.

type vigia struct {
	w       *fsnotify.Watcher
	dirs    map[string]bool
	cambios chan struct{}
}

func nuevaVigia() *vigia {
	v := &vigia{dirs: map[string]bool{}, cambios: make(chan struct{}, 1)}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return v
	}
	v.w = w
	return v
}

// Cambios avisa (sin bloquear, juntando) que algo cambió.
func (v *vigia) Cambios() <-chan struct{} { return v.cambios }

func (v *vigia) correr(ctx context.Context) {
	if v.w == nil {
		return
	}
	defer v.w.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-v.w.Errors:
			if !ok {
				return
			}
		case e, ok := <-v.w.Events:
			if !ok {
				return
			}
			if !relevante(e) {
				continue
			}
			select {
			case v.cambios <- struct{}{}:
			default:
			}
		}
	}
}

// relevante descarta los bloqueos y temporales de las escrituras atómicas: el rename final ya avisa.
func relevante(e fsnotify.Event) bool {
	base := filepath.Base(e.Name)
	return !strings.HasSuffix(base, ".lock") && !strings.HasSuffix(base, ".tmp")
}

// vigilar deja mirando exactamente estas carpetas (las que existen).
func (v *vigia) vigilar(dirs []string) {
	if v.w == nil {
		return
	}
	quiero := map[string]bool{}
	for _, d := range dirs {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			quiero[d] = true
		}
	}
	for d := range v.dirs {
		if !quiero[d] {
			_ = v.w.Remove(d)
			delete(v.dirs, d)
		}
	}
	for d := range quiero {
		if v.dirs[d] {
			continue
		}
		if v.w.Add(d) == nil {
			v.dirs[d] = true
		}
	}
}

// carpetasDe son las carpetas de las que sale el panel de los proyectos del registro.
func carpetasDe(reg registro.Registro) []string {
	var dirs []string
	if home, err := blueprints.Home(); err == nil {
		dirs = append(dirs, home)
		for _, c := range []string{"pedidos", "preguntas", estado.CarpetaActividad, "ciclo"} {
			dirs = append(dirs, filepath.Join(home, "claude", c))
		}
	}
	for _, p := range reg.Proyectos {
		dirs = append(dirs, filepath.Join(p.Ruta, ".qdd", "project"), filepath.Join(p.Ruta, ".qdd", "project", "modules"))
	}
	return dirs
}
