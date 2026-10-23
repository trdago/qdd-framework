package blueprints

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Problema es una falla de integridad o vigencia de los blueprints del proyecto.
type Problema struct {
	Regla       string
	Archivo     string
	Descripcion string
}

// Check revisa los blueprints importados y sus aplicaciones:
//   - BP-01-INVALID: un blueprint importado o su registro no se puede leer o no es válido.
//   - BP-02-ALTERED: la copia del proyecto no coincide con la huella del lockfile
//     (alguien la editó a mano o no pasó por 'qdd blueprint add').
//   - BP-03-OUTDATED: el módulo se generó con una versión del blueprint distinta
//     de la importada; hay que volver a aplicarlo.
//   - BP-04-MISSING: un módulo se generó desde un blueprint que ya no está importado.
func Check(root string) []Problema {
	var out []Problema
	lock, err := LoadLock(root)
	if err != nil {
		out = append(out, Problema{"BP-01-INVALID", LockFile, err.Error()})
	}
	bps, errs := List(root)
	for _, err := range errs {
		out = append(out, Problema{"BP-01-INVALID", Dir, err.Error()})
	}
	versiones := map[string]string{}
	for _, b := range bps {
		versiones[b.ID] = b.Version
		out = append(out, checkBlueprint(root, b, lock)...)
	}
	apps, errs := LoadApplications(root)
	for _, err := range errs {
		out = append(out, Problema{"BP-01-INVALID", ApplicationsDir, err.Error()})
	}
	for _, a := range apps {
		out = append(out, checkApplication(root, a, versiones)...)
	}
	return out
}

func checkBlueprint(root string, b Blueprint, lock Lock) []Problema {
	rel := relTo(root, filepath.Join(b.Dir, ManifestName))
	if problems := b.Validate(); len(problems) > 0 {
		return []Problema{{"BP-01-INVALID", rel, "Blueprint inválido: " + strings.Join(problems, "; ")}}
	}
	entry, ok := lock.Entry(b.ID)
	if !ok {
		return []Problema{{"BP-02-ALTERED", rel, fmt.Sprintf("El blueprint '%s' no figura en %s: impórtalo con 'qdd blueprint add <origen>' en vez de copiarlo a mano.", b.ID, LockFile)}}
	}
	sum, err := HashDir(b.Dir)
	if err != nil || sum != entry.SHA256 || entry.Version != b.Version {
		return []Problema{{"BP-02-ALTERED", rel, fmt.Sprintf("La copia del blueprint '%s' no coincide con la que se importó desde %s. Los cambios se hacen en el repositorio del blueprint y se traen con 'qdd blueprint add'.", b.ID, entry.Origen)}}
	}
	return nil
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

func relTo(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
}
