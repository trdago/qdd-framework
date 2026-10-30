package blueprints

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Diagnósticos de los blueprints que muestra 'qdd doctor' (grupo 4).
const (
	DiagLegado   = "Sin copias de blueprints dentro del proyecto"
	DiagCentral  = "Blueprints del lockfile en la carpeta central de QDD"
	DiagAcceso   = "Acceso al repositorio de blueprints de la organización"
	DiagVigencia = "Blueprints al día con la organización"
	DiagModulos  = "Módulos generados con la versión importada de su blueprint"
)

// Diagnostico es una verificación de salud de los blueprints del proyecto.
type Diagnostico struct {
	Nombre  string
	OK      bool
	Detalle string
}

// Diagnosticar revisa la salud de los blueprints del proyecto para 'qdd
// doctor'. Solo usa la red para comprobar el acceso al repositorio de un
// proyecto que tiene blueprints; uno sin blueprints no sale de la máquina.
func Diagnosticar(root string) []Diagnostico {
	lock, err := LoadLock(root)
	faltan := faltantes(lock)
	return []Diagnostico{
		diagLegado(root, lock),
		diagCentral(lock, err, faltan),
		diagAcceso(lock, faltan),
		diagVigencia(lock),
		diagModulos(root, lock),
	}
}

// faltantes lista los blueprints del lockfile que no están íntegros en la
// carpeta central de esta máquina.
func faltantes(lock Lock) []string {
	var out []string
	for _, e := range lock.Blueprints {
		dest, err := storePath(e.SHA256)
		if err != nil || !intacto(dest, e.SHA256) {
			out = append(out, e.ID)
		}
	}
	return out
}

func diagLegado(root string, lock Lock) Diagnostico {
	probs := checkLegado(root, lock)
	if len(probs) == 0 {
		return Diagnostico{DiagLegado, true, "el proyecto solo versiona " + LockFile}
	}
	var detalle []string
	for _, p := range probs {
		detalle = append(detalle, p.Descripcion)
	}
	return Diagnostico{DiagLegado, false, strings.Join(detalle, " ")}
}

func diagCentral(lock Lock, err error, faltan []string) Diagnostico {
	if err != nil {
		return Diagnostico{DiagCentral, false, err.Error()}
	}
	if len(lock.Blueprints) == 0 {
		return Diagnostico{DiagCentral, true, "sin blueprints de la organización"}
	}
	if len(faltan) == 0 {
		return Diagnostico{DiagCentral, true, fmt.Sprintf("%d blueprint(s) íntegros", len(lock.Blueprints))}
	}
	return Diagnostico{DiagCentral, false, fmt.Sprintf("faltan o están alterados en esta máquina: %s. 'qdd doctor --fix' los reconstruye desde el commit del lockfile.", strings.Join(faltan, ", "))}
}

func diagAcceso(lock Lock, faltan []string) Diagnostico {
	repos := origenesDe(lock)
	if len(repos) == 0 {
		return Diagnostico{DiagAcceso, true, "sin blueprints: no hay repositorio que verificar"}
	}
	for _, repo := range repos {
		_, err := git("", "ls-remote", "--quiet", "--", repo, "HEAD")
		if err == nil {
			continue
		}
		if len(faltan) == 0 {
			return Diagnostico{DiagAcceso, true, fmt.Sprintf("sin conexión con %s; la carpeta central tiene todo lo que fija el lockfile", repo)}
		}
		return Diagnostico{DiagAcceso, false, fmt.Sprintf("no hay acceso a %s (%v). Esta máquina (o el CI) necesita acceso de solo lectura al repositorio: tu llave SSH o credencial git, o una deploy key / token en el CI.", repo, err)}
	}
	return Diagnostico{DiagAcceso, true, "acceso a " + strings.Join(repos, ", ")}
}

func diagVigencia(lock Lock) Diagnostico {
	if len(lock.Blueprints) == 0 {
		return Diagnostico{DiagVigencia, true, "sin blueprints de la organización"}
	}
	var detalle []string
	for _, e := range lock.Blueprints {
		for _, p := range checkVigencia(e) {
			detalle = append(detalle, p.Descripcion)
		}
	}
	if len(detalle) > 0 {
		return Diagnostico{DiagVigencia, false, strings.Join(detalle, " ")}
	}
	return Diagnostico{DiagVigencia, true, "al día con lo último que conoce la carpeta central ('qdd doctor --fix' o 'qdd blueprint sync' la actualizan)"}
}

func origenesDe(lock Lock) []string {
	var out []string
	seen := map[string]bool{}
	for _, e := range lock.Blueprints {
		if seen[e.Origen] {
			continue
		}
		seen[e.Origen] = true
		out = append(out, e.Origen)
	}
	return out
}

// desfasados devuelve los problemas BP-03/BP-04 de los módulos generados con
// blueprints, comparando con las versiones del lockfile (sin red).
func desfasados(root string, lock Lock) []Problema {
	versiones := map[string]string{}
	for _, e := range lock.Blueprints {
		versiones[e.ID] = e.Version
	}
	apps, _ := LoadApplications(root)
	var out []Problema
	for _, a := range apps {
		out = append(out, checkApplication(root, a, versiones)...)
	}
	return out
}

func diagModulos(root string, lock Lock) Diagnostico {
	probs := desfasados(root, lock)
	if len(probs) == 0 {
		return Diagnostico{DiagModulos, true, "cada módulo usa la versión importada de su blueprint"}
	}
	var detalle []string
	for _, p := range probs {
		detalle = append(detalle, p.Descripcion)
	}
	return Diagnostico{DiagModulos, false, strings.Join(detalle, " ") + " 'qdd doctor --fix' los vuelve a aplicar con las respuestas registradas; la ficha vuelve a borrador para que la apruebes."}
}

// Reparar hace lo que 'qdd doctor --fix' sabe resolver, de forma determinista:
//   - lockfile ilegible: lo restaura desde el último commit;
//   - copias del esquema anterior: las migra (nunca borra una editada);
//   - blueprint ausente en esta máquina: lo reconstruye desde el lockfile;
//   - lockfile editado a mano: lo rehace desde la versión publicada que fija;
//   - módulo cuyo blueprint ya no está importado: lo importa del registro;
//   - espejo: trae lo publicado por la organización;
//   - módulo generado con otra versión: lo re-aplica con sus respuestas.
//
// Lo que necesita una decisión humana (preguntas nuevas, actualizar a otra
// versión, llevar cambios al repositorio de la organización) no lo decide:
// dice exactamente qué hacer. Devuelve lo que hizo y lo que queda pendiente.
func Reparar(root string) []string {
	acciones := restaurarLock(root)
	m, err := Migrar(root)
	if err != nil {
		acciones = append(acciones, "no se pudo migrar: "+err.Error())
	}
	if m.Hizo() {
		acciones = append(acciones, fmt.Sprintf("migrados a la carpeta central: %s (commitea la eliminación de %s/, %s y .qdd/state.json)", strings.Join(m.Migrados, ", "), Dir, LockFile))
	}
	for _, p := range m.Pendientes {
		acciones = append(acciones, fmt.Sprintf("pendiente %s: %s", p.ID, p.Motivo))
	}
	lock, err := LoadLock(root)
	if err != nil {
		return append(acciones, err.Error())
	}
	for _, e := range lock.Blueprints {
		acciones = append(acciones, reconstruir(root, e)...)
	}
	acciones = append(acciones, importarFaltantes(root)...)
	lock, _ = LoadLock(root)
	for _, repo := range origenesDe(lock) {
		if _, err := ensureMirror(repo, true); err != nil {
			acciones = append(acciones, "no se pudo sincronizar: "+err.Error())
			continue
		}
		acciones = append(acciones, "sincronizado "+repo)
	}
	return append(acciones, reaplicar(root)...)
}

// restaurarLock devuelve el lockfile a su último commit si quedó ilegible.
func restaurarLock(root string) []string {
	if _, err := LoadLock(root); err == nil {
		return nil
	}
	data, err := gitRaw(root, "show", "HEAD:"+LockFile)
	if err != nil {
		return []string{fmt.Sprintf("%s está ilegible y no hay una versión commiteada: vuelve a importar los blueprints con 'qdd blueprint add <id>'", LockFile)}
	}
	var l Lock
	if err := yaml.Unmarshal(data, &l); err != nil {
		return []string{fmt.Sprintf("%s está ilegible también en el último commit: vuelve a importar los blueprints con 'qdd blueprint add <id>'", LockFile)}
	}
	if err := os.WriteFile(filepath.Join(root, LockFile), data, 0644); err != nil {
		return []string{err.Error()}
	}
	return []string{"restaurado " + LockFile + " desde el último commit"}
}

func reconstruir(root string, e LockEntry) []string {
	dest, err := storePath(e.SHA256)
	if err == nil && intacto(dest, e.SHA256) {
		return nil
	}
	_, err = materialize(e)
	if errors.Is(err, errHuella) || errors.Is(err, errSinHuella) {
		return reimportar(root, e)
	}
	if err != nil {
		return []string{fmt.Sprintf("no se pudo reconstruir %s: %v", e.ID, err)}
	}
	return []string{fmt.Sprintf("reconstruido %s@%s", e.ID, e.Version)}
}

// reimportar rehace la entrada del lockfile desde la versión publicada que
// fija (o la última, si no fija un tag).
func reimportar(root string, e LockEntry) []string {
	origen := origenDeEntrada(e)
	ref := ""
	if tagPattern.MatchString(e.Ref) {
		ref = e.Ref
	}
	entries, problems := Add(AddInput{Root: root, Origen: origen, Ref: ref})
	if len(problems) > 0 {
		return []string{fmt.Sprintf("no se pudo rehacer %s desde %s: %s", e.ID, e.Origen, strings.Join(problems, "; "))}
	}
	return []string{fmt.Sprintf("lockfile rehecho para %s@%s (%s)", e.ID, entries[0].Version, entries[0].Ref)}
}

// importarFaltantes importa del registro los blueprints de módulos generados
// que ya no están en el lockfile (BP-04).
func importarFaltantes(root string) []string {
	lock, _ := LoadLock(root)
	var ids []string
	for _, p := range desfasados(root, lock) {
		if p.Regla != "BP-04-MISSING" {
			continue
		}
		app, err := LoadApplication(root, strings.TrimSuffix(filepath.Base(p.Archivo), ".yaml"))
		if err == nil && !inList(ids, app.Blueprint) {
			ids = append(ids, app.Blueprint)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	res, problems := AddManaged(AddInput{Root: root, Blueprints: ids})
	if len(problems) > 0 && len(res.Entries) == 0 {
		return []string{fmt.Sprintf("no se pudo volver a importar %s: %s", strings.Join(ids, ", "), strings.Join(problems, "; "))}
	}
	return []string{"vuelto a importar: " + strings.Join(ids, ", ")}
}

// reaplicar vuelve a aplicar, con sus respuestas registradas, los módulos
// generados con otra versión de su blueprint (BP-03). Si la versión nueva
// trae preguntas, no las responde: dice cuáles y cómo responderlas.
func reaplicar(root string) []string {
	lock, _ := LoadLock(root)
	var out []string
	for _, p := range desfasados(root, lock) {
		if p.Regla != "BP-03-OUTDATED" {
			continue
		}
		app, err := LoadApplication(root, strings.TrimSuffix(filepath.Base(p.Archivo), ".yaml"))
		if err != nil {
			out = append(out, err.Error())
			continue
		}
		out = append(out, reaplicarModulo(root, app))
	}
	return out
}

func reaplicarModulo(root string, app Aplicacion) string {
	res, problems := Apply(ApplyInput{Root: root, Blueprint: app.Blueprint, Modulo: app.Modulo})
	if len(problems) > 0 {
		return fmt.Sprintf("no se pudo re-aplicar %s al módulo %s: %s", app.Blueprint, app.Modulo, strings.Join(problems, "; "))
	}
	if len(res.Pendientes) == 0 {
		return fmt.Sprintf("re-aplicado %s al módulo %s: la ficha quedó en borrador, revísala y apruébala (qdd modules approve %s)", app.Blueprint, app.Modulo, app.Modulo)
	}
	var preguntas []string
	for _, q := range res.Pendientes {
		preguntas = append(preguntas, fmt.Sprintf("%s (%s: %s)", q.ID, q.Pregunta, strings.Join(q.Valores(), "|")))
	}
	return fmt.Sprintf("el módulo %s necesita tu decisión: la versión nueva de %s pregunta %s. Responde con: qdd blueprint apply %s --modulo %s --respuesta <pregunta>=<valor>", app.Modulo, app.Blueprint, strings.Join(preguntas, "; "), app.Blueprint, app.Modulo)
}
