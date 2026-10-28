package blueprints

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Problema es una falla de integridad o vigencia de los blueprints del proyecto.
type Problema struct {
	Regla       string
	Archivo     string
	Descripcion string
}

// Check revisa los blueprints del lockfile y sus aplicaciones:
//   - BP-01-INVALID: un blueprint, el lockfile o un registro de aplicación no se
//     puede leer, no es válido o no se pudo obtener de su repositorio.
//   - BP-02-ALTERED: lo publicado en el repositorio no coincide con la huella o
//     la versión del lockfile (alguien lo editó a mano).
//   - BP-03-OUTDATED: el módulo se generó con una versión del blueprint distinta
//     de la importada; hay que volver a aplicarlo.
//   - BP-04-MISSING: un módulo se generó desde un blueprint que ya no está importado.
//   - BP-05-LEGACY: queda una copia del esquema anterior dentro del proyecto.
//   - BP-06-BEHIND: la organización publicó una versión más nueva del blueprint.
func Check(root string) []Problema {
	var out []Problema
	lock, err := LoadLock(root)
	if err != nil {
		out = append(out, Problema{"BP-01-INVALID", LockFile, err.Error()})
	}
	versiones := map[string]string{}
	for _, e := range lock.Blueprints {
		b, err := load(e)
		if err != nil {
			out = append(out, problemaDeCarga(e, err))
			continue
		}
		versiones[e.ID] = b.Version
		out = append(out, checkBlueprint(b, e)...)
		out = append(out, checkVigencia(e)...)
	}
	out = append(out, checkLegado(root, lock)...)
	apps, errs := LoadApplications(root)
	for _, err := range errs {
		out = append(out, Problema{"BP-01-INVALID", ApplicationsDir, err.Error()})
	}
	for _, a := range apps {
		out = append(out, checkApplication(root, a, versiones)...)
	}
	return out
}

func problemaDeCarga(e LockEntry, err error) Problema {
	if errors.Is(err, errHuella) {
		return Problema{"BP-02-ALTERED", LockFile, fmt.Sprintf("El blueprint '%s' del lockfile no coincide con lo publicado en %s (commit %.12s): el lockfile se editó a mano. Lo administra QDD; vuelve a importarlo: qdd blueprint add %s --ref %s", e.ID, e.Origen, e.Commit, e.ID, firstNonEmpty(e.Ref, "<versión>"))}
	}
	return Problema{"BP-01-INVALID", LockFile, fmt.Sprintf("No se pudo obtener el blueprint '%s' desde %s: %v. En el CI, el job necesita acceso de solo lectura a ese repositorio (deploy key o token).", e.ID, e.Origen, err)}
}

func checkBlueprint(b Blueprint, e LockEntry) []Problema {
	if problems := b.Validate(); len(problems) > 0 {
		return []Problema{{"BP-01-INVALID", LockFile, fmt.Sprintf("Blueprint '%s' inválido: %s", e.ID, strings.Join(problems, "; "))}}
	}
	if b.ID != e.ID || b.Version != e.Version {
		return []Problema{{"BP-02-ALTERED", LockFile, fmt.Sprintf("El lockfile dice %s@%s, pero lo publicado es %s: el lockfile se editó a mano. Vuelve a importarlo: qdd blueprint add %s", e.ID, e.Version, b.Ref(), e.ID)}}
	}
	return nil
}

// checkVigencia compara con la última versión que el espejo central conoce
// (sin red: el espejo se actualiza al importar y con 'qdd blueprint sync').
func checkVigencia(e LockEntry) []Problema {
	mirror, err := mirrorPath(e.Origen)
	if err != nil || !exists(mirror) {
		return nil
	}
	tags, err := versionTags(mirror)
	if err != nil || len(tags) == 0 || tags[0] == e.Ref {
		return nil
	}
	disponible := versionEn(mirror, tags[0], e.Subdir)
	if disponible == "" || compararVersion(disponible, e.Version) <= 0 {
		return nil
	}
	return []Problema{{"BP-06-BEHIND", LockFile, fmt.Sprintf("La organización publicó %s@%s (%s) y el proyecto usa %s. Actualízalo: qdd blueprint add %s (y vuelve a aplicarlo a sus módulos con qdd blueprint apply).", e.ID, disponible, tags[0], e.Version, e.ID)}}
}

func checkApplication(root string, a Aplicacion, versiones map[string]string) []Problema {
	rel := relTo(root, a.File)
	version, ok := versiones[a.Blueprint]
	if !ok {
		return []Problema{{"BP-04-MISSING", rel, fmt.Sprintf("El módulo '%s' se generó desde el blueprint '%s', que ya no está importado.", a.Modulo, a.Blueprint)}}
	}
	if version == a.Version {
		return nil
	}
	return []Problema{{"BP-03-OUTDATED", rel, fmt.Sprintf("El módulo '%s' se generó con %s@%s, pero el importado es %s@%s. Vuelve a aplicarlo: qdd blueprint apply %s --modulo %s", a.Modulo, a.Blueprint, a.Version, a.Blueprint, version, a.Blueprint, a.Modulo)}}
}

// Estandares resume los blueprints importados y las fichas generadas con
// cada uno, para el contexto que recibe una IA.
func Estandares(root string) []modules.Estandar {
	bps, _ := List(root)
	apps, _ := LoadApplications(root)
	var out []modules.Estandar
	for _, b := range bps {
		e := modules.Estandar{ID: b.ID, Nombre: b.Nombre, Version: b.Version, Rutas: b.Ficha.Rutas}
		for _, a := range apps {
			if a.Blueprint == b.ID {
				e.Modulos = append(e.Modulos, a.Modulo)
			}
		}
		out = append(out, e)
	}
	return out
}

func relTo(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
}
