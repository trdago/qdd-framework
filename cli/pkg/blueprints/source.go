package blueprints

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FormatoLock es la versión del lockfile. El formato 2 (RFC 0004) significa
// que los blueprints viven en la carpeta central y el proyecto solo guarda el
// lockfile; sin formato, el proyecto es del esquema anterior (copias en
// .qdd/blueprints/), que Migrar convierte una sola vez.
const FormatoLock = 2

// LockEntry fija de dónde salió un blueprint importado (repositorio, subcarpeta,
// tag y commit) y su huella, con la que QDD lo reconstruye y lo verifica.
type LockEntry struct {
	ID         string `yaml:"id"`
	Version    string `yaml:"version"`
	Origen     string `yaml:"origen"`
	Subdir     string `yaml:"subdir,omitempty"`
	Ref        string `yaml:"ref,omitempty"`
	Commit     string `yaml:"commit,omitempty"`
	SHA256     string `yaml:"sha256"`
	AgregadoEl string `yaml:"agregado_el"`
}

// Lock es el contenido de .qdd/blueprints.lock.yaml.
type Lock struct {
	Formato    int         `yaml:"formato,omitempty"`
	Blueprints []LockEntry `yaml:"blueprints"`
}

// Entry busca la entrada de un blueprint por id.
func (l Lock) Entry(id string) (LockEntry, bool) {
	for _, e := range l.Blueprints {
		if e.ID == id {
			return e, true
		}
	}
	return LockEntry{}, false
}

// CatalogName es el índice opcional de un repositorio con varios blueprints.
const CatalogName = "catalogo.yaml"

// Catalogo es el índice de un repositorio de blueprints de la organización.
// Sin catálogo, cada carpeta de primer nivel con un blueprint.yaml es un
// blueprint con el nombre de la carpeta.
type Catalogo struct {
	Organizacion string              `yaml:"organizacion,omitempty"`
	Blueprints   map[string]string   `yaml:"blueprints,omitempty"` // id → carpeta
	Paquetes     map[string][]string `yaml:"paquetes,omitempty"`   // nombre → ids
}

// AddInput describe qué importar.
type AddInput struct {
	Root       string
	Origen     string   // repositorio git: URL (opcionalmente repo//subcarpeta) o carpeta local de un repositorio
	Ref        string   // tag de versión vX.Y.Z; vacío = la última versión publicada
	Blueprints []string // ids a importar de un repositorio con varios
	Paquete    string   // paquete del catálogo a importar
	Todos      bool     // importar todos los blueprints del repositorio
	Reemplazar bool     // permite que un id importado desde otra fuente pase a esta
}

// Add importa uno o varios blueprints de una versión publicada (tag vX.Y.Z)
// del repositorio: los valida TODOS antes de guardar nada, guarda cada uno en
// la carpeta central y registra origen, tag, commit y huella en el lockfile.
// El repositorio se clona una vez por máquina con las credenciales git del
// usuario, así que uno privado solo lo puede importar quien tenga acceso.
func Add(in AddInput) ([]LockEntry, []string) {
	origen := strings.TrimSpace(in.Origen)
	if origen == "" || strings.HasPrefix(origen, "-") {
		return nil, []string{"origen: indica una carpeta o un repositorio git (p. ej. git@github.com:org/qdd-blueprints.git)"}
	}
	base, subdir, entry, cleanup, err := fetch(origen, in.Ref)
	defer cleanup()
	if err != nil {
		return nil, []string{err.Error()}
	}
	carpetas, problems := seleccionar(base, subdir, in)
	if len(problems) > 0 {
		return nil, problems
	}
	bps, problems := parseSeleccion(base, carpetas)
	if len(problems) > 0 {
		return nil, problems
	}
	if problems := choques(in, bps, entry.Origen); len(problems) > 0 {
		return nil, problems
	}
	var entries []LockEntry
	for i, b := range bps {
		e := entry
		e.Subdir = carpetas[i]
		installed, err := install(in.Root, b, e)
		if err != nil {
			return entries, []string{err.Error()}
		}
		entries = append(entries, installed)
	}
	return entries, nil
}

// choques rechaza los blueprints cuyo id ya está importado desde otra fuente:
// un id pertenece a una sola fuente y solo se cambia de fuente a pedido
// (Reemplazar), nunca por accidente (FND-064).
func choques(in AddInput, bps []Blueprint, repo string) []string {
	if in.Reemplazar {
		return nil
	}
	lock, err := LoadLock(in.Root)
	if err != nil {
		return []string{err.Error()}
	}
	var problems []string
	for _, b := range bps {
		e, ok := lock.Entry(b.ID)
		if !ok || e.Origen == repo {
			continue
		}
		problems = append(problems, fmt.Sprintf("'%s' ya está importado desde %s: un id pertenece a una sola fuente. Para cambiarlo por el de %s usa --reemplazar y después vuelve a aplicarlo (qdd blueprint apply %s)", b.ID, e.Origen, repo, b.ID))
	}
	return problems
}

// seleccionar decide qué carpetas del origen importar.
func seleccionar(base, subdir string, in AddInput) ([]string, []string) {
	pidio := len(in.Blueprints) > 0 || in.Paquete != "" || in.Todos
	if subdir != "" && pidio {
		return nil, []string{"con 'repo//subcarpeta' se importa solo esa subcarpeta; para elegir varios, indica el repositorio sin '//'"}
	}
	if subdir != "" {
		return []string{subdir}, nil
	}
	if _, err := os.Stat(filepath.Join(base, ManifestName)); err == nil && !pidio {
		return []string{""}, nil
	}
	cat, disponibles, err := catalogo(base)
	if err != nil {
		return nil, []string{err.Error()}
	}
	ids, problems := idsPedidos(in, cat, disponibles)
	if len(problems) > 0 {
		return nil, problems
	}
	var carpetas []string
	for _, id := range ids {
		carpetas = append(carpetas, disponibles[id])
	}
	return carpetas, nil
}

func idsPedidos(in AddInput, cat Catalogo, disponibles map[string]string) ([]string, []string) {
	pedidos := append([]string{}, in.Blueprints...)
	if in.Paquete != "" {
		paquete, ok := cat.Paquetes[in.Paquete]
		if !ok {
			return nil, []string{fmt.Sprintf("el repositorio no tiene el paquete '%s'. %s", in.Paquete, oferta(cat, disponibles))}
		}
		pedidos = append(pedidos, paquete...)
	}
	if in.Todos {
		pedidos = append(pedidos, sortedKeys(disponibles)...)
	}
	pedidos = dedupe(pedidos)
	if len(pedidos) == 0 {
		return nil, []string{"el repositorio tiene varios blueprints: elige cuáles (qdd blueprint add <repo> auth auditoria), un --paquete o --todos. " + oferta(cat, disponibles)}
	}
	var problems []string
	for _, id := range pedidos {
		if _, ok := disponibles[id]; !ok {
			problems = append(problems, fmt.Sprintf("el repositorio no tiene el blueprint '%s'. %s", id, oferta(cat, disponibles)))
		}
	}
	return pedidos, problems
}

func oferta(cat Catalogo, disponibles map[string]string) string {
	s := "Blueprints: " + firstNonEmpty(strings.Join(sortedKeys(disponibles), ", "), "ninguno")
	if len(cat.Paquetes) == 0 {
		return s + "."
	}
	var paquetes []string
	for _, nombre := range sortedKeys(cat.Paquetes) {
		paquetes = append(paquetes, fmt.Sprintf("%s (%s)", nombre, strings.Join(cat.Paquetes[nombre], ", ")))
	}
	return s + ". Paquetes: " + strings.Join(paquetes, "; ") + "."
}

// catalogo lee catalogo.yaml (si existe) y devuelve los blueprints
// disponibles: id → carpeta relativa a la raíz del repositorio.
func catalogo(base string) (Catalogo, map[string]string, error) {
	var cat Catalogo
	data, err := os.ReadFile(filepath.Join(base, CatalogName))
	if err == nil {
		if err := yaml.Unmarshal(data, &cat); err != nil {
			return Catalogo{}, nil, fmt.Errorf("%s: YAML inválido: %w", CatalogName, err)
		}
	}
	disponibles := descubrir(base)
	for id, carpeta := range cat.Blueprints {
		if !isInside(carpeta) {
			return Catalogo{}, nil, fmt.Errorf("%s: la carpeta '%s' de '%s' debe ser relativa al repositorio", CatalogName, carpeta, id)
		}
		disponibles[id] = filepath.ToSlash(filepath.Clean(carpeta))
	}
	for nombre, ids := range cat.Paquetes {
		for _, id := range ids {
			if _, ok := disponibles[id]; ok {
				continue
			}
			return Catalogo{}, nil, fmt.Errorf("%s: el paquete '%s' incluye '%s', que no existe en el repositorio", CatalogName, nombre, id)
		}
	}
	return cat, disponibles, nil
}

func descubrir(base string) map[string]string {
	out := map[string]string{}
	entries, _ := os.ReadDir(base)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if _, err := os.Stat(filepath.Join(base, e.Name(), ManifestName)); err != nil {
			continue
		}
		out[e.Name()] = e.Name()
	}
	return out
}

// parseSeleccion lee y valida todos los blueprints elegidos; si alguno falla,
// no se importa ninguno.
func parseSeleccion(base string, carpetas []string) ([]Blueprint, []string) {
	var bps []Blueprint
	var problems []string
	ids := map[string]string{}
	for _, c := range carpetas {
		b, err := ParseDir(filepath.Join(base, filepath.FromSlash(c)))
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		label := firstNonEmpty(c, b.ID)
		if invalid := b.Validate(); len(invalid) > 0 {
			problems = append(problems, prefix(fmt.Sprintf("blueprint '%s' inválido: ", label), invalid)...)
			continue
		}
		if previa, dup := ids[b.ID]; dup {
			problems = append(problems, fmt.Sprintf("las carpetas '%s' y '%s' declaran el mismo id '%s'", previa, c, b.ID))
			continue
		}
		ids[b.ID] = c
		bps = append(bps, b)
	}
	return bps, problems
}

func install(root string, b Blueprint, entry LockEntry) (LockEntry, error) {
	_, sum, err := storePut(b.Dir)
	if err != nil {
		return LockEntry{}, err
	}
	entry.ID = b.ID
	entry.Version = b.Version
	entry.SHA256 = sum
	entry.AgregadoEl = time.Now().Format("2006-01-02")
	return entry, saveLockEntry(root, entry)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// fetch deja una versión publicada del origen en una carpeta temporal y
// devuelve su raíz, la subcarpeta pedida y los datos del lockfile. Devuelve
// siempre una función de limpieza, aunque falle.
func fetch(origen, ref string) (string, string, LockEntry, func(), error) {
	noop := func() {}
	if ref != "" && !tagPattern.MatchString(ref) {
		return "", "", LockEntry{}, noop, fmt.Errorf("--ref debe ser una versión publicada (tag vX.Y.Z); '%s' no lo es. QDD no importa ramas ni commits sueltos", ref)
	}
	repo, subdir, err := resolverOrigen(origen)
	if err != nil {
		return "", "", LockEntry{}, noop, err
	}
	mirror, err := ensureMirror(repo, true)
	if err != nil {
		return "", "", LockEntry{}, noop, err
	}
	tags, err := versionTags(mirror)
	if err != nil {
		return "", "", LockEntry{}, noop, err
	}
	if len(tags) == 0 {
		return "", "", LockEntry{}, noop, fmt.Errorf("'%s' no tiene versiones publicadas (tags vX.Y.Z): publícalas con 'qdd blueprint release'", repo)
	}
	ref = firstNonEmpty(ref, tags[0])
	commit, err := git(mirror, "rev-parse", "--verify", "--quiet", "refs/tags/"+ref+"^{commit}")
	if err != nil {
		return "", "", LockEntry{}, noop, fmt.Errorf("'%s' no tiene la versión '%s'. Versiones: %s", repo, ref, strings.Join(tags, ", "))
	}
	tmp, err := os.MkdirTemp("", "qdd-blueprint-")
	if err != nil {
		return "", "", LockEntry{}, noop, err
	}
	cleanup := func() { os.RemoveAll(tmp) }
	if err := extraer(mirror, commit, tmp); err != nil {
		return "", "", LockEntry{}, cleanup, err
	}
	return tmp, subdir, LockEntry{Origen: repo, Ref: ref, Commit: commit}, cleanup, nil
}

// resolverOrigen separa el repositorio de la subcarpeta. Una carpeta local
// vale solo si está dentro de un repositorio git: QDD importa versiones
// publicadas, nunca archivos sueltos.
func resolverOrigen(origen string) (string, string, error) {
	origen = strings.TrimSpace(origen)
	if strings.HasPrefix(origen, "~/") {
		home, _ := os.UserHomeDir()
		origen = filepath.Join(home, origen[2:])
	}
	if info, err := os.Stat(origen); err == nil && info.IsDir() {
		return repoLocal(origen)
	}
	repo, subdir := splitSource(origen)
	if subdir != "" && !isInside(subdir) {
		return "", "", fmt.Errorf("origen: la subcarpeta '%s' debe ser relativa al repositorio", subdir)
	}
	return repo, subdir, nil
}

func repoLocal(dir string) (string, string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	top, err := git(abs, "rev-parse", "--show-toplevel")
	if err != nil || top == "" {
		return repoBare(abs, dir)
	}
	realAbs, _ := filepath.EvalSymlinks(abs)
	realTop, _ := filepath.EvalSymlinks(top)
	rel, err := filepath.Rel(realTop, realAbs)
	if err != nil {
		return "", "", err
	}
	if rel == "." {
		return abs, "", nil
	}
	// La raíz se expresa como la escribió el usuario (sin resolver enlaces),
	// para que el registro y el lockfile la nombren igual.
	repo := abs
	for range strings.Split(filepath.ToSlash(rel), "/") {
		repo = filepath.Dir(repo)
	}
	return repo, filepath.ToSlash(rel), nil
}

// repoBare acepta la carpeta de un repositorio bare (p. ej. uno compartido en
// un disco de red): no tiene árbol de trabajo, así que es la raíz y no admite
// subcarpeta (FND-067).
func repoBare(abs, dir string) (string, string, error) {
	noEsRepo := fmt.Errorf("'%s' no es un repositorio git: QDD solo importa blueprints versionados (git init, commit y 'qdd blueprint release')", dir)
	out, err := git(abs, "rev-parse", "--is-bare-repository", "--absolute-git-dir")
	bare, gitDir, _ := strings.Cut(out, "\n")
	if err != nil || bare != "true" {
		return "", "", noEsRepo
	}
	realAbs, _ := filepath.EvalSymlinks(abs)
	realGitDir, _ := filepath.EvalSymlinks(strings.TrimSpace(gitDir))
	if realAbs != realGitDir {
		return "", "", fmt.Errorf("'%s' está dentro de un repositorio bare: indica su carpeta raíz (%s)", dir, gitDir)
	}
	return abs, "", nil
}

// splitSource separa "repo//subcarpeta" (convención de Terraform); el "//"
// de un esquema como https:// no cuenta.
func splitSource(origen string) (string, string) {
	start := 0
	if i := strings.Index(origen, "://"); i >= 0 {
		start = i + 3
	}
	i := strings.Index(origen[start:], "//")
	if i < 0 {
		return origen, ""
	}
	return origen[:start+i], strings.Trim(origen[start+i+2:], "/")
}

// blueprintFiles lista los archivos regulares de un blueprint, ordenados y
// relativos a su carpeta. Omite .git y los enlaces simbólicos: un blueprint
// nunca trae archivos de fuera de su carpeta.
func blueprintFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, err
}

// HashDir calcula la huella de un blueprint: sha256 de cada ruta y contenido.
func HashDir(dir string) (string, error) {
	files, err := blueprintFiles(dir)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, rel := range files {
		f, err := os.Open(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		io.WriteString(h, rel+"\x00")
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyTree(src, dest string, mode os.FileMode) error {
	files, err := blueprintFiles(src)
	if err != nil {
		return err
	}
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, mode); err != nil {
			return err
		}
	}
	return nil
}

// LoadLock lee el lockfile; sin lockfile devuelve uno vacío.
func LoadLock(root string) (Lock, error) {
	data, err := os.ReadFile(filepath.Join(root, LockFile))
	if os.IsNotExist(err) {
		return Lock{}, nil
	}
	if err != nil {
		return Lock{}, err
	}
	var l Lock
	if err := yaml.Unmarshal(data, &l); err != nil {
		return Lock{}, fmt.Errorf("%s: YAML inválido: %w", LockFile, err)
	}
	return l, nil
}

func saveLockEntry(root string, entry LockEntry) error {
	l, err := LoadLock(root)
	if err != nil {
		return err
	}
	var out []LockEntry
	for _, e := range l.Blueprints {
		if e.ID == entry.ID {
			continue
		}
		out = append(out, e)
	}
	l.Blueprints = append(out, entry)
	return saveLock(root, l)
}

// saveLock escribe el lockfile, siempre en el formato vigente.
func saveLock(root string, l Lock) error {
	l.Formato = FormatoLock
	sort.Slice(l.Blueprints, func(i, j int) bool { return l.Blueprints[i].ID < l.Blueprints[j].ID })
	data, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	header := "# Blueprints importados con 'qdd blueprint add'. Lo administra QDD: no se edita a mano.\n# El contenido vive en la carpeta central de QDD (RFC 0004), no en el proyecto.\n"
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(LockFile)), 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, LockFile), append([]byte(header), data...), 0644)
}

// Find lee un blueprint del lockfile desde la carpeta central.
func Find(root, id string) (Blueprint, error) {
	if !idPattern.MatchString(id) {
		return Blueprint{}, fmt.Errorf("'%s' no es un id de blueprint válido", id)
	}
	lock, err := LoadLock(root)
	if err != nil {
		return Blueprint{}, err
	}
	e, ok := lock.Entry(id)
	if !ok {
		return Blueprint{}, fmt.Errorf("el blueprint '%s' no está importado; usa 'qdd blueprint add %s'", id, id)
	}
	return load(e)
}

// List lee los blueprints del lockfile desde la carpeta central.
func List(root string) ([]Blueprint, []error) {
	lock, err := LoadLock(root)
	if err != nil {
		return nil, []error{err}
	}
	var out []Blueprint
	var errs []error
	for _, e := range lock.Blueprints {
		b, err := load(e)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.ID, err))
			continue
		}
		out = append(out, b)
	}
	return out, errs
}
