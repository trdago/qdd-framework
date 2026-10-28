package blueprints

import (
	"fmt"
	"os"
	"path/filepath"
)

// Pendiente es una copia del esquema anterior que QDD no puede migrar sola
// sin perder información; la termina una IA siguiendo su Motivo.
type Pendiente struct {
	ID     string
	Motivo string
}

// Migracion describe lo que hizo Migrar.
type Migracion struct {
	Migrados   []string
	Pendientes []Pendiente
}

// Hizo indica si esta ejecución migró algo (ocurre una sola vez por proyecto).
func (m Migracion) Hizo() bool {
	return len(m.Migrados) > 0
}

// Migrar convierte un proyecto del esquema anterior (copias en
// .qdd/blueprints/<id>, v1.10–v1.11) al de la carpeta central: cada copia
// íntegra se guarda en el store y se borra del proyecto, y el lockfile queda
// en el formato vigente. Sin esa carpeta no hace nada, así que corre una sola
// vez. Nunca descarta información: una copia editada a mano o fuera del
// lockfile queda como Pendiente (y como BP-05-LEGACY en el gate).
func Migrar(root string) (Migracion, error) {
	legado := filepath.Join(root, Dir)
	entries, err := os.ReadDir(legado)
	if os.IsNotExist(err) {
		return Migracion{}, nil
	}
	if err != nil {
		return Migracion{}, err
	}
	lock, err := LoadLock(root)
	if err != nil {
		return Migracion{}, err
	}
	var m Migracion
	for _, d := range entries {
		if !d.IsDir() {
			continue
		}
		motivo, err := migrarCopia(filepath.Join(legado, d.Name()), d.Name(), lock)
		if err != nil {
			return m, fmt.Errorf("no se pudo migrar el blueprint '%s': %w", d.Name(), err)
		}
		if motivo != "" {
			m.Pendientes = append(m.Pendientes, Pendiente{ID: d.Name(), Motivo: motivo})
			continue
		}
		m.Migrados = append(m.Migrados, d.Name())
	}
	if len(m.Pendientes) == 0 {
		if err := os.RemoveAll(legado); err != nil {
			return m, err
		}
	}
	if !m.Hizo() {
		return m, nil
	}
	if err := saveLock(root, lock); err != nil {
		return m, err
	}
	return m, registrarDesdeProyecto(root)
}

func migrarCopia(dir, id string, lock Lock) (string, error) {
	if motivo := motivoLegado(dir, id, lock); motivo != "" {
		return motivo, nil
	}
	if _, _, err := storePut(dir); err != nil {
		return "", err
	}
	return "", os.RemoveAll(dir)
}

// motivoLegado explica por qué una copia del esquema anterior no se puede
// migrar sola, con los pasos para terminar la migración; vacío si se puede.
func motivoLegado(dir, id string, lock Lock) string {
	copia := filepath.ToSlash(filepath.Join(Dir, id))
	e, ok := lock.Entry(id)
	if !ok {
		return fmt.Sprintf("La copia '%s' no figura en el lockfile (se copió a mano). Si el repositorio de la organización ya tiene el blueprint, impórtalo con 'qdd blueprint add %s' y borra la copia; si no, apórtalo: 'qdd blueprint contribute %s', copia la carpeta a la que indica y 'qdd blueprint propose %s -m \"…\"' (pide confirmación al humano antes de proponer); luego borra '%s'.", copia, id, id, id, copia)
	}
	if !intacto(dir, e.SHA256) {
		return fmt.Sprintf("La copia '%s' tiene cambios hechos dentro del proyecto; QDD no la borra para no perderlos. Llévalos al repositorio de la organización: 'qdd blueprint contribute %s', copia los cambios a la carpeta que indica y 'qdd blueprint propose %s -m \"…\"' (pide confirmación al humano antes de proponer); luego borra '%s'.", copia, id, id, copia)
	}
	return ""
}

// registrarDesdeProyecto lleva al registro central el repositorio que el
// proyecto ya tenía en su .qdd/config.yaml.
func registrarDesdeProyecto(root string) error {
	reg, err := registroDelProyecto(root)
	if err != nil || reg.Origen == "" {
		return err
	}
	return RegistrarOrganizacion(root, reg)
}

// checkLegado reporta las copias del esquema anterior que siguen en el proyecto.
func checkLegado(root string, lock Lock) []Problema {
	legado := filepath.Join(root, Dir)
	entries, err := os.ReadDir(legado)
	if err != nil {
		return nil
	}
	var out []Problema
	for _, d := range entries {
		if !d.IsDir() {
			continue
		}
		motivo := motivoLegado(filepath.Join(legado, d.Name()), d.Name(), lock)
		if motivo == "" {
			motivo = fmt.Sprintf("La copia '%s/%s' es del esquema anterior y aún no se migra a la carpeta central: ejecuta cualquier comando qdd (p. ej. 'qdd blueprint list') y commitea su eliminación.", Dir, d.Name())
		}
		out = append(out, Problema{"BP-05-LEGACY", filepath.ToSlash(filepath.Join(Dir, d.Name())), motivo})
	}
	return out
}
