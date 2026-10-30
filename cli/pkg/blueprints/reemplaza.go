package blueprints

import (
	"fmt"
	"path"
	"strings"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'
// RFC 0009: un blueprint puede declarar que reemplaza a otros de su misma fuente (reemplaza: [id…]). El gate avisa
// a quien usa el reemplazado (BP-07-REPLACED) y 'qdd blueprint add <nuevo>' lo migra: saca el viejo del lockfile y
// pasa sus registros de aplicación al nuevo, con el mismo módulo y las mismas respuestas.

func (b Blueprint) validateReemplaza() []string {
	var problems []string
	for _, id := range b.Reemplaza {
		if !idPattern.MatchString(id) {
			problems = append(problems, fmt.Sprintf("reemplaza: '%s' no es un id de blueprint válido", id))
			continue
		}
		if id == b.ID {
			problems = append(problems, "reemplaza: un blueprint no se reemplaza a sí mismo")
		}
	}
	return problems
}

func (b Blueprint) reemplazaA(id string) bool {
	return inList(b.Reemplaza, id)
}

// reemplazoEn busca, en un tag de la fuente, un blueprint que reemplace a id.
func reemplazoEn(repoDir, tag, id string) (Blueprint, bool) {
	listado, err := git(repoDir, "ls-tree", "-r", "--name-only", tag)
	if err != nil {
		return Blueprint{}, false
	}
	for _, archivo := range strings.Split(listado, "\n") {
		if path.Base(archivo) != ManifestName {
			continue
		}
		b, ok := blueprintEn(repoDir, tag, path.Dir(archivo))
		if ok && b.reemplazaA(id) {
			return b, true
		}
	}
	return Blueprint{}, false
}

// problemaReemplazado arma BP-07-REPLACED con los comandos exactos para migrar cada módulo generado con el viejo.
func problemaReemplazado(root string, e LockEntry, sucesor Blueprint, tag string) Problema {
	reaplicar := ""
	apps, _ := LoadApplications(root)
	for _, a := range apps {
		if a.Blueprint == e.ID {
			reaplicar += fmt.Sprintf(" y qdd blueprint apply %s --modulo %s", sucesor.ID, a.Modulo)
		}
	}
	return Problema{"BP-07-REPLACED", LockFile, fmt.Sprintf("%s fue reemplazado por %s en %s (%s). Migra: qdd blueprint add %s%s. El módulo conserva su id y sus respuestas; las preguntas nuevas las respondes al aplicarlo.", e.ID, sucesor.Ref(), e.Origen, tag, sucesor.ID, reaplicar)}
}

// migrarReemplazados saca del lockfile los blueprints que b reemplaza (solo los de su misma fuente) y pasa sus
// registros de aplicación a b. El registro conserva el módulo, las respuestas y la versión vieja, así BP-03 pide
// volver a aplicarlo. Devuelve los ids migrados.
func migrarReemplazados(root string, b Blueprint, origen string) ([]string, error) {
	if len(b.Reemplaza) == 0 {
		return nil, nil
	}
	lock, err := LoadLock(root)
	if err != nil {
		return nil, err
	}
	var migrados []string
	resto := lock.Blueprints[:0:0]
	for _, e := range lock.Blueprints {
		if b.reemplazaA(e.ID) && e.Origen == origen {
			migrados = append(migrados, e.ID)
			continue
		}
		resto = append(resto, e)
	}
	if len(migrados) == 0 {
		return nil, nil
	}
	apps, _ := LoadApplications(root)
	for _, a := range apps {
		if !inList(migrados, a.Blueprint) {
			continue
		}
		a.Blueprint = b.ID
		if _, err := saveApplication(root, a); err != nil {
			return nil, err
		}
	}
	lock.Blueprints = resto
	return migrados, saveLock(root, lock)
}

// FichaDeBlueprint devuelve la ficha de módulo generada por el primero de los blueprints dados que tenga una (en
// orden de preferencia: el nuevo antes que el que reemplazó).
func FichaDeBlueprint(root string, ids ...string) (modules.Module, bool) {
	mods, _ := modules.Load(root)
	for _, id := range ids {
		for _, m := range mods {
			if strings.SplitN(m.Blueprint, "@", 2)[0] == id {
				return m, true
			}
		}
	}
	return modules.Module{}, false
}
