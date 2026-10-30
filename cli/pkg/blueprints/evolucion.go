package blueprints

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// La evolución de un blueprint la dicta QDD (RFC 0004):
//
//	qdd blueprint contribute <id>   prepara la rama aporte/<id> en la carpeta central
//	qdd blueprint propose <id> -m   valida, fija la versión semver, commitea, push y PR
//	qdd blueprint release           etiqueta el repositorio (vX.Y.Z) tras el merge
//	qdd blueprint sync              trae lo publicado y dice qué actualizar
//
// Nadie elige la versión a mano: QDD la calcula según lo que cambió.

// Tipos de cambio, de menor a mayor.
const (
	CambioPatch = "patch"
	CambioMinor = "minor"
	CambioMajor = "major"
	CambioNuevo = "nuevo"
)

var (
	versionLine = regexp.MustCompile(`(?m)^version:.*$`)
	rangoCambio = map[string]int{"": 0, CambioPatch: 1, CambioMinor: 2, CambioMajor: 3}
)

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// compararVersion devuelve -1, 0 o 1 (acepta "v1.2.3" y "1.2.3").
func compararVersion(a, b string) int {
	va, _ := parseVersion(a)
	vb, _ := parseVersion(b)
	for i := range va {
		if va[i] > vb[i] {
			return 1
		}
		if va[i] < vb[i] {
			return -1
		}
	}
	return 0
}

// subirVersion aplica el cambio a la versión (sin la "v").
func subirVersion(v, cambio string) string {
	n, _ := parseVersion(v)
	switch cambio {
	case CambioMajor:
		return fmt.Sprintf("%d.0.0", n[0]+1)
	case CambioMinor:
		return fmt.Sprintf("%d.%d.0", n[0], n[1]+1)
	}
	return fmt.Sprintf("%d.%d.%d", n[0], n[1], n[2]+1)
}

func mayorCambio(a, b string) string {
	if rangoCambio[b] > rangoCambio[a] {
		return b
	}
	return a
}

// versionTags lista los tags vX.Y.Z del repositorio, del más nuevo al más viejo.
func versionTags(repoDir string) ([]string, error) {
	out, err := git(repoDir, "tag", "--list", "v*")
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, t := range strings.Fields(out) {
		if tagPattern.MatchString(t) {
			tags = append(tags, t)
		}
	}
	sort.Slice(tags, func(i, j int) bool { return compararVersion(tags[i], tags[j]) > 0 })
	return tags, nil
}

// blueprintEn lee el blueprint de subdir en ref; false si no existe ahí.
func blueprintEn(repoDir, ref, subdir string) (Blueprint, bool) {
	data, err := gitRaw(repoDir, "show", ref+":"+path.Join(subdir, ManifestName))
	if err != nil {
		return Blueprint{}, false
	}
	b, err := Parse(data)
	return b, err == nil
}

func versionEn(repoDir, ref, subdir string) string {
	b, _ := blueprintEn(repoDir, ref, subdir)
	return b.Version
}

// Clasificar decide qué tan grande es el cambio entre dos versiones de un
// blueprint, con sus razones. Quitar o cambiar lo que un proyecto ya acató
// (invariantes, decisiones, preguntas, opciones, condiciones, vocabulario,
// escenarios) es major; agregar es minor; el resto (redacción, referencias)
// es patch. Vacío si los manifiestos son equivalentes.
func Clasificar(base, nuevo Blueprint) (string, []string) {
	antes, despues := aspectos(base), aspectos(nuevo)
	cambio := ""
	var razones []string
	for _, k := range sortedKeys(antes) {
		v, ok := despues[k]
		if !ok {
			cambio = mayorCambio(cambio, CambioMajor)
			razones = append(razones, "se quitó "+k)
			continue
		}
		if v == antes[k] {
			continue
		}
		tipo := CambioMajor
		if strings.HasPrefix(k, "pregunta ") {
			tipo = CambioPatch
		}
		cambio = mayorCambio(cambio, tipo)
		razones = append(razones, "cambió "+k)
	}
	for _, k := range sortedKeys(despues) {
		if _, ok := antes[k]; ok {
			continue
		}
		cambio = mayorCambio(cambio, CambioMinor)
		razones = append(razones, "se agregó "+k)
	}
	return cambio, razones
}

// aspectos aplana lo que un proyecto acata de un blueprint: clave → contenido.
func aspectos(b Blueprint) map[string]string {
	m := map[string]string{"id": b.ID}
	for _, t := range b.Vocabulario {
		m["término '"+t.Termino+"'"] = t.Definicion
	}
	agregarFicha(m, "", b.Ficha.ObjetivoTecnologico.Decisiones, b.Ficha.ObjetivoTecnologico.Invariantes)
	agregarEscenarios(m, b.Conformidad)
	for _, p := range b.Preguntas {
		m["pregunta '"+p.ID+"'"] = p.Pregunta
		m["condición de '"+p.ID+"'"] = fmt.Sprint(p.Si)
		for _, o := range p.Opciones {
			m["opción '"+p.ID+"="+o.Valor+"'"] = strings.Join(o.DependeDe, "\n")
			agregarFicha(m, " (si "+p.ID+"="+o.Valor+")", o.Decisiones, o.Invariantes)
			agregarEscenarios(m, o.Conformidad)
		}
	}
	return m
}

func agregarFicha(m map[string]string, sufijo string, decisiones, invariantes []string) {
	for _, d := range decisiones {
		m["la decisión \""+d+"\""+sufijo] = ""
	}
	for _, i := range invariantes {
		m["la invariante \""+i+"\""+sufijo] = ""
	}
}

func agregarEscenarios(m map[string]string, escenarios []Escenario) {
	for _, e := range escenarios {
		m["escenario "+e.ID] = strings.Join([]string{e.Titulo, e.Dado, e.Cuando, e.Entonces}, "\n")
	}
}

// Novedad es un blueprint con una versión publicada más nueva que la del proyecto.
type Novedad struct {
	ID         string
	Actual     string
	Disponible string
}

// Sincronizacion describe lo que trajo Sincronizar.
type Sincronizacion struct {
	Origen    string
	Ref       string // última versión publicada del repositorio
	Novedades []Novedad
	Problema  string // por qué no se pudo sincronizar esta fuente
}

// Sincronizar actualiza el espejo central de cada fuente del proyecto (el
// repositorio de su organización y las demás de su lockfile), reconstruye en
// la carpeta central lo que el lockfile fija y dice qué blueprints tienen una
// versión publicada más nueva. Una fuente inaccesible no oculta las demás: su
// resultado lleva el Problema (FND-066).
func Sincronizar(root string) ([]Sincronizacion, error) {
	lock, err := LoadLock(root)
	if err != nil {
		return nil, err
	}
	fuentes, err := fuentesDe(root, lock)
	if err != nil {
		return nil, err
	}
	var res []Sincronizacion
	for _, repo := range fuentes {
		res = append(res, sincronizarFuente(repo, lock))
	}
	if _, errs := List(root); len(errs) > 0 {
		return res, errs[0]
	}
	return res, nil
}

// fuentesDe lista los repositorios del proyecto: primero el registrado, luego
// los del lockfile, sin repetir.
func fuentesDe(root string, lock Lock) ([]string, error) {
	reg, err := LoadRegistro(root)
	if err != nil {
		return nil, err
	}
	var fuentes []string
	if reg.Origen != "" {
		repo, _, err := resolverOrigen(reg.Origen)
		if err != nil {
			return nil, err
		}
		fuentes = append(fuentes, repo)
	}
	for _, o := range origenesDe(lock) {
		if !inList(fuentes, o) {
			fuentes = append(fuentes, o)
		}
	}
	if len(fuentes) == 0 {
		return nil, errors.New("el proyecto no tiene repositorio de blueprints de la organización: impórtalo una vez con 'qdd blueprint add <repositorio> <id>'")
	}
	return fuentes, nil
}

func sincronizarFuente(repo string, lock Lock) Sincronizacion {
	s := Sincronizacion{Origen: repo}
	mirror, err := ensureMirror(repo, true)
	if err != nil {
		s.Problema = err.Error()
		return s
	}
	tags, err := versionTags(mirror)
	if err != nil {
		s.Problema = err.Error()
		return s
	}
	if len(tags) == 0 {
		s.Problema = fmt.Sprintf("'%s' no tiene versiones publicadas (tags vX.Y.Z): publícalas con 'qdd blueprint release'", repo)
		return s
	}
	s.Ref = tags[0]
	for _, e := range lock.Blueprints {
		if e.Origen != repo {
			continue
		}
		if v := versionEn(mirror, tags[0], e.Subdir); v != "" && compararVersion(v, e.Version) > 0 {
			s.Novedades = append(s.Novedades, Novedad{ID: e.ID, Actual: e.Version, Disponible: v})
		}
	}
	return s
}

// Aporte es el espacio de trabajo central donde se edita un blueprint.
type Aporte struct {
	Repo    string
	Espacio string // clon editable en la carpeta central
	Carpeta string // carpeta del blueprint dentro del clon
	Rama    string
}

// Aportar prepara en la carpeta central (nunca en el proyecto) un clon
// editable de la fuente del blueprint (ver origenDe), en la rama aporte/<id> sobre
// lo último publicado. Si el clon tiene cambios sin commitear, no lo toca.
func Aportar(root, id, origen string) (Aporte, error) {
	repo, subdir, err := origenDe(root, id, origen)
	if err != nil {
		return Aporte{}, err
	}
	espacio, err := espacioDeTrabajo(repo)
	if err != nil {
		return Aporte{}, err
	}
	rama := "aporte/" + id
	sucio, err := git(espacio, "status", "--porcelain")
	if err != nil {
		return Aporte{}, err
	}
	if sucio == "" {
		if err := cambiarARama(espacio, rama); err != nil {
			return Aporte{}, err
		}
	}
	actual, err := git(espacio, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return Aporte{}, err
	}
	return Aporte{Repo: repo, Espacio: espacio, Carpeta: filepath.Join(espacio, filepath.FromSlash(subdir)), Rama: actual}, nil
}

// origenDe devuelve el repositorio y la carpeta de un blueprint: la del
// lockfile si está importado, o <id>/ en la fuente elegida (origen) o, sin
// ella, en el repositorio de la organización si es un blueprint nuevo. Un
// blueprint importado se aporta siempre a su propia fuente.
func origenDe(root, id, origen string) (string, string, error) {
	if !idPattern.MatchString(id) {
		return "", "", fmt.Errorf("'%s' no es un id de blueprint válido", id)
	}
	lock, err := LoadLock(root)
	if err != nil {
		return "", "", err
	}
	elegido, err := fuenteElegida(root, origen)
	if err != nil {
		return "", "", err
	}
	e, ok := lock.Entry(id)
	if ok && origen != "" && elegido != e.Origen {
		return "", "", fmt.Errorf("'%s' está importado desde %s: su aporte va a esa fuente (no indiques --origen)", id, e.Origen)
	}
	if ok {
		return e.Origen, e.Subdir, nil
	}
	return elegido, id, nil
}

// fuenteElegida es el repositorio de origen, o el registrado si no se indica.
func fuenteElegida(root, origen string) (string, error) {
	if strings.TrimSpace(origen) == "" {
		reg, err := LoadRegistro(root)
		if err != nil {
			return "", err
		}
		origen = reg.Origen
	}
	if origen == "" {
		return "", errors.New("el proyecto no tiene repositorio de blueprints de la organización: indica la fuente con --origen o impórtala una vez con 'qdd blueprint add <repositorio> <id>'")
	}
	repo, _, err := resolverOrigen(origen)
	return repo, err
}

func espacioPath(repo string) (string, error) {
	dir, err := centralDir("aportes")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, repoKey(repo)), nil
}

func espacioDeTrabajo(repo string) (string, error) {
	espacio, err := espacioPath(repo)
	if err != nil {
		return "", err
	}
	if !exists(espacio) {
		return espacio, clonarAtomico(repo, espacio)
	}
	if _, err := git(espacio, "fetch", "--quiet", "--prune", "--tags", "origin"); err != nil {
		return "", sinAcceso("actualizar", repo, err)
	}
	return espacio, nil
}

// cambiarARama deja el clon en la rama del aporte. Una rama ya integrada en
// lo publicado se reinicia sobre lo último; una con trabajo propio se conserva.
func cambiarARama(espacio, rama string) error {
	if _, err := git(espacio, "rev-parse", "--verify", "--quiet", "refs/heads/"+rama); err != nil {
		_, err := git(espacio, "checkout", "--quiet", "-b", rama, "origin/HEAD")
		return err
	}
	if _, err := git(espacio, "merge-base", "--is-ancestor", rama, "origin/HEAD"); err == nil {
		_, err := git(espacio, "checkout", "--quiet", "-B", rama, "origin/HEAD")
		return err
	}
	_, err := git(espacio, "checkout", "--quiet", rama)
	return err
}

// Propuesta describe lo que hizo Proponer.
type Propuesta struct {
	Rama    string
	Version string
	Cambio  string
	Razones []string
	Commit  string
	PR      string // URL del pull request; vacío si no se pudo abrir
	Aviso   string // por qué no se abrió el PR y cómo abrirlo
	Via     string // cómo se publicó la rama, si no fue por el origin (FND-063)
}

// abrirPR abre el pull request de la rama con la CLI de GitHub.
var abrirPR = func(espacio, rama, titulo, cuerpo string) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", errors.New("no está instalada la CLI de GitHub (gh)")
	}
	cmd := exec.Command("gh", "pr", "create", "--head", rama, "--title", titulo, "--body", cuerpo)
	cmd.Dir = espacio
	out, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(out), "already exists") {
		return "", fmt.Errorf("gh pr create: %s", strings.TrimSpace(string(out)))
	}
	view := exec.Command("gh", "pr", "view", rama, "--json", "url", "--jq", ".url")
	view.Dir = espacio
	url, err := view.Output()
	return strings.TrimSpace(string(url)), err
}

// Proponer publica el aporte como pull request al repositorio de la
// organización. QDD valida el blueprint, calcula el cambio respecto de lo
// publicado y fija la versión (nadie la elige a mano), commitea, hace push de
// la rama aporte/<id> y abre el PR.
func Proponer(root, id, mensaje, origen string) (Propuesta, error) {
	if strings.TrimSpace(mensaje) == "" {
		return Propuesta{}, errors.New("describe el cambio con -m: queda en el commit y en el pull request")
	}
	repo, subdir, err := origenDe(root, id, origen)
	if err != nil {
		return Propuesta{}, err
	}
	espacio, rama, err := espacioEnRama(repo, id)
	if err != nil {
		return Propuesta{}, err
	}
	carpeta := filepath.Join(espacio, filepath.FromSlash(subdir))
	p, err := versionar(espacio, carpeta, subdir, id)
	if err != nil {
		return Propuesta{}, err
	}
	p.Rama = rama
	titulo := fmt.Sprintf("feat(%s): %s (v%s)", id, strings.TrimSpace(mensaje), p.Version)
	if _, err := git(espacio, "commit", "--quiet", "-m", titulo); err != nil {
		return Propuesta{}, err
	}
	if p.Commit, err = git(espacio, "rev-parse", "HEAD"); err != nil {
		return Propuesta{}, err
	}
	if p.Via, err = empujar(espacio, repo, rama, "-u"); err != nil {
		return p, fmt.Errorf("no se pudo publicar la rama %s en %s: %v", rama, repo, err)
	}
	p.PR, err = abrirPR(espacio, rama, titulo, cuerpoPR(id, p))
	if err != nil {
		p.Aviso = fmt.Sprintf("la rama %s quedó publicada, pero no se abrió el pull request (%v): ábrelo en %s", rama, err, repo)
	}
	return p, nil
}

func espacioEnRama(repo, id string) (string, string, error) {
	espacio, err := espacioPath(repo)
	if err != nil {
		return "", "", err
	}
	rama := "aporte/" + id
	if !exists(espacio) {
		return "", "", fmt.Errorf("no hay un aporte en curso: primero 'qdd blueprint contribute %s'", id)
	}
	actual, err := git(espacio, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || actual != rama {
		return "", "", fmt.Errorf("el espacio de trabajo está en '%s', no en '%s': ejecuta 'qdd blueprint contribute %s'", actual, rama, id)
	}
	if _, err := git(espacio, "fetch", "--quiet", "--prune", "--tags", "origin"); err != nil {
		return "", "", fmt.Errorf("no se pudo actualizar '%s': %v", repo, err)
	}
	return espacio, rama, nil
}

// versionar valida el blueprint editado, calcula su cambio respecto de lo
// publicado (origin/HEAD), fija la versión en su manifiesto y lo deja listo
// para commitear.
func versionar(espacio, carpeta, subdir, id string) (Propuesta, error) {
	nuevo, err := ParseDir(carpeta)
	if err != nil {
		return Propuesta{}, err
	}
	if nuevo.ID != id {
		return Propuesta{}, fmt.Errorf("la carpeta %s declara el id '%s', no '%s'", subdir, nuevo.ID, id)
	}
	if _, err := git(espacio, "add", "--all", "--", subdir); err != nil {
		return Propuesta{}, err
	}
	cambiados, err := git(espacio, "diff", "--cached", "--name-only", "origin/HEAD", "--", subdir)
	if err != nil {
		return Propuesta{}, err
	}
	if cambiados == "" {
		return Propuesta{}, fmt.Errorf("'%s' no tiene cambios respecto de lo publicado", id)
	}
	p := Propuesta{Cambio: CambioNuevo, Version: "1.0.0", Razones: []string{"blueprint nuevo"}}
	if base, ok := blueprintEn(espacio, "origin/HEAD", subdir); ok {
		p.Cambio, p.Razones = Clasificar(base, nuevo)
		p.Cambio = firstNonEmpty(p.Cambio, CambioPatch)
		p.Version = subirVersion(base.Version, p.Cambio)
	}
	if err := fijarVersion(filepath.Join(carpeta, ManifestName), p.Version); err != nil {
		return Propuesta{}, err
	}
	nuevo.Version = p.Version
	if problems := nuevo.Validate(); len(problems) > 0 {
		return Propuesta{}, fmt.Errorf("el blueprint '%s' no es válido: %s", id, strings.Join(problems, "; "))
	}
	_, err = git(espacio, "add", "--all", "--", subdir)
	return p, err
}

func fijarVersion(manifest, version string) error {
	data, err := os.ReadFile(manifest)
	if err != nil {
		return err
	}
	if !versionLine.Match(data) {
		return fmt.Errorf("%s no declara 'version:'", manifest)
	}
	return os.WriteFile(manifest, versionLine.ReplaceAll(data, []byte("version: "+version)), 0644)
}

func cuerpoPR(id string, p Propuesta) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Aporte al blueprint `%s` preparado con `qdd blueprint propose`.\n\n", id)
	fmt.Fprintf(&b, "**Cambio %s → v%s** (lo calculó QDD):\n", p.Cambio, p.Version)
	for _, r := range p.Razones {
		fmt.Fprintf(&b, "- %s\n", r)
	}
	b.WriteString("\nTras el merge, un mantenedor publica la versión con `qdd blueprint release`; cada proyecto la recibe con `qdd blueprint sync`.\n")
	return b.String()
}

// Publicacion describe lo que hizo Publicar.
type Publicacion struct {
	Tag     string
	Cambio  string
	Cambios []Novedad
	Via     string // cómo se publicó, si no fue por el origin (FND-063)
}

// Publicar etiqueta lo integrado en la fuente indicada (origen) o, sin ella,
// en el repositorio de la organización. QDD
// valida todos los blueprints, exige que cada uno que cambió haya subido su
// versión como corresponde (es decir, que pasó por 'qdd blueprint propose') y
// calcula el tag: el mayor cambio entre los blueprints decide si sube el
// major, el minor o el patch del repositorio.
func Publicar(root, origen string) (Publicacion, error) {
	repo, err := fuenteElegida(root, origen)
	if err != nil {
		return Publicacion{}, err
	}
	if _, err := ensureMirror(repo, true); err != nil {
		return Publicacion{}, err
	}
	espacio, err := espacioDeTrabajo(repo)
	if err != nil {
		return Publicacion{}, err
	}
	tags, err := versionTags(espacio)
	if err != nil {
		return Publicacion{}, err
	}
	pub, err := cambiosDesde(espacio, tags)
	if err != nil {
		return Publicacion{}, err
	}
	if _, err := git(espacio, "tag", "--annotate", pub.Tag, "origin/HEAD", "-m", "Blueprints "+pub.Tag+" (qdd blueprint release)"); err != nil {
		return Publicacion{}, err
	}
	if pub.Via, err = empujar(espacio, repo, "refs/tags/"+pub.Tag); err != nil {
		git(espacio, "tag", "--delete", pub.Tag)
		return Publicacion{}, fmt.Errorf("no se pudo publicar %s en %s: %v", pub.Tag, repo, err)
	}
	_, err = ensureMirror(repo, true)
	return pub, err
}

func cambiosDesde(espacio string, tags []string) (Publicacion, error) {
	head, err := leerBlueprints(espacio, "origin/HEAD")
	if err != nil {
		return Publicacion{}, err
	}
	if len(tags) == 0 {
		return Publicacion{Tag: "v1.0.0", Cambio: CambioNuevo}, nil
	}
	previos, err := leerBlueprints(espacio, tags[0])
	if err != nil {
		return Publicacion{}, err
	}
	pub := Publicacion{}
	for _, id := range sortedKeys(previos) {
		if _, ok := head[id]; !ok {
			pub.Cambio = mayorCambio(pub.Cambio, CambioMajor)
			pub.Cambios = append(pub.Cambios, Novedad{ID: id, Actual: previos[id].Version, Disponible: "eliminado"})
		}
	}
	for _, id := range sortedKeys(head) {
		cambio, err := cambioPublicable(espacio, tags[0], previos, head[id])
		if err != nil {
			return Publicacion{}, err
		}
		if cambio == "" {
			continue
		}
		pub.Cambio = mayorCambio(pub.Cambio, cambio)
		pub.Cambios = append(pub.Cambios, Novedad{ID: id, Actual: previos[id].Version, Disponible: head[id].Version})
	}
	if pub.Cambio == "" {
		return Publicacion{}, fmt.Errorf("no hay cambios en los blueprints desde %s", tags[0])
	}
	pub.Tag = "v" + subirVersion(tags[0], pub.Cambio)
	return pub, nil
}

// cambioPublicable devuelve el cambio de un blueprint desde el último tag y
// rechaza los que no subieron su versión como QDD exige.
func cambioPublicable(espacio, tag string, previos map[string]Blueprint, b Blueprint) (string, error) {
	base, ok := previos[b.ID]
	if !ok {
		return CambioMinor, nil
	}
	if _, err := git(espacio, "diff", "--quiet", tag, "origin/HEAD", "--", filepath.ToSlash(b.Dir)); err == nil {
		return "", nil
	}
	cambio, _ := Clasificar(base, b)
	cambio = firstNonEmpty(cambio, CambioPatch)
	requerida := subirVersion(base.Version, cambio)
	if compararVersion(b.Version, requerida) < 0 {
		return "", fmt.Errorf("'%s' cambió (%s) pero su versión %s no sube a %s: sus cambios no pasaron por 'qdd blueprint propose'", b.ID, cambio, b.Version, requerida)
	}
	return cambio, nil
}

// leerBlueprints valida y lee todos los blueprints del repositorio en ref.
// Blueprint.Dir queda como la carpeta relativa a la raíz del repositorio.
func leerBlueprints(espacio, ref string) (map[string]Blueprint, error) {
	tmp, err := os.MkdirTemp("", "qdd-release-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	if err := extraer(espacio, ref, tmp); err != nil {
		return nil, err
	}
	_, disponibles, err := catalogo(tmp)
	if err != nil {
		return nil, err
	}
	out := map[string]Blueprint{}
	for _, id := range sortedKeys(disponibles) {
		b, err := ParseDir(filepath.Join(tmp, filepath.FromSlash(disponibles[id])))
		if err != nil {
			return nil, err
		}
		if problems := b.Validate(); len(problems) > 0 {
			return nil, fmt.Errorf("%s: el blueprint '%s' no es válido: %s", ref, id, strings.Join(problems, "; "))
		}
		b.Dir = disponibles[id]
		out[b.ID] = b
	}
	return out, nil
}
